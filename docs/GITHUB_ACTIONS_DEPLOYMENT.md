# GitHub Actions Deployment with Docker Compose + systemd

This guide explains the automated deployment setup for Snippy Backend using GitHub Actions, Docker Compose, and systemd.

## Architecture Overview

The deployment uses a robust three-tier approach:

1. **GitHub Actions**: Compiles the Go binary in CI environment
2. **Docker Compose**: Orchestrates PostgreSQL and API containers
3. **systemd**: Ensures automatic startup and restart on failure

### Why This Approach?

✅ **No server compilation**: Binary is built in GitHub Actions with full resources
✅ **Automatic startup**: systemd starts services on server boot
✅ **Automatic recovery**: systemd restarts services if they crash
✅ **Simple management**: Single command to control everything
✅ **Production-ready**: Industry standard for service management

## Components

### 1. docker-compose.yml

Defines two services:

- **postgres**: PostgreSQL 16 database with health checks
- **api**: Alpine container running the compiled Go binary

The binary and `.env.production` are mounted as read-only volumes from the host.

### 2. snippy-backend.service

systemd unit file that:
- Runs `docker-compose up -d` on service start
- Runs `docker-compose down` on service stop
- Automatically restarts on failure
- Starts after Docker daemon is ready

### 3. GitHub Actions Workflow (build-and-deploy.yml)

On every push to `main`:

1. **Build**: Compiles static Go binary with `CGO_ENABLED=0`
2. **Copy**: Transfers binary, docker-compose.yml, and service file to server via SCP
3. **Deploy**: SSH to server and:
   - Installs systemd service (if not already installed)
   - Restarts the service (which restarts Docker Compose)
   - Verifies health check endpoint

## Prerequisites

- DigitalOcean droplet with SSH access
- Project already deployed once manually on the droplet (at `/root/snippy-backend`)
- GitHub repository with Actions enabled

## Setup Steps

### 1. Generate SSH Key for GitHub Actions

On your **local machine** (or any secure machine), generate a new SSH key pair:

```bash
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/github_actions_deploy
```

This creates two files:

- `~/.ssh/github_actions_deploy` (private key)
- `~/.ssh/github_actions_deploy.pub` (public key)

### 2. Add Public Key to Your Droplet

Copy the public key to your droplet:

```bash
# View the public key
cat ~/.ssh/github_actions_deploy.pub

# SSH to your droplet
ssh root@YOUR_DROPLET_IP

# Add the public key to authorized_keys
echo "PASTE_PUBLIC_KEY_HERE" >> ~/.ssh/authorized_keys

# Or for non-root user (recommended):
ssh deploy@YOUR_DROPLET_IP
echo "PASTE_PUBLIC_KEY_HERE" >> ~/.ssh/authorized_keys

# Set correct permissions
chmod 600 ~/.ssh/authorized_keys
chmod 700 ~/.ssh
```

### 3. Add GitHub Secrets

Go to your GitHub repository:

1. Click **Settings** → **Secrets and variables** → **Actions**
2. Click **New repository secret**

Add these secrets:

