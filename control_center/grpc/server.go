package grpc

import (
	"context"
	"control_center/config"
	"control_center/frontcontrolpb"
	"control_center/internal/auth"
	"control_center/internal/configpool"
	"control_center/internal/gatherdata"
	"control_center/pb"
	"log"
	"net"

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

func Start_grpc(ctx context.Context) {
	log.Println("Démarage du serveur gRPC...")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Erreur lors de l'écoute du port : %v", err)
	}

	s := grpc.NewServer()

	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)

	frontcontrolpb.RegisterAuthServiceServer(s, auth.New(config.Database, client))
	frontcontrolpb.RegisterGatherDataServiceServer(s, gatherdata.New())
	frontcontrolpb.RegisterConfigServiceServer(s, configpool.New())
	frontcontrolpb.RegisterPoolServiceServer(s, &PoolServer{DB: config.Database})
	frontcontrolpb.RegisterUserServiceServer(s, &UserServer{DB: config.Database})

	reflection.Register(s)

	// Lance le serveur dans une goroutine
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Erreur serveur gRPC: %v", err)
		}
	}()

	log.Println("Serveur gRPC lancé sur le port 50051")

	// Attend que le contexte soit annulé
	<-ctx.Done()
	log.Println("Arrêt du serveur gRPC demandé...")

	// Arrêt propre du serveur
	s.GracefulStop()
	log.Println("Serveur gRPC arrêté proprement ✅")
}
