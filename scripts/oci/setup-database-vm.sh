#!/bin/bash
#
# Phase 2: Database Server Setup Script for Oracle Cloud Infrastructure
#
# This script installs and configures PostgreSQL on the database VM
#
# Usage:
#   1. Upload this script to your database VM
#   2. Run: chmod +x setup-database-vm.sh
#   3. Run: sudo ./setup-database-vm.sh
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
DB_NAME="todolist"
DB_USER="todolist"
DB_PASSWORD=""
VCN_CIDR="10.0.0.0/16"  # Default OCI VCN CIDR
PG_VERSION="15"

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

# Prompt for database password
prompt_password() {
    log_info "Setting up database credentials..."

    if [ -z "$DB_PASSWORD" ]; then
        read -sp "Enter password for PostgreSQL user '$DB_USER': " DB_PASSWORD
        echo
        read -sp "Confirm password: " DB_PASSWORD_CONFIRM
        echo

        if [ "$DB_PASSWORD" != "$DB_PASSWORD_CONFIRM" ]; then
            log_error "Passwords do not match"
            exit 1
        fi

        if [ ${#DB_PASSWORD} -lt 12 ]; then
            log_error "Password must be at least 12 characters"
            exit 1
        fi
    fi

    log_success "Password set"
}

# Install PostgreSQL on Oracle Linux
install_postgresql_oracle() {
    log_info "Installing PostgreSQL on Oracle Linux..."

    # Enable PostgreSQL repository
    dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-${OS_VERSION}-x86_64/pgdg-redhat-repo-latest.noarch.rpm

    # Disable built-in PostgreSQL module
    dnf -qy module disable postgresql

    # Install PostgreSQL
    dnf install -y postgresql${PG_VERSION}-server postgresql${PG_VERSION}-contrib

    log_success "PostgreSQL installed"
}

# Install PostgreSQL on Ubuntu
install_postgresql_ubuntu() {
    log_info "Installing PostgreSQL on Ubuntu..."

    apt-get update
    apt-get install -y postgresql postgresql-contrib

    log_success "PostgreSQL installed"
}

# Initialize PostgreSQL
initialize_postgresql() {
    log_info "Initializing PostgreSQL..."

    if [ "$OS" = "oracle" ]; then
        # Initialize database
        /usr/pgsql-${PG_VERSION}/bin/postgresql-${PG_VERSION}-setup initdb

        # Enable and start service
        systemctl enable postgresql-${PG_VERSION}
        systemctl start postgresql-${PG_VERSION}

        PG_DATA_DIR="/var/lib/pgsql/${PG_VERSION}/data"
        PG_SERVICE="postgresql-${PG_VERSION}"
    else
        # Ubuntu
        systemctl enable postgresql
        systemctl start postgresql

        PG_DATA_DIR="/etc/postgresql/${PG_VERSION}/main"
        PG_SERVICE="postgresql"
    fi

    log_success "PostgreSQL initialized and started"
}

# Create database and user
create_database() {
    log_info "Creating database and user..."

    # Create user and database
    sudo -u postgres psql << EOF
-- Create user
CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';

-- Create database
CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};

-- Connect to database and grant schema privileges
\c ${DB_NAME}
GRANT ALL ON SCHEMA public TO ${DB_USER};
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ${DB_USER};
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ${DB_USER};
EOF

    log_success "Database '$DB_NAME' and user '$DB_USER' created"
}

# Configure PostgreSQL for remote access
configure_postgresql() {
    log_info "Configuring PostgreSQL for remote access..."

    # Backup original files
    cp ${PG_DATA_DIR}/postgresql.conf ${PG_DATA_DIR}/postgresql.conf.backup
    cp ${PG_DATA_DIR}/pg_hba.conf ${PG_DATA_DIR}/pg_hba.conf.backup

    # Configure postgresql.conf
    cat >> ${PG_DATA_DIR}/postgresql.conf << EOF

# Custom configuration for TodoList API
listen_addresses = '*'
max_connections = 50
shared_buffers = 256MB
effective_cache_size = 512MB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 2621kB
min_wal_size = 512MB
max_wal_size = 2GB
EOF

    # Configure pg_hba.conf for VCN access
    cat >> ${PG_DATA_DIR}/pg_hba.conf << EOF

# Allow access from VCN
host    ${DB_NAME}    ${DB_USER}    ${VCN_CIDR}    md5
host    all           ${DB_USER}    ${VCN_CIDR}    md5
EOF

    log_success "PostgreSQL configured"
}

# Configure firewall
configure_firewall() {
    log_info "Configuring firewall..."

    if command -v firewall-cmd &> /dev/null; then
        # firewalld (Oracle Linux)
        firewall-cmd --permanent --add-service=postgresql
        firewall-cmd --permanent --add-port=5432/tcp
        firewall-cmd --reload
        log_success "Firewall configured (firewalld)"
    elif command -v ufw &> /dev/null; then
        # ufw (Ubuntu)
        ufw allow 5432/tcp
        log_success "Firewall configured (ufw)"
    else
        log_warning "No firewall detected. Please configure manually."
    fi
}

# Restart PostgreSQL
restart_postgresql() {
    log_info "Restarting PostgreSQL..."
    systemctl restart ${PG_SERVICE}
    sleep 3

    if systemctl is-active --quiet ${PG_SERVICE}; then
        log_success "PostgreSQL restarted successfully"
    else
        log_error "PostgreSQL failed to restart"
        systemctl status ${PG_SERVICE}
        exit 1
    fi
}

# Test connection
test_connection() {
    log_info "Testing database connection..."

    if sudo -u postgres psql -U ${DB_USER} -d ${DB_NAME} -h localhost -c "SELECT version();" &> /dev/null; then
        log_success "Database connection test passed"
    else
        log_warning "Database connection test failed (this might be OK if password auth is required)"
    fi
}

# Show connection info
show_connection_info() {
    PRIVATE_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d/ -f1 | head -1)

    cat << EOF

${GREEN}=================================================================
Database Setup Complete!
=================================================================${NC}

Database Information:
  Database Name: ${DB_NAME}
  Database User: ${DB_USER}
  Database Password: ${DB_PASSWORD}

Connection Details:
  Host: ${PRIVATE_IP} (Private IP)
  Port: 5432

Connection String:
  postgresql://${DB_USER}:${DB_PASSWORD}@${PRIVATE_IP}:5432/${DB_NAME}

Test Connection:
  psql -U ${DB_USER} -h ${PRIVATE_IP} -d ${DB_NAME} -W

${YELLOW}IMPORTANT:${NC}
1. Save the password securely!
2. Use the PRIVATE IP (${PRIVATE_IP}) in your application's .env file
3. Ensure OCI Security List allows port 5432 from your VCN CIDR
4. Configure OS firewall if needed

Next Steps:
  - Set up the application VM (Phase 3)
  - Use the connection details in your .env file

EOF
}

