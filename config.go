package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

// writeEnvFile crée ou remplace un fichier .env avec les variables fournies
func writeEnvFile(path string, vars map[string]string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	for k, v := range vars {
		_, err := file.WriteString(fmt.Sprintf("%s=%s\n", k, v))
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	// -----------------
	// Formulaire Controle Center
	// -----------------
	// Valeur par défaut directement dans la variable
	ccUser := "admin"
	ccPassword := ""

	// Création du formulaire sans Default()
	ccForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Control Center - Postgres User").Value(&ccUser),
			huh.NewInput().Title("Control Center - Postgres Password").Value(&ccPassword),
		),
	)

	if err := ccForm.Run(); err != nil {
		log.Fatal(err)
	}

	ccEnv := map[string]string{
		"POSTGRES_HOST":        "postgres",
		"POSTGRES_PORT":        "5432",
		"POSTGRES_USER":        ccUser,
		"POSTGRES_PASSWORD":    ccPassword,
		"POSTGRES_DB":          "control_center",
		"CONTROL_CENTER_PORT":  "50051",
		"SSH_PUBLIC_KEY_PATH":  os.Getenv("HOME") + "/.ssh/id_ed25519.pub",
		"SSH_PRIVATE_KEY_PATH": os.Getenv("HOME") + "/.ssh/id_ed25519",
	}

	ccEnvPath := filepath.Join("control_center", ".env")
	if err := writeEnvFile(ccEnvPath, ccEnv); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Fichier .env control_center créé")

	// -----------------
	// Formulaire OpenStack
	// -----------------
	var osAPIKey, osOptsCloud, osSecretJWT string

	osForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("OpenStack - API_KEYNAME").Value(&osAPIKey),
			huh.NewInput().Title("OpenStack - OPTS_CLOUD").Value(&osOptsCloud),
			huh.NewInput().Title("OpenStack - SECRET_KEY_JWT").Value(&osSecretJWT),
		),
	)

	if err := osForm.Run(); err != nil {
		log.Fatal(err)
	}

	osEnv := map[string]string{
		// Server base config
		"SERVER_NAME":            "test",
		"SERVER_IMAGE_REF":       "72d05dc3-73ec-405b-b870-48f70782526f",
		"SERVER_FLAVOR_REF":      "a69d1ae1-74cf-4750-8f8c-a621a39f8e24",
		"SERVER_UUID":            "94e241e7-77d3-46e2-b6b7-f87d97d0a28e",
		"SERVER_SECURITY_GROUPS": "default",
		"SERVER_SUBNET":          "fda5f48f-a66e-43a5-8c39-24c598159fe9",

		// Metadata
		"METADATA_SERVERPOOL_ID": "pool_vms",
		"METADATA_USER_ID":       "admin",
		"METADATA_MIN_VM":        "2",
		"METADATA_MAX_VM":        "9",

		// Networks
		"NETWORK_ID": "39aa90ca-163b-4630-9671-9439fefe516f",

		// Volume
		"VOLUME_SIZE":        "1",
		"VOLUME_DESCRIPTION": "test",
		"VOLUME_NAME":        "test",
		"VOLUME_TYPE":        "__DEFAULT__",

		// Keys
		"API_KEYNAME":    osAPIKey,
		"OPTS_CLOUD":     osOptsCloud,
		"SECRET_KEY_JWT": osSecretJWT,

		// SSH Key
		"SSH_PUBLIC_KEY_PATH": os.Getenv("HOME") + "/.ssh/id_ed25519.pub",
	}

	osEnvPath := filepath.Join("microservices/openstack", ".env")
	if err := writeEnvFile(osEnvPath, osEnv); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Fichier .env openstack créé")

	// -----------------
	// Racine .env pour Taskfile
	// -----------------
	rootEnv := map[string]string{
		"CC_ENV_FILE": ccEnvPath,
		"OS_ENV_FILE": osEnvPath,
	}
	rootEnvPath := ".env"
	if err := writeEnvFile(rootEnvPath, rootEnv); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Fichier .env racine créé")
}
