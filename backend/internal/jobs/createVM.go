package jobs

import (
	"PoolManagerVM/backend/models"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/utils/openstack/clientconfig"
)

func CreateVM(serv models.Server, paramID uint) error {

	opts := &clientconfig.ClientOpts{
		Cloud: os.Getenv("OPTS_CLOUD"),
	}

	client, err := clientconfig.NewServiceClient("compute", opts)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	createOpts := servers.CreateOpts{
		Name:      fmt.Sprintf(`%s-%s`, serv.Name, uuid.New().String()),
		FlavorRef: serv.FlavorRef,
		ImageRef:  serv.ImageRef,
		Metadata:  serv.Metadata,
		Networks:  []servers.Network{{UUID: os.Getenv("NETWORK_ID")}},
	}

	createOptsExt := keypairs.CreateOptsExt{
		CreateOptsBuilder: createOpts,
		KeyName:           os.Getenv("API_KEYNAME"),
	}

	server, err := servers.Create(client, createOptsExt).Extract()
	if err != nil {
		log.Println("failed to create VM:", err)
		return fmt.Errorf("failed to create VM: %w", err)
	}

	DecrementPending(paramID)
	log.Println("[VM] Creating server ID=", server.ID, " , Name=", server.Name)

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
