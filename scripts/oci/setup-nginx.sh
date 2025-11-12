#!/bin/bash
#
# Phase 4: Nginx Reverse Proxy Setup Script for Oracle Cloud Infrastructure
#
# This script installs and configures Nginx as a reverse proxy
#
# Usage:
#   Run this on the application VM after Phase 3 is complete
#   sudo ./setup-nginx.sh [domain-name]
#
# Examples:
#   sudo ./setup-nginx.sh                    # Use public IP
#   sudo ./setup-nginx.sh api.example.com    # Use custom domain
#

set -e  # Exit on error
set -u  # Exit on undefined variable

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
DOMAIN="${1:-}"
APP_PORT="8080"
NGINX_CONF_DIR="/etc/nginx"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    log_error "Please run as root (use sudo)"
    exit 1
fi

# Detect OS
detect_os() {
    if [ -f /etc/oracle-release ]; then
        OS="oracle"
        OS_VERSION=$(cat /etc/oracle-release | grep -oP '\d+' | head -1)
        log_info "Detected: Oracle Linux $OS_VERSION"
    elif [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
        log_info "Detected: $NAME $VERSION_ID"
    else
        log_error "Unable to detect OS"
        exit 1
    fi
}

# Determine server name
determine_server_name() {
    if [ -z "$DOMAIN" ]; then
        PUBLIC_IP=$(curl -s ifconfig.me || echo "")
        if [ -n "$PUBLIC_IP" ]; then
            SERVER_NAME="$PUBLIC_IP"
            log_info "Using public IP: $SERVER_NAME"
        else
            SERVER_NAME="_"
            log_warning "Could not determine public IP, using default"
        fi
    else
        SERVER_NAME="$DOMAIN"
        log_info "Using custom domain: $SERVER_NAME"
    fi
}

# Install Nginx
install_nginx() {
    log_info "Installing Nginx..."

    if [ "$OS" = "oracle" ]; then
        dnf install -y nginx
    else
        apt-get update
        apt-get install -y nginx
    fi

    log_success "Nginx installed"
}

# Backup default configuration
backup_default_config() {
    log_info "Backing up default Nginx configuration..."

    if [ -f "${NGINX_CONF_DIR}/nginx.conf" ]; then
        cp "${NGINX_CONF_DIR}/nginx.conf" "${NGINX_CONF_DIR}/nginx.conf.backup.$(date +%Y%m%d_%H%M%S)"
    fi

    log_success "Default configuration backed up"
}

# Create Nginx configuration
create_nginx_config() {
    log_info "Creating Nginx configuration..."

    # Determine config directory
    if [ "$OS" = "oracle" ]; then
        CONF_DIR="${NGINX_CONF_DIR}/conf.d"
    else
        CONF_DIR="${NGINX_CONF_DIR}/sites-available"
        mkdir -p "${NGINX_CONF_DIR}/sites-enabled"
    fi

    # Create configuration file
    cat > ${CONF_DIR}/todolist-api.conf << EOF
# TodoList API Nginx Configuration
# Generated: $(date)

upstream todolist_backend {
    server 127.0.0.1:${APP_PORT};
    keepalive 32;
}

# Rate limiting
limit_req_zone \$binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone \$binary_remote_addr zone=auth_limit:10m rate=5r/s;

server {
    listen 80;
    server_name ${SERVER_NAME};

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # Client limits
    client_max_body_size 10M;
    client_body_timeout 60s;
    client_header_timeout 60s;

    # Logging
    access_log /var/log/nginx/todolist-api-access.log;
    error_log /var/log/nginx/todolist-api-error.log;

    # Health check (no rate limit)
    location /health {
        proxy_pass http://todolist_backend/health;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        access_log off;
    }

    # Authentication endpoints (stricter rate limit)
    location /api/v1/auth {
        limit_req zone=auth_limit burst=10 nodelay;

        proxy_pass http://todolist_backend;
        proxy_http_version 1.1;

        # Headers
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Connection "";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # API endpoints (standard rate limit)
    location /api/ {
        limit_req zone=api_limit burst=20 nodelay;

        proxy_pass http://todolist_backend;
        proxy_http_version 1.1;

        # Headers
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Connection "";

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;

        # Buffering
        proxy_buffering off;
        proxy_request_buffering off;
    }

    # Root redirects to API docs (future)
    location = / {
        return 301 /api/v1;
    }

    # Deny access to hidden files
    location ~ /\. {
        deny all;
        access_log off;
        log_not_found off;
    }
}
EOF

    # Enable site on Ubuntu
    if [ "$OS" != "oracle" ]; then
        ln -sf ${CONF_DIR}/todolist-api.conf ${NGINX_CONF_DIR}/sites-enabled/
    fi

    log_success "Nginx configuration created"
}

# Configure SELinux (Oracle Linux)
configure_selinux() {
    if [ "$OS" = "oracle" ] && command -v getenforce &> /dev/null; then
        if [ "$(getenforce)" != "Disabled" ]; then
            log_info "Configuring SELinux..."

            # Allow Nginx to connect to backend
            setsebool -P httpd_can_network_connect 1

            log_success "SELinux configured"
        fi
    fi
}

# Test Nginx configuration
test_nginx_config() {
    log_info "Testing Nginx configuration..."

    if nginx -t; then
        log_success "Nginx configuration is valid"
    else
        log_error "Nginx configuration test failed"
        exit 1
    fi
}

# Start Nginx
start_nginx() {
    log_info "Starting Nginx..."

    systemctl enable nginx
    systemctl restart nginx
    sleep 2

    if systemctl is-active --quiet nginx; then
        log_success "Nginx started successfully"
    else
        log_error "Nginx failed to start"
        systemctl status nginx
        exit 1
    fi
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."

    if command -v firewall-cmd &> /dev/null; then
        # firewalld (Oracle Linux)
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https
        firewall-cmd --reload
        log_success "Firewall configured (firewalld)"
    elif command -v ufw &> /dev/null; then
        # ufw (Ubuntu)
        ufw allow 'Nginx Full'
        log_success "Firewall configured (ufw)"
    else
        log_warning "No firewall detected. Please configure manually."
    fi
}

# Test reverse proxy
test_reverse_proxy() {
    log_info "Testing reverse proxy..."

    sleep 2

    # Test health endpoint
    if curl -f -s http://localhost/health > /dev/null; then
        log_success "Reverse proxy working (health check passed)"
    else
        log_warning "Reverse proxy test inconclusive"
    fi

    # Test API endpoint (should return 401 without auth)
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost/api/v1/lists)
    if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "200" ]; then
        log_success "API endpoint reachable (HTTP $HTTP_CODE)"
    else
        log_warning "API endpoint returned HTTP $HTTP_CODE"
    fi
}

