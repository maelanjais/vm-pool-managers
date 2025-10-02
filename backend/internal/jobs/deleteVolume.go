package jobs

import (
	"PoolManagerVM/backend/models"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
)

func DeleteVolume(instanceID string) error {

	deleteopts := volumes.DeleteOpts{}

	err := volumes.Delete(context.Background(), models.BlockstorageClient, instanceID, deleteopts).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete Volume %s: %w", instanceID, err)
	}

	return nil
}
