# GitHub Actions CI/CD Workflows

This directory contains the CI/CD pipeline configuration for the TodoList API.

## Overview

The CI/CD pipeline automatically builds, tests, and deploys the TodoList API to Oracle Cloud Infrastructure (OCI) whenever code is pushed to the main branch.

## Workflow: CI/CD Pipeline

**File:** `ci-cd.yml`

### Triggers

- **Push** to `main` or `develop` branches
- **Pull requests** to `main` or `develop` branches
- **Manual trigger** via workflow_dispatch

### Jobs

#### 1. Test & Lint Job

**Runs on:** Every push and pull request

**Services:**
- PostgreSQL 15 database for integration tests

**Steps:**
1. Checkout code
2. Set up Go 1.24
3. Download and verify dependencies
4. Run `go vet` for static analysis
5. Run full test suite with race detection (108 tests)
6. Upload coverage to Codecov (optional)
7. Run golangci-lint for code quality

**Environment Variables:**
- `DB_HOST=localhost`
- `DB_PORT=5432`
- `DB_USER=todouser`
- `DB_PASSWORD=todopassword`
- `DB_NAME=todolist`
- `DB_SSLMODE=disable`

#### 2. Build Job

**Runs on:** After test job passes

**Steps:**
1. Build server binary (`todolist-api`)
2. Build migrate binary (`todolist-migrate`)
3. Upload artifacts for deployment

**Artifacts:**
- `todolist-api` (server binary, 7-day retention)
- `todolist-migrate` (migration tool, 7-day retention)

#### 3. Docker Job

**Runs on:** Push to `main` or `develop` (after tests pass)

**Steps:**
1. Build Docker image using Buildx
2. Use GitHub Actions cache for faster builds
3. Upload Docker image as artifact

**Features:**
- Multi-stage build optimization
- Layer caching for speed
- Tagged with commit SHA

#### 4. Security Job

**Runs on:** After test job passes

**Tools:**
- **Trivy:** Filesystem vulnerability scanning
- **Gosec:** Go security checker

**Results:** Uploaded to GitHub Security tab (SARIF format)

#### 5. Deploy to OCI Job

**Runs on:** Push to `main` branch only (after all jobs pass)

**Environment:** `production`

**Prerequisites:**
- All tests pass
- Build succeeds
- Security scans complete

**Steps:**
1. Download built artifacts
2. Install and configure OCI CLI
3. Set up SSH access to OCI instance
4. Copy binaries to OCI instance
5. Copy deployment scripts
6. Run automated deployment script
7. Verify deployment with health checks

**Post-Deployment:**
- Health check at `/health` endpoint
- Detailed health info logged
- Deployment status notification

## Required GitHub Secrets

Configure these secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):

### OCI Authentication

| Secret | Description | Example |
|--------|-------------|---------|
| `OCI_CLI_USER` | OCI user OCID | `ocid1.user.oc1..aaa...` |
| `OCI_CLI_TENANCY` | OCI tenancy OCID | `ocid1.tenancy.oc1..aaa...` |
| `OCI_CLI_FINGERPRINT` | API key fingerprint | `aa:bb:cc:dd:ee:ff:...` |
| `OCI_CLI_KEY_CONTENT` | Private API key content | `-----BEGIN RSA PRIVATE KEY-----...` |
| `OCI_CLI_REGION` | OCI region | `us-phoenix-1` |

### OCI Instance Access

| Secret | Description | Example |
|--------|-------------|---------|
| `OCI_INSTANCE_IP` | Public IP of OCI instance | `132.145.xxx.xxx` |
| `OCI_USER` | SSH user for instance | `ubuntu` or `opc` |
| `OCI_SSH_PRIVATE_KEY` | SSH private key | `-----BEGIN OPENSSH PRIVATE KEY-----...` |

### Optional Secrets

| Secret | Description | Required |
|--------|-------------|----------|
| `CODECOV_TOKEN` | Codecov API token | No (nice to have) |

## Setting Up OCI Secrets

### 1. Get OCI API Key

```bash
# Generate API key pair
openssl genrsa -out ~/.oci/oci_api_key.pem 2048
openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem

# Get fingerprint
openssl rsa -pubout -outform DER -in ~/.oci/oci_api_key.pem | openssl md5 -c
```

Add public key to OCI Console: `User Settings > API Keys > Add API Key`

### 2. Get SSH Key

```bash
# Generate SSH key pair for OCI instance
ssh-keygen -t rsa -b 4096 -f ~/.ssh/oci_instance_key -N ""

# Copy private key content for GitHub secret
cat ~/.ssh/oci_instance_key
```

Add public key to OCI instance during creation or add to `~/.ssh/authorized_keys`

### 3. Get OCIDs

- **User OCID:** OCI Console > User Settings > User Information
- **Tenancy OCID:** OCI Console > Tenancy Details
- **Region:** e.g., `us-phoenix-1`, `us-ashburn-1`

