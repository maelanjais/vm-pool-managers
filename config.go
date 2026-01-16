package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
)

/*
UTILS
*/

func writeEnvFile(path string, vars map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for k, v := range vars {
		if _, err := f.WriteString(fmt.Sprintf("%s=%s\n", k, v)); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

/*
MAIN
*/

func main() {
	home := os.Getenv("HOME")

	/*
		--------------------------------
		SSH CONFIG (GLOBAL)
		--------------------------------
	*/
	sshPrivateKeyPath := home + "/.ssh/id_ed25519"
	sshPublicKeyPath := home + "/.ssh/id_ed25519.pub"

	sshForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Chemin de la clé SSH PRIVÉE").
				Description("Ex: ~/.ssh/id_ed25519").
				Value(&sshPrivateKeyPath).
				Validate(func(v string) error {
					if !fileExists(v) {
						return fmt.Errorf("clé privée introuvable")
					}
					return nil
				}),

			huh.NewInput().
				Title("Chemin de la clé SSH PUBLIQUE").
				Description("Ex: ~/.ssh/id_ed25519.pub").
				Value(&sshPublicKeyPath).
				Validate(func(v string) error {
					if !fileExists(v) {
						return fmt.Errorf("clé publique introuvable")
					}
					return nil
				}),
		),
	)

	if err := sshForm.Run(); err != nil {
		log.Fatal(err)
	}

	/*
		--------------------------------
		CONTROL CENTER
		--------------------------------
	*/
	ccUser := "admin"
	ccPassword := ""

	ccForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Controle Center - Postgres User").
				Value(&ccUser),

			huh.NewInput().
				Title("Controle Center - Postgres Password").
				EchoMode(huh.EchoModePassword).
				Value(&ccPassword),
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
		"SSH_PUBLIC_KEY_PATH":  sshPublicKeyPath,
		"SSH_PRIVATE_KEY_PATH": sshPrivateKeyPath,
	}

	ccEnvPath := filepath.Join("control_center", ".env")
	if err := writeEnvFile(ccEnvPath, ccEnv); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ control_center/.env généré")

	/*
		--------------------------------
		OPENSTACK
		--------------------------------
	*/
	var apiKeyName, optsCloud, secretJWT string

	osForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("OpenStack - API_KEYNAME").
				Value(&apiKeyName),

			huh.NewInput().
				Title("OpenStack - OPTS_CLOUD").
				Value(&optsCloud),

			huh.NewInput().
				Title("OpenStack - SECRET_KEY_JWT").
				EchoMode(huh.EchoModePassword).
				Value(&secretJWT),
		),
	)

	if err := osForm.Run(); err != nil {
		log.Fatal(err)
	}

	osEnv := map[string]string{
		// Server
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

		// Network
		"NETWORK_ID": "39aa90ca-163b-4630-9671-9439fefe516f",

		// Volume
		"VOLUME_SIZE":        "1",
		"VOLUME_DESCRIPTION": "test",
		"VOLUME_NAME":        "test",
		"VOLUME_TYPE":        "__DEFAULT__",

		// Secrets
		"API_KEYNAME":    apiKeyName,
		"OPTS_CLOUD":     optsCloud,
		"SECRET_KEY_JWT": secretJWT,

		// SSH
		"SSH_PUBLIC_KEY_PATH": sshPublicKeyPath,
	}

	osEnvPath := filepath.Join("openstack", ".env")
	if err := writeEnvFile(osEnvPath, osEnv); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ openstack/.env généré")

	/*
		--------------------------------
		ROOT .env (Taskfile)
		--------------------------------
	*/
	rootEnv := map[string]string{
		"CC_ENV_FILE": ccEnvPath,
		"OS_ENV_FILE": osEnvPath,
	}

	if err := writeEnvFile(".env", rootEnv); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ .env racine généré")

	fmt.Println("\n🎉 Configuration terminée avec succès")
}
