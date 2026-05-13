#!/bin/bash
set -e
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

# ═══ CONFIGURATION ═══
# Note: These paths must match your VPS structure (from screenshot)
BE_PATH="/var/www/locolive/locolive-backend-main"
FE_PATH="/var/www/locolive/locolive-frontend-main"

echo "🚀 Starting System Update (VPS Version)..."

# 1. Update Backend Repo
echo "═══ Updating Backend ═══"
cd "$BE_PATH"
# Detect current branch or default to main
BE_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")
git fetch origin && git reset --hard origin/$BE_BRANCH

# 2. Update Frontend Repo
if [ -d "$FE_PATH" ]; then
    echo "═══ Updating Frontend ═══"
    cd "$FE_PATH"
    FE_BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "main")
    git fetch origin && git reset --hard origin/$FE_BRANCH
else
    echo "❌ Frontend path not found: $FE_PATH"
    echo "Please update FE_PATH in update.sh"
    exit 1
fi

# 4. Build Backend
cd "$BE_PATH"
go build -o locolive-api cmd/server/main.go
# Run migrations
DB_URL=$(grep "^DB_SOURCE=" app.env | cut -d'=' -f2- | sed 's/"//g' | sed "s/'//g")
migrate -path db/migrations -database "$DB_URL" -verbose up || true
pm2 restart locolive-backend

# 5. Build Frontend (SECURE - Using local .env.production)
cd "$FE_PATH"
if [ ! -f ".env.production" ]; then
    echo "⚠️ .env.production missing! Creating from template (Manually edit this later on VPS)"
    echo "VITE_API_URL=https://locolive.appnity.co.in/api" > .env.production
fi
npm install

# 6. Inject Firebase Key into Service Worker (Securely from .env.production)
if [ -f "public/firebase-messaging-sw.js" ]; then
    FIREBASE_KEY=$(grep "^VITE_FIREBASE_API_KEY=" .env.production | cut -d'=' -f2- | sed 's/"//g' | sed "s/'//g")
    if [ ! -z "$FIREBASE_KEY" ]; then
        sed -i "s/REPLACE_WITH_YOUR_FIREBASE_API_KEY/$FIREBASE_KEY/g" public/firebase-messaging-sw.js
        echo "✅ Firebase key injected into Service Worker"
    fi
fi

npm run build -- --mode production

# 7. Finalize
nginx -t && systemctl reload nginx
echo "✅ DEEP UPDATE COMPLETED! (No secrets leaked to GitHub)"