### 4. Add Secrets to GitHub

1. Go to repository: `Settings > Secrets and variables > Actions`
2. Click `New repository secret`
3. Add each secret listed above

## Deployment Process

### Automatic Deployment

1. Push code to `main` branch
2. GitHub Actions triggers automatically
3. Tests run (2-3 minutes)
4. Build creates binaries (1-2 minutes)
5. Security scans complete (1-2 minutes)
6. Deployment to OCI (2-3 minutes)
7. Health check verifies deployment

**Total time:** ~7-12 minutes

### Manual Deployment

Trigger workflow manually:

1. Go to `Actions` tab in GitHub
2. Select `CI/CD Pipeline`
3. Click `Run workflow`
4. Select branch
5. Click `Run workflow` button

## Monitoring Deployments

### View Workflow Runs

- GitHub repository > `Actions` tab
- See status, logs, and artifacts
- Download build artifacts

### Check Deployment Status

```bash
# SSH to OCI instance
ssh ubuntu@<OCI_INSTANCE_IP>

# Check service status
sudo systemctl status todolist-api

# View logs
sudo journalctl -u todolist-api -f

# Check health
curl http://localhost:8080/health/detailed
```

### Health Endpoints

- **Basic:** `http://<IP>:8080/health`
- **Detailed:** `http://<IP>:8080/health/detailed`
- **Readiness:** `http://<IP>:8080/health/ready`
- **Liveness:** `http://<IP>:8080/health/live`

## Rollback Procedure

If deployment fails or issues arise:

```bash
# SSH to OCI instance
ssh ubuntu@<OCI_INSTANCE_IP>

# List backups
ls -lah ~/todolist-api/backups/

# Restore previous version
sudo systemctl stop todolist-api
cp ~/todolist-api/backups/todolist-api-<timestamp> ~/todolist-api/bin/todolist-api
chmod +x ~/todolist-api/bin/todolist-api
sudo systemctl start todolist-api
sudo systemctl status todolist-api
```

## Troubleshooting

### Deployment Fails

1. Check workflow logs in GitHub Actions
2. Verify all secrets are configured correctly
3. SSH to OCI instance and check logs: `sudo journalctl -u todolist-api -n 100`
4. Verify database is running: `sudo systemctl status postgresql`

### Tests Fail

1. Review test output in workflow logs
2. Check if database service started correctly
3. Run tests locally: `go test ./...`

### Security Scan Warnings

1. Review security tab for details
2. Update dependencies: `go get -u && go mod tidy`
3. Fix reported vulnerabilities

### SSH Connection Issues

1. Verify `OCI_SSH_PRIVATE_KEY` secret is correct
2. Check instance security list allows SSH (port 22)
3. Verify `OCI_INSTANCE_IP` is correct
4. Test SSH manually: `ssh -i ~/.ssh/key ubuntu@<IP>`

## Local Testing

### Test Linting

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

### Test Build

```bash
# Build server
go build -o todolist-api ./cmd/server

# Build migrate
go build -o todolist-migrate ./cmd/migrate
```

### Test Deployment Script

```bash
# Copy script to OCI instance
scp scripts/oci/deploy.sh ubuntu@<IP>:~/todolist-api/

# SSH to instance
ssh ubuntu@<IP>

# Run deployment
cd ~/todolist-api
./deploy.sh
```

## Workflow Badges

Add to your README.md:

```markdown
[![CI/CD Pipeline](https://github.com/<username>/<repo>/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/<username>/<repo>/actions/workflows/ci-cd.yml)
```

## Cost Estimate

**GitHub Actions Free Tier:**
- 2,000 minutes/month (private repos)
- Unlimited for public repos

**Estimated Usage:**
- ~7 minutes per deployment
- ~20 deployments/month = 140 minutes
- **Well within free tier**

## Performance Optimization

- **Caching:** Go modules and Docker layers cached
- **Parallel Jobs:** Test, Build, Docker, Security run in parallel
- **Artifact Reuse:** Binaries built once, deployed to OCI
- **Fast Tests:** Tests complete in ~2-3 minutes

## Security Best Practices

✅ Secrets stored in GitHub Secrets (encrypted)
✅ SSH keys use 4096-bit RSA
✅ OCI API keys rotated regularly
✅ Vulnerability scanning on every push
✅ Security results in GitHub Security tab
✅ Service runs as non-root user
✅ Systemd security hardening enabled

## Next Steps

1. **Set up secrets** in GitHub repository
2. **Test workflow** with a push to develop branch
3. **Monitor** first deployment to production
4. **Configure** Codecov for coverage reports (optional)
5. **Add status badge** to README.md

## Support

For issues with the CI/CD pipeline:
1. Check workflow logs in Actions tab
2. Review this documentation
3. Check OCI instance logs
4. Verify all secrets are configured

---

**Last Updated:** 2025-11-12
**Maintained by:** TodoList API Team
