package jobs

import (
	"PoolManagerVM/backend/models"
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

// DeleteVM deletes an existing virtual machine (VM) from OpenStack.
//
// Workflow:
//  1. Reads the cloud configuration name from the OPTS_CLOUD environment variable.
//  2. Initializes an OpenStack compute client using the clouds.yaml configuration.
//  3. Sends a delete request for the VM with the given instance ID.
//  4. Returns an error if the environment variable is missing, the client cannot be created,
//     or the deletion request fails.
//
// Parameters:
//   - instanceID: The unique identifier of the VM to be deleted.
//
// Returns:
//   - error: An error if the client setup fails or the VM deletion request fails.
//     Returns nil if the VM is successfully deleted.
func DeleteVM(instanceID string) error {

	// Supprime la VM
	err := servers.Delete(context.Background(), models.ComputeClient, instanceID).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete VM %s: %w", instanceID, err)
	}

	return nil
}
