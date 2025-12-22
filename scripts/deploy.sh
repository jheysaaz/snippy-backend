#!/bin/bash
set -e

# Deploy script for Snippy API
# This script is executed on the remote server via SSH

DEPLOY_DIR="/root/snippy-api"
SERVICE_NAME="snippy-api"

# Detect docker compose command (v2 vs v1)
if docker compose version >/dev/null 2>&1; then
  DC="docker compose"
else
  DC="docker-compose"
fi

echo "Starting deployment..."
echo "Using: $DC"

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

# Initialize SSL certificates
echo "Initializing SSL certificates..."
docker volume create snippy-backend_certbot_certs >/dev/null 2>&1 || true
docker volume create snippy-backend_certbot_www >/dev/null 2>&1 || true

# Check if certificates exist in the volume
CERT_EXISTS=$(docker run --rm -v snippy-backend_certbot_certs:/etc/letsencrypt alpine sh -c "test -f /etc/letsencrypt/live/cert/fullchain.pem && echo 'yes' || echo 'no'")

if [ "$CERT_EXISTS" = "no" ]; then
  if [ -z "${DOMAIN:-}" ] || [ "$DOMAIN" = "localhost" ]; then
    # Create self-signed cert for local/no domain
    echo "Creating self-signed certificate..."
    docker run --rm \
      -v snippy-backend_certbot_certs:/etc/letsencrypt \
      alpine sh -c "
        apk add --no-cache openssl >/dev/null 2>&1
        mkdir -p /etc/letsencrypt/live/cert
        openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
          -keyout /etc/letsencrypt/live/cert/privkey.pem \
          -out /etc/letsencrypt/live/cert/fullchain.pem \
          -subj '/CN=localhost/O=Snippy/C=US' 2>/dev/null
      "
    echo "Self-signed certificate created"
  else
    # For Let's Encrypt, we need nginx running first
    echo "Starting services for Let's Encrypt challenge..."
    
    # Create temporary self-signed cert so nginx can start
    docker run --rm \
      -v snippy-backend_certbot_certs:/etc/letsencrypt \
      alpine sh -c "
        apk add --no-cache openssl >/dev/null 2>&1
        mkdir -p /etc/letsencrypt/live/cert
        openssl req -x509 -nodes -newkey rsa:2048 -days 1 \
          -keyout /etc/letsencrypt/live/cert/privkey.pem \
          -out /etc/letsencrypt/live/cert/fullchain.pem \
          -subj '/CN=${DOMAIN}/O=Snippy/C=US' 2>/dev/null
      "
    
    # Start nginx to handle ACME challenge
    systemctl restart ${SERVICE_NAME}
    sleep 10
    
    # Request Let's Encrypt certificate
    echo "Requesting Let's Encrypt certificate for ${DOMAIN}..."
    STAGING_FLAG=""
    if [ "${CERTBOT_STAGING:-false}" = "true" ]; then
      STAGING_FLAG="--staging"
      echo "Using Let's Encrypt staging server"
    fi
    
    $DC --profile ssl run --rm certbot certonly --webroot \
      -w /var/www/certbot \
      -d ${DOMAIN} \
      --email ${CERTBOT_EMAIL} \
      --agree-tos --no-eff-email --non-interactive \
      --cert-name cert \
      $STAGING_FLAG
    
    echo "Let's Encrypt certificate obtained!"
    
    # Reload nginx with new cert
    $DC exec nginx nginx -s reload || true
  fi
else
  echo "SSL certificates already exist"
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
    $DC exec -T postgres psql -U "${POSTGRES_USER:-snippy_user}" -d "${POSTGRES_DB:-snippy_production}" -f "/migrations/$(basename $migration)" || {
      echo "Warning: Migration $(basename $migration) may have already been applied or encountered an error"
    }
  fi
done
echo "Migrations completed"

# Check status
echo "Checking service status..."
systemctl status ${SERVICE_NAME} --no-pager || true

echo "Checking Docker containers..."
$DC ps || true

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
  $DC logs --tail=50 api || true
  exit 1
fi
