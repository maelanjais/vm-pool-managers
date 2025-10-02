package internal

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal/jobs"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"gorm.io/gorm"
)

// Monitor periodically checks server pools and ensures the minimum number of VMs are running.
//
// Parameters:
//   - c: Context used to stop monitoring gracefully.
//
// Workflow:
//  1. Creates a ticker that triggers every 15 seconds.
//  2. On each tick, calls CheckAndCreate() to inspect server pools and create missing VMs.
//  3. Stops monitoring if the context is cancelled.
func Monitor(c context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Done():
			log.Println("Monitoring stopped")
			return

		case <-ticker.C:
			log.Println("Checking serverpools...")
			CheckAndCreate()
			attachVolume()
			volnotattached()
		}
	}
}

// CheckAndCreate inspects all server pools and their servers, and schedules VM creation jobs if needed.
//
// Workflow:
//  1. Fetches all servers and server pools from the database.
//  2. For each pool, counts existing servers that match the pool configuration.
//  3. Calculates how many additional VMs are missing (taking PendingJobs into account).
//  4. Creates jobs for missing VMs and increments the pending counter.
//  5. Ensures a default "pool_vms" server pool exists for the admin user. If missing, it creates it
//     from environment variables and schedules its VMs.
func CheckAndCreate() {

	var (
		servs        []models.Server
		pools        []models.Serverpool
		servadminmap = make(map[string]bool)
	)

	res_servs := config.Database.Find(&servs)
	if res_servs.Error != nil {
		log.Println(res_servs.Error)
	}
	res_pools := config.Database.Find(&pools)
	if res_pools.Error != nil {
		log.Println(res_pools.Error)
	}

	countadmin := 0
	for _, p := range pools {
		count := 0
		for _, s := range servs {
			if serverisinpool(p, s) {
				count++
			}
			if s.UserID == "admin" {
				if !servadminmap[s.ID] {
					servadminmap[s.ID] = true
					countadmin++
				}
			}
		}
		missing := p.MinVM - (count + p.PendingJobs)
		for i := 0; i < missing; i++ {
			if p.ImageRef == os.Getenv("SERVER_IMAGE_REF") && p.FlavorRef == os.Getenv("SERVER_FLAVOR_REF") && len(p.Networks) == 1 && p.Networks[0] == os.Getenv("NETWORK_ID") && countadmin > 0 && p.UserID != "admin" {
				worker.AddJob((*worker.CreateJob(models.AttribVM, map[string]string{
					"ID":            fmt.Sprint(p.ID),
					"serverpool_id": p.ServerpoolID,
					"user_id":       p.UserID,
					"min_vm":        fmt.Sprint(p.MinVM),
					"max_vm":        fmt.Sprint(p.MaxVM),
				})), true)
				countadmin--
				jobs.IncrementPending(p.ID)
			} else {
				log.Println("Creating VM for pool:", p)
				worker.AddJob(*worker.CreateJob(models.CreateVM, utils.BuildDataMap(utils.FlatstringSP(p))), false)
				jobs.IncrementPending(p.ID)
			}
		}
	}

	found := false
	for _, sp := range pools {
		if sp.ServerpoolID == "pool_vms" && sp.UserID == "admin" {
			found = true
			break
		}
	}
	if !found {
		base_p, err := CreateServerpoolFromEnv()
		if err != nil {
			log.Println("Error: can't create param from env: ", err)
		}
		if err := config.Database.First(&base_p, "serverpool_id = ? AND user_id = ?", base_p.ServerpoolID, base_p.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				config.Database.Create(&base_p)
			} else {
				log.Println("Error Database: ", err)
			}
		}
		for i := 0; i < base_p.MinVM; i++ {
			worker.AddJob(*worker.CreateJob(models.CreateVM, utils.BuildDataMap(utils.FlatstringSP(base_p))), false)
			jobs.IncrementPending(base_p.ID)
		}
	}
}

