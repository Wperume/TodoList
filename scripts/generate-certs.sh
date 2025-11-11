#!/bin/bash
#
# Generate self-signed SSL certificates for development
#
# Usage: ./scripts/generate-certs.sh [domain]
#

set -e

# Default domain
DOMAIN="${1:-localhost}"
CERT_DIR="./certs"
DAYS=365

echo "🔐 Generating self-signed SSL certificate for development"
echo "Domain: $DOMAIN"
echo "Valid for: $DAYS days"
echo ""

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate private key
echo "📝 Generating private key..."
openssl genrsa -out "$CERT_DIR/server.key" 2048

# Generate certificate signing request
echo "📝 Generating certificate signing request..."
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "/C=US/ST=State/L=City/O=Organization/OU=Development/CN=$DOMAIN"

# Generate self-signed certificate
echo "📝 Generating self-signed certificate..."
openssl x509 -req -days $DAYS -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" \
  -extfile <(printf "subjectAltName=DNS:$DOMAIN,DNS:*.$DOMAIN,DNS:localhost,IP:127.0.0.1")

# Set proper permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

# Clean up CSR
rm "$CERT_DIR/server.csr"

echo ""
echo "✅ Certificate generated successfully!"
echo ""
echo "Files created:"
echo "  - Private key: $CERT_DIR/server.key"
echo "  - Certificate: $CERT_DIR/server.crt"
echo ""
echo "⚠️  WARNING: This is a self-signed certificate for DEVELOPMENT ONLY"
echo "   DO NOT use in production!"
echo ""
echo "To use with the API:"
echo "  export TLS_ENABLED=true"
echo "  export TLS_CERT_FILE=$CERT_DIR/server.crt"
echo "  export TLS_KEY_FILE=$CERT_DIR/server.key"
echo "  ./todolist-api"
echo ""
echo "To trust this certificate (macOS):"
echo "  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain $CERT_DIR/server.crt"
echo ""
echo "To trust this certificate (Linux):"
echo "  sudo cp $CERT_DIR/server.crt /usr/local/share/ca-certificates/"
echo "  sudo update-ca-certificates"
echo ""
