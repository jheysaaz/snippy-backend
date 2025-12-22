# Deployment Guide

## Deployment Strategy

This project uses a **manual deployment** approach to ensure stability:

- **CI runs on every commit** to `main` and `develop` branches
- **Deployment only happens**:
  - Manually via GitHub Actions UI
  - Automatically when creating version tags (e.g., `v1.0.0`)

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Nginx     │────▶│   API       │────▶│  PostgreSQL │
│  (SSL/TLS)  │     │  (Go/Gin)   │     │             │
│  :80/:443   │     │  :8080      │     │  :5432      │
└─────────────┘     └─────────────┘     └─────────────┘
```

- **Nginx**: Handles SSL termination (Let's Encrypt), reverse proxy
- **API**: Snippy backend service
- **PostgreSQL**: Database

## How to Deploy

### Option 1: Manual Deployment via GitHub Actions

1. Go to [Actions](../../actions/workflows/deploy.yml) tab
2. Click "Run workflow"
3. Select the branch to deploy
4. Enter the CI build run ID (from a successful CI run)
5. Click "Run workflow" button

### Option 2: Deploy with Version Tag

1. Create and push a version tag:

   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. This will automatically:
   - Run all CI checks (tests, linting, security)
   - Build the binary
   - Deploy to production if checks pass

## Pre-deployment Checklist

- [ ] All tests pass locally: `make test-with-db`
- [ ] Code is formatted: `make format`
- [ ] Linter passes: `make lint`
- [ ] Security scan passes: `make security`
- [ ] Changes are committed and pushed to `main`
- [ ] `.env.production` is configured on the server

## Server Configuration

### Required Environment Variables (`.env.production`)

```env
# Database
POSTGRES_USER=snippy_user
POSTGRES_PASSWORD=<secure-password>
POSTGRES_DB=snippy_production
DATABASE_URL=postgres://snippy_user:<password>@postgres:5432/snippy_production?sslmode=disable

# API
PORT=8080
GIN_MODE=release
JWT_SECRET=<generate-with: openssl rand -base64 32>
CORS_ALLOWED_ORIGINS=https://yourdomain.com

# SSL (for Let's Encrypt)
DOMAIN=snippy.yourdomain.com
CERTBOT_EMAIL=your@email.com
```

### SSL Certificates

**First deployment** (self-signed cert created automatically):

```bash
make up
```

**Get Let's Encrypt certificate** (after DNS is configured):

```bash
# Ensure DOMAIN and CERTBOT_EMAIL are set in .env.production
make ssl-clean
make ssl-init
make restart
```

**Renew certificates**:

```bash
make ssl-renew
```

## CI/CD Workflow

1. **CI runs on every commit** to `main`/`develop` (tests, linting, security)
2. **Deploy runs only when**: Manual trigger via UI or version tag pushed

## Rollback

Deploy previous version tag:

```bash
git tag v1.0.0-rollback v1.0.0^
git push origin v1.0.0-rollback
```

## Monitoring Deployment

After deployment:

```bash
# Check health endpoint (via nginx)
curl https://your-domain.com/api/v1/health

# Check service status
systemctl status snippy-api

# View all container logs
docker compose logs -f

# View specific service logs
docker compose logs -f api
docker compose logs -f nginx

# Check SSL certificate status
make ssl-status

# Check container status
docker compose ps
```

## Troubleshooting

### Containers not starting

```bash
docker compose logs --tail=50
```

### SSL certificate issues

```bash
# Check certificate
make ssl-status

# Regenerate certificate
make ssl-clean
make ssl-init
docker compose restart nginx
```

### Database connection issues

```bash
docker compose exec postgres psql -U snippy_user -d snippy_production -c '\conninfo'
```
