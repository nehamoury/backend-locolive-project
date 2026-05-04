# LocoLive VPS Deployment — Step by Step

> For domain: `locolive.appnity.co.in`

## Prerequisites on VPS
- Ubuntu 22.04/24.04
- Root or sudo access
- Domain DNS pointed to VPS IP

---

## Step 1: SSH into VPS
```bash
ssh root@YOUR_VPS_IP
```

## Step 2: Run Initial Setup (first time only)
```bash
# If you haven't already setup the server, run:
cd /tmp
# Upload vps-setup.sh via scp, then:
chmod +x vps-setup.sh
./vps-setup.sh
```

This will install: PostgreSQL, Redis, Nginx, Go, Node.js, UFW, Fail2Ban.

## Step 3: Clone/Upload Code
```bash
# Backend
mkdir -p /var/www/locolive-backend
# Upload your backend files here (via scp or git clone)

# Frontend
mkdir -p /var/www/locolive-frontend
# Upload your frontend files here
```

## Step 4: Configure Backend Environment
```bash
cd /var/www/locolive-backend

# Copy production env template
cp app.env.production.example app.env

# Edit with your actual values
nano app.env
```

**Key values to change in `app.env`:**
- `DB_SOURCE` → use actual DB password
- `REDIS_ADDRESS` → use actual Redis password  
- `JWT_SECRET` → generate random 64-char string
- `GOOGLE_CLIENT_ID/SECRET` → your production Google OAuth credentials
- `R2_*` → your Cloudflare R2 credentials (or set `STORAGE_PROVIDER=local`)
- `EMAIL_SENDER_PASSWORD` → Gmail App Password (16 digits)
- `FRONTEND_URL` → `https://locolive.appnity.co.in`
- `ENVIRONMENT` → `production`
- `FORCE_HTTPS` → `true`

## Step 5: Configure Frontend Environment
```bash
cd /var/www/locolive-frontend

# Create production env
cat > .env.production << 'EOF'
VITE_API_URL=https://locolive.appnity.co.in/api
VITE_GOOGLE_CLIENT_ID=YOUR_PROD_GOOGLE_CLIENT_ID
EOF
```

## Step 6: Build & Deploy
```bash
cd /var/www/locolive-backend

# Make deploy script executable
chmod +x deploy.sh

# Run deployment
./deploy.sh
```

## Step 7: Setup SSL (Let's Encrypt)
```bash
# Install certbot if not already installed
apt install -y certbot python3-certbot-nginx

# Get SSL certificate
certbot --nginx -d locolive.appnity.co.in

# Restart nginx
systemctl restart nginx
```

## Step 8: Verify Deployment

```bash
# Check backend health
curl http://127.0.0.1:8080/api/health
# Expected: {"status":"healthy"}

# Check HTTPS frontend
curl -I https://locolive.appnity.co.in
# Expected: HTTP/2 200

# Check API through Nginx
curl https://locolive.appnity.co.in/api/health
# Expected: {"status":"healthy"}

# Check WebSocket endpoint
curl -I -H "Upgrade: websocket" -H "Connection: Upgrade" https://locolive.appnity.co.in/api/ws/chat
# Expected: 101 Switching Protocols (if backend running)
```

## Step 9: View Logs
```bash
# Backend logs
journalctl -u locolive-api -f

# Nginx access logs
tail -f /var/log/nginx/access.log

# Nginx error logs
tail -f /var/log/nginx/error.log

# PostgreSQL logs
tail -f /var/log/postgresql/postgresql-*-main.log
```

## Step 10: Setup Auto-Renewal for SSL
```bash
# Certbot auto-renewal is already configured via systemd timer
# Verify:
systemctl status certbot.timer

# Test dry-run:
certbot renew --dry-run
```

---

## Quick Fix: API Not Working (404/HTML Response)

If `/api/*` returns HTML instead of JSON:

```bash
# 1. Check if backend is running
systemctl status locolive-api

# 2. Check if backend responds directly
curl http://127.0.0.1:8080/api/health

# 3. Check nginx proxy config
nginx -T | grep -A 10 "location /api"

# 4. If proxy not configured, deploy the site config:
cp /var/www/locolive-backend/nginx-locolive-site.conf /etc/nginx/sites-available/locolive
ln -sf /etc/nginx/sites-available/locolive /etc/nginx/sites-enabled/locolive
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl restart nginx
```

---

## Redeploy After Code Changes

```bash
# On your local machine:
scp -r backend/* root@VPS_IP:/var/www/locolive-backend/
scp -r frontend/* root@VPS_IP:/var/www/locolive-frontend/

# On VPS:
cd /var/www/locolive-backend
./deploy.sh
```

---

## Backup Database

```bash
# Manual backup
pg_dump -U locolive privacy_social > /var/backups/locolive_$(date +%Y%m%d).sql

# Setup daily backup cron
echo "0 2 * * * pg_dump -U locolive privacy_social | gzip > /var/backups/locolive_\$(date +\%Y\%m\%d).sql.gz" | crontab -
```
