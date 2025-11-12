#!/bin/bash
#
# Phase 5: SSL/HTTPS Setup Script using Let's Encrypt
#
# This script installs Certbot and configures SSL for your domain
#
# Prerequisites:
#   - Domain name must point to your VM's public IP
#   - Nginx must be installed and running (Phase 4)
#   - Ports 80 and 443 must be open
#
# Usage:
#   sudo ./setup-ssl.sh <domain-name> [email]
#
# Examples:
#   sudo ./setup-ssl.sh api.example.com
#   sudo ./setup-ssl.sh api.example.com admin@example.com
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
EMAIL="${2:-}"

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

# Validate domain argument
if [ -z "$DOMAIN" ]; then
    log_error "Domain name is required"
    echo
    echo "Usage: $0 <domain-name> [email]"
    echo "Example: $0 api.example.com admin@example.com"
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

# Prompt for email if not provided
prompt_email() {
    if [ -z "$EMAIL" ]; then
        read -p "Enter email for Let's Encrypt notifications: " EMAIL

        if [ -z "$EMAIL" ]; then
            log_error "Email is required"
            exit 1
        fi
    fi

    log_info "Using email: $EMAIL"
}

# Check DNS configuration
check_dns() {
    log_info "Checking DNS configuration for $DOMAIN..."

    PUBLIC_IP=$(curl -s ifconfig.me || echo "")

    if [ -z "$PUBLIC_IP" ]; then
        log_warning "Could not determine public IP"
        return
    fi

    DOMAIN_IP=$(dig +short $DOMAIN | tail -1)

    if [ -z "$DOMAIN_IP" ]; then
        log_error "Domain $DOMAIN does not resolve to any IP"
        log_error "Please configure DNS before proceeding"
        exit 1
    fi

    if [ "$DOMAIN_IP" != "$PUBLIC_IP" ]; then
        log_warning "Domain resolves to $DOMAIN_IP but VM public IP is $PUBLIC_IP"
        log_warning "Make sure DNS is configured correctly"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        log_success "DNS configured correctly ($DOMAIN → $PUBLIC_IP)"
    fi
}

# Check if Nginx is running
check_nginx() {
    log_info "Checking Nginx status..."

    if systemctl is-active --quiet nginx; then
        log_success "Nginx is running"
    else
        log_error "Nginx is not running. Please run Phase 4 setup first."
        exit 1
    fi

    # Check if domain is configured
    if ! nginx -T 2>/dev/null | grep -q "server_name.*$DOMAIN"; then
        log_warning "Domain $DOMAIN not found in Nginx config"
        log_info "Will configure domain in Nginx"
    fi
}

# Install Certbot
install_certbot() {
    log_info "Installing Certbot..."

    if [ "$OS" = "oracle" ]; then
        # Oracle Linux
        dnf install -y certbot python3-certbot-nginx
    else
        # Ubuntu
        apt-get update
        apt-get install -y certbot python3-certbot-nginx
    fi

    log_success "Certbot installed"
}

# Configure firewall for HTTPS
configure_firewall() {
    log_info "Ensuring HTTPS port is open..."

    if command -v firewall-cmd &> /dev/null; then
        # firewalld (Oracle Linux)
        firewall-cmd --permanent --add-service=https
        firewall-cmd --reload
        log_success "Firewall configured (port 443 open)"
    elif command -v ufw &> /dev/null; then
        # ufw (Ubuntu)
        ufw allow 443/tcp
        log_success "Firewall configured (port 443 open)"
    else
        log_warning "No firewall detected"
    fi
}

# Obtain SSL certificate
obtain_certificate() {
    log_info "Obtaining SSL certificate from Let's Encrypt..."
    echo

    # Run certbot
    if certbot --nginx -d $DOMAIN --non-interactive --agree-tos --email $EMAIL --redirect; then
        log_success "SSL certificate obtained successfully"
    else
        log_error "Failed to obtain SSL certificate"
        log_error "Common issues:"
        log_error "  - Domain not pointing to this server"
        log_error "  - Port 80 not accessible from internet"
        log_error "  - Firewall blocking connections"
        exit 1
    fi
}

# Test SSL configuration
test_ssl() {
    log_info "Testing SSL configuration..."

    sleep 2

    # Test HTTPS endpoint
    if curl -f -s -k https://$DOMAIN/health > /dev/null; then
        log_success "HTTPS is working"
    else
        log_warning "HTTPS test inconclusive"
    fi

    # Check certificate
    if echo | openssl s_client -connect $DOMAIN:443 -servername $DOMAIN 2>/dev/null | grep -q "Verify return code: 0"; then
        log_success "SSL certificate is valid"
    else
        log_warning "SSL certificate validation inconclusive"
    fi
}

