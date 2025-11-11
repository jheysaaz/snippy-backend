# Deployment Guide - DigitalOcean Droplet

This guide covers deploying Snippy Backend to a DigitalOcean droplet using Docker Compose.

## Prerequisites

- DigitalOcean droplet with Ubuntu 22.04 LTS (minimum 2GB RAM recommended)
- Domain name pointed to your droplet's IP (optional, but recommended)
- SSH access to your droplet

## Initial Server Setup

### 1. Connect to Your Droplet

```bash
ssh root@your-droplet-ip
```

### 2. Update System

```bash
apt update && apt upgrade -y
```

### 3. Install Docker

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Start Docker service
systemctl start docker
systemctl enable docker

# Verify installation
docker --version
```

### 4. Install Docker Compose

```bash
# Install Docker Compose
apt install docker-compose-plugin -y

# Verify installation
docker-compose version
```

### 5. Install Git

```bash
apt install git -y
```

### 6. Create Non-Root User (Recommended)

```bash
# Create user
adduser deploy
usermod -aG sudo,docker deploy

# Switch to new user
su - deploy
```

## Application Deployment

### 1. Clone Repository

```bash
cd /opt
sudo git clone https://github.com/jheysaaz/snippy-backend.git
sudo chown -R deploy:deploy snippy-backend
cd snippy-backend
```

### 2. Configure Environment Variables

**IMPORTANT: Never commit `.env.production` to Git! It contains secrets.**

```bash
# Copy the example template (this is in Git)
cp .env.production.example .env.production

# Edit with your actual production values
nano .env.production
```

**Required Changes:**

```bash
# Generate strong password for PostgreSQL
POSTGRES_PASSWORD=$(openssl rand -base64 32)

# Generate JWT secret
JWT_SECRET=$(openssl rand -base64 64)

# Set your domain
CORS_ALLOWED_ORIGINS=https://yourdomain.com

# Update database URL with the generated password
DATABASE_URL=postgres://snippy_user:YOUR_POSTGRES_PASSWORD@postgres:5432/snippy_production?sslmode=require
```

**Security Note:** The `.env.production` file you create will only exist on your server and is excluded from Git via `.gitignore`.

### 3. Create Backup Directory

```bash
mkdir -p backups
chmod 700 backups
```

### 4. Deploy Application

```bash
# Make deploy script executable
chmod +x deploy.sh

# Run deployment
./deploy.sh production
```

### 5. Verify Deployment

```bash
# Check running containers
docker-compose ps

# Check API health
curl http://localhost:8080/health

# View logs
docker-compose logs -f
```

## Setting Up Nginx Reverse Proxy (Recommended)

### 1. Install Nginx

```bash
sudo apt install nginx -y
```

### 2. Configure Nginx

```bash
sudo nano /etc/nginx/sites-available/snippy
```

Add this configuration:

```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # API proxy
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint
    location /health {
        access_log off;
        proxy_pass http://localhost:8080/health;
    }
}
```

### 3. Enable Site and Restart Nginx

```bash
sudo ln -s /etc/nginx/sites-available/snippy /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### 4. Install SSL Certificate (Let's Encrypt)

```bash
# Install Certbot
sudo apt install certbot python3-certbot-nginx -y

# Obtain certificate
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Auto-renewal is configured automatically
# Test renewal
sudo certbot renew --dry-run
```

## Firewall Configuration

```bash
# Install UFW if not present
sudo apt install ufw -y

# Allow SSH
sudo ufw allow 22/tcp

# Allow HTTP and HTTPS
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

## Database Management

### Backup Database

```bash
# Manual backup
docker exec snippy-postgres pg_dump -U snippy_user snippy_production > backups/backup-$(date +%Y%m%d-%H%M%S).sql

# Compress backup
gzip backups/backup-*.sql
```

### Restore Database

```bash
# Restore from backup
gunzip < backups/backup-20250111-120000.sql.gz | docker exec -i snippy-postgres psql -U snippy_user snippy_production
```

### Automated Backups

Create backup script:

```bash
sudo nano /usr/local/bin/backup-snippy-db.sh
```

Add content:

```bash
#!/bin/bash
BACKUP_DIR="/opt/snippy-backend/backups"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="$BACKUP_DIR/backup-$TIMESTAMP.sql"

# Create backup
docker exec snippy-postgres pg_dump -U snippy_user snippy_production > "$BACKUP_FILE"

# Compress
gzip "$BACKUP_FILE"

# Keep only last 7 days of backups
find "$BACKUP_DIR" -name "backup-*.sql.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_FILE.gz"
```

Make executable and add to crontab:

```bash
sudo chmod +x /usr/local/bin/backup-snippy-db.sh

