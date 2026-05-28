package main

import (
	"context"
	"control_center/config"
	cc "control_center/grpc"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// Chargement du fichier .env (cherche dans le répertoire courant puis dans le parent)
	if err := godotenv.Load(); err != nil {
		if err2 := godotenv.Load("../.env"); err2 != nil {
			log.Fatalf("Error loading .env file: %v", err2)
		}
	}

	// Initialisation de la base de données
	config.Start_DB(context.Background())

	// Création d’un contexte annulé sur SIGINT ou SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Remplissage initial de la base de données
	cc.PopulateDBImageMicroOpen()
	cc.PopulateDBFlavorMicroOpen()
	cc.PopulateDBNetworkMicroOpen()

	go cc.Start_grpc(ctx)
	go cc.ConnectToMicroOpen(ctx)

	// Attente du signal d’arrêt
	<-ctx.Done()

	// Annule explicitement le contexte (au cas où)
	stop()

	log.Println("Arrêt terminé proprement ✅")
}
