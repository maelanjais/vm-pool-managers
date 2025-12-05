package models

import (
	"control_center/frontcontrolpb"
	"control_center/pb"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
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
	AttachVolumeID string
	VolPending     bool `gorm:"default:false; not null"`
	Reattrib       bool `gorm:"default:false; not null"`
	Progress       int  `gorm:"default:0; not null"`
	ConfigID       int
}

func FromGopherServer(s servers.Server) Server {
	var networks []string
	for netName, netAddrs := range s.Addresses {
		for _, addr := range netAddrs.([]any) {
			if addrMap, ok := addr.(map[string]any); ok {
				if ip, ok := addrMap["addr"].(string); ok {
					networks = append(networks, fmt.Sprintf("%s:%s", netName, ip))
				}
			}
		}
	}

	metadata := make(map[string]string)
	for k, v := range s.Metadata {
		metadata[k] = v
	}

	return Server{
		ID:           s.ID,
		Name:         s.Name,
		Status:       s.Status,
		FlavorRef:    s.Flavor["id"].(string),
		ImageRef:     s.Image["id"].(string),
		Networks:     networks,
		Metadata:     metadata,
		ServerpoolID: s.Metadata["serverpool_id"],
		UserID:       s.Metadata["user_id"],
	}
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
		"attach_volume": s.AttachVolumeID,
		"vol_pending":   fmt.Sprintf("%t", s.VolPending),
		"reattrib":      fmt.Sprintf("%t", s.Reattrib),
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
	s.AttachVolumeID = pbs.Data["attach_volume"]

	if v, ok := pbs.Data["vol_pending"]; ok {
		s.VolPending = (v == "true")
	}
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
}

func (s *Server) ToFrontControlPb() *frontcontrolpb.Server {
	metadata := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		metadata[k] = v
	}
	return &frontcontrolpb.Server{
		Id:        s.ID,
		Name:      s.Name,
		Status:    s.Status,
		Image:     s.ImageRef,
		Flavor:    s.FlavorRef,
		Network:   "",
		IpAddress: "",
		CreatedAt: nil,
		UpdatedAt: nil,
		Metadata:  metadata,
	}
}

func PrintServer(server Server) error {
	fmt.Println("=== Server Data ===")
	fmt.Printf("ID: %s\n", server.ID)
	fmt.Printf("Name: %s\n", server.Name)
	fmt.Printf("Status: %s\n", server.Status)
	fmt.Printf("FlavorRef: %s\n", server.FlavorRef)
	fmt.Printf("ImageRef: %s\n", server.ImageRef)
	fmt.Printf("Networks: %+v\n", server.Networks)
	fmt.Printf("Metadata: %+v\n", server.Metadata)
	fmt.Printf("ServerpoolID: %s\n", server.ServerpoolID)
	fmt.Printf("UserID: %s\n", server.UserID)
	return nil
}

func (s *Server) IsEqual(other Server) bool {
	if s.ID != other.ID ||
		s.Name != other.Name ||
		s.Status != other.Status ||
		s.FlavorRef != other.FlavorRef ||
		s.ImageRef != other.ImageRef ||
		s.ServerpoolID != other.ServerpoolID ||
		s.UserID != other.UserID {
		return false
	}

	if len(s.Networks) != len(other.Networks) {
		return false
	}
	for i, v := range s.Networks {
		if v != other.Networks[i] {
			return false
		}
	}

	if len(s.Metadata) != len(other.Metadata) {
		return false
	}
	for k, v := range s.Metadata {
		if otherVal, ok := other.Metadata[k]; !ok || v != otherVal {
			return false
		}
	}

	return true
}
