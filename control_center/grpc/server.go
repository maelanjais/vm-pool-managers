package grpc

import (
	"context"
	"control_center/config"
	"control_center/frontcontrolpb"
	"control_center/internal/attribvm"
	"control_center/internal/auth"
	"control_center/internal/configpool"
	"control_center/internal/gatherdata"
	"control_center/internal/guacamole"
	"control_center/internal/monitoring"
	oidcmw "control_center/internal/oidc"
	"control_center/internal/pool"
	"control_center/internal/user"
	"control_center/pb"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	grpcweb "github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"
)

type GatherDataServer struct {
	frontcontrolpb.UnimplementedGatherDataServiceServer
	DB *gorm.DB
}

type ConfigServer struct {
	frontcontrolpb.UnimplementedConfigServiceServer
	DB *gorm.DB
}

type PoolServer struct {
	frontcontrolpb.UnimplementedPoolServiceServer
	DB *gorm.DB
}

type UserServer struct {
	frontcontrolpb.UnimplementedUserServiceServer
	DB *gorm.DB
}

func handleInventory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result, err := buildInventory()
	if err != nil {
		log.Printf("inventory error: %v", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleVMActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Hostname string `json:"hostname"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Hostname == "" || req.Status == "" {
		http.Error(w, "bad request: need hostname and status", http.StatusBadRequest)
		return
	}

	RecordVMActivity(req.Hostname, req.Status)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
}

func Start_grpc(ctx context.Context) {
	log.Println("Demarrage du serveur gRPC...")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Erreur lors de l'ecoute du port : %v", err)
	}

	// Public gRPC methods that don't require authentication
	publicMethods := []string{
		"/frontcontrol.AttribVMService/AttribVMinPool",
		"/frontcontrol.AttribVMService/ReturnPoolWithKey",
		"/frontcontrol.AuthService/AuthenticateUser",
		"/frontcontrol.AuthService/CreateUser",
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(oidcmw.UnaryInterceptor(publicMethods)),
		grpc.ChainStreamInterceptor(oidcmw.StreamInterceptor(publicMethods)),
	)

	conn, err := grpc.NewClient("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)

	gc, err := guacamole.NewClientFromEnv()
	if err != nil {
		log.Printf("[guac] init error: %v", err)
	}
	guacClient = gc

	frontcontrolpb.RegisterAuthServiceServer(s,
		auth.New(config.Database, client))
	frontcontrolpb.RegisterGatherDataServiceServer(s,
		gatherdata.New(client, config.Database))
	frontcontrolpb.RegisterConfigServiceServer(s,
		configpool.New(client, config.Database))
	poolService := pool.New(config.Database, client, gc)
	frontcontrolpb.RegisterPoolServiceServer(s, poolService)
	frontcontrolpb.RegisterUserServiceServer(s,
		user.New(config.Database, config.Broker))
	frontcontrolpb.RegisterAttribVMServiceServer(s,
		attribvm.New(config.Database))

	reflection.Register(s)

	// gRPC server (HTTP/2) on port 50051 for internal use
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Erreur serveur gRPC: %v", err)
		}
	}()

	// gRPC-Web + REST API server on port 50055
	wrappedGrpc := grpcweb.WrapServer(s,
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	registerMetrics()
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/api/inventory", handleInventory)
	mux.HandleFunc("/api/vm-activity", handleVMActivity)
	mux.HandleFunc("/api/guac-url", handleGuacURL)
	mux.HandleFunc("/api/app-status", handleAppStatus)
	mux.HandleFunc("/api/github/session", handleGitHubSession)
	mux.HandleFunc("/api/github/students", handleGitHubStudents)
	mux.HandleFunc("/api/github/public-keys", handleGitHubPublicKeys)
	mux.HandleFunc("/api/github/login", handleGitHubLogin)
	mux.HandleFunc("/auth/github/callback", handleGitHubCallback)
	mux.HandleFunc("/api/nbgrader/assignments", handleNbgraderAssignments)
	mux.HandleFunc("/api/nbgrader/submit", handleNbgraderSubmit)
	mux.HandleFunc("/api/nbgrader/collect", handleNbgraderCollect)
	mux.HandleFunc("/api/nbgrader/autograde", handleNbgraderAutograde)
	mux.HandleFunc("/api/nbgrader/grades", handleNbgraderGrades)
	mux.HandleFunc("/api/nbgrader/release", handleNbgraderRelease)
	mux.HandleFunc("/api/nbgrader/export-csv", handleNbgraderExportCSV)
	mux.HandleFunc("/api/nbgrader/jupyter-url", handleNbgraderJupyterURL)
	mux.HandleFunc("/api/nbgrader/submission-url", handleNbgraderSubmissionURL)
	mux.HandleFunc("/api/image-proposals", handleImageProposals)
	mux.HandleFunc("/api/jupyter-proxy/", handleJupyterProxy)
	mux.HandleFunc("/vm-registrar", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "vm-registrar")
	})

	httpServer := &http.Server{
		Addr: ":50055",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") ||
				strings.HasPrefix(r.URL.Path, "/auth/") ||
				r.URL.Path == "/vm-registrar" ||
				r.URL.Path == "/metrics" {
				mux.ServeHTTP(w, r)
				return
			}
			wrappedGrpc.ServeHTTP(w, r)
		}),
		ReadHeaderTimeout: 30 * time.Second,
	}
	go func() {
		log.Println("Serveur gRPC-Web + REST API sur le port 50055")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erreur serveur gRPC-Web: %v", err)
		}
	}()

	log.Println("Serveur gRPC lance sur le port 50051")
	go monitoring.Start_Monitoring(ctx, client, gc)

	<-ctx.Done()
	log.Println("Arret du serveur gRPC demande...")

	s.GracefulStop()
	httpServer.Shutdown(ctx)
	log.Println("Serveur gRPC arrete proprement")
}
