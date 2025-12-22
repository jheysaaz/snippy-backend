#!/bin/bash
# Initialize SSL certificates for Snippy
# Creates self-signed certs for local dev, or requests Let's Encrypt for production

set -e

CERT_DIR="/etc/letsencrypt/live/cert"
DOMAIN="${1:-}"
EMAIL="${2:-}"
STAGING="${3:-false}"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[SSL]${NC} $1"; }
warn() { echo -e "${YELLOW}[SSL]${NC} $1"; }

# Create directory
mkdir -p "$CERT_DIR"

# Check if certs already exist
if [ -f "$CERT_DIR/fullchain.pem" ] && [ -f "$CERT_DIR/privkey.pem" ]; then
    log "Certificates already exist"
    exit 0
fi

# If no domain provided, create self-signed cert for local development
if [ -z "$DOMAIN" ] || [ "$DOMAIN" = "localhost" ]; then
    log "Creating self-signed certificate for local development..."
    openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
        -keyout "$CERT_DIR/privkey.pem" \
        -out "$CERT_DIR/fullchain.pem" \
        -subj "/CN=localhost/O=Snippy/C=US" \
        2>/dev/null
    log "Self-signed certificate created"
    exit 0
fi

# For production, we need email
if [ -z "$EMAIL" ]; then
    warn "CERTBOT_EMAIL required for Let's Encrypt"
    warn "Creating self-signed cert as fallback..."
    openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
        -keyout "$CERT_DIR/privkey.pem" \
        -out "$CERT_DIR/fullchain.pem" \
        -subj "/CN=$DOMAIN/O=Snippy/C=US" \
        2>/dev/null
    exit 0
fi

# Request Let's Encrypt certificate
log "Requesting Let's Encrypt certificate for $DOMAIN..."

CERTBOT_ARGS="certonly --webroot -w /var/www/certbot"
CERTBOT_ARGS="$CERTBOT_ARGS -d $DOMAIN --email $EMAIL"
CERTBOT_ARGS="$CERTBOT_ARGS --agree-tos --no-eff-email --non-interactive"
CERTBOT_ARGS="$CERTBOT_ARGS --cert-name cert"

if [ "$STAGING" = "true" ]; then
    warn "Using Let's Encrypt staging server"
    CERTBOT_ARGS="$CERTBOT_ARGS --staging"
fi

certbot $CERTBOT_ARGS

log "Let's Encrypt certificate obtained!"
