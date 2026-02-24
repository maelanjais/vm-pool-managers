package rclone

import (
	"bytes"
	"control_center/internal/sshinject"
	"control_center/models"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

//
// ===========================
// Utilities
// ===========================
//

func runLocalCmd(cmd string) error {
	log.Printf("runLocalCmd: executing\n%s", cmd)

	c := exec.Command("bash", "-c", cmd)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	log.Printf("STDOUT:\n%s\nSTDERR:\n%s", stdout.String(), stderr.String())

	if err != nil {
		return fmt.Errorf("runLocalCmd failed: %w", err)
	}
	return nil
}

func RunSSHcmd(client *ssh.Client, cmd string) error {
	log.Println("runSSHcmd: executing\n%s", cmd)
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("new ssh session failed: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		log.Printf("SSH stdout: %s\nSSH stderr: %s", stdout.String(), stderr.String())
		return fmt.Errorf("ssh command failed: %w", err)
	}
	return nil
}

func RunSSHcmdWithOutput(client *ssh.Client, cmd string) (string, error) {
	log.Println("runSSHcmdWithOutput: executing\n%s", cmd)
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create ssh session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("ssh command failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func UsernameFromEmail(email string) string {
	local := strings.Split(email, "@")[0]
	local = strings.ToLower(local)
	re := regexp.MustCompile(`[^a-z0-9_.-]`)
	local = re.ReplaceAllString(local, "")
	if len(local) > 32 {
		local = local[:32]
	}
	return local
}

//
// ===========================
// Prof functions
// ===========================
//

func CreateProfLocal(prof models.User) error {
	username := UsernameFromEmail(prof.Email)

	cmd := fmt.Sprintf(`
set -e
if ! id "%[1]s" >/dev/null 2>&1; then
	sudo useradd -m -s /bin/bash "%[1]s"
fi

sudo install -d -m 700 -o "%[1]s" -g "%[1]s" /home/%[1]s/.ssh
sudo touch /home/%[1]s/.ssh/authorized_keys
sudo chown "%[1]s":"%[1]s" /home/%[1]s/.ssh/authorized_keys
sudo chmod 600 /home/%[1]s/.ssh/authorized_keys

if ! sudo grep -qF "%[2]s" /home/%[1]s/.ssh/authorized_keys; then
	echo "%[2]s" | sudo tee -a /home/%[1]s/.ssh/authorized_keys > /dev/null
fi
`, username, prof.Keypubuser)

	return runLocalCmd(cmd)
}

func CreatePoolLocal(profName, poolName string) error {
	profUsername := UsernameFromEmail(profName)

	cmd := fmt.Sprintf(`
set -e
POOL_DIR="/home/%[1]s/%[2]s"

sudo install -d -m 750 -o "%[1]s" -g "%[1]s" "$POOL_DIR"

# mask + héritage
sudo setfacl -m m:rwx "$POOL_DIR"
sudo setfacl -d -m m:rwx "$POOL_DIR"
`, profUsername, poolName)

	return runLocalCmd(cmd)
}

func DeletePool(profName, poolName string) error {
	profUsername := UsernameFromEmail(profName)

	cmd := fmt.Sprintf(`
set -e
sudo rm -rf /home/%[1]s/%[2]s
`, profUsername, poolName)

	return runLocalCmd(cmd)
}

//
// ===========================
// Student functions
// ===========================
//

func CreateUserLocal(user string) error {
	username := UsernameFromEmail(user)

	cmd := fmt.Sprintf(`
set -e
if ! id "%[1]s" >/dev/null 2>&1; then
	sudo useradd -m -s /bin/bash "%[1]s"
fi
`, username)

	return runLocalCmd(cmd)
}

func AuthorizeStudentKey(studentName, pubKey string) error {
	username := UsernameFromEmail(studentName)

	cmd := fmt.Sprintf(`
set -e
SSH_DIR="/home/%[1]s/.ssh"

sudo install -d -m 700 -o "%[1]s" -g "%[1]s" "$SSH_DIR"
sudo touch "$SSH_DIR/authorized_keys"
sudo chown "%[1]s":"%[1]s" "$SSH_DIR/authorized_keys"
sudo chmod 600 "$SSH_DIR/authorized_keys"

if ! sudo grep -qF "%[2]s" "$SSH_DIR/authorized_keys"; then
	echo "%[2]s" | sudo tee -a "$SSH_DIR/authorized_keys" > /dev/null
fi
`, username, pubKey)

	return runLocalCmd(cmd)
}

func AddStudentToPool(profName, studentName, poolName string) error {
	profUsername := UsernameFromEmail(profName)
	studentUsername := UsernameFromEmail(studentName)

	cmd := fmt.Sprintf(`
set -e
BASE="/home/%[1]s/%[2]s"
STUD_DIR="$BASE/%[3]s"

sudo install -d -m 750 -o "%[1]s" -g "%[1]s" "$STUD_DIR"

# permettre traversal du home du prof
sudo setfacl -m u:%[3]s:x /home/%[1]s

# accès lecture au pool
sudo setfacl -m u:%[3]s:rx "$BASE"

# accès complet au dossier étudiant pour student
sudo setfacl -m u:%[3]s:rwx "$STUD_DIR"

# le prof doit pouvoir lire ce que le student crée
sudo setfacl -m u:%[1]s:rwx "$STUD_DIR"

# ACL par défaut (héritage)
sudo setfacl -d -m u:%[3]s:rwx "$STUD_DIR"
sudo setfacl -d -m u:%[1]s:rwx "$STUD_DIR"

# mask (important sinon les ACL sont réduites)
sudo setfacl -m m:rwx "$STUD_DIR"
sudo setfacl -d -m m:rwx "$STUD_DIR"
`, profUsername, poolName, studentUsername)

	return runLocalCmd(cmd)
}



func RemoveStudentFromPool(profName, studentName, poolName string) error {
	profUsername := UsernameFromEmail(profName)
	studentUsername := UsernameFromEmail(studentName)

	cmd := fmt.Sprintf(`
set -e
BASE="/home/%[1]s/%[2]s"
STUD_DIR="$BASE/%[3]s"

sudo setfacl -x u:%[3]s "$BASE" || true
sudo setfacl -x u:%[3]s "$STUD_DIR" || true
sudo setfacl -k "$STUD_DIR" || true
`, profUsername, poolName, studentUsername)

	return runLocalCmd(cmd)
}

//
// ===========================
// Remote / VM functions
// ===========================
//

func EnsureRemoteSSHKey(client *ssh.Client, username string) error {
	cmd := fmt.Sprintf(`
set -e
HOME="/home/%[1]s"
SSH="$HOME/.ssh"

sudo -u %[1]s mkdir -p "$SSH"
sudo -u %[1]s chmod 700 "$SSH"

if [ ! -f "$SSH/id_ed25519" ]; then
	sudo -u %[1]s ssh-keygen -t ed25519 -f "$SSH/id_ed25519" -N ""
fi

sudo -u %[1]s chmod 600 "$SSH/id_ed25519"
sudo -u %[1]s chmod 644 "$SSH/id_ed25519.pub"
`, username)

	return RunSSHcmd(client, cmd)
}

func ReadRemotePubKey(client *ssh.Client, username string) (string, error) {
	cmd := fmt.Sprintf(`sudo cat /home/%[1]s/.ssh/id_ed25519.pub`, username)
	return RunSSHcmdWithOutput(client, cmd)
}

func PrepareRemoteMountDirs(client *ssh.Client, username string) error {
	cmd := fmt.Sprintf(`
set -e
HOME="/home/%[1]s"
sudo -u %[1]s mkdir -p "$HOME/depot"
`, username)

	return RunSSHcmd(client, cmd)
}

func RestrictUserToSFTP(username string) error {
	cmd := fmt.Sprintf(`
echo '
Match User %[1]s
	ForceCommand internal-sftp
	AllowTcpForwarding no
	X11Forwarding no
' | sudo tee -a /etc/ssh/sshd_config
sudo systemctl restart ssh
`, username)

	return runLocalCmd(cmd)
}

//
// ===========================
// Setup Rclone workflow
// ===========================
//

func SetupRcloneForStudent(server models.Server, student models.Student, profName, poolName string) error {
	username := sshinject.UsernameFromEmail(student.Name)

	// 1. Charger clé privée
	signer, err := sshinject.LoadPrivateKey(os.Getenv("SSH_PRIVATE_KEY_PATH"))
	if err != nil {
		return fmt.Errorf("load private key failed: %w", err)
	}

	config := sshinject.SshConfig("vmuser", signer)
	addr := fmt.Sprintf("%s:22", server.IP_Address)

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	defer client.Close()

	// 2. Créer clé SSH sur VM distante
	if err := EnsureRemoteSSHKey(client, username); err != nil {
		return fmt.Errorf("ensure remote ssh key failed: %w", err)
	}

	// 3. Récupérer clé publique
	pubkey, err := ReadRemotePubKey(client, username)
	if err != nil {
		return fmt.Errorf("read remote pub key failed: %w", err)
	}
	log.Printf("Public key for %s:\n%s", student.Name, pubkey)

	// 4. Préparer dossier montage distant
	if err := PrepareRemoteMountDirs(client, username); err != nil {
		return fmt.Errorf("prepare remote mount dirs failed: %w", err)
	}

	// 5. Créer utilisateur local sur hôte
	if err := CreateUserLocal(student.Name); err != nil {
		return fmt.Errorf("create local user failed: %w", err)
	}

	// 6. Ajouter sa clé publique pour SFTP
	if err := AuthorizeStudentKey(student.Name, pubkey); err != nil {
		return fmt.Errorf("authorize student key failed: %w", err)
	}

	// 7. Ajouter étudiant au pool
	if err := AddStudentToPool(profName, student.Name, poolName); err != nil {
		return fmt.Errorf("add student to pool failed: %w", err)
	}

	// 8. Configurer rclone sur la VM distante
	if err := SetupRcloneMount(student, client, profName, poolName); err != nil {
		return fmt.Errorf("setup rclone mount failed: %w", err)
	}

	return nil
}

func RcloneConfigCmd(username, hostIP string) string {
	return fmt.Sprintf(`
set -e
CONF_DIR="/home/%[1]s/.config/rclone"
CONF_FILE="$CONF_DIR/rclone.conf"

sudo -u %[1]s mkdir -p "$CONF_DIR"

sudo -u %[1]s tee "$CONF_FILE" > /dev/null << EOF
[depot_%[1]s]
type = sftp
host = %[2]s
user = %[1]s
key_file = /home/%[1]s/.ssh/id_ed25519
shell_type = unix
EOF

sudo chown %[1]s:%[1]s "$CONF_FILE"
sudo chmod 600 "$CONF_FILE"
`, username, hostIP)
}

func RcloneSystemdCmd(username, profName, poolName string) string {
	return fmt.Sprintf(`
set -e

REMOTE_PATH="/home/%[2]s/%[3]s/%[1]s"
MOUNT_PATH="/home/%[1]s/depot"
SERVICE="/etc/systemd/system/rclone-depot-%[1]s.service"

# créer le dossier de montage si nécessaire
sudo -u %[1]s mkdir -p "$MOUNT_PATH"

# créer le service systemd
sudo tee "$SERVICE" > /dev/null << EOF
[Unit]
Description=Rclone depot mount for %[1]s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%[1]s

ExecStart=/usr/bin/rclone mount depot_%[1]s:$REMOTE_PATH $MOUNT_PATH \
    --vfs-cache-mode writes \
    --dir-cache-time 30s \
    --poll-interval 15s \
    --umask 002 \
    --log-file /home/%[1]s/.rclone_mount.log \
    --log-level INFO

ExecStop=/bin/fusermount3 -u $MOUNT_PATH

Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo chmod 644 "$SERVICE"
sudo systemctl daemon-reload
sudo systemctl enable rclone-depot-%[1]s.service
sudo systemctl restart rclone-depot-%[1]s.service
`, username, profName, poolName)
}

func SetupRcloneMount(student models.Student, client *ssh.Client, profname, poolname string) error {
	username := sshinject.UsernameFromEmail(student.Name)

	// 1. Créer config rclone sur la VM distante
	hostIP := os.Getenv("IP_ADDRESS")
	log.Println("hostIP = ", hostIP)

	cmdConfig := RcloneConfigCmd(username, hostIP)
	if err := RunSSHcmd(client, cmdConfig); err != nil {
		return fmt.Errorf("rclone config failed: %w", err)
	}

	// 2. Créer service systemd pour mount sur la VM distante
	cmdService := RcloneSystemdCmd(UsernameFromEmail(student.Name), UsernameFromEmail(profname), poolname)
	if err := RunSSHcmd(client, cmdService); err != nil {
		return fmt.Errorf("rclone systemd service failed: %w", err)
	}

	return nil
}
