package models

import (
	"PoolManagerVM/backend/events"
	"PoolManagerVM/backend/notifier"
	"PoolManagerVM/backend/pb"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"gorm.io/gorm"
)

type Server struct {
	ID             string `gorm:"primaryKey"`
	Name           string
	Status         string
	FlavorRef      string
	ImageRef       string
	Networks       JSONStringSlice `gorm:"type:text"`
	IP_Address     string
	Metadata       JSONStringMap `gorm:"type:text"`
	ServerpoolID   string
	UserID         string
	ServerPool     *Serverpool `gorm:"foreignKey:ServerpoolID,UserID;references:ServerpoolID,UserID"`
	Reattrib bool `gorm:"default:false; not null"`
	Progress       int  `gorm:"default:0; not null"`
	ConfigID       string
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
		"config_id":     s.ConfigID,
		"ip_address":    s.IP_Address,
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

func FromGopherServer(s servers.Server) Server {
	var networks []string
	var ipaddr string
	for netName, netAddrs := range s.Addresses {
		for _, addr := range netAddrs.([]any) {
			if addrMap, ok := addr.(map[string]any); ok {
				if ip, ok := addrMap["addr"].(string); ok {
					networks = append(networks, fmt.Sprintf("%s:%s", netName, ip))
					ipaddr = ip
				}
			}
		}
	}
	if raw, ok := s.Metadata["network_uuid"]; ok {
		var arr []string
		if err := json.Unmarshal([]byte(raw), &arr); err == nil && len(arr) > 0 {
			networks = []string{arr[0]}
		}
	}
	metadata := make(map[string]string)
	maps.Copy(metadata, s.Metadata)
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
		IP_Address:   ipaddr,
	}
}

func (s *Server) AfterCreate(tx *gorm.DB) (err error) {
	notifier.GlobalChan <- events.RessourceEvent{
		Action:    "created",
		Type:      pb.Type_SERVER,
		Ressource: *s}
	return nil
}

func (s *Server) AfterUpdate(tx *gorm.DB) (err error) {
	notifier.GlobalChan <- events.RessourceEvent{
		Action:    "updated",
		Type:      pb.Type_SERVER,
		Ressource: *s}
	return nil
}

func (s *Server) AfterDelete(tx *gorm.DB) (err error) {
	notifier.GlobalChan <- events.RessourceEvent{
		Action:    "deleted",
		Type:      pb.Type_SERVER,
		Ressource: *s}
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
