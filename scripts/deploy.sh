#!/bin/bash
set -e

# Deploy script for Snippy API
# This script is executed on the remote server via SSH

DEPLOY_DIR="/root/snippy-api"
SERVICE_NAME="snippy-api"

echo "Starting deployment..."

cd "$DEPLOY_DIR"

# Load environment variables
if [ -f .env.production ]; then
  echo "Loading environment variables..."
  set -a
  source .env.production
  set +a
fi

# Make binary executable
echo "Setting binary permissions..."
chmod +x snippy-api

# Make scripts executable
echo "Setting script permissions..."
chmod +x scripts/*.sh

# Install systemd service if it doesn't exist
if [ ! -f /etc/systemd/system/${SERVICE_NAME}.service ]; then
  echo "Installing systemd service..."
  cp ${SERVICE_NAME}.service /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable ${SERVICE_NAME}
fi

# Initialize SSL certificates (creates self-signed if no domain, or uses existing Let's Encrypt)
echo "Initializing SSL certificates..."
docker volume create snippy-backend_certbot_certs >/dev/null 2>&1 || true
if [ ! -f /etc/letsencrypt/live/cert/fullchain.pem ]; then
  docker run --rm \
    -v snippy-backend_certbot_certs:/etc/letsencrypt \
    -v "$DEPLOY_DIR/scripts/init-ssl.sh:/init-ssl.sh:ro" \
    alpine sh -c "apk add --no-cache openssl bash >/dev/null && bash /init-ssl.sh '${DOMAIN:-}' '${CERTBOT_EMAIL:-}' '${CERTBOT_STAGING:-false}'"
fi

# Restart service (this runs docker compose)
echo "Restarting service..."
systemctl restart ${SERVICE_NAME}

# Wait for services to start
echo "Waiting for services to start..."
sleep 15

# Run migrations
echo "Running database migrations..."
for migration in migrations/*.sql; do
  if [[ ! "$migration" =~ rollback\.sql$ ]] && [[ -f "$migration" ]]; then
    echo "Applying migration: $(basename $migration)"
    docker compose exec -T postgres psql -U "${POSTGRES_USER:-snippy_user}" -d "${POSTGRES_DB:-snippy_production}" -f "/migrations/$(basename $migration)" || {
      echo "Warning: Migration $(basename $migration) may have already been applied or encountered an error"
    }
  fi
done
echo "Migrations completed"

# Check status
echo "Checking service status..."
systemctl status ${SERVICE_NAME} --no-pager || true

echo "Checking Docker containers..."
docker compose ps || true

# Health check
echo "Running health check..."
if curl -sf http://localhost/api/v1/health; then
  echo ""
  echo "Deploy successful!"
  echo "Application is healthy and running"
  exit 0
else
  echo "Health check failed"
  echo "Recent logs:"
  docker compose logs --tail=50 api || true
  exit 1
fi
