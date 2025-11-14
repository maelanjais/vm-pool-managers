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
	// Chargement du fichier .env
	if err := godotenv.Load(); err != nil {
		panic("Error on loading .env")
	}

	// Initialisation de la base de données
	config.Start_DB(context.Background())

	// Création d’un contexte annulé sur SIGINT ou SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Lancement des goroutines
	go config.Sync_DB(ctx)

	// Remplissage initial de la base de données
	cc.PopulateDBImageMicroOpen()
	cc.PopulateDBFlavorMicroOpen()
	cc.PopulateDBNetworkMicroOpen()

	go cc.Start_grpc(ctx)
	go cc.ConnectToMicroOpen(ctx)

	// Attente du signal d’arrêt
	<-ctx.Done()
	log.Println("Signal reçu, arrêt du streaming, du serveur et des tâches en cours...")

	// Annule explicitement le contexte (au cas où)
	stop()

	log.Println("Arrêt terminé proprement ✅")
}
