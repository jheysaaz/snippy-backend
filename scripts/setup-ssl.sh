#!/bin/bash
# Complete SSL Setup Script
# This script sets up SSL for both PostgreSQL and the API

set -e

echo "🔐 Setting up SSL for Snippy Backend..."
echo "This will configure SSL for both PostgreSQL and the Go API"

# Make scripts executable
chmod +x scripts/setup-postgres-ssl.sh
chmod +x scripts/setup-api-ssl.sh

# Setup PostgreSQL SSL
echo ""
echo "1️⃣ Setting up PostgreSQL SSL certificates..."
./scripts/setup-postgres-ssl.sh

# Setup API SSL
echo ""
echo "2️⃣ Setting up API SSL certificates..."
./scripts/setup-api-ssl.sh

# Create .gitignore entry for SSL certificates (security)
echo ""
echo "3️⃣ Adding SSL certificates to .gitignore..."
if ! grep -q "ssl/" .gitignore 2>/dev/null; then
    echo "" >> .gitignore
    echo "# SSL Certificates (never commit private keys)" >> .gitignore
    echo "ssl/" >> .gitignore
    echo "*.key" >> .gitignore
    echo "*.crt" >> .gitignore
    echo "*.pem" >> .gitignore
fi

echo ""
echo "✅ SSL setup completed successfully!"
echo ""
echo "📋 Next steps:"
echo "   1. Build your application: go build"
echo "   2. Start with Docker: docker-compose up -d"
echo "   3. Your API will be available at: https://yourdomain.com"
echo "   4. PostgreSQL will use SSL connections"
echo ""
echo "🔒 Security notes:"
echo "   - SSL certificates are excluded from Git"
echo "   - Database connections now require SSL"
echo "   - API serves on HTTPS (port 443)"
echo "   - For production, consider using real certificates from Let's Encrypt"
echo ""
echo "🔧 To use Let's Encrypt instead of self-signed certificates:"
echo "   sudo apt install certbot"
echo "   sudo certbot certonly --standalone -d yourdomain.com"
echo "   Then update SSL_CERT_FILE and SSL_KEY_FILE paths in .env.production"
