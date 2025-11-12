# OCI Deployment Checklist

Use this checklist to track your deployment progress.

## Pre-Deployment

- [ ] OCI account created
- [ ] 2 VMs provisioned (Database + Application)
- [ ] VCN and Security Lists configured
- [ ] SSH key pair generated
- [ ] Git repository accessible
- [ ] Domain name purchased (optional, for SSL)

## Phase 1: OCI Infrastructure Setup

### VCN and Security Configuration

- [ ] Create VCN (e.g., `todolist-vcn`)
- [ ] Create Public Subnet
- [ ] Create Internet Gateway
- [ ] Configure Route Table

### Security List - Database VM

**Ingress Rules:**
- [ ] Port 22 (SSH) from 0.0.0.0/0
- [ ] Port 5432 (PostgreSQL) from VCN CIDR (e.g., 10.0.0.0/16)

### Security List - Application VM

**Ingress Rules:**
- [ ] Port 22 (SSH) from 0.0.0.0/0
- [ ] Port 80 (HTTP) from 0.0.0.0/0
- [ ] Port 443 (HTTPS) from 0.0.0.0/0

### VM Creation

**Database VM:**
- [ ] Name: `todolist-db`
- [ ] Shape: VM.Standard.E2.1.Micro (Always Free)
- [ ] Image: Oracle Linux 8 or Ubuntu 22.04
- [ ] VCN: todolist-vcn
- [ ] SSH key uploaded
- [ ] Note Private IP: `__________________`
- [ ] Note Public IP: `__________________`

**Application VM:**
- [ ] Name: `todolist-app`
- [ ] Shape: VM.Standard.E2.1.Micro (Always Free)
- [ ] Image: Oracle Linux 8 or Ubuntu 22.04
- [ ] VCN: todolist-vcn
- [ ] SSH key uploaded
- [ ] Note Private IP: `__________________`
- [ ] Note Public IP: `__________________`

## Phase 2: Database Server Setup

- [ ] SSH into database VM
- [ ] Upload `setup-database-vm.sh`
- [ ] Run script: `sudo ./setup-database-vm.sh`
- [ ] Enter secure database password (min 12 chars)
- [ ] Script completes successfully
- [ ] Save database credentials from `/root/db-credentials.txt`

**Credentials to Save:**
```
DB_HOST: _______________________ (Private IP)
DB_PORT: 5432
DB_USER: todolist
DB_PASSWORD: _______________________
DB_NAME: todolist
```

**Verification:**
- [ ] PostgreSQL service running: `sudo systemctl status postgresql-15`
- [ ] Can connect locally: `psql -U todolist -d todolist -h localhost -W`

## Phase 3: Application Server Setup

- [ ] SSH into application VM
- [ ] Upload `setup-application-vm.sh`
- [ ] Run script: `sudo ./setup-application-vm.sh`
- [ ] Enter Git repository URL
- [ ] Enter database host (DB VM private IP)
- [ ] Enter database password
- [ ] Script completes successfully

**Verification:**
- [ ] Go installed: `/usr/local/go/bin/go version`
- [ ] Application built: `/opt/todolist-api/todolist-api` exists
- [ ] Migrations completed successfully
- [ ] Service running: `sudo systemctl status todolist-api`
- [ ] Health check works: `curl http://localhost:8080/health`

## Phase 4: Nginx Setup

- [ ] Upload `setup-nginx.sh` to application VM
- [ ] Run script with domain or IP: `sudo ./setup-nginx.sh [domain]`
- [ ] Script completes successfully

**Verification:**
- [ ] Nginx running: `sudo systemctl status nginx`
- [ ] Config valid: `sudo nginx -t`
- [ ] Health check through Nginx: `curl http://localhost/health`
- [ ] Can access from public IP: `curl http://<PUBLIC_IP>/health`

## Phase 5: SSL/HTTPS Setup (Optional)

**DNS Configuration (if using domain):**
- [ ] DNS A record created
- [ ] Points to application VM public IP
- [ ] DNS propagated: `dig api.example.com`

