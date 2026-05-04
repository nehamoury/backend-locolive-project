#!/bin/bash
# ==========================================
# LocoLive VPS Deployment Script
# Usage: chmod +x deploy.sh && ./deploy.sh
# ==========================================
set -e

echo "🚀 LocoLive VPS Deployment"
echo "=========================="

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ─── Pre-flight Checks ──────────────────────────────────────────────
check_command() {
    if ! command -v $1 &> /dev/null; then
        log_error "$1 is not installed. Please install it first."
        exit 1
    fi
}

log_info "Checking prerequisites..."
check_command go
check_command node
check_command npm
check_command make
check_command nginx

# ─── Configuration ──────────────────────────────────────────────────
BACKEND_DIR="/var/www/locolive-backend"
FRONTEND_DIR="/var/www/locolive-frontend"
UPLOADS_DIR="/var/www/uploads"

echo ""
log_warn "Before running this script:"
log_warn "1. Copy app.env.production.example to app.env and fill in production values"
log_warn "2. Copy frontend .env.production.example to .env.production and update API URL"
log_warn "3. Ensure PostgreSQL, Redis are running"
log_warn "4. Ensure you have SSL certificates (run certbot first)"
echo ""
read -p "Have you completed the above? (y/N): " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    log_error "Aborting. Complete the prerequisites first."
    exit 1
fi

# ─── Backend Build ──────────────────────────────────────────────────
log_info "Building backend..."
cd "$BACKEND_DIR"

# Run migrations
log_info "Running database migrations..."
make migrateup || { log_error "Migration failed!"; exit 1; }

# Build Go binary
log_info "Compiling Go binary..."
go build -o locolive-api cmd/server/main.go || { log_error "Backend build failed!"; exit 1; }

# Set permissions
chown www-data:www-data locolive-api
chmod 755 locolive-api

# ─── Frontend Build ─────────────────────────────────────────────────
log_info "Building frontend..."
cd "$FRONTEND_DIR"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    log_info "Installing frontend dependencies..."
    npm ci --production
fi

# Build with production env
log_info "Compiling frontend..."
npm run build -- --mode production || { log_error "Frontend build failed!"; exit 1; }

# Copy dist to where backend/nginx expects it
log_info "Deploying frontend assets..."
mkdir -p "$FRONTEND_DIR/dist"
# The build output is already in dist/

# ─── Uploads Directory ──────────────────────────────────────────────
log_info "Setting up uploads directory..."
mkdir -p "$UPLOADS_DIR"
chown www-data:www-data "$UPLOADS_DIR"
chmod 755 "$UPLOADS_DIR"

# ─── Systemd Service ────────────────────────────────────────────────
log_info "Installing systemd service..."
cp "$BACKEND_DIR/locolive-api.service" /etc/systemd/system/locolive-api.service
systemctl daemon-reload
systemctl enable locolive-api
systemctl restart locolive-api

# ─── Nginx ──────────────────────────────────────────────────────────
log_info "Configuring nginx..."
# Backup current config
cp /etc/nginx/nginx.conf /etc/nginx/nginx.conf.backup.$(date +%Y%m%d%H%M%S)
cp "$BACKEND_DIR/nginx.conf" /etc/nginx/nginx.conf

# Test nginx config
nginx -t || { log_error "Nginx config test failed!"; exit 1; }
systemctl restart nginx

# ─── Verification ───────────────────────────────────────────────────
echo ""
log_info "Deployment complete! Running verification..."
sleep 3

# Check backend health
if curl -sf http://127.0.0.1:8080/api/health > /dev/null; then
    log_info "✅ Backend is running"
else
    log_error "❌ Backend health check failed. Check: journalctl -u locolive-api -f"
fi

# Check nginx
if curl -sf http://localhost/ > /dev/null; then
    log_info "✅ Nginx is serving frontend"
else
    log_warn "⚠️  Nginx may need SSL certs. Run: certbot --nginx -d your-domain.com"
fi

# Check systemd
if systemctl is-active --quiet locolive-api; then
    log_info "✅ Systemd service is active"
else
    log_error "❌ Systemd service failed. Check: systemctl status locolive-api"
fi

echo ""
log_info "📋 Post-deploy checklist:"
log_info "1. Update nginx.conf: replace 'your-domain.com' with actual domain"
log_info "2. Run: certbot --nginx -d your-domain.com"
log_info "3. Verify HTTPS: https://your-domain.com"
log_info "4. Test auth flow: signup → login → refresh → logout"
log_info "5. Check logs: journalctl -u locolive-api -f"
log_info "6. Setup DB backup: pg_dump cron job"
log_info "7. Rotate all secrets in app.env"

echo ""
log_info "🎉 Deployment finished!"