// serverisinpool checks if a server belongs to a given server pool.
//
// Parameters:
//   - p: Serverpool to check against.
//   - s: Server to check.
//
// Returns:
//   - true if the server belongs to the pool (matching ServerpoolID, UserID, FlavorRef, and ImageRef), false otherwise.
func serverisinpool(p models.Serverpool, s models.Server) bool {
	if s.ServerpoolID == p.ServerpoolID && s.UserID == p.UserID && s.FlavorRef == p.FlavorRef && s.ImageRef == p.ImageRef {
		return true
	} else {
		return false
	}
}

// CreateServerpoolFromEnv creates a Serverpool struct using environment variables.
//
// Returns:
//   - models.Serverpool: The server pool built from environment variables.
//   - error: If any conversion or missing environment variable fails.
//
// Required environment variables:
//   - SERVER_IMAGE_REF
//   - SERVER_FLAVOR_REF
//   - METADATA_SERVERPOOL_ID
//   - METADATA_USER_ID
//   - METADATA_MIN_VM
//   - METADATA_MAX_VM
//   - NETWORK_ID
func CreateServerpoolFromEnv() (models.Serverpool, error) {
	// Lire les variables d'environnement
	imageRef := os.Getenv("SERVER_IMAGE_REF")
	flavorRef := os.Getenv("SERVER_FLAVOR_REF")
	poolID := os.Getenv("METADATA_SERVERPOOL_ID")
	userID := os.Getenv("METADATA_USER_ID")
	minVMStr := os.Getenv("METADATA_MIN_VM")
	maxVMStr := os.Getenv("METADATA_MAX_VM")

	// Convertir MinVM et MaxVM en int
	minVM, err := strconv.Atoi(minVMStr)
	if err != nil {
		return models.Serverpool{}, err
	}
	maxVM, err := strconv.Atoi(maxVMStr)
	if err != nil {
		return models.Serverpool{}, err
	}

	// Construire le pool
	pool := models.Serverpool{
		ServerpoolID: poolID,
		UserID:       userID,
		ImageRef:     imageRef,
		FlavorRef:    flavorRef,
		Networks:     models.JSONStringSlice{os.Getenv("NETWORK_ID")},
		MinVM:        minVM,
		MaxVM:        maxVM,
		PendingJobs:  0,
	}

	return pool, nil
}

func attachVolume() {
	allServ, err := utils.GetAllServers()
	if err != nil {
		log.Println("Failed to get all servers:", err)
		return
	}
	for _, serv := range allServ {
		var server models.Server
		if err := config.Database.Select("vol_pending").Where("id = ?", serv.ID).First(&server).Error; err != nil {
			log.Println("Error fetching updated vol_pending:", err)
			return
		}
		if utils.NoVolAttached(serv) && utils.NoVolAttachedDB(models.FromGopherServer(serv), config.Database) && serv.Status == "ACTIVE" && !server.VolPending {
			log.Printf("Attaching volume to server %s\n", serv.ID)
			jobs.ChangePendingVol(serv.ID)
			worker.AddJob(*worker.CreateJob(models.CreateVolumeAndAttach, map[string]string{
				"size":        os.Getenv("VOLUME_SIZE"),
				"description": os.Getenv("VOLUME_DESCRIPTION"),
				"name":        os.Getenv("VOLUME_NAME"),
				"volume_type": os.Getenv("VOLUME_TYPE"),
				"server_id":   serv.ID,
			}), false)
		}
	}
}

func volnotattached() {
	allVol := utils.GetAllVolumes(context.Background())
	if allVol == nil {
		log.Println("Failed to get all volumes")
		return
	}
	for _, vol := range allVol {
		if len(vol.Attachments) == 0 && vol.Status == "available" && !servstillinuse(vol) {
			worker.AddJob(*worker.CreateJob(models.DeleteVolume, map[string]string{
				"instance_id": vol.ID,
			}), false)
		}
	}
}

func servstillinuse(v volumes.Volume) bool {
	allserv, err := utils.GetAllServers()
	if err != nil {
		log.Println("Failed to get all servers:", err)
		return true
	}
	for _, serv := range allserv {
		if v.Metadata["instance_id"] == serv.ID {
			return true
		}
	}
	return false
}
