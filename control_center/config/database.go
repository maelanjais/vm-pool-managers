package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"control_center/models"
)

// global variable to get access to the database anywhere in the code
var (
	Database *gorm.DB
	DBmu     sync.Mutex
)

// boot the database
func Start_DB() {
	// host := os.Getenv("POSTGRES_HOST")
	host := "localhost"
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	pw := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC", host, user, pw, dbname, port)

	var err error

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Erreur lors de l'accès à la DB SQL : %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Impossible de ping PostgreSQL : %v", err)
	}

	Database = db
	log.Println("✅ Connexion à PostgreSQL réussie avec GORM")

	Database.AutoMigrate(&models.User{}, &models.Serverpool{}, &models.Server{}, &models.ConfigPool{}, &models.Image{}, &models.Flavor{}, &models.Network{})
}

func Sync_DB(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Arrêt de la synchronisation DB")
			return
		case <-ticker.C:
			log.Println("🔄 Synchronisation de la base de données...")
			// Exemple : synchroniser les serveurs, configurations, etc.
		}
	}
}
