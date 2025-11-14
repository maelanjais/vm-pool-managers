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
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
			if err := stream.CloseSend(); err != nil {
				log.Printf("Erreur lors de la fermeture du stream: %v", err)
			}
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
			log.Printf("message recieved type : %s", resp.GetType().String())
			HandleStreamEvent(resp)
		}
	}
}

func HandleStreamEvent(resp *pb.StreamRessourceResponse) {
	switch resp.Type {

	case pb.Type_SERVER:
		var serv models.Server
		serv.FromPb(resp)
		handleDBEvent(&serv, resp.Status)

	case pb.Type_SERVERPOOL:
		var pool models.Serverpool
		pool.FromPb(resp)
		handleDBEvent(&pool, resp.Status)

	case pb.Type_CONFIG:
		var conf models.ConfigPool
		conf.FromPb(resp)
		handleDBEvent(&conf, resp.Status)

	default:
		log.Printf("⚠️ Type inconnu reçu : %v", resp.Type)
	}
}

func handleDBEvent(model any, status pb.Status) {
	switch status {

	case pb.Status_CREATE:
		if err := config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(model).Error; err != nil {
			log.Printf("Erreur CREATE %T : %v", model, err)
		} else {
			log.Printf("CREATE %T OK", model)
		}

	case pb.Status_UPDATE:
		if err := config.Database.Save(model).Error; err != nil {
			log.Printf("Erreur UPDATE %T : %v", model, err)
		} else {
			log.Printf("UPDATE %T OK ", model)
		}

	case pb.Status_DELETE:
		if err := config.Database.Delete(model).Error; err != nil {
			log.Printf("Erreur DELETE %T : %v", model, err)
		} else {
			log.Printf("DELETE %T OK", model)
		}

	default:
		log.Printf("Status inconnu : %v", status)
	}
}

func PopulateDBImageMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(&img)
	}
}

func PopulateDBFlavorMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(&flavor)
	}
}

func PopulateDBNetworkMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		config.Database.Clauses(clause.OnConflict{UpdateAll: true}).Create(&network)
	}
}
