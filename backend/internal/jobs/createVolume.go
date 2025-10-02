package jobs

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/volumeattach"
)

func CreateVolumeAndAttach(workerID int, job models.Job) error {

	log.Println("Worker ", workerID, " takes the job to create a volume")

	volumeOpts := volumes.CreateOpts{
		Size:        utils.ParseInt(job.Data["size"]),
		Description: job.Data["description"],
		Name:        job.Data["name"],
		VolumeType:  job.Data["volume_type"],
		Metadata: map[string]string{
			"instance_id": job.Data["server_id"],
		},
	}

	volumeSchedulerHintOpts := volumes.SchedulerHintOpts{}

	newVolume, err := volumes.Create(context.Background(), models.BlockstorageClient, volumeOpts, volumeSchedulerHintOpts).Extract()
	if err != nil {
		log.Println("Failed to create volume:", err)
		log.Println(volumeOpts)
		ChangePendingVol(job.Data["server_id"])
		return err
	}

	for {
		current, errGet := volumes.Get(context.Background(), models.BlockstorageClient, newVolume.ID).Extract()
		if errGet != nil {
			log.Println("Erreur lors de la récupération du volume :", errGet)
			ChangePendingVol(job.Data["server_id"])
			return errGet
		}

		if current.Status == "available" {
			break
		}

		if current.Status == "error" {
			log.Println("error")
			ChangePendingVol(job.Data["server_id"])
			return fmt.Errorf("volume creation failed")
		}

		log.Println("En attente... statut actuel :", current.Status)
		time.Sleep(3 * time.Second)
	}

	createopts := volumeattach.CreateOpts{
		Device:   "/dev/vdc",
		VolumeID: newVolume.ID,
	}

	_, err = volumeattach.Create(context.TODO(), models.ComputeClient, job.Data["server_id"], createopts).Extract()
	if err != nil {
		log.Println("error :", err)
		ChangePendingVol(job.Data["server_id"])
		DeleteVolume(newVolume.ID)
		return err
	}

	result := config.Database.Model(&models.Server{}).
		Where("id = ?", job.Data["server_id"]).
		Update("attach_volume_id", newVolume.ID).Error
	if result != nil {
		ChangePendingVol(job.Data["server_id"])
		return result
	}

	ChangePendingVol(job.Data["server_id"])
	log.Printf("Worker %d completed the job of creating and attaching a volume\n", workerID)

	return nil

}
