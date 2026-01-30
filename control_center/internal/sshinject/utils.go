package sshinject

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func LoadPrivateKey(path string) (ssh.Signer, error) {
	// log.Printf("path: %s\n", path)
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(key)
}

func SshConfig(user string, signer ssh.Signer) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
}

func RunSSHcmd(client *ssh.Client, cmd string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		log.Printf("SSH stdout: %s", stdout.String())
		log.Printf("SSH stderr: %s", stderr.String())
		if stderr.Len() > 0 {
			return fmt.Errorf("ssh command error: %s", stderr.String())
		}
		return fmt.Errorf("ssh command failed: %w", err)
	}

	return nil
}

func UsernameFromEmail(email string) string {
	local := strings.Split(email, "@")[0]
	local = strings.ToLower(local)

	// remplacer caractères interdits
	re := regexp.MustCompile(`[^a-z0-9_.-]`)
	local = re.ReplaceAllString(local, "")

	if len(local) > 32 {
		local = local[:32]
	}

	return local
}
