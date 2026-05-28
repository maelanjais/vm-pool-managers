package models

import (
	"control_center/frontcontrolpb"
	"control_center/pb"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Server struct {
	ID             string `gorm:"primaryKey"`
	Name           string
	Status         string
	FlavorRef      string
	ImageRef       string
	Networks       JSONStringSlice `gorm:"type:text"`
	Metadata       JSONStringMap   `gorm:"type:text"`
	ServerpoolID   string
	UserID         string
	Reattrib bool `gorm:"default:false; not null"`
	Progress       int  `gorm:"default:0; not null"`
	ConfigID       int
	IP_Address     string
	Locked         bool `gorm:"default:false; not null"`
	SshKeyAssigned string
	Configured     bool `gorm:"default:false; not null"`
	PendingConf    bool `gorm:"default:false; not null"`
}

func (s *Server) ToMap() map[string]string {
	result := map[string]string{
		"id":            s.ID,
		"name":          s.Name,
		"status":        s.Status,
		"flavor_ref":    s.FlavorRef,
		"image_ref":     s.ImageRef,
		"serverpool_id": s.ServerpoolID,
		"user_id":       s.UserID,
		"reattrib": fmt.Sprintf("%t", s.Reattrib),
		"progress":      fmt.Sprintf("%d", s.Progress),
		"config_id":     fmt.Sprintf("%d", s.ConfigID),
	}

	if s.Networks != nil {
		if b, err := json.Marshal(s.Networks); err == nil {
			result["networks"] = string(b)
		}
	}
	if s.Metadata != nil {
		if b, err := json.Marshal(s.Metadata); err == nil {
			result["metadata"] = string(b)
		}
	}

	result["host"] = "OpenStack"
	return result
}

func (s *Server) FromPb(pbs *pb.StreamRessourceResponse) {
	s.ID = pbs.Data["id"]
	s.Name = pbs.Data["name"]
	s.Status = pbs.Data["status"]
	s.FlavorRef = pbs.Data["flavor_ref"]
	s.ImageRef = pbs.Data["image_ref"]
	s.ServerpoolID = pbs.Data["serverpool_id"]
	s.UserID = pbs.Data["user_id"]
	if v, ok := pbs.Data["reattrib"]; ok {
		s.Reattrib = (v == "true")
	}
	if v, ok := pbs.Data["progress"]; ok {
		if num, err := strconv.Atoi(v); err == nil {
			s.Progress = num
		}
	}
	if v, ok := pbs.Data["config_id"]; ok {
		if num, err := strconv.Atoi(v); err == nil {
			s.ConfigID = num
		}
	}
	if v, ok := pbs.Data["networks"]; ok && v != "" {
		var arr []string
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			s.Networks = arr
		}
	}
	if v, ok := pbs.Data["metadata"]; ok && v != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(v), &m); err == nil {
			s.Metadata = m
		}
	}
	if v, ok := pbs.Data["ip_address"]; ok {
		s.IP_Address = v
	}
}

func (s *Server) ToFrontControlPb() *frontcontrolpb.Server {
	metadata := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		metadata[k] = v
	}

	networkStr := ""

	if len(s.Networks) > 0 {
		networkStr = s.Networks[0]

		if len(s.Networks) > 1 {
			networkStr = strings.Join(s.Networks, ",")
		}
	}

	return &frontcontrolpb.Server{
		Id:          s.ID,
		Name:        s.Name,
		Status:      s.Status,
		Image:       s.ImageRef,
		Flavor:      s.FlavorRef,
		Network:     networkStr,
		IpAddress:   s.IP_Address,
		CreatedAt:   nil,
		UpdatedAt:   nil,
		Metadata:    metadata,
		UserId:      s.UserID,
		AddressedIp: s.IP_Address,
	}
}

