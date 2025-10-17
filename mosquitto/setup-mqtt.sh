#!/bin/bash
# Setup script for Mosquitto MQTT Broker
# Creates password file for user authentication

set -e

echo "==================================="
echo "Mosquitto MQTT Broker Setup"
echo "==================================="
echo

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "docker-compose.yml" ]; then
    echo "Error: This script must be run from the storagecore root directory"
    echo "Usage: ./mosquitto/setup-mqtt.sh"
    exit 1
fi

# Create directories if they don't exist
echo "Creating Mosquitto directories..."
mkdir -p mosquitto/config
mkdir -p mosquitto/data
mkdir -p mosquitto/log

# Prompt for username
read -p "Enter MQTT username [leduser]: " MQTT_USER
MQTT_USER=${MQTT_USER:-leduser}

# Create password file
echo
echo "Creating password file for user: $MQTT_USER"
echo "You will be prompted to enter the password twice."
echo

docker run -it --rm \
  -v "$(pwd)/mosquitto/config:/mosquitto/config" \
  eclipse-mosquitto:2.0 \
  mosquitto_passwd -c /mosquitto/config/passwords "$MQTT_USER"

if [ $? -eq 0 ]; then
    echo
    echo "✓ Password file created successfully!"
    echo

    # Set correct permissions
    echo "Setting permissions..."
    chmod 755 mosquitto/config
    chmod 644 mosquitto/config/passwords 2>/dev/null || true
    chmod 644 mosquitto/config/mosquitto.conf

    echo
    echo "==================================="
    echo "Setup Complete!"
    echo "==================================="
    echo
    echo "Next steps:"
    echo "1. Update your .env file with MQTT credentials:"
    echo "   LED_MQTT_HOST=mosquitto"
    echo "   LED_MQTT_PORT=1883"
    echo "   LED_MQTT_TLS=false"
    echo "   LED_MQTT_USER=$MQTT_USER"
    echo "   LED_MQTT_PASS=<your_password>"
    echo
    echo "2. Start the MQTT broker:"
    echo "   docker-compose up -d mosquitto"
    echo
    echo "3. Verify it's running:"
    echo "   docker-compose logs -f mosquitto"
    echo
    echo "4. Update your ESP32 secrets.h with:"
    echo "   MQTT_HOST=\"your-server-domain-or-ip\""
    echo "   MQTT_PORT=1883"
    echo "   MQTT_USER=\"$MQTT_USER\""
    echo "   MQTT_PASS=\"<your_password>\""
    echo
    echo "See mosquitto/README.md for more information."
    echo
else
    echo
    echo "✗ Failed to create password file"
    exit 1
fi