# Show summary
show_summary() {
    PUBLIC_IP=$(curl -s ifconfig.me || echo "Unable to determine")

    cat << EOF

${GREEN}=================================================================
Nginx Reverse Proxy Setup Complete!
=================================================================${NC}

Configuration:
  Server Name: ${SERVER_NAME}
  Backend: http://127.0.0.1:${APP_PORT}
  Config File: ${CONF_DIR}/todolist-api.conf

Access URLs:
  Health Check: http://${SERVER_NAME}/health
  API Base URL: http://${SERVER_NAME}/api/v1

Test Commands:
  # Health check
  curl http://${SERVER_NAME}/health

  # Register user
  curl -X POST http://${SERVER_NAME}/api/v1/auth/register \\
    -H "Content-Type: application/json" \\
    -d '{"email":"test@example.com","password":"SecurePass123!"}'

Nginx Management:
  Status:  sudo systemctl status nginx
  Restart: sudo systemctl restart nginx
  Reload:  sudo systemctl reload nginx
  Test:    sudo nginx -t
  Logs:    sudo tail -f /var/log/nginx/todolist-api-error.log

${YELLOW}Next Steps:${NC}
1. Test the API through Nginx
2. Set up SSL with Let's Encrypt (Phase 5)
3. Update DNS records if using custom domain
4. Configure rate limiting if needed

${YELLOW}Security Features Enabled:${NC}
✓ Rate limiting (10 req/s for API, 5 req/s for auth)
✓ Security headers
✓ Request size limits
✓ Timeouts configured
✓ Hidden file access denied

${YELLOW}SSL Setup (Phase 5):${NC}
To enable HTTPS with Let's Encrypt, run:
  sudo ./setup-ssl.sh ${DOMAIN}

EOF
}

# Main installation
main() {
    log_info "Starting Nginx setup for OCI..."
    echo

    detect_os
    determine_server_name

    echo
    install_nginx
    backup_default_config
    create_nginx_config
    configure_selinux
    test_nginx_config
    start_nginx
    configure_firewall
    test_reverse_proxy

    echo
    show_summary
}

# Run main installation
main
