#!/bin/bash
set -e

echo "🚀 Deploying..."

# Start PostgreSQL
docker-compose up -d postgres
sleep 3

# Restart API
pkill -f snippy-api || true
nohup ./snippy-api > api.log 2>&1 &
sleep 2

# Test
curl -f http://localhost:8080/health && echo "✅ Deployed!" || echo "❌ Failed"
