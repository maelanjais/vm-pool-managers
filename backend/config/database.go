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

var Database *gorm.DB

func Start_DB() {
	var err error
	Database, err = gorm.Open(sqlite.Open("PoolManagerVM.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	Database.AutoMigrate(&models.User{}, &models.Serverpool{}, &models.Param{}, &models.Server{})
}

func Sync_DB(ctx context.Context) {
	do_sync()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("Resync stopped")
			return
		case <-ticker.C:
			do_sync()
		}
	}
}

func delete_serv() {
	allserv, err := utils.GetAllServers()
	if err != nil {
		panic("failed to connect to Openstack")
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

func do_sync() {
	// log.Println("Resync !")
	allpool, err := utils.GetAllServerPool()
	if err != nil {
		panic("failed to connect to OpenStack")
	}

	for _, p := range allpool {
		var existed models.Serverpool
		if err := Database.First(&existed, "serverpool_id = ? AND user_id = ?", p.ServerpoolID, p.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				Database.Create(&p)
			} else {
				log.Println("Error Database: ", err)
			}
		} else {
			Database.Model(&existed).Updates(p)
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
		for _, param := range p.Params {
			var existedp models.Param
			if err := Database.First(&existedp, "serverpool_id = ? AND user_id = ? AND flavor_ref = ? AND image_ref = ?", param.ServerpoolID, param.UserID, param.FlavorRef, param.ImageRef).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					Database.Create(&param)
				} else {
					log.Println("Error Database param: ", err)
				}
			} else {
				Database.Model(&existedp).Updates(param)
			}
		}
	}
	delete_serv()
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
