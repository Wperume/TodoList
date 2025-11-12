# Cloud VM Deployment Guide

This guide walks you through deploying the TodoList API on a cloud VM (AWS EC2, DigitalOcean, Google Cloud, etc.).

## Prerequisites

- Linux VM (Ubuntu 20.04+ or similar)
- PostgreSQL installed
- Go 1.21+ installed
- Git installed
- Sudo/root access

## Quick Start

```bash
# Clone and run the setup script
git clone https://github.com/yourusername/todolist-api.git /opt/todolist-api
cd /opt/todolist-api
sudo ./scripts/setup-vm.sh
```

## Manual Setup

### 1. System Preparation

```bash
# Update system
sudo apt-get update
sudo apt-get upgrade -y

# Install required packages
sudo apt-get install -y \
    postgresql postgresql-contrib \
    nginx \
    git \
    curl \
    build-essential
```

### 2. Install Go

```bash
# Download Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

# Extract and install
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
```

### 3. Setup PostgreSQL

```bash
# Start PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE todolist;
CREATE USER todolist WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE todolist TO todolist;
\q
EOF

# Test connection
psql -U todolist -d todolist -h localhost -W
```

### 4. Create Service User

```bash
# Create dedicated user for the application
sudo useradd -r -s /bin/false todolist

# Create application directory
sudo mkdir -p /opt/todolist-api
sudo chown todolist:todolist /opt/todolist-api
```

### 5. Clone and Build Application

```bash
# Clone repository
cd /opt
sudo git clone https://github.com/yourusername/todolist-api.git
sudo chown -R todolist:todolist todolist-api
cd todolist-api

# Build application
go mod download
go build -o todolist-api cmd/server/main.go
go build -o bin/migrate cmd/migrate/main.go

# Make binaries executable
chmod +x todolist-api bin/migrate
```

### 6. Configure Environment

```bash
# Create environment file
sudo nano /opt/todolist-api/.env
```

Add the following (adjust values):

```bash
# Server Configuration
PORT=8080
GIN_MODE=release

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=todolist
DB_PASSWORD=your_secure_password
DB_NAME=todolist
DB_SSL_MODE=disable

# JWT Configuration
JWT_SECRET_KEY=your-very-long-random-secret-key-min-32-chars
JWT_ACCESS_TOKEN_MINUTES=15
JWT_REFRESH_TOKEN_DAYS=7
JWT_ISSUER=todolist-api

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# TLS (optional)
TLS_ENABLED=false
# TLS_CERT_FILE=/etc/ssl/certs/todolist.crt
# TLS_KEY_FILE=/etc/ssl/private/todolist.key
```

Secure the file:

```bash
sudo chown todolist:todolist /opt/todolist-api/.env
sudo chmod 600 /opt/todolist-api/.env
```

### 7. Run Database Migrations

```bash
cd /opt/todolist-api
./bin/migrate up
```

You should see:
```
✅ Migrations applied successfully
```

### 8. Install Systemd Service

```bash
# Copy service file
sudo cp todolist-api.service /etc/systemd/system/

# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable todolist-api

# Start service
sudo systemctl start todolist-api

# Check status
sudo systemctl status todolist-api
```

### 9. Setup Nginx Reverse Proxy (Optional but Recommended)

```bash
# Install Nginx
sudo apt-get install -y nginx

# Create Nginx configuration
sudo nano /etc/nginx/sites-available/todolist-api
```

Add this configuration:

```nginx
server {
    listen 80;
    server_name your-domain.com;  # Replace with your domain

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check endpoint (no auth required)
    location /health {
        proxy_pass http://localhost:8080/health;
        access_log off;
    }
}
```

Enable the site:

```bash
# Create symlink
sudo ln -s /etc/nginx/sites-available/todolist-api /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Restart Nginx
sudo systemctl restart nginx
```

### 10. Setup SSL with Let's Encrypt (Optional)

```bash
# Install Certbot
sudo apt-get install -y certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d your-domain.com

# Auto-renewal is configured automatically
# Test renewal:
sudo certbot renew --dry-run
```

### 11. Configure Firewall

```bash
# Allow SSH (if not already allowed)
sudo ufw allow OpenSSH

# Allow HTTP and HTTPS
sudo ufw allow 'Nginx Full'

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

## Automated Deployment

Once set up, use the deployment script for updates:

```bash
cd /opt/todolist-api
./deploy.sh production
```

This will:
- ✅ Pull latest code
- ✅ Build new binary
- ✅ Backup old binary
- ✅ Run migrations
- ✅ Restart service
- ✅ Run health checks

## Monitoring and Logs

### View Application Logs

```bash
# Real-time logs
sudo journalctl -u todolist-api -f

# Last 100 lines
sudo journalctl -u todolist-api -n 100

# Logs since specific time
sudo journalctl -u todolist-api --since "1 hour ago"

# Export logs
sudo journalctl -u todolist-api > app.log
```

### Monitor Service Status

```bash
# Service status
sudo systemctl status todolist-api

