#!/bin/bash
# ============================================================
# LocoLive — Production Update Script (VPS)
# Fixes: Wrong API URL, missing R2, no nginx reload, no health check
# ============================================================
set -e

BE_PATH="/var/www/locolive-backend"
FE_PATH="/var/www/locolive-frontend"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info()  { echo -e "${GREEN}[✓]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[!]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }

# ─── 1. Backend Update ─────────────────────────────────────────
echo ""
echo "═══ 1. Updating Backend ═══"

cd "$BE_PATH"

# Pull latest code (stash first to avoid conflicts)
git stash 2>/dev/null || true
git pull origin main
log_info "Code pulled"

# Ensure R2 storage is enabled (critical — was local before)
if grep -q "^STORAGE_PROVIDER=local" app.env 2>/dev/null; then
    sed -i 's/^STORAGE_PROVIDER=local/STORAGE_PROVIDER=r2/' app.env
    log_warn "Switched STORAGE_PROVIDER from local → r2"
fi

# Ensure production environment
sed -i 's/^ENVIRONMENT=.*/ENVIRONMENT=production/' app.env
sed -i 's/^FORCE_HTTPS=.*/FORCE_HTTPS=true/' app.env

# Ensure correct frontend URL
sed -i 's|^FRONTEND_URL=.*|FRONTEND_URL=https://locolive.appnity.co.in|' app.env

# Build backend
rm -f locolive-api
go build -o locolive-api cmd/server/main.go
if [ $? -ne 0 ]; then
    log_error "Backend build FAILED"
    exit 1
fi
log_info "Backend built"

# Set permissions
chown www-data:www-data locolive-api 2>/dev/null || true
chmod 755 locolive-api

# Restart backend (systemd — not pm2)
if systemctl is-active --quiet locolive-api; then
    systemctl restart locolive-api
    log_info "Backend restarted (systemd)"
else
    log_warn "Systemd service not found, using pm2 fallback"
    pm2 delete locolive-backend 2>/dev/null || true
    pm2 start ./locolive-api --name locolive-backend
    log_info "Backend started (pm2)"
fi

# Wait for backend to start
sleep 3

# Health check
HEALTH=$(curl -sf http://127.0.0.1:8080/api/health 2>/dev/null)
if [ -n "$HEALTH" ]; then
    log_info "Backend health check passed: $HEALTH"
else
    log_error "Backend health check FAILED — check logs: journalctl -u locolive-api -f"
    exit 1
fi

# ─── 2. Frontend Update ────────────────────────────────────────
echo ""
echo "═══ 2. Updating Frontend ═══"

cd "$FE_PATH"

# Pull latest code
git stash 2>/dev/null || true
git pull origin main
log_info "Code pulled"

# Set production environment variables (CORRECT URL with /api suffix)
cat > .env.production << 'EOF'
VITE_API_URL=https://locolive.appnity.co.in/api
VITE_GOOGLE_CLIENT_ID=481033555872-lu12apmuqjjeolsmkec81204reiafgcq.apps.googleusercontent.com
EOF
log_info "Production env set"

# Build
rm -rf dist
npm ci --production
npm run build -- --mode production
if [ $? -ne 0 ]; then
    log_error "Frontend build FAILED"
    exit 1
fi
log_info "Frontend built"

# ─── 3. Nginx Reload ───────────────────────────────────────────
echo ""
echo "═══ 3. Reloading Nginx ═══"

nginx -t && systemctl reload nginx
log_info "Nginx reloaded"

# ─── 4. Final Verification ─────────────────────────────────────
echo ""
echo "═══ 4. Verification ═══"

sleep 2

# Check API through nginx
API_CHECK=$(curl -sf https://locolive.appnity.co.in/api/health 2>/dev/null)
if [ -n "$API_CHECK" ]; then
    log_info "API accessible: $API_CHECK"
else
    log_error "API NOT accessible through nginx — check nginx config"
fi

# Check frontend
FE_CHECK=$(curl -sI https://locolive.appnity.co.in 2>/dev/null | head -1)
if echo "$FE_CHECK" | grep -q "200"; then
    log_info "Frontend accessible: $FE_CHECK"
else
    log_error "Frontend NOT accessible"
fi

# Check OG meta tags
OG_COUNT=$(curl -s https://locolive.appnity.co.in/login 2>/dev/null | grep -c "og:")
if [ "$OG_COUNT" -gt 0 ]; then
    log_info "OpenGraph tags found: $OG_COUNT tags"
else
    log_warn "OpenGraph tags NOT found — frontend may need rebuild"
fi

echo ""
echo -e "${GREEN}═══════════════════════════════════${NC}"
echo -e "${GREEN}  LocoLive Update Complete! 🚀${NC}"
echo -e "${GREEN}═══════════════════════════════════${NC}"
echo ""
echo "Check logs: journalctl -u locolive-api -f"
echo "Check nginx: tail -f /var/log/nginx/error.log"
