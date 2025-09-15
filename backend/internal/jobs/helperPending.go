package jobs

import (
	"PoolManagerVM/backend/config"
	"PoolManagerVM/backend/models"
	"log"

	"gorm.io/gorm"
)

func IncrementPending(paramID uint) {
	result := config.Database.Model(&models.Param{}).
		Where("id = ?", paramID).
		UpdateColumn("pending_jobs", gorm.Expr("pending_jobs + ?", 1))

	if result.Error != nil {
		log.Println("Error: ", result.Error)
	}
}

func DecrementPending(paramID uint) {
	result := config.Database.Model(&models.Param{}).
		Where("id = ?", paramID).
		UpdateColumn("pending_jobs", gorm.Expr("pending_jobs - ?", 1))

	if result.Error != nil {
		log.Println("Error: ", result.Error)
	}
}
