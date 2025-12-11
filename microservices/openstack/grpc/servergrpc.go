package grpc

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/events"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/notifier"
	"PoolManagerVM/backend/pb"
	"PoolManagerVM/backend/utils"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

type ServerMicroOpenstack struct {
	pb.UnimplementedPoolManagerServer
	DB *gorm.DB
}

func Start_grpc() {
	log.Println("gRPC started")
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Error listening: %v", err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterPoolManagerServer(grpcServer, &ServerMicroOpenstack{DB: config.Database})
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Erreur serveur gRPC: %v", err)
	}
}

func (s *ServerMicroOpenstack) handleUser(db *gorm.DB, req *pb.RessourceRequest) error {
	data := req.GetData()
	switch req.GetStatus() {
	case pb.Status_CREATE:
		user := models.User{
			Name:     data["name"],
			Email:    data["email"],
			Password: data["password"],
		}
		return db.Create(&user).Error
	case pb.Status_UPDATE:
		return db.Model(&models.User{}).
			Where("email = ?", data["email"]).
			Updates(map[string]interface{}{
				"name":     data["name"],
				"password": data["password"],
			}).Error
	case pb.Status_DELETE:
		return db.Where("email = ?", data["email"]).Delete(&models.User{}).Error
	default:
		return fmt.Errorf("unknown status for USER: %v", req.GetStatus())
	}
}

