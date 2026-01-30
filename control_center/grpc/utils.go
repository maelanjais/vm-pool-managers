package grpc

import (
	ccconfig "control_center/config"
	"control_center/internal/sshinject"
	"control_center/models"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

func retryConfigureSSHUserNFS(server *models.Server, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	delay := 5 * time.Second

	for {
		err := configureSSHUserNFS(server)
		if err == nil {
			log.Printf("[SSH][OK] %s configured or already configured\n", server.IP_Address)
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout after %s: %w", timeout, err)
		}

		log.Printf("[SSH][WAIT] %s not ready yet: %v\n", server.IP_Address, err)
		time.Sleep(delay)

		if delay < 30*time.Second {
			delay *= 2
		}
	}
}

func configureSSHUserNFS(server *models.Server) error {
	signer, err := sshinject.LoadPrivateKey(os.Getenv("SSH_PRIVATE_KEY_PATH"))
	if err != nil {
		return err
	}

	config := sshinject.SshConfig("vmuser", signer)

	addr := fmt.Sprintf("%s:22", server.IP_Address)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	defer client.Close()

	var user models.User
	if err := ccconfig.Database.
		Where("email = ?", server.UserID).
		First(&user).Error; err != nil {
		return fmt.Errorf("fetch user failed: %w", err)
	}

	cmd := cmdInit(user)
	if err := sshinject.RunSSHcmd(client, cmd); err != nil {
		return fmt.Errorf("run ssh cmd failed: %w", err)
	}
	return nil
}

func cmdInit(user models.User) string {
	userUsername := sshinject.UsernameFromEmail(user.Email)

	return fmt.Sprintf(`
set -e

MARKER="/var/lib/poolmanager/ssh_config_done"
POOL_MOUNT="/mnt/pool"
POOL_GROUP="pool_prof"

if [ -f "$MARKER" ]; then
  echo "Already configured"
  exit 0
fi

# --- sécurité SSH ---
sudo sed -i 's/^#\\?PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config
sudo systemctl reload ssh || true

ensure_group() {
  if ! getent group "$POOL_GROUP" >/dev/null; then
    sudo groupadd "$POOL_GROUP"
  fi
}

create_user() {
  USERNAME="$1"
  PUBKEY="$2"
  ROLE="$3"

  if ! id "$USERNAME" >/dev/null 2>&1; then
    sudo useradd -m -s /bin/bash "$USERNAME"
  fi

  HOME="/home/$USERNAME"
  SSH="$HOME/.ssh"
  AUTH="$SSH/authorized_keys"

  sudo mkdir -p "$SSH"
  sudo chmod 700 "$SSH"
  sudo chown "$USERNAME:$USERNAME" "$SSH"

  # IMPORTANT : écriture propre de la clé
  echo "$PUBKEY" | sudo tee "$AUTH" > /dev/null

  sudo chmod 600 "$AUTH"
  sudo chown "$USERNAME:$USERNAME" "$AUTH"

  if [ "$ROLE" = "prof" ]; then
    sudo usermod -aG sudo "$USERNAME"
    sudo usermod -aG "$POOL_GROUP" "$USERNAME"
  fi

  # lien vers le pool
  if [ ! -L "$HOME/pool" ]; then
    sudo ln -s "$POOL_MOUNT" "$HOME/pool"
  fi

  sudo chown -h "$USERNAME:$USERNAME" "$HOME/pool"
}

ensure_group

create_user "%s" "%s" "prof"

sudo mkdir -p /var/lib/poolmanager
sudo touch "$MARKER"
sudo chmod 644 "$MARKER"

echo "SSH configuration done"
`, userUsername, user.Keypubuser)
}
