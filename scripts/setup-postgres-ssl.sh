#!/bin/bash
# PostgreSQL SSL Setup Script
# This script generates SSL certificates for PostgreSQL

set -e

echo "🔐 Setting up PostgreSQL SSL certificates..."

# Create SSL directory
mkdir -p ./ssl/postgres
cd ./ssl/postgres

# Generate private key for PostgreSQL server
openssl genrsa -out server.key 2048
chmod 600 server.key

# Generate certificate signing request
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=State/L=City/O=Organization/CN=postgres"

# Generate self-signed certificate (valid for 10 years)
openssl x509 -req -in server.csr -signkey server.key -out server.crt -days 3650

# Set proper permissions
chmod 600 server.key
chmod 644 server.crt

# Create root certificate (for client verification)
cp server.crt root.crt

echo "✅ PostgreSQL SSL certificates generated successfully!"
echo "📁 Certificates location: ./ssl/postgres/"
echo "   - server.key (private key)"
echo "   - server.crt (server certificate)"
echo "   - root.crt (root certificate for clients)"

cd ../..
