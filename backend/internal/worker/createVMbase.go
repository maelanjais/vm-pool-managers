package worker

import (
	"PoolManagerVM/backend/internal/jobs"
	"PoolManagerVM/backend/models"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

func CreateVMbase(cfg models.Config) error {
	opts := &clientconfig.ClientOpts{
		Cloud: "idcs-stage-dev@stratuslab.production.virtualdata",
	}

	client, err := clientconfig.NewServiceClient("compute", opts)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	createOpts := servers.CreateOpts{
		Name:      fmt.Sprintf(`%s-%s`, cfg.Server.Name, uuid.New().String()),
		FlavorRef: cfg.Server.FlavorRef,
		ImageRef:  cfg.Server.ImageRef,
		Networks:  []servers.Network{{UUID: cfg.Network.NetworkID}},
		Metadata: map[string]string{
			"serverpool-id": cfg.Metadata["serverpool-id"],
			"minVM":         cfg.Metadata["minVM"],
			"maxVM":         cfg.Metadata["maxVm"],
			"userID":        cfg.Metadata["userID"],
		},
	}

	createOptsExt := keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           cfg.Server.Keyname,
	}

	server, err := servers.Create(client, createOptsExt).Extract()
	if err != nil {
		return fmt.Errorf("failed to create VM : %w", err)
	}

	jobs.DecrementPending("admin")
	log.Printf("[VM] Creating server ID=%s Name=%s\n", server.ID, server.Name)

	// Waiting for server to start
	for {
		current, err := servers.Get(client, server.ID).Extract()
		if err != nil {
			return fmt.Errorf("failed to get server status: %w", err)
		}

		if current.Status == "ACTIVE" {
			log.Printf("[VM] Server %s is ACTIVE\n", current.ID)
			break
		}

		if current.Status == "ERROR" {
			return fmt.Errorf("server %s failed to boot (ERROR state)", current.ID)
		}

		log.Printf("[VM] Waiting for server %s (status=%s)\n", current.ID, current.Status)
		time.Sleep(3 * time.Second)
	}

	return nil
}
