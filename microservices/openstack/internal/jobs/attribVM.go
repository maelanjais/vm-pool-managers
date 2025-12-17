package jobs

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"gorm.io/gorm"
)

type MetadataUpdate struct {
	Metadata map[string]string
}

func (m MetadataUpdate) ToMetadataUpdateMap() (map[string]any, error) {
	return map[string]any{
		"metadata": m.Metadata,
	}, nil
}

func AttribVM(workerID int, job models.Job) error {

	fmt.Println("Worker ", workerID, " takes the job of attributing a VM")

	allServers, err := utils.GetAllServers()
	if err != nil {
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur récupération des serveurs: %w", err)
	}

	var target *servers.Server
	config.DBmu.Lock()
	println("coucou")
	for i := range allServers {
		srv := &allServers[i]
		if srv.Metadata["user_id"] == "admin" &&
			srv.Metadata["serverpool_id"] == "pool_vms" &&
			!checkReattrib(*srv) {
			target = srv
			updateReattrib(models.FromGopherServer(*target))
			break
		}
	}
	config.DBmu.Unlock()

	if target == nil {
		log.Println("No suitable server found for attribution")
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("aucun serveur cible trouvé")
	}

	newMetadata := MetadataUpdate{
		Metadata: map[string]string{
			"user_id":       job.Data["user_id"],
			"serverpool_id": job.Data["serverpool_id"],
			"min_vm":        job.Data["min_vm"],
			"max_vm":        job.Data["max_vm"],
		},
	}

	newUpdateOpts := servers.UpdateOpts{
		Name: fmt.Sprintf(`%s-%s`,
			job.Data["serverpool_id"], uuid.New().String()),
	}
	_, err = servers.Update(context.Background(),
		models.ComputeClient, target.ID, newUpdateOpts).Extract()
	if err != nil {
		log.Println("Failed to update server name:", err)
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur mise à jour nom serveur: %w", err)
	}

	_, err = servers.UpdateMetadata(context.Background(),
		models.ComputeClient, target.ID, newMetadata).Extract()
	if err != nil {
		log.Println("Failed to update server metadata:", err)
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur mise à jour serveur: %w", err)
	}
	DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
	//dire a la DB que le serveur a ete modifie
	config.DBmu.Lock()
	config.Database.Model(&models.Server{}).Where("id = ?", target.ID).
		Update("serverpool_id", job.Data["serverpool_id"])
	config.Database.Model(&models.Server{}).Where("id = ?", target.ID).
		Update("user_id", job.Data["user_id"])
	time.Sleep(5 * time.Second)
	config.DBmu.Unlock()

	log.Println("Successfully attributed VM", target.ID)

	return nil
}

func checkReattrib(serv servers.Server) bool {
	var s models.Server

	if err := config.Database.Select("reattrib").Where("id = ?", serv.ID).
		First(&s).Error; err != nil {
		log.Println("Error fetching updated reattrib:", err)
		return true
	}
	return s.Reattrib

}

func updateReattrib(serv models.Server) {
	res := config.Database.Model(&models.Server{}).
		Where("id = ?", serv.ID).
		UpdateColumn("reattrib", gorm.Expr("NOT reattrib"))

	if res.Error != nil {
		log.Println("Error: ", res.Error)
	}
}
