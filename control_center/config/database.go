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
	log.Println("Connexion à PostgreSQL réussie avec GORM")

	Database.AutoMigrate(&models.User{}, &models.Serverpool{}, &models.Server{}, &models.ConfigPool{}, &models.Image{}, &models.Flavor{}, &models.Network{})
	createNotifyTriggers()
}

func Sync_DB(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Arrêt de la synchronisation DB")
			return
		case <-ticker.C:
			log.Println("Synchronisation de la base de données...")
			// Exemple : synchroniser les serveurs, configurations, etc.
		}
	}
}

func createNotifyTriggers() {
	// Fonction générique pour notifier les changements
	triggerFuncSQL := `
CREATE OR REPLACE FUNCTION notify_table_change()
RETURNS trigger AS $$
DECLARE
    payload json;
    action text;
    table_name text;
BEGIN
    table_name := TG_TABLE_NAME;

    IF TG_OP = 'INSERT' THEN
        action := 'create';
        payload := row_to_json(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        action := 'update';
        payload := row_to_json(NEW);
    ELSIF TG_OP = 'DELETE' THEN
        action := 'delete';
        payload := row_to_json(OLD);
    END IF;

    payload := json_build_object(
        'table', table_name,
        'action', action,
        'data', payload
    );

    PERFORM pg_notify(table_name || '_events', payload::text);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
`
	if err := Database.Exec(triggerFuncSQL).Error; err != nil {
		log.Fatalf("Erreur lors de la création de la fonction trigger : %v", err)
	}

	// Liste des tables pour lesquelles créer les triggers
	tables := []string{"servers", "serverpools", "config_pools"}

	for _, table := range tables {
		triggerSQL := fmt.Sprintf(`
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = '%s_change_trigger') THEN
        CREATE TRIGGER %s_change_trigger
        AFTER INSERT OR UPDATE OR DELETE ON %s
        FOR EACH ROW EXECUTE FUNCTION notify_table_change();
    END IF;
END$$;
`, table, table, table)

		if err := Database.Exec(triggerSQL).Error; err != nil {
			log.Fatalf("Erreur lors de la création du trigger pour %s : %v", table, err)
		}
	}

	log.Println("Triggers PostgreSQL créés pour Server, Serverpool et ConfigPool")
}
