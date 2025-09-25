package jobs

import (
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
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
	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	}

	provider, err := clientconfig.AuthenticatedClient(context.Background(), opts)
	if err != nil {
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur auth OpenStack: %w", err)
	}

	// 2. Créer un client Compute (Nova)
	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur init ComputeV2: %w", err)
	}

	allServers, err := utils.GetAllServers()
	if err != nil {
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur récupération des serveurs: %w", err)
	}

	var target *servers.Server
	for i := range allServers {
		srv := &allServers[i]
		log.Printf("Checking server ID: %s, Metadata: %v\n", srv.ID, srv.Metadata)
		if srv.Metadata["user_id"] == "admin" && srv.Metadata["serverpool_id"] == "pool_vms" {
			target = srv
			break
		}
	}

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
		Name: fmt.Sprintf(`%s-%s`, job.Data["serverpool_id"], uuid.New().String()),
	}
	_, err = servers.Update(context.Background(), client, target.ID, newUpdateOpts).Extract()
	if err != nil {
		log.Println("Failed to update server name:", err)
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur mise à jour nom serveur: %w", err)
	}

	_, err = servers.UpdateMetadata(context.Background(), client, target.ID, newMetadata).Extract()
	if err != nil {
		log.Println("Failed to update server metadata:", err)
		DecrementPending(uint(utils.ParseInt(job.Data["ID"])))
		return fmt.Errorf("erreur mise à jour serveur: %w", err)
	}
	DecrementPending(uint(utils.ParseInt(job.Data["ID"])))

	return nil
}
