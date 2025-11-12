# Oracle Cloud Infrastructure (OCI) Deployment Scripts

Automated setup scripts for deploying the TodoList API on Oracle Cloud Infrastructure's Always Free tier.

## Overview

These scripts automate the entire deployment process from bare VMs to production-ready API with HTTPS.

## Prerequisites

- **2 OCI VMs** (Always Free tier):
  - VM 1: Database Server (PostgreSQL)
  - VM 2: Application Server (API + Nginx)
- **SSH access** to both VMs
- **Domain name** (optional, for SSL)
- **Public SSH key** uploaded to OCI

## Quick Start

### Step 1: Prepare VMs

1. **Create VMs in OCI Console**:
   - Navigate to: Compute → Instances → Create Instance
   - Create 2 instances with the same VCN
   - Configure security lists (see below)

2. **Configure Security Lists**:

   **Database VM** (Ingress):
   ```
   Port 22 (SSH)    - Source: 0.0.0.0/0
   Port 5432 (PostgreSQL) - Source: 10.0.0.0/16 (VCN CIDR)
   ```

   **Application VM** (Ingress):
   ```
   Port 22 (SSH)    - Source: 0.0.0.0/0
   Port 80 (HTTP)   - Source: 0.0.0.0/0
   Port 443 (HTTPS) - Source: 0.0.0.0/0
   ```

### Step 2: Setup Database Server

```bash
# SSH into database VM
ssh -i ~/.ssh/your-key.pem opc@<DB_VM_PUBLIC_IP>

# Upload script
scp -i ~/.ssh/your-key.pem setup-database-vm.sh opc@<DB_VM_PUBLIC_IP>:~

# Run setup
chmod +x setup-database-vm.sh
sudo ./setup-database-vm.sh
```

**What this does**:
- ✅ Installs PostgreSQL 15
- ✅ Creates `todolist` database and user
- ✅ Configures remote access from VCN
- ✅ Optimizes PostgreSQL settings
- ✅ Configures firewall
- ✅ Saves credentials to `/root/db-credentials.txt`

**Save the output** - you'll need the database credentials!

### Step 3: Setup Application Server

```bash
# SSH into application VM
ssh -i ~/.ssh/your-key.pem opc@<APP_VM_PUBLIC_IP>

# Upload script
scp -i ~/.ssh/your-key.pem setup-application-vm.sh opc@<APP_VM_PUBLIC_IP>:~

# Run setup (you'll be prompted for Git repo and DB credentials)
chmod +x setup-application-vm.sh
sudo ./setup-application-vm.sh
```

**You'll be prompted for**:
- Git repository URL
- Database host (private IP from Step 2)
- Database password

**What this does**:
- ✅ Installs Go 1.21.5
- ✅ Clones your repository
- ✅ Builds the application
- ✅ Runs database migrations
- ✅ Installs systemd service
- ✅ Starts the API on port 8080
- ✅ Configures firewall

### Step 4: Setup Nginx Reverse Proxy

```bash
# On application VM
scp -i ~/.ssh/your-key.pem setup-nginx.sh opc@<APP_VM_PUBLIC_IP>:~
ssh -i ~/.ssh/your-key.pem opc@<APP_VM_PUBLIC_IP>

# Run setup
chmod +x setup-nginx.sh
sudo ./setup-nginx.sh                    # Using public IP
# OR
sudo ./setup-nginx.sh api.example.com    # Using custom domain
```

**What this does**:
- ✅ Installs Nginx
- ✅ Configures reverse proxy
- ✅ Sets up rate limiting
- ✅ Adds security headers
- ✅ Configures logging
- ✅ Opens ports 80/443

### Step 5: Setup SSL/HTTPS (Optional but Recommended)

**Prerequisites**:
- Custom domain name
- DNS A record pointing to your VM's public IP

```bash
# On application VM
scp -i ~/.ssh/your-key.pem setup-ssl.sh opc@<APP_VM_PUBLIC_IP>:~
ssh -i ~/.ssh/your-key.pem opc@<APP_VM_PUBLIC_IP>

# Run setup
chmod +x setup-ssl.sh
sudo ./setup-ssl.sh api.example.com your-email@example.com
```

**What this does**:
- ✅ Installs Certbot
- ✅ Obtains Let's Encrypt certificate
- ✅ Configures HTTPS in Nginx
- ✅ Sets up auto-renewal
- ✅ Redirects HTTP to HTTPS
- ✅ Optionally enables HSTS

## Scripts Reference

### Phase 2: `setup-database-vm.sh`

**Purpose**: Install and configure PostgreSQL on database VM

**Usage**:
```bash
sudo ./setup-database-vm.sh
```

**Interactive Prompts**:
- Database password (min 12 characters)

**Output Files**:
- `/root/db-credentials.txt` - Contains all connection details

**Configuration**:
```bash
# Edit these variables if needed (in script)
DB_NAME="todolist"
DB_USER="todolist"
VCN_CIDR="10.0.0.0/16"  # Your VCN CIDR
PG_VERSION="15"
```

### Phase 3: `setup-application-vm.sh`

**Purpose**: Install Go, clone repo, build app, run migrations, start service

**Usage**:
```bash
sudo ./setup-application-vm.sh
```

**Interactive Prompts**:
- Git repository URL
- Database host (private IP)
- Database password

**Configuration**:
```bash
# Edit these variables if needed (in script)
APP_NAME="todolist-api"
APP_DIR="/opt/todolist-api"
APP_USER="todolist"
GO_VERSION="1.21.5"
GIT_BRANCH="main"
```

### Phase 4: `setup-nginx.sh`

**Purpose**: Install and configure Nginx reverse proxy

**Usage**:
```bash
sudo ./setup-nginx.sh [domain-name]
```

