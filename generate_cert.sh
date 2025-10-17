#!/bin/bash
# Generate self-signed TLS certificates for IRC server

CERT_DIR="certs"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"

echo "🔐 Generating self-signed TLS certificates for IRC server"
echo

# Create certs directory if it doesn't exist
mkdir -p "$CERT_DIR"

# Generate private key and certificate
openssl req -x509 -newkey rsa:4096 -keyout "$KEY_FILE" -out "$CERT_FILE" \
    -days 365 -nodes -subj "/CN=localhost/O=IRCServer/C=US" 2>/dev/null

if [ $? -eq 0 ]; then
    echo "✅ Certificate generated successfully!"
    echo
    echo "   Certificate: $CERT_FILE"
    echo "   Private Key: $KEY_FILE"
    echo "   Valid for: 365 days"
    echo
    echo "To enable TLS, update config/config.yaml:"
    echo "   server.tls.enabled: true"
    echo
    echo "Then connect with:"
    echo "   openssl s_client -connect localhost:6697"
else
    echo "❌ Failed to generate certificates"
    echo "   Make sure openssl is installed"
    exit 1
fi
