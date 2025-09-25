package config

import (
	"context"
	"errors"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"PoolManagerVM/backend/models"
	"PoolManagerVM/backend/utils"
)

// global variable to get access to the database anywhere in the code
var Database *gorm.DB

// boot the database
func Start_DB() {
	var err error
	Database, err = gorm.Open(sqlite.Open("PoolManagerVM.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	Database.AutoMigrate(&models.User{}, &models.Serverpool{}, &models.Server{})
}

// routine to maintain a cohesive database with the reality on OpenStack
func Sync_DB(ctx context.Context) {
	do_sync()
	first = false
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("Resync stopped")
			return
		case <-ticker.C:
			do_sync()
			delete_serv()
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
				log.Println("Ligne supprimee")
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

//quand passage a Postgres, utiliser :
// Database.Clauses(clause.OnConflict{
//     Columns:   []clause.Column{{Name: "serverpool_id"}, {Name: "user_id"}},
//     UpdateAll: true, // fait un UPDATE si conflit
// }).Create(&sp)

// Database.Clauses(clause.OnConflict{
//     Columns:   []clause.Column{
//         {Name: "serverpool_id"},
//         {Name: "user_id"},
//         {Name: "image_ref"},
//         {Name: "flavor_ref"},
//     },
//     UpdateAll: true,
// }).Create(&paramsFromOpenStack)
