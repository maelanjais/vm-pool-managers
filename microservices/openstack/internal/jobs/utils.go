package jobs

import "fmt"


func baseUserConfig(sshKey string) string {
	return fmt.Sprintf(`#cloud-config
users:
  - name: vmuser
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    ssh_authorized_keys:
      - %s

package_update: true
package_upgrade: true
packages:
  - fuse3
  - unzip

runcmd:
  - curl https://rclone.org/install.sh | bash || wget -qO- https://rclone.org/install.sh | bash
  - echo "Installation de rclone terminee"
  - sudo groupadd -f fuse || true
  - sudo usermod -aG fuse vmuser
  - sudo sed -i 's/^#user_allow_other/user_allow_other/' /etc/fuse.conf
  - sudo mkdir -p /home/vmuser/depot
  - sudo chown vmuser:vmuser /home/vmuser/depot
  - sudo chmod 700 /home/vmuser/depot
`, sshKey)
}

// baseUserConfigSnapshot is a lighter version for VMs booted from a pre-baked snapshot
// (e.g. jupyter-snapshot-*) where apt upgrade, rclone, and fuse are already installed.
func baseUserConfigSnapshot(sshKey string) string {
	return fmt.Sprintf(`#cloud-config
users:
  - name: vmuser
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    groups: sudo
    ssh_authorized_keys:
      - %s
`, sshKey)
}


// registrarCloudConfig generates a cloud-init script that installs and starts
// the vm-registrar agent on the VM. The agent will auto-register itself
// into the PostgreSQL inventory using OpenStack metadata.
func registrarCloudConfig(pgDSN string, healthPort int, controlCenterURL string, downloadBaseURL string) string {
	return fmt.Sprintf(`#!/bin/bash

# ─── vm-registrar agent setup (runs in background to not block cloud-init) ───
(
REGISTRAR_DIR="/etc/registrar"
REGISTRAR_BIN="/usr/local/bin/vm-registrar"

mkdir -p ${REGISTRAR_DIR}

# Write environment config
cat > ${REGISTRAR_DIR}/registrar.env << 'ENVEOF'
REGISTRAR_PG_DSN=%s
REGISTRAR_CC_URL=%s
REGISTRAR_HEARTBEAT_INTERVAL=15s
REGISTRAR_HEALTH_TIMEOUT=2s
REGISTRAR_DRAIN_TIMEOUT=5s
REGISTRAR_HEALTH_PORT=%d
ENVEOF

chmod 600 ${REGISTRAR_DIR}/registrar.env

# Download pre-compiled vm-registrar (hard 60s total timeout)
DOWNLOAD_URL="%s"
if [ -n "$DOWNLOAD_URL" ]; then
  timeout 60 curl -fsSLk --max-time 30 ${DOWNLOAD_URL}/vm-registrar -o ${REGISTRAR_BIN} || \
  timeout 60 wget -qO ${REGISTRAR_BIN} --no-check-certificate --timeout=30 --tries=1 ${DOWNLOAD_URL}/vm-registrar || true
fi
[ -f ${REGISTRAR_BIN} ] && chmod +x ${REGISTRAR_BIN} || echo "[registrar] download skipped"

# Create systemd service
cat > /etc/systemd/system/vm-registrar.service << 'SVCEOF'
[Unit]
Description=VM Registrar Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/registrar/registrar.env
ExecStart=/usr/local/bin/vm-registrar
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable vm-registrar
systemctl start vm-registrar

# ─── ssh-monitor agent setup ───────────────────────────────
cat > /usr/local/bin/ssh-monitor.sh << 'MOF'
#!/bin/bash
while true; do
  COUNT=$(who | wc -l)
  if [ "$COUNT" -gt 0 ]; then
    STATUS="connected"
  else
    STATUS="idle"
  fi
  HOST=$(hostname)
  if [ -n "$REGISTRAR_CC_URL" ]; then
    curl -sfk -X POST "${REGISTRAR_CC_URL}/api/vm-activity" \
      -H "Content-Type: application/json" \
      -d "{\"hostname\":\"${HOST}\",\"status\":\"${STATUS}\"}" > /dev/null 2>&1 || true
  fi
  if [ -n "$REGISTRAR_PG_DSN" ]; then
    psql "$REGISTRAR_PG_DSN" -c "UPDATE vm_instances SET activity_status = '${STATUS}' WHERE name = '${HOST}'" > /dev/null 2>&1 || true
  fi
  sleep 10
done
MOF

chmod +x /usr/local/bin/ssh-monitor.sh

cat > /etc/systemd/system/ssh-monitor.service << 'SMOF'
[Unit]
Description=SSH Connection Monitor
After=network-online.target

[Service]
Type=simple
EnvironmentFile=/etc/registrar/registrar.env
ExecStart=/usr/local/bin/ssh-monitor.sh
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
SMOF

systemctl daemon-reload
systemctl enable ssh-monitor
systemctl start ssh-monitor

echo "[cloud-init] vm-registrar & ssh-monitor started"
) &

echo "[cloud-init] registrar setup launched in background"
`, pgDSN, controlCenterURL, healthPort, downloadBaseURL)
}
