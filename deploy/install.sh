#!/bin/bash
# Installation script for IRC server with systemd
# Run with sudo

set -e

echo "=== IRC Server Installation ==="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Please run as root (sudo)"
    exit 1
fi

# Configuration
INSTALL_DIR="/opt/ircd"
USER="ircd"
GROUP="ircd"
SERVICE_NAME="ircd.service"

# Create user and group if they don't exist
if ! id -u $USER > /dev/null 2>&1; then
    echo "Creating user '$USER'..."
    useradd --system --home $INSTALL_DIR --shell /bin/false $USER
else
    echo "User '$USER' already exists"
fi

# Create installation directory
echo "Creating installation directory..."
mkdir -p $INSTALL_DIR/{bin,config,certs,logs}

# Copy files
echo "Copying server files..."
cp bin/ircd $INSTALL_DIR/bin/
cp config/config.yaml $INSTALL_DIR/config/

# Copy certificates if they exist
if [ -f "certs/server.crt" ] && [ -f "certs/server.key" ]; then
    echo "Copying TLS certificates..."
    cp certs/server.crt $INSTALL_DIR/certs/
    cp certs/server.key $INSTALL_DIR/certs/
    chmod 600 $INSTALL_DIR/certs/server.key
else
    echo "Warning: TLS certificates not found. You'll need to generate them."
    echo "Run: cd $INSTALL_DIR && ./generate_cert.sh"
fi

# Set permissions
echo "Setting permissions..."
chown -R $USER:$GROUP $INSTALL_DIR
chmod 755 $INSTALL_DIR/bin/ircd
chmod 644 $INSTALL_DIR/config/config.yaml

# Make logs writable
chmod 755 $INSTALL_DIR/logs

# Install systemd service
echo "Installing systemd service..."
cp deploy/ircd.service /etc/systemd/system/
systemctl daemon-reload

# Enable and start service
echo ""
echo "Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Edit configuration: sudo nano $INSTALL_DIR/config/config.yaml"
echo "  2. Generate TLS certificates (if needed): cd $INSTALL_DIR && sudo -u $USER ./generate_cert.sh"
echo "  3. Enable service: sudo systemctl enable $SERVICE_NAME"
echo "  4. Start service: sudo systemctl start $SERVICE_NAME"
echo "  5. Check status: sudo systemctl status $SERVICE_NAME"
echo "  6. View logs: sudo journalctl -u $SERVICE_NAME -f"
echo ""
echo "Firewall configuration (if needed):"
echo "  sudo ufw allow 6667/tcp  # IRC plaintext"
echo "  sudo ufw allow 7000/tcp  # IRC TLS"
echo ""