**SSL Setup:**
- [ ] Upload `setup-ssl.sh` to application VM
- [ ] Run script: `sudo ./setup-ssl.sh <domain> <email>`
- [ ] Certificate obtained successfully
- [ ] Script completes successfully

**Verification:**
- [ ] HTTPS works: `curl https://<domain>/health`
- [ ] Certificate valid: Check in browser
- [ ] HTTP redirects to HTTPS
- [ ] Auto-renewal configured: `sudo certbot renew --dry-run`

## Testing

### Basic Functionality

- [ ] Health endpoint works: `curl https://<domain>/health`
- [ ] Register user works
- [ ] Login works and returns tokens
- [ ] Create list works (with token)
- [ ] Create todo works (with token)
- [ ] Get lists works (with token)

### Security

- [ ] Endpoints without auth return 401
- [ ] Rate limiting works (test with rapid requests)
- [ ] HTTPS enforced
- [ ] Security headers present

### Performance

- [ ] Response times acceptable
- [ ] Database queries efficient
- [ ] No memory leaks (monitor over time)

## Post-Deployment

### Documentation

- [ ] API base URL documented: `_______________________`
- [ ] Admin credentials stored securely
- [ ] Architecture diagram created
- [ ] Deployment procedure documented

### Monitoring Setup

- [ ] Log rotation configured
- [ ] Database backup scheduled
- [ ] Health check monitoring
- [ ] SSL expiry monitoring
- [ ] Disk space monitoring

### Security Hardening

- [ ] Changed default passwords
- [ ] Enabled automatic security updates
- [ ] Fail2Ban configured (optional)
- [ ] SSH key-only authentication
- [ ] Root SSH disabled
- [ ] Firewall rules reviewed

### Backups

- [ ] Database backup script created
- [ ] Backup cron job scheduled
- [ ] Backup restoration tested
- [ ] Backup storage configured

## Commands Reference

### Database VM

```bash
# Service management
sudo systemctl status postgresql-15
sudo systemctl restart postgresql-15

# Connect to database
psql -U todolist -d todolist -h localhost -W

# Backup database
sudo -u postgres pg_dump todolist | gzip > backup_$(date +%Y%m%d).sql.gz
```

### Application VM

```bash
# Service management
sudo systemctl status todolist-api
sudo systemctl restart todolist-api
sudo journalctl -u todolist-api -f

# Nginx management
sudo systemctl status nginx
sudo systemctl reload nginx
sudo nginx -t

# View logs
sudo tail -f /opt/todolist-api/logs/app.log
sudo tail -f /var/log/nginx/todolist-api-error.log

# Run migrations
cd /opt/todolist-api
sudo -u todolist ./bin/migrate version
sudo -u todolist ./bin/migrate up

# Deploy updates
cd /opt/todolist-api
sudo -u todolist ./deploy.sh production
```

## Troubleshooting

### Database Issues

- [ ] PostgreSQL service status checked
- [ ] Database connection tested from app VM
- [ ] Firewall rules verified
- [ ] pg_hba.conf allows VCN connections
- [ ] Credentials verified

### Application Issues

- [ ] Service logs reviewed
- [ ] Environment file checked
- [ ] Database connectivity tested
- [ ] Migrations status checked
- [ ] Disk space sufficient

### Nginx Issues

- [ ] Configuration syntax tested
- [ ] Error logs reviewed
- [ ] Backend health check works
- [ ] Firewall allows ports 80/443
- [ ] SELinux configured (Oracle Linux)

### SSL Issues

- [ ] DNS resolves correctly
- [ ] Port 80 accessible publicly
- [ ] Certbot logs reviewed
- [ ] Certificate files exist
- [ ] Nginx config updated

## Success Criteria

- [ ] All services running and healthy
- [ ] API accessible via HTTPS
- [ ] Users can register and login
- [ ] CRUD operations work
- [ ] Performance acceptable
- [ ] Backups configured
- [ ] Monitoring in place
- [ ] Documentation complete

## Notes

```
Date Deployed: _________________
Deployed By: _________________
Version: _________________
Issues Encountered:






Actions Taken:






```