func (s *ServerMicroOpenstack) handleServerpool(db *gorm.DB, req *pb.RessourceRequest) error {
	data := req.GetData()
	switch req.GetStatus() {
	case pb.Status_CREATE:
		pool := models.Serverpool{
			ServerpoolID: data["serverpool_id"],
			UserID:       req.GetUser(),
			ImageRef:     data["image_ref"],
			FlavorRef:    data["flavor_ref"],
			Networks:     models.ParseJSONStringSlice(data["networks"]),
			MinVM:        parseInt(data["min_vm"]),
			MaxVM:        parseInt(data["max_vm"]),
		}
		return db.Create(&pool).Error
	case pb.Status_UPDATE:
		return db.Model(&models.Serverpool{}).
			Where("serverpool_id = ? AND user_id = ?", data["serverpool_id"], req.GetUser()).
			Updates(map[string]any{
				"image_ref":  data["image"],
				"flavor_ref": data["flavor"],
				"min_vm":     parseInt(data["min_vm"]),
				"max_vm":     parseInt(data["max_vm"]),
			}).Error
	case pb.Status_DELETE:
		var pool models.Serverpool
		if err := db.Where(" user_id = ? AND serverpool_id = ? ", req.GetUser(), data["serverpool_id"]).First(&pool).Error; err != nil {
			return err
		}
		err := db.Where("user_id = ? AND serverpool_id = ?", req.GetUser(), data["serverpool_id"]).Delete(&models.Serverpool{}).Error
		if err == nil {
			notifier.GlobalChan <- events.RessourceEvent{Action: "deleted", Type: pb.Type_SERVERPOOL, Ressource: pool}
		}
		// Also delete all servers linked to this serverpool
		ops, err := utils.GetAllServers()
		if err != nil {
			return err
		}
		for _, serv := range ops {
			if serv.Metadata["serverpool_id"] == data["serverpool_id"] && serv.Metadata["user_id"] == req.GetUser() {
				var args []string
				args = append(args, "instance_id")
				args = append(args, serv.ID)
				worker.AddJob(*worker.CreateJob(models.DeleteVM, utils.BuildDataMap(args)), true)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown status for SERVERPOOL: %v", req.GetStatus())
	}
}

func (s *ServerMicroOpenstack) handleServer(db *gorm.DB, req *pb.RessourceRequest) error {
	data := req.GetData()
	switch req.GetStatus() {
	case pb.Status_CREATE:
		var pool models.Serverpool
		if err := db.Where(" user_id = ? AND serverpool_id = ? ", req.GetUser(), data["serverpool_id"]).First(&pool).Error; err != nil {
			return err
		}
		worker.AddJob(*worker.CreateJob(models.CreateVM, utils.BuildDataMap(utils.FlatstringSP(pool))), true)
		return nil
	case pb.Status_UPDATE:
		opts := &clientconfig.ClientOpts{
			Cloud: os.Getenv("OPTS_CLOUD"),
		}
		client, err := clientconfig.NewServiceClient(context.Background(), "compute", opts)
		if err != nil {
			return err
		}
		rebuildOpts := servers.RebuildOpts{
			ImageRef: req.GetData()["image_ref"],
			Name:     req.GetData()["name"],
		}
		_, err = servers.Rebuild(context.Background(), client, req.GetData()["id"], rebuildOpts).Extract()
		if err != nil {
			return err
		}
		return nil
	case pb.Status_DELETE:
		var serv models.Server
		if err := db.Where(" user_id = ? AND name = ? ", req.GetUser(), data["name"]).First(&serv).Error; err != nil {
			return err
		}
		err := db.Where("user_id = ? AND name = ?", req.GetUser(), data["name"]).Delete(&models.Server{}).Error
		if err == nil {
			notifier.GlobalChan <- events.RessourceEvent{Action: "deleted", Type: pb.Type_SERVER, Ressource: serv}
		}
		return db.Where("id = ?", data["server_id"]).Delete(&models.Server{}).Error
	default:
		return fmt.Errorf("unknown status for SERVER: %v", req.GetStatus())
	}
}

func (s *ServerMicroOpenstack) handleConfig(db *gorm.DB, req *pb.RessourceRequest) error {
	data := req.GetData()
	switch req.GetStatus() {
	case pb.Status_CREATE:
		cfg := models.ConfigPool{
			UserID: req.GetUser(),
			Name:   data["name"],
			Data:   data["data"],
		}
		return db.Create(&cfg).Error
	case pb.Status_UPDATE:
		var cfg models.ConfigPool
		if err := db.Where(" user_id = ? AND name = ? ", req.GetUser(), data["name"]).First(&cfg).Error; err != nil {
			return err
		}
		cfg.Data = data["data"]
		return db.Save(&cfg).Error
	case pb.Status_DELETE:
		var cfg models.ConfigPool
		if err := db.Where(" user_id = ? AND name = ? ", req.GetUser(), data["name"]).First(&cfg).Error; err != nil {
			return err
		}
		err := db.Where("user_id = ? AND name = ?", req.GetUser(), data["name"]).
			Delete(&models.ConfigPool{}).Error
		if err == nil {
			notifier.GlobalChan <- events.RessourceEvent{Action: "deleted", Type: pb.Type_CONFIG, Ressource: cfg}
		}
		return err
	default:
		return fmt.Errorf("unknown status for CONFIG: %v", req.GetStatus())
	}
}

func (s *ServerMicroOpenstack) SendRessources(ctx context.Context, req *pb.RessourceRequest) (*pb.RessourceResponse, error) {
	log.Printf("[SendRessources] User=%s Data=%v Status=%v Type=%v", req.GetUser(), req.GetData(), req.GetStatus(), req.GetType())
	err := s.DB.Transaction(func(db *gorm.DB) error {
		switch req.GetType() {
		case pb.Type_USER:
			return s.handleUser(db, req)
		case pb.Type_SERVERPOOL:
			return s.handleServerpool(db, req)
		case pb.Type_SERVER:
			return s.handleServer(db, req)
		case pb.Type_CONFIG:
			return s.handleConfig(db, req)
		default:
			return fmt.Errorf("Type unknown: %v", req.GetType())
		}
	})
	if err != nil {
		return &pb.RessourceResponse{Success: false}, err
	}
	return &pb.RessourceResponse{Success: true}, nil
}

func sendAllServer(s *ServerMicroOpenstack, stream pb.PoolManager_GetStreamRessourcesServer) error {
	rows, err := s.DB.Model(&models.Server{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var serv models.Server
		if err := s.DB.ScanRows(rows, &serv); err != nil {
			log.Println("Error rows server")
			return err
		}
		ret := &pb.StreamRessourceResponse{
			User:   serv.UserID,
			Status: pb.Status_CREATE,
			Type:   pb.Type_SERVER,
			Data:   serv.ToMap(),
		}

		if err := stream.Send(ret); err != nil {
			log.Println("error sending server")
			return err
		}
	}
	return nil
}

func sendAllPool(s *ServerMicroOpenstack, stream pb.PoolManager_GetStreamRessourcesServer) error {
	rows, err := s.DB.Model(&models.Serverpool{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var pool models.Serverpool
		if err := s.DB.ScanRows(rows, &pool); err != nil {
			log.Println("Error rows server")
			return err
		}
		ret := &pb.StreamRessourceResponse{
			User:   pool.UserID,
			Status: pb.Status_CREATE,
			Type:   pb.Type_SERVERPOOL,
			Data:   pool.ToMap(),
		}

		if err := stream.Send(ret); err != nil {
			log.Println("error sending server")
			return err
		}
	}
	return nil
}

func sendAllConfig(s *ServerMicroOpenstack, stream pb.PoolManager_GetStreamRessourcesServer) error {
	rows, err := s.DB.Model(&models.ConfigPool{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var pool models.ConfigPool
		if err := s.DB.ScanRows(rows, &pool); err != nil {
			log.Println("Error rows server")
			return err
		}
		ret := &pb.StreamRessourceResponse{
			User:   pool.UserID,
			Status: pb.Status_CREATE,
			Type:   pb.Type_CONFIG,
			Data:   pool.ToMap(),
		}

		if err := stream.Send(ret); err != nil {
			log.Println("error sending server")
			return err
		}
	}
	return nil
}

func (s *ServerMicroOpenstack) GetStreamRessources(req *emptypb.Empty, stream pb.PoolManager_GetStreamRessourcesServer) error {
	log.Println("[GetStreamRessources] Stream global started")

	// Send all ressources at first connection to ensure synchronize
	if err := sendAllServer(s, stream); err != nil {
		log.Printf("Error Server: %v", err)
		return err
	}
	if err := sendAllPool(s, stream); err != nil {
		log.Printf("Error Serverpool: %v", err)
		return err
	}
	if err := sendAllConfig(s, stream); err != nil {
		log.Printf("Error Serverpool: %v", err)
		return err
	}

	// Infinite loop to send all modification on the database
	for {
		select {
		case evt := <-notifier.GlobalChan:
			switch evt.Type {
			case pb.Type_SERVER:
				server, ok := evt.Ressource.(models.Server)
				if !ok {
					continue
				}
				var status pb.Status
				switch evt.Action {
				case "created":
					status = pb.Status_CREATE
				case "updated":
					status = pb.Status_UPDATE
				case "deleted":
					status = pb.Status_DELETE
				default:
					status = pb.Status_STATUS_UNKNOWN
				}
				log.Println("Sending message now")
				err := stream.Send(&pb.StreamRessourceResponse{
					User:   server.UserID,
					Type:   pb.Type_SERVER,
					Status: status,
					Data:   server.ToMap(),
				})
				if err != nil {
					log.Printf("Stream closed for client: %v", err)
					return err
				}

			case pb.Type_SERVERPOOL:
				pool, ok := evt.Ressource.(models.Serverpool)
				if !ok {
					continue
				}
				var status pb.Status
				switch evt.Action {
				case "created":
					status = pb.Status_CREATE
				case "updated":
					status = pb.Status_UPDATE
				case "deleted":
					status = pb.Status_DELETE
				default:
					status = pb.Status_STATUS_UNKNOWN
				}
				log.Println("Sending message now")
				err := stream.Send(&pb.StreamRessourceResponse{
					User:   pool.UserID,
					Type:   pb.Type_SERVERPOOL,
					Status: status,
					Data:   pool.ToMap(),
				})
				if err != nil {
					log.Printf("Stream closed for client: %v", err)
					return err
				}

			case pb.Type_CONFIG:
				config, ok := evt.Ressource.(models.ConfigPool)
				if !ok {
					continue
				}
				var status pb.Status
				switch evt.Action {
				case "created":
					status = pb.Status_CREATE
				case "updated":
					status = pb.Status_UPDATE
				case "deleted":
					status = pb.Status_DELETE
				default:
					status = pb.Status_STATUS_UNKNOWN
				}
				log.Println("Sending message now")
				log.Printf("user = %s, Name = %s", config.UserID, config.Name)
				err := stream.Send(&pb.StreamRessourceResponse{
					User:   config.UserID,
					Type:   pb.Type_CONFIG,
					Status: status,
					Data:   config.ToMap(),
				})
				if err != nil {
					log.Printf("Stream closed for client: %v", err)
					return err
				}
			}

		case <-stream.Context().Done():
			log.Println("[GetStreamRessources] Client disconnected, end of stream")
			return nil
		}
	}
}

func (s *ServerMicroOpenstack) GetStreamRessourcesUser(req *pb.UserRequest, stream grpc.ServerStreamingServer[pb.StreamRessourceResponse]) error {
	log.Println("[GetStreamRessourcesUser] Stream User started")
	// stream user-specific ressources
	return nil
}

func (s *ServerMicroOpenstack) GetAllImages(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Image]) error {
	rows, err := s.DB.Model(&models.Image{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var img models.Image
		if err := s.DB.ScanRows(rows, &img); err != nil {
			log.Println("Error rows server")
			return err
		}
		if err := stream.Send(img.ToPb()); err != nil {
			log.Println("error sending server")
			return err
		}
	}
	return nil
}

func (s *ServerMicroOpenstack) GetAllFlavors(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Flavor]) error {
	rows, err := s.DB.Model(&models.Flavor{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var f models.Flavor
		if err := s.DB.ScanRows(rows, &f); err != nil {
			log.Println("Error rows server")
			return err
		}
		if err := stream.Send(f.ToPb()); err != nil {
			log.Println("error sending server")
			return err
		}
	}
	return nil
}

func (s *ServerMicroOpenstack) GetAllNetworks(req *emptypb.Empty, stream grpc.ServerStreamingServer[pb.Network]) error {
	rows, err := s.DB.Model(&models.Network{}).Rows()
	if err != nil {
		log.Println("Error retrieving servers")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var n models.Network
		if err := s.DB.ScanRows(rows, &n); err != nil {
			log.Println("Error rows server")
			return err
		}
		if err := stream.Send(n.ToPb()); err != nil {
			log.Println("error sending server")
			return err
		}
	}
	return nil
}

// parseInt helper
func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
