#!/bin/bash
set -e

# Deploy script for Snippy API
# This script is executed on the remote server via SSH

DEPLOY_DIR="/root/snippy-api"
SERVICE_NAME="snippy-api"

echo "Starting deployment..."

cd "$DEPLOY_DIR"

# Make binary executable
echo "Setting binary permissions..."
chmod +x snippy-api

# Install systemd service if it doesn't exist
if [ ! -f /etc/systemd/system/${SERVICE_NAME}.service ]; then
  echo "Installing systemd service..."
  cp ${SERVICE_NAME}.service /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable ${SERVICE_NAME}
fi

# Restart service (this runs docker-compose)
echo "Restarting service..."
systemctl restart ${SERVICE_NAME}

# Wait for services to start
echo "Waiting for services to start..."
sleep 10

# Check status
echo "Checking service status..."
systemctl status ${SERVICE_NAME} --no-pager || true

echo "Checking Docker containers..."
docker-compose ps || true

# Health check
echo "Running health check..."
if curl -f http://localhost:80/api/v1/health; then
  echo "Deploy successful!"
  echo "Application is healthy and running"
  exit 0
else
  echo "Health check failed"
  echo "Recent logs:"
  docker-compose logs --tail=50 api || true
  exit 1
fi
