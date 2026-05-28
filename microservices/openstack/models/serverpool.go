package models

import (
	"PoolManagerVM/backend/events"
	"PoolManagerVM/backend/notifier"
	"PoolManagerVM/backend/pb"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type Serverpool struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	ServerpoolID string `gorm:"index:idx_pool_user,unique"`
	UserID       string `gorm:"index:idx_pool_user,unique"`
	ImageRef     string
	FlavorRef    string
	Networks     JSONStringSlice `gorm:"type:text"`
	MinVM        int
	MaxVM        int
	PendingJobs  int
	ListServ     []Server `gorm:"foreignKey:ServerpoolID,UserID;references:ServerpoolID,UserID"`
	ConfigID    string
	NetworkUuid string
	TimeStart   string
}

func (sp *Serverpool) ToMap() map[string]string {
	result := map[string]string{
		"id":            fmt.Sprintf("%d", sp.ID),
		"serverpool_id": sp.ServerpoolID,
		"user_id":       sp.UserID,
		"image_ref":     sp.ImageRef,
		"flavor_ref":    sp.FlavorRef,
		"min_vm":        fmt.Sprintf("%d", sp.MinVM),
		"max_vm":        fmt.Sprintf("%d", sp.MaxVM),
		"pending_jobs":  fmt.Sprintf("%d", sp.PendingJobs),
		"config_id":     sp.ConfigID,
	}

	// Sérialiser les champs JSON custom
	if sp.Networks != nil {
		if b, err := json.Marshal(sp.Networks); err == nil {
			result["networks"] = string(b)
		}
	}
	result["host"] = "OpenStack"
	return result
}

func (s *Serverpool) AfterCreate(tx *gorm.DB) (err error) {
	notifier.GlobalChan <- events.RessourceEvent{
		Action:    "created",
		Type:      pb.Type_SERVERPOOL,
		Ressource: *s}
	return nil
}

func (s *Serverpool) AfterUpdate(tx *gorm.DB) (err error) {
	notifier.GlobalChan <- events.RessourceEvent{
		Action:    "updated",
		Type:      pb.Type_SERVERPOOL,
		Ressource: *s}
	return nil
}

func (s *Serverpool) AfterDelete(tx *gorm.DB) (err error) {
	notifier.GlobalChan <- events.RessourceEvent{
		Action:    "deleted",
		Type:      pb.Type_SERVERPOOL,
		Ressource: *s}
	return nil
}
