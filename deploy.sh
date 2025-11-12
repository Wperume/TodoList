#!/bin/bash
#
# TodoList API Deployment Script
#
# This script automates the deployment process on cloud VMs
# It handles code updates, database migrations, and service restart
#
# Usage:
#   ./deploy.sh                 # Deploy to current machine
#   ./deploy.sh production      # Deploy to production environment
#   ./deploy.sh staging         # Deploy to staging environment
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
APP_NAME="todolist-api"
APP_DIR="/opt/${APP_NAME}"
SERVICE_NAME="${APP_NAME}.service"
BACKUP_DIR="/opt/${APP_NAME}/backups"

# Environment (default to production)
ENVIRONMENT="${1:-production}"

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

# Check if running as appropriate user (not root for safety)
check_user() {
    if [ "$EUID" -eq 0 ]; then
        log_warning "Running as root. Consider using a dedicated service account."
    fi
}

# Load environment variables
load_env() {
    if [ -f "${APP_DIR}/.env.${ENVIRONMENT}" ]; then
        log_info "Loading environment from .env.${ENVIRONMENT}"
        export $(cat "${APP_DIR}/.env.${ENVIRONMENT}" | grep -v '^#' | xargs)
    elif [ -f "${APP_DIR}/.env" ]; then
        log_info "Loading environment from .env"
        export $(cat "${APP_DIR}/.env" | grep -v '^#' | xargs)
    else
        log_warning "No .env file found. Using system environment variables."
    fi
}

# Backup current binary
backup_binary() {
    if [ -f "${APP_DIR}/${APP_NAME}" ]; then
        local timestamp=$(date +%Y%m%d_%H%M%S)
        local backup_file="${BACKUP_DIR}/${APP_NAME}_${timestamp}"

        mkdir -p "${BACKUP_DIR}"

        log_info "Backing up current binary..."
        cp "${APP_DIR}/${APP_NAME}" "${backup_file}"
        log_success "Backup created: ${backup_file}"

        # Keep only last 5 backups
        ls -t "${BACKUP_DIR}/${APP_NAME}_"* 2>/dev/null | tail -n +6 | xargs -r rm
    fi
}

# Pull latest code
pull_code() {
    log_info "Pulling latest code from git..."
    cd "${APP_DIR}"

    # Stash any local changes
    if ! git diff-index --quiet HEAD --; then
        log_warning "Local changes detected. Stashing..."
        git stash
    fi

    # Pull latest
    git pull origin main || git pull origin master

    log_success "Code updated successfully"
}

# Build application
build_app() {
    log_info "Building application..."
    cd "${APP_DIR}"

    # Download dependencies
    go mod download

    # Build server binary
    go build -o "${APP_NAME}" cmd/server/main.go

    # Build migration tool
    go build -o bin/migrate cmd/migrate/main.go

    log_success "Build completed successfully"
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."
    cd "${APP_DIR}"

    # Check current version
    if [ -f "bin/migrate" ]; then
        log_info "Current migration version:"
        ./bin/migrate version || true

        # Run migrations
        ./bin/migrate up

        log_success "Migrations completed successfully"
    else
        log_error "Migration tool not found. Build failed?"
        exit 1
    fi
}

# Run tests
run_tests() {
    log_info "Running tests..."
    cd "${APP_DIR}"

    if go test ./... -short -count=1; then
        log_success "All tests passed"
    else
        log_error "Tests failed!"
        read -p "Continue deployment anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_error "Deployment aborted"
            exit 1
        fi
    fi
}

# Stop service
stop_service() {
    log_info "Stopping ${SERVICE_NAME}..."

    if systemctl is-active --quiet "${SERVICE_NAME}"; then
        sudo systemctl stop "${SERVICE_NAME}"
        log_success "Service stopped"
    else
        log_info "Service is not running"
    fi
}

# Start service
start_service() {
    log_info "Starting ${SERVICE_NAME}..."
    sudo systemctl start "${SERVICE_NAME}"

    # Wait a moment for startup
    sleep 2

    # Check status
    if systemctl is-active --quiet "${SERVICE_NAME}"; then
        log_success "Service started successfully"
    else
        log_error "Service failed to start!"
        log_info "Checking service status:"
        sudo systemctl status "${SERVICE_NAME}" --no-pager
        exit 1
    fi
}

# Reload service (graceful restart)
reload_service() {
    log_info "Reloading ${SERVICE_NAME}..."
    sudo systemctl reload-or-restart "${SERVICE_NAME}"

    sleep 2

    if systemctl is-active --quiet "${SERVICE_NAME}"; then
        log_success "Service reloaded successfully"
    else
        log_error "Service failed to reload!"
        exit 1
    fi
}

# Health check
health_check() {
    log_info "Running health check..."

    local max_attempts=10
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if curl -f -s http://localhost:${PORT:-8080}/health > /dev/null; then
            log_success "Health check passed!"
            return 0
        fi

        log_info "Health check attempt $attempt/$max_attempts failed. Waiting..."
        sleep 2
        ((attempt++))
    done

    log_error "Health check failed after $max_attempts attempts"
    log_info "Checking service logs:"
    sudo journalctl -u "${SERVICE_NAME}" -n 50 --no-pager
    exit 1
}

# Rollback
rollback() {
    log_error "Rolling back to previous version..."

    local latest_backup=$(ls -t "${BACKUP_DIR}/${APP_NAME}_"* 2>/dev/null | head -n 1)

    if [ -n "$latest_backup" ]; then
        log_info "Restoring from: $latest_backup"
        cp "$latest_backup" "${APP_DIR}/${APP_NAME}"

        # Rollback one migration
        log_info "Rolling back last migration..."
        ./bin/migrate down || log_warning "Migration rollback failed"

        reload_service
        log_success "Rollback completed"
    else
        log_error "No backup found for rollback!"
        exit 1
    fi
}

# Main deployment process
main() {
    log_info "Starting deployment to ${ENVIRONMENT} environment..."

    check_user
    load_env

    # Create necessary directories
    mkdir -p "${BACKUP_DIR}"
    mkdir -p "${APP_DIR}/bin"

    # Deployment steps
    backup_binary
    pull_code
    build_app

    # Optional: run tests (comment out if too slow)
    # run_tests

    # Database migrations
    run_migrations

    # Restart service
    stop_service
    start_service

    # Verify deployment
    health_check

    log_success "🎉 Deployment completed successfully!"
    log_info "Application is running on port ${PORT:-8080}"
}

# Trap errors and attempt rollback
trap 'log_error "Deployment failed! Check logs above."; exit 1' ERR

# Run main deployment
main

# Show final status
log_info "Final service status:"
sudo systemctl status "${SERVICE_NAME}" --no-pager | head -n 10
