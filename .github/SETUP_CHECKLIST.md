# CI/CD Setup Checklist

Use this checklist to set up the CI/CD pipeline step by step.

## Prerequisites

- [ ] GitHub repository with admin access
- [ ] OCI account created
- [ ] OCI compute instance running
- [ ] PostgreSQL installed on OCI instance
- [ ] Domain name configured (optional)

## Phase 1: Local Preparation

### Generate OCI API Key
- [ ] Create `~/.oci` directory
- [ ] Generate RSA key pair: `openssl genrsa -out ~/.oci/oci_api_key.pem 2048`
- [ ] Generate public key: `openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem`
- [ ] Get fingerprint: `openssl rsa -pubout -outform DER -in ~/.oci/oci_api_key.pem | openssl md5 -c`
- [ ] Save fingerprint (you'll need it later)

### Add API Key to OCI
- [ ] Login to OCI Console
- [ ] Go to: Profile Icon > User Settings
- [ ] Click "API Keys" > "Add API Key"
- [ ] Paste public key content
- [ ] Save configuration

### Generate SSH Key for OCI Instance
- [ ] Generate SSH key: `ssh-keygen -t rsa -b 4096 -f ~/.ssh/oci_todolist_key -N ""`
- [ ] Save private key location
- [ ] Copy public key content: `cat ~/.ssh/oci_todolist_key.pub`

### Add SSH Key to OCI Instance
Choose one method:

**Method A - During instance creation:**
- [ ] Paste public key when creating instance

**Method B - Existing instance:**
- [ ] SSH to instance with existing key
- [ ] Run: `echo "<PUBLIC_KEY>" >> ~/.ssh/authorized_keys`
- [ ] Run: `chmod 600 ~/.ssh/authorized_keys`
- [ ] Test new key: `ssh -i ~/.ssh/oci_todolist_key ubuntu@<IP>`

## Phase 2: Collect Required Information

### OCI OCIDs
- [ ] **User OCID**: OCI Console > Profile > User Settings > Copy OCID
  - Format: `ocid1.user.oc1..aaaaaaaaXXX...`
  - Value: `_________________________________`

- [ ] **Tenancy OCID**: OCI Console > Profile > Tenancy > Copy OCID
  - Format: `ocid1.tenancy.oc1..aaaaaaaaXXX...`
  - Value: `_________________________________`

- [ ] **Region**: Found in OCI Console (top-right)
  - Examples: `us-phoenix-1`, `us-ashburn-1`, `uk-london-1`
  - Value: `_________________________________`

### OCI Instance Details
- [ ] **Public IP**: OCI Console > Compute > Instances > Your Instance
  - Format: `132.145.xxx.xxx`
  - Value: `_________________________________`

- [ ] **SSH Username**: Usually `ubuntu` or `opc`
  - Value: `_________________________________`

### Key Files
- [ ] **API Fingerprint**: From earlier step
  - Format: `aa:bb:cc:dd:ee:ff:...`
  - Value: `_________________________________`

- [ ] **API Private Key**: `cat ~/.oci/oci_api_key.pem`
  - Saved location: `_________________________________`

- [ ] **SSH Private Key**: `cat ~/.ssh/oci_todolist_key`
  - Saved location: `_________________________________`

## Phase 3: Configure GitHub Secrets

Go to: GitHub Repository > Settings > Secrets and variables > Actions > New repository secret

### OCI Authentication Secrets

- [ ] **OCI_CLI_USER**
  - Value: [User OCID from above]
  - Status: ⬜ Added

- [ ] **OCI_CLI_TENANCY**
  - Value: [Tenancy OCID from above]
  - Status: ⬜ Added

- [ ] **OCI_CLI_FINGERPRINT**
  - Value: [API Fingerprint from above]
  - Status: ⬜ Added

- [ ] **OCI_CLI_KEY_CONTENT**
  - Value: [Complete private key including BEGIN/END lines]
  - Status: ⬜ Added

- [ ] **OCI_CLI_REGION**
  - Value: [Region from above]
  - Status: ⬜ Added

### OCI Instance Access Secrets

- [ ] **OCI_INSTANCE_IP**
  - Value: [Public IP from above]
  - Status: ⬜ Added

- [ ] **OCI_USER**
  - Value: [SSH username from above]
  - Status: ⬜ Added

- [ ] **OCI_SSH_PRIVATE_KEY**
  - Value: [Complete SSH private key including BEGIN/END lines]
  - Status: ⬜ Added

### Optional Secrets

- [ ] **CODECOV_TOKEN** (optional)
  - Go to: https://codecov.io
  - Sign in with GitHub
  - Add repository
  - Copy token
  - Status: ⬜ Added / ⬜ Skipped

## Phase 4: Verify Configuration

### Test SSH Access
```bash
ssh -i ~/.ssh/oci_todolist_key <OCI_USER>@<OCI_INSTANCE_IP>
```
- [ ] SSH connection successful
- [ ] Can access instance without password

### Test OCI CLI (Optional)
```bash
oci iam region list --config-file ~/.oci/config
```
- [ ] OCI CLI configured
- [ ] Can access OCI API

### Verify OCI Instance Setup
SSH to instance and check:
```bash
# Check PostgreSQL
sudo systemctl status postgresql
```
- [ ] PostgreSQL is installed and running

```bash
# Check disk space
df -h
```
- [ ] At least 5GB free space

```bash
# Check user permissions
sudo -l
```
- [ ] User has sudo access

## Phase 5: Test the Pipeline

### Initial Test Run

- [ ] Commit and push these CI/CD changes to `develop` branch
  ```bash
  git checkout -b develop
  git add .github/ .golangci.yml scripts/oci/deploy.sh README.md
  git commit -m "Add CI/CD pipeline"
  git push origin develop
  ```

- [ ] Go to GitHub > Actions tab
- [ ] Watch the workflow run
- [ ] All jobs should be visible: Test, Build, Docker, Security

### Expected Results for Develop Branch

- [ ] ✅ Test & Lint job passes (2-3 min)
- [ ] ✅ Build job passes (1-2 min)
- [ ] ✅ Docker job passes (1-2 min)
- [ ] ✅ Security job completes (warnings OK)
- [ ] ⏭️ Deploy job skipped (only runs on main)

### Review Test Output

- [ ] Click on "Test & Lint" job
- [ ] Verify all 108 tests passed
- [ ] Check linting results
- [ ] Review coverage report

### Check Security Scan

- [ ] Go to repository Security tab
- [ ] Review any findings from Trivy/Gosec
- [ ] Address critical issues if any

## Phase 6: Production Deployment

### Create Environment Protection (Optional but Recommended)

- [ ] Go to: Settings > Environments > New environment
- [ ] Name: `production`
- [ ] Add protection rules:
  - [ ] Required reviewers (optional)
  - [ ] Wait timer (optional)
  - [ ] Deployment branches: `main` only

### Deploy to Production

- [ ] Merge develop to main:
  ```bash
  git checkout main
  git merge develop
  git push origin main
  ```

- [ ] Go to GitHub > Actions tab
- [ ] Watch deployment workflow

### Expected Results for Main Branch

- [ ] ✅ Test & Lint passes
- [ ] ✅ Build passes
- [ ] ✅ Docker passes
- [ ] ✅ Security completes
- [ ] ✅ Deploy to OCI passes

### Verify Deployment on OCI

SSH to instance:
```bash
ssh -i ~/.ssh/oci_todolist_key <OCI_USER>@<OCI_INSTANCE_IP>
```

Check service:
```bash
sudo systemctl status todolist-api
```
- [ ] Service is active and running

Check logs:
```bash
sudo journalctl -u todolist-api -n 50
```
- [ ] No errors in logs
- [ ] Service started successfully

Test health endpoint:
```bash
curl http://localhost:8080/health/detailed | jq
```
- [ ] Health check returns 200 OK
- [ ] Database status is healthy
- [ ] All checks pass

## Phase 7: Add Status Badge

Update your README.md:
```markdown
[![CI/CD Pipeline](https://github.com/YOUR_USERNAME/TodoList/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/YOUR_USERNAME/TodoList/actions/workflows/ci-cd.yml)
```

- [ ] Replace `YOUR_USERNAME` with actual username
- [ ] Commit and push
- [ ] Badge shows "passing" status

## Phase 8: Documentation Review

- [ ] Read [.github/workflows/README.md](.github/workflows/README.md)
- [ ] Bookmark for future reference
- [ ] Share with team members (if applicable)

## Troubleshooting

If something fails, check:

### Workflow Fails at Test Stage
- [ ] Review test output in Actions logs
- [ ] Check if PostgreSQL service started
- [ ] Run tests locally: `go test ./...`

### Workflow Fails at Deploy Stage
- [ ] Verify all secrets are configured correctly
- [ ] Check SSH connection manually
- [ ] Review OCI instance security list (port 22 open)
- [ ] Check OCI instance logs: `sudo journalctl -u todolist-api -n 100`

### Health Check Fails After Deployment
- [ ] SSH to instance
- [ ] Check service status: `sudo systemctl status todolist-api`
- [ ] Check logs: `sudo journalctl -u todolist-api -f`
- [ ] Verify database is running
- [ ] Check .env file configuration

## Maintenance Schedule

### Weekly
- [ ] Review GitHub Actions runs
- [ ] Check for failed deployments
- [ ] Monitor security scan results

### Monthly
- [ ] Review and update dependencies: `go get -u && go mod tidy`
- [ ] Check for new security vulnerabilities
- [ ] Review workflow performance

### Quarterly
- [ ] Rotate OCI API keys
- [ ] Update GitHub Actions versions
- [ ] Review and update linter rules
- [ ] Performance optimization review

## Rollback Procedure

If deployment causes issues:

1. [ ] SSH to OCI instance
2. [ ] List backups: `ls -lah ~/todolist-api/backups/`
3. [ ] Stop service: `sudo systemctl stop todolist-api`
4. [ ] Restore backup: `cp ~/todolist-api/backups/todolist-api-<timestamp> ~/todolist-api/bin/todolist-api`
5. [ ] Make executable: `chmod +x ~/todolist-api/bin/todolist-api`
6. [ ] Start service: `sudo systemctl start todolist-api`
7. [ ] Verify: `sudo systemctl status todolist-api`
8. [ ] Test health: `curl http://localhost:8080/health`

## Success Criteria

✅ All items checked above
✅ Pipeline runs successfully
✅ Deployment to OCI works
✅ Health checks pass
✅ Service runs without errors
✅ Documentation understood

## Get Help

If stuck:
1. Review [.github/workflows/README.md](.github/workflows/README.md)
2. Check [.github/SECRETS_SETUP.md](.github/SECRETS_SETUP.md)
3. Review workflow logs in Actions tab
4. Check this checklist for missed steps

---

**Completion Date:** _______________

**Notes:**
```
[Add any notes or issues encountered during setup]




```
