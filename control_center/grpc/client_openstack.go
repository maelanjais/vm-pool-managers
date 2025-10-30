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
)

func ConnectToMicroOpen() {
	conn, err := grpc.NewClient("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Erreur de connexion: %v", err)
	}
	defer conn.Close()

	client := pb.NewPoolManagerClient(conn)

	stream, err := client.GetStreamRessources(context.Background(), &emptypb.Empty{})
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
		switch resp.Type {
		case pb.Type_SERVER:
			// createserver
		case pb.Type_SERVERPOOL:
			// createpool
		case pb.Type_CONFIG:
			// createconfig
		}
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
		config.Database.Create(&img)
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
		config.Database.Create(&flavor)
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
		config.Database.Create(&network)
	}
}
