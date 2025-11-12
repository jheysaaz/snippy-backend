#!/bin/bash
# SSL Certificate Renewal Script for api.snippy.jheysonsaavedra.com
# This script renews Let's Encrypt certificates and restarts services

set -e

DOMAIN="api.snippy.jheysonsaavedra.com"
CERT_PATH="/etc/letsencrypt/live/$DOMAIN"
API_DIR="/root/snippy-api"

echo "🔄 SSL Certificate Renewal for $DOMAIN"

# Check if certificate exists and when it expires
if [ -f "$CERT_PATH/fullchain.pem" ]; then
    EXPIRY=$(openssl x509 -enddate -noout -in "$CERT_PATH/fullchain.pem" | cut -d= -f2)
    echo "📅 Current certificate expires: $EXPIRY"

    # Check if certificate expires in less than 30 days
    if openssl x509 -checkend 2592000 -noout -in "$CERT_PATH/fullchain.pem"; then
        echo "✅ Certificate is still valid for more than 30 days"
        exit 0
    else
        echo "⚠️ Certificate expires soon, renewing..."
    fi
else
    echo "❌ No certificate found, obtaining new one..."
fi

# Stop services temporarily for renewal
echo "🛑 Stopping services for renewal..."
cd "$API_DIR"
docker-compose down 2>/dev/null || true

# Renew or obtain certificate
echo "🔄 Renewing Let's Encrypt certificate..."
certbot certonly --standalone --non-interactive --force-renewal \
    --agree-tos \
    --email "$LETSENCRYPT_EMAIL" \
    -d "$DOMAIN"

# Update API SSL certificates
echo "🔐 Updating API SSL certificates..."
cp "$CERT_PATH/fullchain.pem" "$API_DIR/ssl/api/api.crt"
cp "$CERT_PATH/privkey.pem" "$API_DIR/ssl/api/api.key"
chmod 644 "$API_DIR/ssl/api/api.crt"
chmod 600 "$API_DIR/ssl/api/api.key"

# Restart services
echo "🚀 Restarting services..."
docker-compose up -d

# Wait for startup
sleep 15

# Verify renewal
echo "🏥 Verifying renewed certificate..."
if curl -f https://$DOMAIN/api/v1/health; then
    echo "✅ SSL renewal successful!"

    # Log renewal
    echo "$(date): SSL certificate renewed successfully" >> /var/log/ssl-renewal.log
else
    echo "❌ SSL renewal verification failed"
    # Log failure
    echo "$(date): SSL certificate renewal failed" >> /var/log/ssl-renewal.log
    exit 1
fi

echo "🎉 Certificate renewal completed successfully!"
