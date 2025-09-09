package internal

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"log"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"gorm.io/gorm"
)

// Backwork is a background loop that monitors admin servers and ensures a minimum number of VMs are running.
// It fetches all servers, filters those owned by the admin, and compares the current count to a configured minimum.
// If there are too few, it adds jobs to create additional VMs. The loop repeats every 20 seconds.

func Backwork(ctx context.Context) {

	for {
		allServers, err := utils.GetAllServers()
		if err != nil {
			log.Printf("Error : %v", err)
			return
		}

		// get all admin VMs
		var myPool []servers.Server
		for _, s := range allServers {
			if s.Metadata["userID"] == "admin" {
				myPool = append(myPool, s)
			}
		}

		var minVM int
		// if no VMs, create some with config file
		if len(myPool) == 0 {
			cfg, err := utils.LoadConfig("config.toml")
			if err != nil {
				log.Printf("Error")
				return
			}
			minVM, err = strconv.Atoi(cfg.Metadata["minVM"])
			if err != nil {
				log.Printf("Error : %v", err)
			}
		} else {
			minVM, err = strconv.Atoi(myPool[0].Metadata["minVM"])
			if err != nil {
				log.Printf("Error : %v", err)
			}
		}

		err = config.Database.Transaction(func(tx *gorm.DB) error {
			var pool models.ServerPool
			if err := tx.FirstOrCreate(&pool, models.ServerPool{ID: "admin"}).Error; err != nil {
				return err
			}

			current := len(myPool) + pool.PendingJobs
			if current < minVM {
				numToCreate := minVM - current
				for range numToCreate {
					worker.AddJob(*worker.CreateJob("base", worker.CreateVMAdmin, nil), false)
					pool.PendingJobs++
				}
				if err := tx.Model(&pool).Update("pending_jobs", pool.PendingJobs).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			log.Println("DB error: ", err)
		}

		select {
		case <-ctx.Done():
			log.Println("Backwork stopped")
			return
		case <-time.After(10 * time.Second):
			// next cycle
		}
	}
}
