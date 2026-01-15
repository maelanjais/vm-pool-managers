package monitoring

import (
	"context"
	"control_center/config"
	"control_center/models"
	"control_center/pb"
	"log"
	"time"
)

func Start_Monitoring(
	ctx context.Context,
	clientMicroservice pb.PoolManagerClient,
) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Monitoring stopped")
			return
		case <-ticker.C:
			log.Println("Monitoring tick...")
			checkallpool(clientMicroservice)
		}
	}
}

func checkallpool(client pb.PoolManagerClient) {
	var pools []models.Serverpool
	err := config.Database.Find(&pools).Error
	if err != nil {
		log.Println("Error fetching server pools:", err)
		return
	}
	for _, pool := range pools {
		checkpool(&pool, client)
	}
}

func checkpool(pool *models.Serverpool, client pb.PoolManagerClient) {
	now := time.Now().UTC()

	switch pool.Status {
	case "scheduled":
		if shouldStartPool(pool, now) {
			startPool(pool, client)
		}
	case "running":
		if shouldDeletePool(pool, now) {
			deletePool(pool, client)
		}
	}
}

func startPool(pool *models.Serverpool, client pb.PoolManagerClient) {
	log.Printf("Starting pool ID %s as per schedule", pool.ServerpoolID)
	err := config.Database.Model(pool).
		Where("status = ?", "scheduled").
		Update("status", "creating").Error
	if err != nil {
		log.Println("Failed to change pool status:", err)
		return
	}
	go launchCreatePool(pool, client)
}

func shouldDeletePool(pool *models.Serverpool, now time.Time) bool {
	if pool.TimeStart == nil || pool.Timewindow == nil {
		return false
	}

	endTime := pool.TimeStart.Add(*pool.Timewindow)
	return now.After(endTime)
}

func deletePool(pool *models.Serverpool, client pb.PoolManagerClient) {
	log.Printf("Deleting pool ID %s as per schedule", pool.ServerpoolID)
	err := config.Database.Model(pool).
		Where("status = ?", "running").
		Update("status", "deleting").Error
	if err != nil {
		log.Println("Failed to change pool status:", err)
		return
	}

	go launchDeletePool(pool, client)
}

func shouldStartPool(pool *models.Serverpool, now time.Time) bool {
	if pool.TimeStart == nil || pool.Timewindow == nil {
		return false
	}

	startWindow := pool.TimeStart.Add(-30 * time.Minute)
	return now.After(startWindow) && now.Before(*pool.TimeStart)
}

func launchCreatePool(p *models.Serverpool, client pb.PoolManagerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rep, err := client.SendRessources(
		ctx,
		&pb.RessourceRequest{
			User:   p.UserID,
			Data:   p.ToMap(),
			Status: pb.Status_CREATE,
			Type:   pb.Type_SERVERPOOL,
		},
	)
	if err != nil || rep.GetSuccess() == false {
		log.Println("Error on creating pool as planned")
		err := config.Database.Model(p).
			Where("status = ?", "creating").
			Update("status", "schedlued").Error
		if err != nil {
			log.Println("Failed to update pool status:", err)
		}
		return
	}
	log.Printf("Pool ID %s created successfully", p.ServerpoolID)
	err = config.Database.Model(p).
		Where("status = ?", "creating").
		Update("status", "running").Error
	if err != nil {
		log.Println("Failed to update pool status to running:", err)
	}
}

func launchDeletePool(p *models.Serverpool, client pb.PoolManagerClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rep, err := client.SendRessources(
		ctx,
		&pb.RessourceRequest{
			User:   p.UserID,
			Data:   p.ToMap(),
			Status: pb.Status_DELETE,
			Type:   pb.Type_SERVERPOOL,
		},
	)
	if err != nil || rep.GetSuccess() == false {
		log.Println("Error on deleting pool as planned")
		err := config.Database.Model(p).
			Where("status = ?", "deleting").
			Update("status", "running").Error
		if err != nil {
			log.Println("Failed to update pool status:", err)
		}
		return
	}
	log.Printf("Pool ID %s deleted successfully", p.ServerpoolID)
	var nextTimeStart *time.Time
	if p.TimeStart != nil {
		t := p.TimeStart.AddDate(0, 0, 7)
		nextTimeStart = &t
	}
	err = config.Database.Model(p).
		Where("status = ?", "deleting").
		Updates(map[string]any{
			"status":     "scheduled",
			"time_start": nextTimeStart,
		}).Error
	if err != nil {
		log.Println("Failed to update pool status:", err)
	}
}
