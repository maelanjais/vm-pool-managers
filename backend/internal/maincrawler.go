package internal

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/internal/jobs"
	"PoolManagerVM/backend/internal/worker"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"log"
	"os"
	"strconv"
	"time"
)

func Monitor(c context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Done():
			log.Println("Monitoring stopped")
			return

		case <-ticker.C:
			log.Println("Checking serverpools...")
			CheckAndCreate()
		}
	}
}

// Look in the DB if some servers are missing, create job to add news ones
func CheckAndCreate() {

	var (
		servs  []models.Server
		params []models.Param
		pools  []models.Serverpool
	)
	res_servs := config.Database.Find(&servs)
	if res_servs.Error != nil {
		log.Println(res_servs.Error)
	}
	res_params := config.Database.Find(&params)
	if res_params.Error != nil {
		log.Println(res_params.Error)
	}
	res_pools := config.Database.Find(&pools)
	if res_pools.Error != nil {
		log.Println(res_pools.Error)
	}

	for _, p := range params {
		count := 0
		for _, s := range servs {
			if serverisinparams(p, s) {
				count++
			}
		}
		missing := p.MinVM - (count + p.PendingJobs)
		for i := 0; i < missing; i++ {
			worker.AddJob(*worker.CreateJob(worker.CreateVM, utils.BuildDataMap(utils.FlatstringParam(p))), false)
			jobs.IncrementPending(p.ID)
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
		for i := 0; i < base_p.MinVM; i++ {
			worker.AddJob(*worker.CreateJob(worker.CreateVM, utils.BuildDataMap(utils.FlatstringParam(base_p))), false)
			jobs.IncrementPending(base_p.ID)
		}
	}
}

func serverisinparams(p models.Param, s models.Server) bool {
	if s.ServerpoolID == p.ServerpoolID && s.UserID == p.UserID && s.FlavorRef == p.FlavorRef && s.ImageRef == p.ImageRef {
		return true
	} else {
		return false
	}
}

func CreateServerpoolFromEnv() (models.Param, error) {
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
		return models.Param{}, err
	}
	maxVM, err := strconv.Atoi(maxVMStr)
	if err != nil {
		return models.Param{}, err
	}

	// Construire le param
	param := models.Param{
		ServerpoolID: poolID,
		UserID:       userID,
		ImageRef:     imageRef,
		FlavorRef:    flavorRef,
		Networks:     models.JSONStringSlice{os.Getenv("NETWORK_ID")},
		MinVM:        minVM,
		MaxVM:        maxVM,
		PendingJobs:  0,
	}

	return param, nil
}
