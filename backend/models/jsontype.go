package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

type JSONStringSlice []string
type JSONStringMap map[string]string
type JSONAttachments []volumes.Attachment

// -------- JSONStringSlice --------
func (j *JSONStringSlice) Scan(value any) error {
	if value == nil {
		*j = JSONStringSlice{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSONStringSlice: %v", value)
	}

	if len(bytes) == 0 {
		*j = JSONStringSlice{}
		return nil
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONStringSlice) Value() (driver.Value, error) {
	if j == nil {
		return "[]", nil
	}
	return json.Marshal(j)
}

// -------- JSONStringMap --------
func (j *JSONStringMap) Scan(value any) error {
	if value == nil {
		*j = JSONStringMap{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSONStringMap: %v", value)
	}

	if len(bytes) == 0 {
		*j = JSONStringMap{}
		return nil
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONStringMap) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

func (j JSONStringSlice) ToNetworks() []servers.Network {
	nets := make([]servers.Network, 0, len(j))
	for _, id := range j {
		nets = append(nets, servers.Network{UUID: id})
	}
	log.Println("j:", j, "nets:", nets)
	return nets
}

// -------- JSONAttachments --------

func (j *JSONAttachments) Scan(value any) error {
	if value == nil {
		*j = JSONAttachments{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSONAttachments: %v", value)
	}

	if len(bytes) == 0 {
		*j = JSONAttachments{}
		return nil
	}

	return json.Unmarshal(bytes, j)
}

func (j JSONAttachments) Value() (driver.Value, error) {
	if j == nil {
		return "[]", nil
	}
	return json.Marshal(j)
}