**Examples**:
```bash
sudo ./setup-nginx.sh                    # Use public IP
sudo ./setup-nginx.sh api.example.com    # Use custom domain
```

**Features**:
- Rate limiting (10 req/s API, 5 req/s auth)
- Security headers
- Request size limits
- Logging
- SELinux configuration (Oracle Linux)

### Phase 5: `setup-ssl.sh`

**Purpose**: Obtain and configure SSL certificate from Let's Encrypt

**Usage**:
```bash
sudo ./setup-ssl.sh <domain-name> [email]
```

**Examples**:
```bash
sudo ./setup-ssl.sh api.example.com
sudo ./setup-ssl.sh api.example.com admin@example.com
```

**Prerequisites**:
- Domain must point to VM's public IP
- Port 80 must be accessible
- Nginx must be running

## Testing Your Deployment

### 1. Health Check

```bash
# HTTP
curl http://<YOUR_DOMAIN_OR_IP>/health

# HTTPS (after Phase 5)
curl https://<YOUR_DOMAIN>/health
```

Expected output:
```json
{"status":"healthy"}
```

### 2. Register User

```bash
curl -X POST https://<YOUR_DOMAIN>/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123!",
    "firstName": "John",
    "lastName": "Doe"
  }'
```

### 3. Login

```bash
curl -X POST https://<YOUR_DOMAIN>/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123!"
  }'
```

Save the `accessToken` from the response!

### 4. Create List

```bash
curl -X POST https://<YOUR_DOMAIN>/api/v1/lists \
  -H "Authorization: Bearer <YOUR_ACCESS_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My First List",
    "description": "Getting started"
  }'
```

## Troubleshooting

### Script Fails to Run

```bash
# Check script has execute permissions
chmod +x setup-*.sh

# Check you're running as root
sudo ./setup-script.sh
```

### Database Connection Fails

```bash
# Test from application VM
psql -U todolist -h <DB_PRIVATE_IP> -d todolist -W

# Check PostgreSQL is running (on DB VM)
sudo systemctl status postgresql-15

# Check firewall (on DB VM)
sudo firewall-cmd --list-all
```

### Application Won't Start

```bash
# Check service status
sudo systemctl status todolist-api

# View logs
sudo journalctl -u todolist-api -n 100

# Check environment file
sudo cat /opt/todolist-api/.env

# Test manually
sudo su - todolist
cd /opt/todolist-api
./todolist-api
```

### Nginx Issues

```bash
# Test configuration
sudo nginx -t

# Check error logs
sudo tail -f /var/log/nginx/todolist-api-error.log

# Restart Nginx
sudo systemctl restart nginx
```

### SSL Certificate Fails

```bash
# Check DNS
dig api.example.com

# Check port 80 is accessible
curl -I http://api.example.com

# Check Certbot logs
sudo tail -f /var/log/letsencrypt/letsencrypt.log

# Verify OCI security list allows port 80
```

## File Locations

### Database VM

```
/var/lib/pgsql/15/data/           PostgreSQL data directory
/var/lib/pgsql/15/data/postgresql.conf  PostgreSQL config
/var/lib/pgsql/15/data/pg_hba.conf      Access control
/root/db-credentials.txt          Database credentials
```

### Application VM

```
/opt/todolist-api/                Application directory
/opt/todolist-api/.env            Environment configuration
/opt/todolist-api/todolist-api    Application binary
/opt/todolist-api/bin/migrate     Migration tool
/opt/todolist-api/logs/           Application logs
/etc/systemd/system/todolist-api.service  Systemd service
/etc/nginx/conf.d/todolist-api.conf       Nginx config (Oracle Linux)
/etc/nginx/sites-available/todolist-api.conf  Nginx config (Ubuntu)
/etc/letsencrypt/live/<domain>/   SSL certificates
```

## Maintenance

### Update Application

```bash
# SSH to application VM
ssh opc@<APP_VM_PUBLIC_IP>

# Run deployment script
cd /opt/todolist-api
sudo -u todolist ./deploy.sh production
```

### Database Backup

```bash
# SSH to database VM
ssh opc@<DB_VM_PUBLIC_IP>

# Manual backup
sudo -u postgres pg_dump todolist | gzip > backup_$(date +%Y%m%d).sql.gz
```

### View Logs

```bash
# Application logs
sudo journalctl -u todolist-api -f

# Nginx access logs
sudo tail -f /var/log/nginx/todolist-api-access.log

# Nginx error logs
sudo tail -f /var/log/nginx/todolist-api-error.log
```

### Restart Services

```bash
# Application
sudo systemctl restart todolist-api

# Nginx
sudo systemctl reload nginx

# PostgreSQL (DB VM)
sudo systemctl restart postgresql-15
```

## Security Recommendations

1. **Change default passwords** immediately
2. **Enable automatic security updates**:
   ```bash
   sudo dnf install -y dnf-automatic
   sudo systemctl enable --now dnf-automatic.timer
   ```
3. **Setup Fail2Ban** (optional):
   ```bash
   sudo dnf install -y fail2ban
   sudo systemctl enable --now fail2ban
   ```
4. **Regular backups** of database
5. **Monitor logs** for suspicious activity
6. **Keep SSL certificates** renewed (auto-renewal is configured)

## Cost

**Total Monthly Cost: $0.00** 🎉

- 2x VM.Standard.E2.1.Micro (Always Free)
- 100 GB Block Storage (Always Free)
- 10 TB/month outbound data transfer (Always Free)

## Support

For issues with the scripts:
1. Check the troubleshooting section above
2. Review the script output for error messages
3. Check service logs (`journalctl` or `/var/log/`)
4. Open an issue on GitHub

## License

These scripts are part of the TodoList API project and follow the same license.