# CPU and memory usage
top -p $(pgrep todolist-api)

# Or use htop
sudo apt-get install htop
htop -p $(pgrep todolist-api)
```

### Database Monitoring

```bash
# Connect to database
psql -U todolist -d todolist

# Check connections
SELECT * FROM pg_stat_activity WHERE datname = 'todolist';

# Check database size
SELECT pg_size_pretty(pg_database_size('todolist'));

# Check table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

## Backup and Restore

### Automated Backups

Create a backup script:

```bash
sudo nano /opt/todolist-api/scripts/backup.sh
```

```bash
#!/bin/bash
BACKUP_DIR="/opt/todolist-api/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/todolist_${TIMESTAMP}.sql"

mkdir -p "${BACKUP_DIR}"

# Backup database
pg_dump -U todolist -h localhost todolist > "${BACKUP_FILE}"

# Compress
gzip "${BACKUP_FILE}"

# Keep only last 7 days of backups
find "${BACKUP_DIR}" -name "todolist_*.sql.gz" -mtime +7 -delete

echo "Backup completed: ${BACKUP_FILE}.gz"
```

Make it executable:

```bash
chmod +x /opt/todolist-api/scripts/backup.sh
```

Schedule with cron:

```bash
sudo crontab -e
```

Add:

```bash
# Daily backup at 2 AM
0 2 * * * /opt/todolist-api/scripts/backup.sh >> /var/log/todolist-backup.log 2>&1
```

### Manual Backup

```bash
# Backup
pg_dump -U todolist todolist > backup_$(date +%Y%m%d).sql

# Backup with compression
pg_dump -U todolist todolist | gzip > backup_$(date +%Y%m%d).sql.gz
```

### Restore

```bash
# Restore from backup
psql -U todolist todolist < backup_20231112.sql

# Restore from compressed backup
gunzip -c backup_20231112.sql.gz | psql -U todolist todolist
```

## Performance Tuning

### PostgreSQL Optimization

Edit `/etc/postgresql/14/main/postgresql.conf`:

```conf
# For 2GB RAM server
shared_buffers = 512MB
effective_cache_size = 1536MB
maintenance_work_mem = 128MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 5242kB
min_wal_size = 1GB
max_wal_size = 4GB
max_connections = 100
```

Restart PostgreSQL:

```bash
sudo systemctl restart postgresql
```

### Application Performance

Adjust in `.env`:

```bash
# For production
GIN_MODE=release

# Connection pooling (adjust based on load)
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=5m
```

## Security Hardening

### 1. Firewall Rules

```bash
# Only allow specific ports
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

### 2. Fail2Ban (Prevent Brute Force)

```bash
sudo apt-get install fail2ban
sudo systemctl enable fail2ban
sudo systemctl start fail2ban
```

### 3. Regular Updates

```bash
# Set up auto-updates
sudo apt-get install unattended-upgrades
sudo dpkg-reconfigure --priority=low unattended-upgrades
```

### 4. Limit PostgreSQL Access

Edit `/etc/postgresql/14/main/pg_hba.conf`:

```conf
# Only allow local connections
local   all   todolist   md5
host    all   todolist   127.0.0.1/32   md5
```

Restart:

```bash
sudo systemctl restart postgresql
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u todolist-api -xe

# Check if port is in use
sudo lsof -i :8080

# Check binary permissions
ls -l /opt/todolist-api/todolist-api

# Try running manually
cd /opt/todolist-api
./todolist-api
```

### Database Connection Issues

```bash
# Test connection
psql -U todolist -d todolist -h localhost -W

# Check PostgreSQL status
sudo systemctl status postgresql

# Check PostgreSQL logs
sudo tail -f /var/log/postgresql/postgresql-14-main.log
```

### Nginx Issues

```bash
# Test configuration
sudo nginx -t

# Check logs
sudo tail -f /var/log/nginx/error.log

# Restart
sudo systemctl restart nginx
```

### High Memory Usage

```bash
# Check process memory
ps aux | grep todolist-api

# Monitor in real-time
top -p $(pgrep todolist-api)

# Check for memory leaks (if persistent)
# Consider adding memory limit in systemd service
```

## Rolling Back

If deployment fails:

```bash
# The deploy script automatically backs up binaries
cd /opt/todolist-api/backups

# List backups
ls -lah

# Restore previous binary
cp todolist-api_20231112_140000 ../todolist-api

# Rollback migration
./bin/migrate down

# Restart service
sudo systemctl restart todolist-api
```

## Health Checks

```bash
# Check if API is responding
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy"}

# With Nginx
curl https://your-domain.com/health
```

## Next Steps

- Set up monitoring (Prometheus + Grafana)
- Configure log aggregation (ELK Stack or similar)
- Set up alerting (email/Slack notifications)
- Implement blue-green or canary deployments
- Add load balancer for multiple instances

## Support

For help:
- Review logs: `sudo journalctl -u todolist-api -n 200`
- Check [MIGRATIONS.md](./MIGRATIONS.md) for database issues
- Open an issue on GitHub