# Save credentials to file
save_credentials() {
    CREDS_FILE="/root/db-credentials.txt"
    PRIVATE_IP=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | awk '{print $2}' | cut -d/ -f1 | head -1)

    cat > ${CREDS_FILE} << EOF
TodoList Database Credentials
Generated: $(date)

Database Name: ${DB_NAME}
Database User: ${DB_USER}
Database Password: ${DB_PASSWORD}
Private IP: ${PRIVATE_IP}
Port: 5432

Connection String:
DB_HOST=${PRIVATE_IP}
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}
DB_SSL_MODE=disable
EOF

    chmod 600 ${CREDS_FILE}
    log_info "Credentials saved to: ${CREDS_FILE}"
}

# Main installation
main() {
    log_info "Starting PostgreSQL setup for OCI..."
    echo

    detect_os
    prompt_password

    echo
    log_info "Installing PostgreSQL..."

    if [ "$OS" = "oracle" ]; then
        install_postgresql_oracle
    elif [ "$OS" = "ubuntu" ]; then
        install_postgresql_ubuntu
    else
        log_error "Unsupported OS: $OS"
        exit 1
    fi

    initialize_postgresql
    create_database
    configure_postgresql
    configure_firewall
    restart_postgresql
    test_connection
    save_credentials

    echo
    show_connection_info
}

# Run main installation
main
