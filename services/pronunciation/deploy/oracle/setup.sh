#!/usr/bin/env bash
# Sets up the pronunciation service on a fresh Oracle Cloud VM — Ubuntu or
# Oracle Linux, ARM (Ampere A1, Always Free) or x86. Run it from this
# directory on the VM:
#
#   bash setup.sh
#
# It installs Docker, opens ports 80/443 in the VM's firewall, picks an
# HTTPS hostname (your own PRON_DOMAIN, or <ip>.sslip.io), creates a token,
# and starts the service. The first build downloads the model (~1.3 GB)
# and takes several minutes.
set -euo pipefail
cd "$(dirname "$0")"

. /etc/os-release
case "$ID" in
  ol | rhel | centos | rocky | almalinux | fedora) family=rhel ;;
  ubuntu | debian) family=debian ;;
  *) echo "Unsupported OS: $ID (use Ubuntu or Oracle Linux)"; exit 1 ;;
esac

if ! command -v docker >/dev/null; then
  echo "==> Installing Docker"
  if [ "$family" = rhel ]; then
    sudo dnf install -y dnf-plugins-core
    sudo dnf config-manager --add-repo https://download.docker.com/linux/rhel/docker-ce.repo
    # Oracle Linux ships podman/runc, which conflict with Docker's packages.
    sudo dnf install -y --allowerasing docker-ce docker-ce-cli containerd.io docker-compose-plugin
    sudo systemctl enable --now docker
  else
    curl -fsSL https://get.docker.com | sudo sh
  fi
  sudo usermod -aG docker "$USER"
fi

# Oracle's images block everything but SSH in the VM's own firewall, on top
# of the VCN security list (which you open in the console, see README):
# firewalld on Oracle Linux, iptables on Ubuntu.
echo "==> Opening ports 80 and 443"
if [ "$family" = rhel ]; then
  if systemctl is-active --quiet firewalld; then
    sudo firewall-cmd --permanent --add-service=http --add-service=https
    sudo firewall-cmd --reload
  fi
else
  for port in 80 443; do
    sudo iptables -C INPUT -p tcp -m state --state NEW --dport "$port" -j ACCEPT 2>/dev/null ||
      sudo iptables -I INPUT 5 -p tcp -m state --state NEW --dport "$port" -j ACCEPT
  done
  if command -v netfilter-persistent >/dev/null; then
    sudo netfilter-persistent save
  fi
fi

if [ ! -f .env ]; then
  echo "==> Writing .env"
  ip="$(curl -fsS https://api.ipify.org)"
  domain="${PRON_DOMAIN:-${ip//./-}.sslip.io}"
  token="$(openssl rand -hex 32)"
  cat > .env <<ENV
PRON_DOMAIN=$domain
PRON_TOKEN=$token
TORCH_THREADS=$(nproc)
ENV
fi
source .env

echo "==> Building and starting (the first build takes a while)"
sudo docker compose up -d --build

echo
echo "Waiting for the model to load..."
for _ in $(seq 1 60); do
  if curl -fsS "https://$PRON_DOMAIN/health" >/dev/null 2>&1; then
    break
  fi
  sleep 10
done
curl -fsS "https://$PRON_DOMAIN/health" && echo

cat <<MSG

Done. Set these on the IELTS Arena API:

  PRON_SERVICE_URL=https://$PRON_DOMAIN
  PRON_SERVICE_TOKEN=$PRON_TOKEN

MSG