| Secret Name        | Value                                      | Description                                        |
| ------------------ | ------------------------------------------ | -------------------------------------------------- |
| `DROPLET_HOST`     | `YOUR_DROPLET_IP`                          | Your droplet's IP address (e.g., `164.90.xxx.xxx`) |
| `DROPLET_USERNAME` | `root` or `deploy`                         | SSH username (use `root` if that's what you use)   |
| `DROPLET_SSH_KEY`  | Contents of `~/.ssh/github_actions_deploy` | The **private key** (entire file contents)         |
| `DROPLET_PORT`     | `22`                                       | SSH port (optional, defaults to 22)                |

**⚠️ IMPORTANT: Copy the private key correctly:**

```bash
# Display the private key
cat ~/.ssh/github_actions_deploy

# The key should look like this:
# -----BEGIN OPENSSH PRIVATE KEY-----
# b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz
# ... (many lines) ...
# -----END OPENSSH PRIVATE KEY-----

# Copy the ENTIRE output including the BEGIN/END lines
# Paste it EXACTLY as shown into the DROPLET_SSH_KEY secret
# Do NOT add or remove any spaces, newlines, or characters
```

**Common mistakes to avoid:**
- ❌ Copying only part of the key
- ❌ Adding extra spaces or newlines
- ❌ Copying the `.pub` file (that's the public key, not private)
- ❌ Missing the BEGIN/END lines
- ✅ Copy the entire key with all lines intact

### 4. Create .env.production on Server

⚠️ **Important**: Never commit `.env.production` to your repository!

SSH to your droplet and create the production environment file:

```bash
ssh root@YOUR_DROPLET_IP
cd /root/snippy-api

# Create .env.production with your production configuration
nano .env.production
```

Example `.env.production` content:

```bash
# PostgreSQL Configuration
POSTGRES_USER=your_production_user
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_DB=snippy_production

# Database URL for application
DATABASE_URL=postgres://your_production_user:your_secure_password@postgres:5432/snippy_production?sslmode=disable

# Server Configuration
PORT=8080
GIN_MODE=release

# JWT Configuration - Generate with: openssl rand -base64 64
JWT_SECRET=your_production_jwt_secret_here

# CORS Configuration
CORS_ALLOWED_ORIGINS=https://yourdomain.com

# Logging
LOG_LEVEL=info
```

**Security tips**:
- Use strong, unique passwords
- Never use the same credentials as development
- Generate a secure JWT_SECRET: `openssl rand -base64 64`
- Restrict CORS to your actual domain

### 5. Test SSH Connection

Test that GitHub Actions can connect:

```bash
# From your local machine, test with the GitHub Actions key
ssh -i ~/.ssh/github_actions_deploy root@YOUR_DROPLET_IP "echo 'Connection successful!'"
```

## How It Works

### Deployment Flow

1. **Push to main**: Developer pushes code to `main` branch
2. **Build binary**: GitHub Actions compiles static Go binary
3. **Transfer files**: SCP copies binary, docker-compose.yml, and systemd service to server
4. **Install service**: Script installs systemd service (first time only)
5. **Restart**: systemd restarts the service, which runs `docker-compose up -d`
6. **Verify**: Health check confirms API is responding

### systemd Service Management

The `snippy-backend.service` file defines:

```ini
[Unit]
Description=Snippy Backend API
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/root/snippy-backend
ExecStart=/usr/bin/docker-compose up -d
ExecStop=/usr/bin/docker-compose down
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

This ensures:
- Service starts after Docker
- Runs docker-compose commands
- Restarts on failure
- Starts automatically on boot

## Server Management Commands

Once deployed, use these commands on the server:

### View Service Status

```bash
# Check systemd service status
systemctl status snippy-backend

# View service logs
journalctl -u snippy-backend -f

# Check container status
docker-compose ps

# View container logs
docker-compose logs -f
docker-compose logs -f api      # API only
docker-compose logs -f postgres # Postgres only
```

### Control the Service

```bash
# Restart everything
systemctl restart snippy-backend

# Stop everything
systemctl stop snippy-backend

# Start everything
systemctl start snippy-backend

# View recent logs (last 100 lines)
docker-compose logs --tail=100 api
```

### Health Check

```bash
# Test API health endpoint
curl http://localhost:8080/api/v1/health

# Should return:
# {"status":"healthy","timestamp":"2024-01-15T10:30:00Z"}
```

## Manual Trigger

You can manually trigger deployment:

1. Go to **Actions** tab in your GitHub repository
2. Click **Build and Deploy** workflow
3. Click **Run workflow** → **Run workflow**

## Monitoring Deployments

To view deployment logs:

1. Go to **Actions** tab in your repository
2. Click on the latest workflow run
3. Expand the deployment steps to see detailed logs

The deploy step shows:
- systemd service status
- Container status (`docker-compose ps`)
- Health check results
- Container logs (if health check fails)

## Rollback

If a deployment fails or introduces bugs:

### Option 1: Re-run Previous Workflow

1. Go to **Actions** → Find the last successful deployment
2. Click **Re-run all jobs**

### Option 2: Manual Rollback on Server

```bash
ssh root@YOUR_DROPLET_IP
cd /root/snippy-backend

# View git history
git log --oneline -10

# Checkout previous version
git checkout PREVIOUS_COMMIT_HASH

# Restart service
systemctl restart snippy-backend

# Verify
docker-compose ps
curl http://localhost:8080/api/v1/health
```

## Automatic Startup on Server Boot

The systemd service ensures your application starts automatically when the server boots:

```bash
# Enable autostart (done automatically by deployment)
systemctl enable snippy-backend

# Check if enabled
systemctl is-enabled snippy-backend
# Should return: enabled

# Test by rebooting
sudo reboot

# After reboot, check status
systemctl status snippy-backend
docker-compose ps
```


## Security Best Practices

✅ **DO:**

- Use a dedicated SSH key for GitHub Actions (never reuse personal keys)
- Use a non-root user (`deploy`) for deployments
- Store private keys in GitHub Secrets (never commit them)
- Limit SSH key permissions on the droplet
- Enable 2FA on your GitHub account

❌ **DON'T:**

- Commit SSH private keys to the repository
- Use root user for deployments (create `deploy` user instead)
- Share SSH keys between different services
- Disable host key checking (security risk)

## Troubleshooting

### Deployment Fails with "Permission Denied"

```bash
# On droplet, check file ownership
ls -la /opt/snippy-backend
sudo chown -R deploy:deploy /opt/snippy-backend

# Verify SSH key permissions
chmod 600 ~/.ssh/authorized_keys
```

### Deployment Fails with "Repository not found"

```bash
# On droplet, ensure git remote is set correctly
cd /opt/snippy-backend
git remote -v

# Should show your GitHub repository
# If not, set it:
git remote set-url origin git@github.com:jheysaaz/snippy-backend.git
```

### Docker Commands Fail

```bash
# Add deploy user to docker group
sudo usermod -aG docker deploy

# Logout and login for changes to take effect
# Or restart SSH session
```

### Health Check Fails

```bash
# On droplet, check if API is running
docker-compose ps

# Check API logs
docker-compose logs api

# Manually test health endpoint
curl http://localhost:8080/health
```

## Workflow Customization

### Deploy Only on Tagged Releases

To deploy only when you create a release tag:

```yaml
on:
  push:
    tags:
      - "v*" # Matches v1.0.0, v2.1.3, etc.
```

### Add Slack/Discord Notifications

Add notification steps to the workflow:

```yaml
- name: Notify Slack
  if: always()
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

### Run Database Migrations

Add migration step before deployment:

```yaml
- name: Run migrations
  uses: appleboy/ssh-action@v1.0.3
  with:
    host: ${{ secrets.DROPLET_HOST }}
    username: ${{ secrets.DROPLET_USERNAME }}
    key: ${{ secrets.DROPLET_SSH_KEY }}
    script: |
      cd /opt/snippy-backend
      docker-compose exec -T postgres psql -U $POSTGRES_USER -d $POSTGRES_DB -f migrations/001_add_column.sql
```

## Cost

GitHub Actions is free for public repositories and includes:

- 2,000 minutes/month for private repositories (free tier)
- This deployment workflow typically uses ~2-3 minutes per deployment

## Next Steps

After setting up automated deployment:

1. Make a small change to your code
2. Push to `main` branch
3. Go to **Actions** tab and watch your first automated deployment
4. Verify the deployment succeeded by checking your API

## Support

If you encounter issues:

1. Check the workflow logs in GitHub Actions
2. SSH into your droplet and check Docker logs: `docker-compose logs`
3. Verify all secrets are correctly set in GitHub
4. Test SSH connection manually with the GitHub Actions key