# Run daily at 2 AM
sudo crontab -e
# Add this line:
0 2 * * * /usr/local/bin/backup-snippy-db.sh >> /var/log/snippy-backup.log 2>&1
```

## Monitoring and Logs

### View Application Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f api
docker-compose logs -f postgres

# Last 100 lines
docker-compose logs --tail=100
```

### System Monitoring

```bash
# Install htop for better monitoring
sudo apt install htop -y

# Check resource usage
htop

# Docker stats
docker stats
```

### Log Rotation

Docker Compose is configured with log rotation (10MB max, 3 files), but you can also configure system-wide:

```bash
sudo nano /etc/logrotate.d/docker-compose
```

Add:

```
/var/lib/docker/containers/*/*.log {
    rotate 7
    daily
    compress
    size=10M
    missingok
    delaycompress
    copytruncate
}
```

## Updating the Application

### 1. Pull Latest Changes

```bash
cd /opt/snippy-backend
git pull origin main
```

### 2. Rebuild and Redeploy

```bash
./deploy.sh production
```

### 3. Zero-Downtime Update (Alternative)

```bash
# Build new image
docker-compose build

# Create new containers
docker-compose up -d --no-deps --build api

# Old containers are automatically removed
```

## Security Checklist

- [ ] Changed all default passwords in `.env.production`
- [ ] Generated strong JWT secret (64+ characters)
- [ ] Configured firewall (UFW)
- [ ] Installed SSL certificate (HTTPS)
- [ ] Set up automated backups
- [ ] Configured log rotation
- [ ] Limited PostgreSQL exposure (not exposed to internet)
- [ ] Updated CORS_ALLOWED_ORIGINS to your domain
- [ ] Enabled automatic security updates
- [ ] Set up monitoring/alerting

### Enable Automatic Security Updates

```bash
sudo apt install unattended-upgrades -y
sudo dpkg-reconfigure -plow unattended-upgrades
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
docker-compose logs

# Check if ports are in use
sudo netstat -tulpn | grep -E '8080|5432'

# Restart services
docker-compose restart
```

### Database Connection Issues

```bash
# Check PostgreSQL logs
docker-compose logs postgres

# Test database connection
docker exec -it snippy-postgres psql -U snippy_user snippy_production

# Verify environment variables
docker-compose config
```

### High Memory Usage

```bash
# Check container stats
docker stats

# Adjust resource limits in docker-compose.prod.yml
# Restart services
docker-compose restart
```

### Reset Everything (Nuclear Option)

```bash
# Stop and remove everything
docker-compose down -v

# Remove images
docker rmi $(docker images -q snippy-*)

# Clean system
docker system prune -af --volumes

# Redeploy
./deploy.sh production
```

## Performance Optimization

### PostgreSQL Tuning

The production compose file includes optimized PostgreSQL settings. For larger droplets, adjust in `docker-compose.prod.yml`:

```yaml
shared_buffers=512MB      # 25% of RAM
effective_cache_size=1536MB  # 75% of RAM
```

### Enable Redis Caching (Optional)

Add Redis service to docker-compose.yml for session/rate limit caching.

## Maintenance

### Weekly Maintenance Checklist

- [ ] Check disk space: `df -h`
- [ ] Review logs for errors
- [ ] Verify backups are running
- [ ] Update packages: `sudo apt update && sudo apt upgrade`
- [ ] Check SSL certificate expiry
- [ ] Monitor resource usage

### Monthly Maintenance

- [ ] Review and rotate logs
- [ ] Test backup restoration
- [ ] Update Docker images: `docker-compose pull`
- [ ] Security audit: `docker scan snippy-api:latest`
- [ ] Update application dependencies

## Scaling Considerations

### Vertical Scaling (Upgrade Droplet)

1. Create snapshot of droplet
2. Resize droplet in DigitalOcean panel
3. Update resource limits in `docker-compose.prod.yml`
4. Restart services: `docker-compose restart`

### Horizontal Scaling (Multiple Droplets)

For high traffic, consider:

- Managed PostgreSQL database (DigitalOcean Managed Database)
- Load balancer (DigitalOcean Load Balancer)
- Multiple API droplets
- Redis for session management
- CDN for static assets

## Support

For issues or questions:

- GitHub Issues: https://github.com/jheysaaz/snippy-backend/issues
- Check logs: `docker-compose logs`
- Review documentation in `/docs` folder

## Cost Estimation

**Minimum Setup (1 Droplet):**

- Basic Droplet (2GB RAM): $12/month
- Backups: $2.40/month (20% of droplet cost)
- Domain: ~$12/year
- **Total: ~$15/month**

**Recommended Setup:**

- Regular Droplet (4GB RAM): $24/month
- Managed Database: $15/month
- Backups: $4.80/month
- Domain: ~$12/year
- **Total: ~$45/month**
