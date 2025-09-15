package utils

import (
	"PoolManagerVM/backend/models"
	"fmt"
)

// BuildDataMap construit une map[string]string à partir d'arguments variadiques.
// Usage : BuildDataMap("clé1", "val1", "clé2", "val2", ...)
func BuildDataMap(kv []string) map[string]string {
	if len(kv)%2 != 0 {
		panic("BuildDataMap requires an even number of arguments (clé, valeur)")
	}

	data := make(map[string]string, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key := kv[i]
		value := kv[i+1]
		data[key] = value
	}
	return data
}

func FlatstringParam(p models.Param) []string {
	var flat []string
	flat = append(flat,
		"ID", fmt.Sprint(p.ID),
		"serverpool_id", p.ServerpoolID,
		"user_id", p.UserID,
		"image_ref", p.ImageRef,
		"flavor_ref", p.FlavorRef,
		"networks", fmt.Sprint(p.Networks), // JSONStringSlice, converti en string
		"min_vm", fmt.Sprint(p.MinVM),
		"max_vm", fmt.Sprint(p.MaxVM))
	return flat
}
