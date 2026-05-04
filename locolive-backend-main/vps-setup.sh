#!/bin/bash
# ==========================================
# LocoLive VPS Initial Server Setup
# Run as root on a fresh VPS
# ==========================================
set -e

echo "🔧 LocoLive VPS Initial Setup"
echo "=============================="

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }

# ─── 1. System Update ───────────────────────────────────────────────
log_info "Updating system packages..."
apt update && apt upgrade -y

# ─── 2. Install Dependencies ────────────────────────────────────────
log_info "Installing dependencies..."
apt install -y \
    postgresql postgresql-contrib \
    redis-server \
    nginx \
    certbot python3-certbot-nginx \
    git curl wget \
    build-essential \
    fail2ban \
    ufw

# ─── 3. Install Go ──────────────────────────────────────────────────
if ! command -v go &> /dev/null; then
    log_info "Installing Go..."
    GO_VERSION="1.22.5"
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
    rm go${GO_VERSION}.linux-amd64.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    source /etc/profile.d/go.sh
    log_info "Go installed: $(go version)"
fi

# ─── 4. Install Node.js ─────────────────────────────────────────────
if ! command -v node &> /dev/null; then
    log_info "Installing Node.js..."
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    apt install -y nodejs
    log_info "Node.js installed: $(node --version)"
fi

# ─── 5. Setup PostgreSQL ────────────────────────────────────────────
log_info "Setting up PostgreSQL..."
read -p "Enter database password for locolive user: " DB_PASSWORD

sudo -u postgres psql <<EOF
CREATE USER locolive WITH PASSWORD '${DB_PASSWORD}';
CREATE DATABASE privacy_social OWNER locolive;
\c privacy_social
CREATE EXTENSION IF NOT EXISTS postgis;
GRANT ALL PRIVILEGES ON DATABASE privacy_social TO locolive;
EOF

log_info "PostgreSQL setup complete"

# ─── 6. Setup Redis ─────────────────────────────────────────────────
log_info "Setting up Redis..."
read -p "Enter Redis password: " REDIS_PASSWORD

sed -i "s/^# requirepass.*/requirepass ${REDIS_PASSWORD}/" /etc/redis/redis.conf
echo "requirepass ${REDIS_PASSWORD}" >> /etc/redis/redis.conf
systemctl restart redis-server
systemctl enable redis-server

log_info "Redis setup complete"

# ─── 7. Firewall (UFW) ──────────────────────────────────────────────
log_info "Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

log_info "Firewall configured (ports 22, 80, 443 open)"

# ─── 8. Fail2Ban ────────────────────────────────────────────────────
log_info "Setting up Fail2Ban..."
cat > /etc/fail2ban/jail.local <<EOF
[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 3600
EOF
systemctl restart fail2ban
systemctl enable fail2ban

log_info "Fail2Ban configured"

# ─── 9. Create App Directories ──────────────────────────────────────
log_info "Creating application directories..."
mkdir -p /var/www/locolive-backend
mkdir -p /var/www/locolive-frontend
mkdir -p /var/www/uploads

chown -R www-data:www-data /var/www/uploads
chmod 755 /var/www/uploads

# ─── 10. Swap (if needed) ───────────────────────────────────────────
if [ $(free -m | awk '/^Mem:/{print $2}') -lt 2048 ]; then
    log_warn "Low memory detected (< 2GB). Creating 2GB swap..."
    fallocate -l 2G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
    log_info "Swap created"
fi

echo ""
log_info "✅ VPS initial setup complete!"
log_info "Next steps:"
log_info "1. Clone your backend to /var/www/locolive-backend"
log_info "2. Clone your frontend to /var/www/locolive-frontend"
log_info "3. Copy app.env.production.example to app.env and fill values"
log_info "4. Copy .env.production.example to .env.production and update API URL"
log_info "5. Run: ./deploy.sh"
