# LocoLive — Final VPS Deployment Guide
> All local fixes ready. Follow these exact steps to deploy.

---

## 🔴 Problems on Live VPS Right Now

| Issue | Evidence |
|-------|----------|
| API proxy broken | `/api/health` returns HTML instead of JSON |
| No OG meta tags | `/login` shows no og:title, og:image, twitter:card |
| Old build deployed | Same asset hash `index-BGoa0ULp.js` — none of the new changes |
| Nginx not routing `/api/` to backend on port 8080 | All API calls fall through to SPA |

---

## 📦 What's Fixed Locally (Ready to Deploy)

| Fix | File |
|-----|------|
| Nginx proxy config with SSL | `nginx.conf`, `nginx-locolive-site.conf` |
| SameSite cookie security | `user.go`, `security.go` |
| Missing admin methods | `router.go` |
| OpenGraph + Twitter meta tags | `index.html` |
| SEOHead reusable component | `SEOHead.tsx` |
| Dynamic OG per page | `Login.tsx`, `Signup.tsx`, `Profile.tsx` |
| Hardcoded DB URL removed | `Makefile` |
| Production env templates | `app.env.production.example` |
| Systemd service file | `locolive-api.service` |
| Deploy scripts | `deploy.sh`, `vps-setup.sh` |

---

## 🚀 DEPLOYMENT STEPS

### Step 1: Build Frontend Locally
```bash
cd frontend/frontend

# Set production API URL
echo 'VITE_API_URL=https://locolive.appnity.co.in/api' > .env.production
echo 'VITE_GOOGLE_CLIENT_ID=481033555872-lu12apmuqjjeolsmkec81204reiafgcq.apps.googleusercontent.com' >> .env.production

# Build
npm run build -- --mode production

# Verify dist/ exists
ls dist/
```

### Step 2: Build Backend Locally (Optional — or build on VPS)
```bash
cd backend/locolive-backend-main
go build -o locolive-api cmd/server/main.go
```

### Step 3: Upload to VPS
```bash
# Replace with your VPS IP
VPS_IP="YOUR_VPS_IP"

# Frontend
scp -r frontend/frontend/dist/* root@$VPS_IP:/var/www/locolive-frontend/dist/

# Backend source (if building on VPS)
scp -r backend/locolive-backend-main/* root@$VPS_IP:/var/www/locolive-backend/

# OR compiled binary
scp backend/locolive-backend-main/locolive-api root@$VPS_IP:/var/www/locolive-backend/
```

### Step 4: SSH into VPS & Fix Nginx (MOST IMPORTANT)
```bash
ssh root@$VPS_IP

# Check if backend is running
systemctl status locolive-api
curl http://127.0.0.1:8080/api/health
# If connection refused → start backend first

# Deploy the FIXED nginx config
cp /var/www/locolive-backend/nginx-locolive-site.conf /etc/nginx/sites-available/locolive
ln -sf /etc/nginx/sites-available/locolive /etc/nginx/sites-enabled/locolive
rm -f /etc/nginx/sites-enabled/default

# Test and restart
nginx -t
systemctl restart nginx

# Verify API proxy works
curl https://locolive.appnity.co.in/api/health
# Should return: {"status":"healthy"}
```

### Step 5: Set Up Systemd Service (if not already)
```bash
# On VPS
cp /var/www/locolive-backend/locolive-api.service /etc/systemd/system/locolive-api.service
systemctl daemon-reload
systemctl enable locolive-api
systemctl restart locolive-api

# Verify
systemctl status locolive-api
journalctl -u locolive-api -f
```

### Step 6: Verify Everything
```bash
# 1. API health through Nginx
curl https://locolive.appnity.co.in/api/health
# Expected: {"status":"healthy"}

# 2. OG meta tags present
curl https://locolive.appnity.co.in/login | grep -o 'og:title.*>' 
# Expected: og:title, og:description, og:image tags

# 3. Frontend loads
curl -I https://locolive.appnity.co.in
# Expected: HTTP/2 200

# 4. WebSocket endpoint accessible
curl -I -H "Upgrade: websocket" -H "Connection: Upgrade" https://locolive.appnity.co.in/api/ws/chat
# Expected: connection reaches backend
```

---

## 🎯 Quick Fix If API Still Returns HTML

```bash
ssh root@YOUR_VPS_IP

# Check what nginx is actually using
nginx -T 2>/dev/null | grep -A 20 "location /api"

# If no proxy_pass found → the nginx config on VPS is wrong
# Force the correct config:
cat > /etc/nginx/sites-available/locolive << 'NGINX_EOF'
upstream locolive_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name locolive.appnity.co.in;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name locolive.appnity.co.in;

    ssl_certificate /etc/letsencrypt/live/locolive.appnity.co.in/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/locolive.appnity.co.in/privkey.pem;

    # API → Backend (CRITICAL)
    location /api/ {
        proxy_pass http://locolive_backend/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
    }

    # Static assets
    location /assets/ {
        alias /var/www/locolive-frontend/dist/assets/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # SPA fallback
    location / {
        root /var/www/locolive-frontend/dist;
        try_files $uri $uri/ /index.html;
        add_header Cache-Control "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0";
    }

    # Uploads
    location /uploads/ {
        alias /var/www/uploads/;
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
}
NGINX_EOF

ln -sf /etc/nginx/sites-available/locolive /etc/nginx/sites-enabled/locolive
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl restart nginx
```

---

## ✅ Deployment Checklist

- [ ] Frontend built with `.env.production` (correct API URL)
- [ ] Backend running on VPS (check: `curl http://127.0.0.1:8080/api/health`)
- [ ] Nginx proxying `/api/` to `127.0.0.1:8080` (NOT returning HTML)
- [ ] HTTPS working with Let's Encrypt certs
- [ ] OG meta tags visible in page source (check `/login` HTML)
- [ ] `SEOHead` component working on Login, Signup, Profile pages
- [ ] WebSocket `/api/ws/chat` accessible through Nginx
- [ ] Systemd service running and auto-restarting
- [ ] Static assets loading from correct paths
- [ ] No Eruda debug console in production build

---

## 🔍 Quick Diagnostic Commands

```bash
# Is backend running?
systemctl status locolive-api

# Is backend responding directly?
curl http://127.0.0.1:8080/api/health

# Is nginx proxying correctly?
curl -s https://locolive.appnity.co.in/api/health | head -1

# Is SSL valid?
curl -I https://locolive.appnity.co.in 2>&1 | grep -i "HTTP\|SSL"

# Are meta tags in HTML?
curl -s https://locolive.appnity.co.in/login | grep -c "og:"

# Nginx error log
tail -20 /var/log/nginx/error.log

# Backend logs
journalctl -u locolive-api --since "5 minutes ago"
```
