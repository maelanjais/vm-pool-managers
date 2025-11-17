package models

import (
	"PoolManagerVM/backend/events"
	"PoolManagerVM/backend/notifier"
	"PoolManagerVM/backend/pb"
	"PoolManagerVM/backend/websockethandler"
	"encoding/json"
	"fmt"
	"log"

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
	ConfigID     int
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
		"config_id":     fmt.Sprintf("%d", sp.ConfigID),
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

func PrintServerpool(sp Serverpool) error {
	fmt.Println("=== Serverpool Data ===")
	fmt.Println("ID: ", sp.ID)
	fmt.Println("ServerpoolID: ", sp.ServerpoolID)
	fmt.Println("UserID: ", sp.UserID)
	fmt.Println("ImageRef: ", sp.ImageRef)
	fmt.Println("FlavorRef: ", sp.FlavorRef)
	fmt.Println("Networks: ", sp.Networks)
	fmt.Println("MinVM: ", sp.MinVM)
	fmt.Println("MaxVm: ", sp.MaxVM)
	fmt.Println("PendingJobs: ", sp.PendingJobs)
	fmt.Println("ConfigID: ", sp.ConfigID)
	for _, s := range sp.ListServ {
		PrintServer(s)
	}

	return nil
}

func PrintMapServerpool(m []Serverpool) error {
	fmt.Println("=== Print Map Serverpool ===")
	for _, p := range m {
		PrintServerpool(p)
		fmt.Println("=====================================")
	}
	return nil
}

func (s *Serverpool) AfterCreate(tx *gorm.DB) (err error) {
	websockethandler.SendMessageToUser(s.UserID, "created", s, "serverpool")
	notifier.GlobalChan <- events.RessourceEvent{Action: "created", Type: pb.Type_SERVERPOOL, Ressource: *s}
	return nil
}

func (s *Serverpool) AfterUpdate(tx *gorm.DB) (err error) {
	websockethandler.SendMessageToUser(s.UserID, "updated", s, "serverpool")
	notifier.GlobalChan <- events.RessourceEvent{Action: "updated", Type: pb.Type_SERVERPOOL, Ressource: *s}
	return nil
}

func (s *Serverpool) AfterDelete(tx *gorm.DB) (err error) {
	log.Println("Sending delete message to user:", s.UserID, "for serverpool:", s.ServerpoolID)
	websockethandler.SendMessageToUser(s.UserID, "deleted", s, "serverpool")
	notifier.GlobalChan <- events.RessourceEvent{Action: "deleted", Type: pb.Type_SERVERPOOL, Ressource: *s}
	return nil
}
