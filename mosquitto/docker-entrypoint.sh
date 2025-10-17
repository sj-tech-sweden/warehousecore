#!/bin/sh
set -e

# Mosquitto Docker Entrypoint with automatic password file generation
# This script reads MQTT_USER and MQTT_PASS from environment variables
# and automatically creates/updates the password file on container startup

echo "[Mosquitto] Starting with automatic password configuration..."

# Check if MQTT_USER and MQTT_PASS are set
if [ -n "$MQTT_USER" ] && [ -n "$MQTT_PASS" ]; then
    echo "[Mosquitto] Generating password file for user: $MQTT_USER"

    # Generate password file from environment variables
    mosquitto_passwd -c -b /mosquitto/config/passwords "$MQTT_USER" "$MQTT_PASS"

    # Set correct permissions
    chmod 644 /mosquitto/config/passwords

    echo "[Mosquitto] Password file generated successfully"
else
    echo "[Mosquitto] WARNING: MQTT_USER or MQTT_PASS not set in environment"
    echo "[Mosquitto] Using existing password file or allowing anonymous access"
fi

# Start Mosquitto with the provided arguments
echo "[Mosquitto] Starting Mosquitto broker..."
exec /usr/sbin/mosquitto -c /mosquitto/config/mosquitto.conf
