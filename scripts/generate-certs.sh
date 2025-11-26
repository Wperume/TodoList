#!/bin/bash
#
# Generate self-signed SSL certificates for development
#
# Usage: ./scripts/generate-certs.sh [domain-or-ip] [owner]
#
# Examples:
#   ./scripts/generate-certs.sh                    # localhost
#   ./scripts/generate-certs.sh example.com        # domain
#   ./scripts/generate-certs.sh 192.168.1.100      # IP address
#   ./scripts/generate-certs.sh localhost todolist # with specific owner
#

set -e

# Default domain or IP
DOMAIN_OR_IP="${1:-localhost}"
CERT_DIR="./certs"
DAYS=365
OWNER="${2:-}"

echo "🔐 Generating self-signed SSL certificate for development"
echo "Domain/IP: $DOMAIN_OR_IP"
echo "Valid for: $DAYS days"
if [ -n "$OWNER" ]; then
    echo "Owner: $OWNER"
fi
echo ""

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate private key
echo "📝 Generating private key..."
openssl genrsa -out "$CERT_DIR/server.key" 2048

# Detect if input is IP address or domain
if [[ $DOMAIN_OR_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    # It's an IP address
    CN="IP-$DOMAIN_OR_IP"
    SAN="IP:$DOMAIN_OR_IP,IP:127.0.0.1,DNS:localhost"
    echo "Detected IP address: $DOMAIN_OR_IP"
else
    # It's a domain
    CN="$DOMAIN_OR_IP"
    SAN="DNS:$DOMAIN_OR_IP,DNS:*.$DOMAIN_OR_IP,DNS:localhost,IP:127.0.0.1"
    echo "Detected domain: $DOMAIN_OR_IP"
fi

# Generate certificate signing request
echo "📝 Generating certificate signing request..."
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "/C=US/ST=State/L=City/O=Organization/OU=Development/CN=$CN"

# Generate self-signed certificate
echo "📝 Generating self-signed certificate..."
openssl x509 -req -days $DAYS -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" \
  -extfile <(printf "subjectAltName=$SAN")

# Set proper permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

# Set ownership if specified
if [ -n "$OWNER" ]; then
    echo "📝 Setting ownership to $OWNER..."
    if id "$OWNER" &>/dev/null; then
        chown "$OWNER:$OWNER" "$CERT_DIR/server.key" "$CERT_DIR/server.crt"
    else
        echo "⚠️  Warning: User '$OWNER' does not exist, skipping ownership change"
    fi
fi

# Clean up CSR
rm "$CERT_DIR/server.csr"

echo ""
echo "✅ Certificate generated successfully!"
echo ""
echo "Files created:"
echo "  - Private key: $CERT_DIR/server.key"
echo "  - Certificate: $CERT_DIR/server.crt"
if [ -n "$OWNER" ]; then
    echo "  - Ownership: $OWNER:$OWNER"
fi
echo ""
echo "⚠️  WARNING: This is a self-signed certificate for DEVELOPMENT ONLY"
echo "   DO NOT use in production!"
echo ""
echo "To use with the API (.env file):"
echo "  TLS_ENABLED=true"
echo "  TLS_CERT_FILE=$CERT_DIR/server.crt"
echo "  TLS_KEY_FILE=$CERT_DIR/server.key"
echo "  TLS_PORT=8443"
echo "  TLS_REDIRECT_HTTP=true"
echo ""
echo "Or via environment variables:"
echo "  export TLS_ENABLED=true"
echo "  export TLS_CERT_FILE=$CERT_DIR/server.crt"
echo "  export TLS_KEY_FILE=$CERT_DIR/server.key"
echo "  ./todolist-api"
echo ""
echo "To test:"
echo "  curl -k https://localhost:8443/health"
if [[ $DOMAIN_OR_IP =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "  curl -k https://$DOMAIN_OR_IP:8443/health"
else
    echo "  curl -k https://$DOMAIN_OR_IP:8443/health"
fi
echo ""
echo "To trust this certificate (macOS):"
echo "  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CERT_DIR/server.crt"
echo ""
echo "To trust this certificate (Linux):"
echo "  sudo cp $CERT_DIR/server.crt /usr/local/share/ca-certificates/"
echo "  sudo update-ca-certificates"
echo ""
