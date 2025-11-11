# Deployment Guide

## Deployment Strategy

This project uses a **manual deployment** approach to ensure stability:

- **CI runs on every commit** to `main` and `develop` branches
- **Deployment only happens**:
  - Manually via GitHub Actions UI
  - Automatically when creating version tags (e.g., `v1.0.0`)

## How to Deploy

### Option 1: Manual Deployment via GitHub Actions

1. Go to [Actions](../../actions/workflows/build-and-deploy.yml) tab
2. Click "Run workflow"
3. Select the branch to deploy
4. Choose environment (production/staging)
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

Before deploying, ensure:

- ✅ All tests pass locally: `make test-with-db`
- ✅ Code is formatted: `make format`
- ✅ Linter passes: `make lint`
- ✅ Security scan passes: `make security`
- ✅ Changes are committed and pushed to `main`

## CI/CD Workflow

```
┌─────────────────┐
│  Push to main   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Run CI Tests   │ ◄── Runs on every commit
│  - Unit tests   │
│  - Linting      │
│  - Security     │
└─────────────────┘
         │
         │ (Manual trigger or tag)
         ▼
┌─────────────────┐
│ Deploy Workflow │ ◄── Runs ONLY when:
│  - Run CI again │     - Manual trigger
│  - Build binary │     - Version tag
│  - Deploy       │
└─────────────────┘
```

## Rollback

If deployment fails or issues arise:

1. **Quick rollback**: Deploy previous version tag

   ```bash
   git tag v1.0.0-rollback v1.0.0^  # Tag previous commit
   git push origin v1.0.0-rollback
   ```

2. **SSH to server**:
   ```bash
   ssh root@your-server
   cd /root/snippy-api
   docker-compose down
   # Fix issues or restore previous binary
   docker-compose up -d
   ```

## Monitoring Deployment

After deployment:

1. Check GitHub Actions logs
2. Verify health endpoint: `curl http://your-server:8080/api/v1/health`
3. Check service status: `systemctl status snippy-api`
4. View logs: `docker-compose logs -f api`
