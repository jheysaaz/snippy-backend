#!/bin/bash
# API SSL Setup Script
# This script generates SSL certificates for the Go API

set -e

echo "🔐 Setting up API SSL certificates..."

# Create SSL directory for API
mkdir -p ./ssl/api
cd ./ssl/api

# Get domain from user input
read -p "Enter your domain name (e.g., api.yourdomain.com): " DOMAIN
if [[ -z "$DOMAIN" ]]; then
    DOMAIN="localhost"
    echo "Using localhost as domain..."
fi

# Generate private key for API server
openssl genrsa -out api.key 2048
chmod 600 api.key

# Create certificate configuration with Subject Alternative Names
cat > api.conf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C=US
ST=State
L=City
O=Organization
CN=${DOMAIN}

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${DOMAIN}
DNS.2 = www.${DOMAIN}
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF

# Generate certificate signing request
openssl req -new -key api.key -out api.csr -config api.conf

# Generate self-signed certificate (valid for 1 year)
openssl x509 -req -in api.csr -signkey api.key -out api.crt -days 365 -extensions v3_req -extfile api.conf

# Set proper permissions
chmod 600 api.key
chmod 644 api.crt

echo "✅ API SSL certificates generated successfully!"
echo "📁 Certificates location: ./ssl/api/"
echo "   - api.key (private key)"
echo "   - api.crt (server certificate)"
echo "   - Domain: ${DOMAIN}"

cd ../..
