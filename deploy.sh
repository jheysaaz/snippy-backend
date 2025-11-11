#!/bin/bash

# Deployment script for DigitalOcean droplet
# Usage: ./deploy.sh [production|staging]

set -e

ENV=${1:-production}
COMPOSE_FILE="docker-compose.yml"
PROD_FILE="docker-compose.prod.yml"

echo "🚀 Starting deployment for $ENV environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env.production exists
if [ ! -f ".env.production" ]; then
    echo -e "${RED}❌ Error: .env.production file not found!${NC}"
    echo "Please create .env.production with your production settings."
    exit 1
fi

# Check for required secrets
if grep -q "CHANGE_ME" .env.production; then
    echo -e "${RED}❌ Error: Please update all CHANGE_ME values in .env.production${NC}"
    exit 1
fi

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed!${NC}"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed!${NC}"
    exit 1
fi

echo "📦 Pulling latest Docker images..."
# No need to build - images are pre-built in GitHub Actions
docker-compose -f $COMPOSE_FILE pull

echo "🔄 Stopping existing containers..."
docker-compose -f $COMPOSE_FILE -f $PROD_FILE down

echo "🗑️  Cleaning up unused Docker resources..."
docker system prune -f

echo "🆙 Starting services..."
if [ "$ENV" = "production" ]; then
    docker-compose -f $COMPOSE_FILE -f $PROD_FILE up -d
else
    docker-compose -f $COMPOSE_FILE up -d
fi

echo "⏳ Waiting for services to be healthy..."
sleep 5

# Check if services are running
if docker-compose ps | grep -q "Up"; then
    echo -e "${GREEN}✅ Services are running!${NC}"
    
    echo ""
    echo "📊 Container Status:"
    docker-compose ps
    
    echo ""
    echo "📝 Logs (last 20 lines):"
    docker-compose logs --tail=20
    
    echo ""
    echo -e "${GREEN}✨ Deployment completed successfully!${NC}"
    echo ""
    echo "Useful commands:"
    echo "  - View logs: docker-compose logs -f"
    echo "  - Check status: docker-compose ps"
    echo "  - Stop services: docker-compose down"
    echo "  - Restart services: docker-compose restart"
    echo "  - Database backup: docker exec snippy-postgres pg_dump -U \$POSTGRES_USER \$POSTGRES_DB > backup.sql"
else
    echo -e "${RED}❌ Deployment failed! Check logs with: docker-compose logs${NC}"
    exit 1
fi
