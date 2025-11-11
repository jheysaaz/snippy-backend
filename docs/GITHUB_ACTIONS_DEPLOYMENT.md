# GitHub Actions Deployment Setup

This guide explains how to set up automated deployment to your DigitalOcean droplet using GitHub Actions.

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

### 4. Verify Project Path on Droplet

Make sure your project is at `/opt/snippy-backend` on the droplet:

```bash
ssh deploy@YOUR_DROPLET_IP
ls -la /opt/snippy-backend
```

If it's in a different location, update the workflow file (`.github/workflows/deploy.yml`) with the correct path.

### 5. Test SSH Connection

Test that GitHub Actions can connect:

```bash
# From your local machine, test with the GitHub Actions key
ssh -i ~/.ssh/github_actions_deploy deploy@YOUR_DROPLET_IP "echo 'Connection successful!'"
```

## How It Works

The deployment workflow (`.github/workflows/deploy.yml`) automatically:

1. **Triggers** on every push to `main` branch (or manual workflow dispatch)
2. **Connects** to your droplet via SSH
3. **Pulls** the latest code from GitHub
4. **Runs** the `deploy.sh` script to build and restart containers
5. **Verifies** the API is healthy after deployment
6. **Notifies** if the deployment fails

## Manual Trigger

You can also trigger the deployment manually:

1. Go to **Actions** tab in your GitHub repository
2. Click **Deploy to DigitalOcean** workflow
3. Click **Run workflow** → **Run workflow**

## Monitoring Deployments

To view deployment logs:

1. Go to **Actions** tab in your repository
2. Click on the latest workflow run
3. Expand the deployment steps to see detailed logs

## Rollback

If a deployment fails, you can rollback on the droplet:

```bash
ssh deploy@YOUR_DROPLET_IP
cd /opt/snippy-backend

# View git history
git log --oneline -10

# Rollback to previous commit
git reset --hard PREVIOUS_COMMIT_HASH

# Redeploy
./deploy.sh production
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
