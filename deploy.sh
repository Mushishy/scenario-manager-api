#!/bin/bash

set -e 

SERVICE_USER="ludus"
SERVICE_GROUP="ludus"
SERVICE_NAME="scenario-manager-api"

# Check if the binary exists
if [ ! -f "/opt/scenario-manager-api/server/scenario-manager-api" ]; then
    echo "Error: Binary '/opt/scenario-manager-api/server/scenario-manager-api' not found. Please run the build script first with the CWD /opt/scenario-manager-api"
    exit 1
fi

echo "Deploying $SERVICE_NAME..."

# Stop service if running
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Stopping $SERVICE_NAME service..."
    sudo systemctl stop "$SERVICE_NAME"
fi

# Set permissions
sudo chown -R "$SERVICE_USER":"$SERVICE_GROUP" "/opt/scenario-manager-api"

# Install systemd service
echo "Installing systemd service..."
sudo cp "/opt/scenario-manager-api/server/scenario-manager-api.service" /etc/systemd/system/
sudo systemctl daemon-reload

# Enable and start service
sudo systemctl enable "$SERVICE_NAME"
sudo systemctl start "$SERVICE_NAME"

echo "Deployment complete!"
echo "Service status:"
sudo systemctl status "$SERVICE_NAME" --no-pager
