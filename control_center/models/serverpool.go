package models

import (
	"control_center/frontcontrolpb"
	"control_center/pb"
	"encoding/json"
	"fmt"
	"strconv"
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
	ConfigID     string
}

func (sp *Serverpool) FromPb(pbs *pb.StreamRessourceResponse) error {
	data := pbs.Data
	if data == nil {
		return fmt.Errorf("empty data map in StreamRessourceResponse")
	}

	if v, ok := data["id"]; ok && v != "" {
		if id, err := strconv.ParseUint(v, 10, 32); err == nil {
			sp.ID = uint(id)
		} else {
			return fmt.Errorf("invalid id value: %v", err)
		}
	}

	sp.ServerpoolID = data["serverpool_id"]
	sp.UserID = data["user_id"]
	sp.ImageRef = data["image_ref"]
	sp.FlavorRef = data["flavor_ref"]

	if v, ok := data["networks"]; ok && v != "" {
		var networks []string
		if err := json.Unmarshal([]byte(v), &networks); err != nil {
			return fmt.Errorf("error unmarshaling networks: %v", err)
		}
		sp.Networks = networks
	}

	if v, ok := data["min_vm"]; ok && v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			sp.MinVM = val
		}
	}
	if v, ok := data["max_vm"]; ok && v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			sp.MaxVM = val
		}
	}
	if v, ok := data["pending_jobs"]; ok && v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			sp.PendingJobs = val
		}
	}
	sp.ConfigID = data["config_id"]

	return nil
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

	if sp.Networks != nil {
		if b, err := json.Marshal(sp.Networks); err == nil {
			result["networks"] = string(b)
		}
	}
	result["host"] = "OpenStack"
	return result
}

func (sp *Serverpool) ToFrontControlPb() *frontcontrolpb.ServerPool {
	var network string
	if len(sp.Networks) > 0 {
		network = sp.Networks[0]
	}

	return &frontcontrolpb.ServerPool{
		Id:       strconv.FormatUint(uint64(sp.ID), 10),
		Name:     sp.ServerpoolID,
		Image:    sp.ImageRef,
		Flavor:   sp.FlavorRef,
		Network:  network,
		Config:   sp.ConfigID,
		MinVm:    int32(sp.MinVM),
		MaxVm:    int32(sp.MaxVM),
		Metadata: map[string]string{},
	}
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
