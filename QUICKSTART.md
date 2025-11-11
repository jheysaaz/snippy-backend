# Quick Deployment Reference

## 🚀 Quick Start Commands

### First Time Deployment

```bash
# 1. Update environment variables
cp .env.production .env.production
nano .env.production  # Update all CHANGE_ME values

# 2. Deploy
./deploy.sh production
```

### Common Commands

```bash
# View logs
docker-compose logs -f

# Restart services
docker-compose restart

# Stop services
docker-compose down

# Update application
git pull && ./deploy.sh production

# Backup database
make -f Makefile.docker backup

# Check health
curl http://localhost:8080/health
```

## 🔑 Security Checklist

Before deploying, ensure you've updated `.env.production`:

```bash
# Generate secure passwords
POSTGRES_PASSWORD=$(openssl rand -base64 32)
JWT_SECRET=$(openssl rand -base64 64)

# Update these values in .env.production
```

**Must Change:**

- [ ] `POSTGRES_PASSWORD` - Strong password for database
- [ ] `JWT_SECRET` - 64+ character random string
- [ ] `CORS_ALLOWED_ORIGINS` - Your actual domain(s)
- [ ] `DATABASE_URL` - Match the POSTGRES_PASSWORD

## 📊 Monitoring

### Check Status

```bash
docker-compose ps                    # Container status
docker stats                         # Resource usage
docker-compose logs --tail=50       # Recent logs
```

### Health Checks

```bash
# API Health
curl http://localhost:8080/health

# Database Health
docker exec snippy-postgres pg_isready -U snippy_user
```

## 🔄 Backup & Restore

### Manual Backup

```bash
docker exec snippy-postgres pg_dump -U snippy_user snippy_production > backup.sql
gzip backup.sql
```

### Restore

```bash
gunzip < backup.sql.gz | docker exec -i snippy-postgres psql -U snippy_user snippy_production
```

## 🐛 Troubleshooting

### Services Won't Start

```bash
docker-compose logs              # Check errors
docker-compose down -v           # Clean slate
./deploy.sh production           # Redeploy
```

### Port Already in Use

```bash
sudo netstat -tulpn | grep 8080  # Find process
sudo kill -9 <PID>               # Kill process
```

### Database Connection Failed

```bash
# Check if PostgreSQL is ready
docker exec snippy-postgres pg_isready

# Verify credentials
docker-compose config | grep DATABASE_URL
```

## 📈 Resource Limits

Current configuration (adjust in `docker-compose.prod.yml`):

| Service  | CPU Limit | Memory Limit | Recommended For     |
| -------- | --------- | ------------ | ------------------- |
| API      | 1 CPU     | 256MB        | Basic Droplet (2GB) |
| Postgres | 1 CPU     | 512MB        | Basic Droplet (2GB) |

For larger droplets (4GB+):

- API: 2 CPU, 512MB
- Postgres: 2 CPU, 1GB

## 🌐 Nginx Setup (Optional but Recommended)

### Install and Configure

```bash
sudo apt install nginx -y
sudo nano /etc/nginx/sites-available/snippy
# Copy configuration from DEPLOYMENT.md
sudo ln -s /etc/nginx/sites-available/snippy /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### Add SSL

```bash
sudo apt install certbot python3-certbot-nginx -y
sudo certbot --nginx -d yourdomain.com
```

## 🔒 Firewall

```bash
sudo ufw allow 22/tcp     # SSH
sudo ufw allow 80/tcp     # HTTP
sudo ufw allow 443/tcp    # HTTPS
sudo ufw enable
```

## 📝 Important Files

- `docker-compose.yml` - Main configuration (development + production base)
- `docker-compose.prod.yml` - Production overrides
- `.env.production` - Production environment variables
- `Dockerfile` - Multi-stage build configuration
- `deploy.sh` - Automated deployment script
- `DEPLOYMENT.md` - Complete deployment guide

## 🎯 Production Deployment Workflow

```bash
# On your local machine
git push origin main

# On DigitalOcean droplet
cd /opt/snippy-backend
git pull origin main
./deploy.sh production

# Verify
curl https://yourdomain.com/health
```

## 💰 Cost Estimate

**Minimum Setup:**

- Basic Droplet (2GB): $12/month
- Total: ~$15/month with backups

**Recommended Setup:**

- Regular Droplet (4GB): $24/month
- Total: ~$27/month with backups

## 📞 Support

- Full Guide: See `DEPLOYMENT.md`
- Issues: https://github.com/jheysaaz/snippy-backend/issues
- Logs: `docker-compose logs -f`
