package config

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
)

// global variable to get access to the database anywhere in the code
var (
	Database *gorm.DB
	DBmu     sync.Mutex
)

// boot the database
func Start_DB() {
	var err error
	Database, err = gorm.Open(sqlite.Open("PoolManagerVM.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	Database.AutoMigrate(&models.User{}, &models.Serverpool{}, &models.Server{}, &models.Image{}, &models.Flavor{}, &models.Network{}, &models.VolumeDB{})
}

func SyncImage(ctx context.Context) {
	allImages := utils.GetAllImages(ctx)

	for _, img := range allImages {
		imageRecord := models.Image{
			ID:               img.ID,
			Name:             img.Name,
			Status:           string(img.Status),     // si c’est un type custom
			Visibility:       string(img.Visibility), // idem
			CreatedAt:        img.CreatedAt,
			UpdatedAt:        img.UpdatedAt,
			Owner:            img.Owner,
			Protected:        img.Protected,
			Hidden:           img.Hidden,
			Checksum:         img.Checksum,
			File:             img.File,
			Schema:           img.Schema,
			ContainerFormat:  img.ContainerFormat,
			DiskFormat:       img.DiskFormat,
			MinDiskGigabytes: img.MinDiskGigabytes,
			MinRAMMegabytes:  img.MinRAMMegabytes,
			SizeBytes:        img.SizeBytes,
			VirtualSize:      img.VirtualSize,
			Tags:             strings.Join(img.Tags, ","),
			ImportMethods:    strings.Join(img.OpenStackImageImportMethods, ","),
			StoreIDs:         strings.Join(img.OpenStackImageStoreIDs, ","),
		}

		Database.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&imageRecord)
	}
}

func SyncFlavor(ctx context.Context) {
	allFlavors := utils.GetallFlavors(ctx) // À adapter au bon nom

	for _, fl := range allFlavors {
		var extraSpecsJSON string
		if fl.ExtraSpecs != nil {
			data, _ := json.Marshal(fl.ExtraSpecs)
			extraSpecsJSON = string(data)
		}

		flavorRecord := models.Flavor{
			ID:          fl.ID,
			Name:        fl.Name,
			Disk:        fl.Disk,
			RAM:         fl.RAM,
			VCPUs:       fl.VCPUs,
			RxTxFactor:  fl.RxTxFactor,
			Swap:        fl.Swap,
			Ephemeral:   fl.Ephemeral,
			IsPublic:    fl.IsPublic,
			Description: fl.Description,
			ExtraSpecs:  extraSpecsJSON,
		}

		Database.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&flavorRecord)
	}
}

func SyncNetwork(ctx context.Context) {
	allNetworks := utils.GetAllNetworks(ctx)

	for _, net := range allNetworks {
		networkRecord := models.Network{
			ID:                    net.ID,
			Name:                  net.Name,
			Description:           net.Description,
			AdminStateUp:          net.AdminStateUp,
			Status:                net.Status,
			TenantID:              net.TenantID,
			ProjectID:             net.ProjectID,
			Shared:                net.Shared,
			RevisionNumber:        net.RevisionNumber,
			Subnets:               strings.Join(net.Subnets, ","),
			AvailabilityZoneHints: strings.Join(net.AvailabilityZoneHints, ","),
			Tags:                  strings.Join(net.Tags, ","),
		}

		Database.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&networkRecord)
	}
}

