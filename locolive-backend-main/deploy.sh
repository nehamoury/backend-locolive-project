#!/bin/bash

# Locolive Backend Auto-Deployment Script
# This script builds and restarts the Docker containers for Locolive.

# --- Configuration ---
PROJECT_DIR="/var/www/locolive/backend/locolive-backend-main"
ENV_FILE=".env"
DOCKER_COMPOSE_FILE="docker-compose.yml"

# --- Colors for Output ---
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Starting Deployment for Locolive Backend...${NC}"

# Navigate to project directory
if [ -d "$PROJECT_DIR" ]; then
    cd "$PROJECT_DIR"
else
    echo -e "${RED}Error: Project directory $PROJECT_DIR not found!${NC}"
    exit 1
fi

# Check for .env file
if [ ! -f "$ENV_FILE" ]; then
    echo -e "${RED}Error: $ENV_FILE not found!${NC}"
    echo -e "${YELLOW}Please create a .env file from app.env.example before running this script.${NC}"
    exit 1
fi

# Pull latest changes (Uncomment if using Git)
# echo -e "${YELLOW}Pulling latest changes from Git...${NC}"
# git pull origin main

# Build and Start Containers
echo -e "${YELLOW}Building and starting Docker containers...${NC}"
docker-compose -f "$DOCKER_COMPOSE_FILE" down
docker-compose -f "$DOCKER_COMPOSE_FILE" up -d --build

# Check if containers are running
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Deployment Successful!${NC}"
    echo -e "${GREEN}Services are running on:${NC}"
    echo -e " - API/Nginx: http://localhost:8090"
    echo -e " - Postgres:  localhost:5433"
    echo -e " - Redis:     localhost:6380"
else
    echo -e "${RED}Deployment failed! Check docker logs for details.${NC}"
    exit 1
fi

# Cleanup unused images to save space
echo -e "${YELLOW}Cleaning up unused Docker images...${NC}"
docker image prune -f

echo -e "${GREEN}All done!${NC}"
