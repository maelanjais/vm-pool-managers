package grpc

import (
	"context"
	"control_center/config"
	"control_center/models"
	"control_center/pb"
	"io"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm/clause"
)

func ConnectToMicroOpen(ctx context.Context) {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)
	stream, err := client.GetStreamRessources(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Erreur stream: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Arrêt du streaming ConnectToMicroOpen")
			_ = stream.CloseSend()
			return
		default:
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				if ctx.Err() != nil {
					log.Println("Stream ended due to context")
					return
				}
				log.Fatalf("Error listening stream: %v", err)
			}
			HandleStreamEvent(resp)
		}
	}
}

func HandleStreamEvent(resp *pb.StreamRessourceResponse) {
	switch resp.Type {
	case pb.Type_SERVER:
		var serv models.Server
		serv.FromPb(resp)
		handleDBServerEvent(&serv, resp.Status)
	case pb.Type_SERVERPOOL:
		var pool models.Serverpool
		pool.FromPb(resp)
		handleDBServerpoolEvent(&pool, resp.Status)
	case pb.Type_CONFIG:
		var conf models.ConfigPool
		conf.FromPb(resp)
		handleDBConfigEvent(&conf, resp.Status)
	default:
		log.Printf("⚠️ Type inconnu reçu : %v", resp.Type)
	}
}

func handleDBServerEvent(server *models.Server, status pb.Status) {
	switch status {
	case pb.Status_CREATE:
		if err := config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(server).Error; err != nil {
			log.Printf("Erreur CREATE %T : %v", server, err)
		}
	case pb.Status_UPDATE:
		err := config.Database.Model(&models.Server{}).
			Where("user_id = ? AND name = ?", server.UserID, server.Name).
			Updates(server).Error
		if err != nil {
			log.Printf("Erreur UPDATE %T : %v", server, err)
		}
	case pb.Status_DELETE:
		if err := config.Database.Delete(server).Error; err != nil {
			log.Printf("Erreur DELETE %T : %v", server, err)
		}
	default:
		log.Printf("Status inconnu : %v", status)
	}
}

func handleDBServerpoolEvent(serverpool *models.Serverpool, status pb.Status) {
	switch status {
	case pb.Status_CREATE:
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(serverpool).Error
	case pb.Status_UPDATE:
		_ = config.Database.Model(&models.Serverpool{}).
			Where("user_id = ? AND serverpool_id = ?", serverpool.UserID, serverpool.ServerpoolID).
			Updates(serverpool).Error
	case pb.Status_DELETE:
		_ = config.Database.Delete(serverpool).Error
	}
}

func handleDBConfigEvent(configpool *models.ConfigPool, status pb.Status) {
	switch status {
	case pb.Status_CREATE:
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(configpool).Error
	case pb.Status_UPDATE:
		_ = config.Database.Model(&models.ConfigPool{}).
			Where("user_id = ? AND name = ?", configpool.UserID, configpool.Name).
			Update("data", configpool.Data).Error
	case pb.Status_DELETE:
		_ = config.Database.
			Where("user_id = ? AND name = ?", configpool.UserID, configpool.Name).
			Delete(configpool).Error
	}
}

func PopulateDBImageMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)
	stream, err := client.GetAllImages(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Erreur stream: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error listening stream: %v", err)
		}
		var img models.Image
		img.FromPb(resp, "Openstack")
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(&img).Error
	}
}

func PopulateDBFlavorMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)
	stream, err := client.GetAllFlavors(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Erreur stream: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error listening stream: %v", err)
		}
		var flavor models.Flavor
		flavor.FromPb(resp, "Openstack")
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(&flavor).Error
	}
}

func PopulateDBNetworkMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)
	stream, err := client.GetAllNetworks(context.Background(), &emptypb.Empty{})
	if err != nil {
		log.Fatalf("Erreur stream: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error listening stream: %v", err)
		}
		var network models.Network
		network.FromPb(resp, "Openstack")
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(&network).Error
	}
}
