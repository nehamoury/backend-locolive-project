#!/bin/bash

# Locolive VPS Initial Setup Script
# This script configures the Host Nginx and directories. Run this ONLY ONCE.

# --- Configuration ---
DOMAIN="api.locolive.com"  # Change this to your domain or VPS IP
PROJECT_DIR="/var/www/locolive"
NGINX_CONF_PATH="/etc/nginx/sites-available/locolive"

# --- Colors ---
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}Setting up Locolive VPS environment...${NC}"

# 1. Create project directories
sudo mkdir -p $PROJECT_DIR
sudo chown -R $USER:$USER $PROJECT_DIR

# 2. Create Host Nginx Config for Reverse Proxy
echo -e "${YELLOW}Creating Host Nginx configuration...${NC}"

sudo bash -c "cat > $NGINX_CONF_PATH <<EOF
server {
    listen 80;
    server_name $DOMAIN;

    location / {
        proxy_pass http://127.0.0.1:8090;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection \"upgrade\";
    }
}
EOF"

# 3. Enable Nginx Config
sudo ln -sf $NGINX_CONF_PATH /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl restart nginx

echo -e "${GREEN}Initial setup complete!${NC}"
echo -e "${GREEN}Domain $DOMAIN is now proxying to port 8090.${NC}"
echo -e "${YELLOW}Next steps:${NC}"
echo -e "1. Upload your code to $PROJECT_DIR"
echo -e "2. Create .env file in $PROJECT_DIR/backend/locolive-backend-main/"
echo -e "3. Run ./deploy.sh"