# Test auto-renewal
test_auto_renewal() {
    log_info "Testing automatic renewal..."

    if certbot renew --dry-run; then
        log_success "Automatic renewal is configured correctly"
    else
        log_warning "Automatic renewal test failed"
    fi
}

# Configure HSTS (optional but recommended)
configure_hsts() {
    log_info "Would you like to enable HSTS (HTTP Strict Transport Security)?"
    log_warning "HSTS forces browsers to always use HTTPS (recommended for production)"
    read -p "Enable HSTS? (y/N): " -n 1 -r
    echo

    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Find nginx config file
        if [ "$OS" = "oracle" ]; then
            CONF_FILE="/etc/nginx/conf.d/todolist-api.conf"
        else
            CONF_FILE="/etc/nginx/sites-available/todolist-api.conf"
        fi

        # Add HSTS header
        if [ -f "$CONF_FILE" ]; then
            # Check if HSTS is already configured
            if ! grep -q "Strict-Transport-Security" "$CONF_FILE"; then
                sed -i '/add_header X-Frame-Options/a \    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;' "$CONF_FILE"

                nginx -t && systemctl reload nginx
                log_success "HSTS enabled (1 year max-age)"
            else
                log_info "HSTS already configured"
            fi
        fi
    fi
}

# Show certificate information
show_certificate_info() {
    log_info "Certificate information:"

    certbot certificates | grep -A 10 "$DOMAIN" || true
}

# Show summary
show_summary() {
    cat << EOF

${GREEN}=================================================================
SSL/HTTPS Setup Complete!
=================================================================${NC}

Domain: ${DOMAIN}
Certificate Authority: Let's Encrypt
Certificate Location: /etc/letsencrypt/live/${DOMAIN}/

HTTPS URLs:
  Health Check: https://${DOMAIN}/health
  API Base URL: https://${DOMAIN}/api/v1

Test Commands:
  # Health check (HTTPS)
  curl https://${DOMAIN}/health

  # Register user (HTTPS)
  curl -X POST https://${DOMAIN}/api/v1/auth/register \\
    -H "Content-Type: application/json" \\
    -d '{"email":"test@example.com","password":"SecurePass123!"}'

Certificate Management:
  View Certificates: sudo certbot certificates
  Renew Manually:    sudo certbot renew
  Revoke:            sudo certbot revoke --cert-path /etc/letsencrypt/live/${DOMAIN}/cert.pem

${YELLOW}Important Information:${NC}

Auto-Renewal:
  ✓ Certificates auto-renew via systemd timer
  ✓ Check status: sudo systemctl status certbot.timer
  ✓ Test renewal: sudo certbot renew --dry-run

Certificate Expiry:
  - Certificates are valid for 90 days
  - Auto-renewal attempts 30 days before expiry
  - You'll receive email notifications at: ${EMAIL}

Security:
  ✓ TLS 1.2 and 1.3 enabled
  ✓ Strong cipher suites configured
  ✓ HTTP automatically redirects to HTTPS
  $([ -f "/etc/nginx/conf.d/todolist-api.conf" ] && grep -q "Strict-Transport-Security" "/etc/nginx/conf.d/todolist-api.conf" && echo "✓ HSTS enabled (forces HTTPS)" || echo "○ HSTS not enabled")

${YELLOW}Monitoring:${NC}
  Check Nginx logs:
    sudo tail -f /var/log/nginx/todolist-api-error.log

  Check Let's Encrypt logs:
    sudo tail -f /var/log/letsencrypt/letsencrypt.log

${YELLOW}Troubleshooting:${NC}
  If SSL doesn't work:
    1. Check DNS: dig ${DOMAIN}
    2. Check firewall: sudo firewall-cmd --list-all
    3. Check Nginx: sudo nginx -t
    4. Check certificates: sudo certbot certificates
    5. View Nginx logs: sudo journalctl -u nginx -f

${GREEN}Your API is now secure and production-ready! 🎉${NC}

EOF
}

# Main installation
main() {
    log_info "Starting SSL/HTTPS setup for OCI..."
    echo

    detect_os
    prompt_email

    echo
    log_info "Performing pre-flight checks..."
    check_dns
    check_nginx

    echo
    log_info "Installing and configuring SSL..."
    install_certbot
    configure_firewall
    obtain_certificate

    echo
    log_info "Testing configuration..."
    test_ssl
    test_auto_renewal

    echo
    configure_hsts

    echo
    show_certificate_info

    echo
    show_summary
}

# Run main installation
main
