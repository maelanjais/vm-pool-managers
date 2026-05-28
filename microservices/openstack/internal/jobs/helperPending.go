package jobs

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"
	"log"

	"gorm.io/gorm"
)

func IncrementPending(ServerpoolID uint) {
	result := config.Database.Model(&models.Serverpool{}).
		Where("id = ?", ServerpoolID).
		UpdateColumn("pending_jobs", gorm.Expr("pending_jobs + ?", 1))

	if result.Error != nil {
		log.Println("Error: ", result.Error)
	}
}

func DecrementPending(ServerpoolID uint) {
	result := config.Database.Model(&models.Serverpool{}).
		Where("id = ? AND pending_jobs > 0", ServerpoolID).
		UpdateColumn("pending_jobs", gorm.Expr("pending_jobs - ?", 1))

	if result.Error != nil {
		log.Println("Error: ", result.Error)
	}
}
