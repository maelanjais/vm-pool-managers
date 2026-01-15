package grpc

import (
	"context"
	"control_center/config"
	"control_center/models"
	"control_center/pb"
	"encoding/json"
	"io"
	"log"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm/clause"
)

func ConnectToMicroOpen(ctx context.Context) {
	conn, err := grpc.NewClient("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		handleDBServerEvent(&serv, resp.Status, resp.Data)
	case pb.Type_SERVERPOOL:
		var pool models.Serverpool
		pool.FromPb(resp)
		handleDBServerpoolEvent(resp.GetData()["serverpool_id"], resp.GetUser(), resp.Data, resp.Status)
	case pb.Type_CONFIG:
		var conf models.ConfigPool
		conf.FromPb(resp)
		handleDBConfigEvent(&conf, resp.Status)
	default:
		log.Printf("Type inconnu reçu : %v", resp.Type)
	}
}

func handleDBServerEvent(server *models.Server, status pb.Status, data map[string]string) {
	// A retravailler
	switch status {
	case pb.Status_CREATE:
		// if err := config.Database.Clauses(clause.OnConflict{UpdateAll: true}).
		// 	Create(server).Error; err != nil {
		// 	log.Printf("Erreur CREATE %T : %v", server, err)
		// }
		updates := serverUpdatesFromMap(data)
		err := config.Database.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.Assignments(updates),
			}).
			Create(server).Error
		if err != nil {
			log.Printf("Erreur CREATE %T : %v", server, err)
		}

	case pb.Status_UPDATE:
		// err := config.Database.Model(&models.Server{}).
		// 	Where("id = ?", server.ID).
		// 	Updates(server).Error
		// if err != nil {
		// 	log.Printf("Erreur UPDATE %T : %v", server, err)
		// }
		updates := serverUpdatesFromMap(data)
		if len(updates) == 0 {
			return
		}
		err := config.Database.
			Model(&models.Server{}).
			Where("id = ?", server.ID).
			Updates(updates).Error
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

func handleDBServerpoolEvent(serverpoolID, userID string, data map[string]string, status pb.Status) {
	switch status {

	case pb.Status_CREATE:
		updates := serverpoolUpdatesFromMap(data)
		updates["serverpool_id"] = serverpoolID
		updates["user_id"] = userID

		_ = config.Database.
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "serverpool_id"}, {Name: "user_id"}},
				DoUpdates: clause.Assignments(updates),
			}).
			Create(&models.Serverpool{}).Error

	case pb.Status_UPDATE:
		updates := serverpoolUpdatesFromMap(data)
		if len(updates) == 0 {
			return
		}

		_ = config.Database.
			Model(&models.Serverpool{}).
			Where("serverpool_id = ? AND user_id = ?", serverpoolID, userID).
			Updates(updates).Error

	case pb.Status_DELETE:
		log.Println("Pool deleted in microservice")
	}
}

func handleDBConfigEvent(configpool *models.ConfigPool, status pb.Status) {
	switch status {
	case pb.Status_CREATE:
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).
			Create(configpool).Error
	case pb.Status_UPDATE:
		_ = config.Database.Model(&models.ConfigPool{}).
			Where("user_id = ? AND name = ?", configpool.UserID,
				configpool.Name).Update("data", configpool.Data).Error
	case pb.Status_DELETE:
		_ = config.Database.
			Where("user_id = ? AND name = ?", configpool.UserID,
				configpool.Name).Delete(configpool).Error
	}
}

func PopulateDBImageMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).
			Create(&img).Error
	}
}

func PopulateDBFlavorMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).
			Create(&flavor).Error
	}
}

func PopulateDBNetworkMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		_ = config.Database.Clauses(clause.OnConflict{UpdateAll: true}).
			Create(&network).Error
	}
}

func serverpoolUpdatesFromMap(data map[string]string) map[string]any {
	updates := map[string]any{}

	for k, v := range data {
		switch k {

		case "image_ref":
			updates["image_ref"] = v

		case "flavor_ref":
			updates["flavor_ref"] = v

		case "min_vm":
			if i, err := strconv.Atoi(v); err == nil {
				updates["min_vm"] = i
			}

		case "max_vm":
			if i, err := strconv.Atoi(v); err == nil {
				updates["max_vm"] = i
			}

		case "pending_jobs":
			if i, err := strconv.Atoi(v); err == nil {
				updates["pending_jobs"] = i
			}

		case "config_id":
			updates["config_id"] = v

		case "networks":
			var networks models.JSONStringSlice
			if err := json.Unmarshal([]byte(v), &networks); err == nil {
				updates["networks"] = networks
			}

		case "status":
			updates["status"] = v
		}
	}

	return updates
}

func serverUpdatesFromMap(data map[string]string) map[string]any {
	updates := map[string]any{}
	for k, v := range data {
		switch k {
		case "status":
			updates["status"] = v
		case "attach_volume_id":
			updates["attach_volume_id"] = v
		case "name":
			updates["name"] = v
		case "user_id":
			updates["user_id"] = v
		case "serverpool_id":
			updates["serverpool_id"] = v
		case "metadata":
			updates["metadata"] = v
		case "ip_address":
			updates["ip_address"] = v
		case "reattrib":
			updates["reattrib"] = v
		}
	}
	return updates
}