func SyncVolumes(ctx context.Context) {
	allVolumes := utils.GetAllVolumes(ctx)

	for _, vol := range allVolumes {
		volRecord := models.VolumeDB{
			ID:                  vol.ID,
			Status:              vol.Status,
			Size:                vol.Size,
			AvailabilityZone:    vol.AvailabilityZone,
			CreatedAt:           vol.CreatedAt,
			UpdatedAt:           vol.UpdatedAt,
			Name:                vol.Name,
			Description:         vol.Description,
			VolumeType:          vol.VolumeType,
			SnapshotID:          vol.SnapshotID,
			SourceVolID:         vol.SourceVolID,
			BackupID:            vol.BackupID,
			Metadata:            models.JSONStringMap(vol.Metadata),
			UserID:              vol.UserID,
			Bootable:            vol.Bootable,
			Encrypted:           vol.Encrypted,
			ReplicationStatus:   vol.ReplicationStatus,
			ConsistencyGroupID:  vol.ConsistencyGroupID,
			Multiattach:         vol.Multiattach,
			VolumeImageMetadata: models.JSONStringMap(vol.VolumeImageMetadata),
			Host:                vol.Host,
			TenantID:            vol.TenantID,
			Attachments:         models.JSONAttachments(vol.Attachments),
		}

		Database.Clauses(clause.OnConflict{
			UpdateAll: true,
		}).Create(&volRecord)
	}
}

// routine to maintain a cohesive database with the reality on OpenStack
func Sync_DB(ctx context.Context) {
	do_sync()
	first = false
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	count := 12
	for {
		select {
		case <-ctx.Done():
			log.Println("Resync stopped")
			return
		case <-ticker.C:
			DBmu.Lock()
			do_sync()
			delete_serv()
			delete_volumes()
			count++
			if count >= 12 {
				SyncImage(ctx)
				SyncFlavor(ctx)
				SyncNetwork(ctx)
				SyncVolumes(ctx)
				models.CreateParams()
				count = 0
			}
			DBmu.Unlock()
		}
	}
}

// check if server in the database still exist on Openstack, update the database if not
func delete_serv() {
	allserv, err := utils.GetAllServers()
	if err != nil {
		log.Println("Failed to connect to OpenStack, will retry:", err)
		return
	}

	var ops []models.Server
	for _, gs := range allserv {
		ops = append(ops, models.FromGopherServer(gs))
	}

	var dbs []models.Server
	res_servs := Database.Find(&dbs)
	if res_servs.Error != nil {
		log.Println(res_servs.Error)
		return
	}

	for _, s := range dbs {
		found := false
		for _, ts := range ops {
			if s.ID == ts.ID {
				found = true
				break
			}
		}
		if !found {
			result := Database.Delete(&s)
			if result.Error != nil {
				log.Println("Error: can't delete serv: ", result.Error)
			} else {
				log.Println("Server supprimee")
			}
		}
	}
}

func delete_volumes() {
	allvol := utils.GetAllVolumes(context.Background())
	if allvol == nil {
		log.Println("Failed to connect to OpenStack, will retry")
		return
	}

	var dbv []models.VolumeDB
	res_vol := Database.Find(&dbv)
	if res_vol.Error != nil {
		log.Println(res_vol.Error)
		return
	}

	for _, v := range dbv {
		found := false
		for _, opv := range allvol {
			if opv.ID == v.ID {
				found = true
				break
			}
		}
		if !found {
			res := Database.Delete(&v)
			if res.Error != nil {
				log.Println("Error: can't delete volume: ", res.Error)
			} else {
				log.Println("Volume supprimee")
			}
		}
	}
}

var first = true

// synchronize the database to Openstack, creating entries if news instances are made on Openstack
// create pools only if it is the first occurence of the routine
func do_sync() {
	log.Println("Resync !")
	allpool, err := utils.GetAllServerPool()
	if err != nil {
		log.Println("Failed to connect to OpenStack, will retry:", err)
		return
	}

	for _, p := range allpool {
		if first {
			var existed models.Serverpool
			if err := Database.First(&existed, "serverpool_id = ? AND user_id = ?", p.ServerpoolID, p.UserID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					Database.Create(&p)
				} else {
					log.Println("Error Database: ", err)
				}
			}
		}
		for _, s := range p.ListServ {
			var existeds models.Server
			if err := Database.First(&existeds, "id = ?", s.ID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					Database.Create(&s)
				} else {
					log.Println("Error Database param: ", err)
				}
			} else {
				Database.Model(&existeds).Updates(s)
			}
		}
	}
}
