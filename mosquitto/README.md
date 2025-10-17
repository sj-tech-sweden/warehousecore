# Mosquitto MQTT Broker Setup

This directory contains the configuration for the self-hosted Mosquitto MQTT broker used by StorageCore's LED warehouse highlighting system.

## Quick Setup

### 1. Create Password File

Before starting the broker, you need to create a password file for authentication:

```bash
# Create initial password file with a user (replace 'leduser' and password as needed)
docker run -it --rm -v $(pwd)/mosquitto/config:/mosquitto/config eclipse-mosquitto:2.0 \
  mosquitto_passwd -c /mosquitto/config/passwords leduser
```

You'll be prompted to enter the password twice.

### 2. Add Additional Users (Optional)

To add more users:

```bash
# Add another user (without -c flag to append, not replace)
docker run -it --rm -v $(pwd)/mosquitto/config:/mosquitto/config eclipse-mosquitto:2.0 \
  mosquitto_passwd /mosquitto/config/passwords anotheruser
```

### 3. Set Correct Permissions

```bash
# Make sure mosquitto can read/write the files
chmod -R 755 mosquitto/
chmod 644 mosquitto/config/passwords
```

### 4. Start the Broker

```bash
docker-compose up -d mosquitto
```

### 5. Verify It's Running

```bash
# Check logs
docker-compose logs -f mosquitto

# Test connection (requires mosquitto-clients installed locally)
mosquitto_sub -h localhost -p 1883 -t test/topic -u leduser -P yourpassword
```

## Configuration

### Environment Variables (.env)

Update your `.env` file to use the self-hosted broker:

```env
# LED MQTT Configuration - Self-Hosted
LED_MQTT_HOST=mosquitto
LED_MQTT_PORT=1883
LED_MQTT_TLS=false
LED_MQTT_USER=leduser
LED_MQTT_PASS=yourpassword
LED_TOPIC_PREFIX=weidelbach
WAREHOUSE_ID=weidelbach
```

### ESP32 Configuration

In your ESP32 `secrets.h`, use your server's public IP or domain:

```cpp
#define MQTT_HOST "your-server.example.com"  // or your server's IP
#define MQTT_PORT 1883
#define MQTT_USER "leduser"
#define MQTT_PASS "yourpassword"
```

## Ports

- **1883**: Plain MQTT (default, no TLS)
- **8883**: MQTT over TLS (requires SSL certificates - see TLS Setup below)
- **9001**: WebSockets (for web-based MQTT clients)

## TLS Setup (Production Recommended)

For production environments, enable TLS:

### 1. Get SSL Certificates

Using Let's Encrypt (recommended):

```bash
# Install certbot
sudo apt install certbot

# Get certificate (replace with your domain)
sudo certbot certonly --standalone -d mqtt.your-domain.com
```

### 2. Copy Certificates to Mosquitto

```bash
# Create certs directory
mkdir -p mosquitto/certs

# Copy certificates (replace paths with your actual cert paths)
sudo cp /etc/letsencrypt/live/mqtt.your-domain.com/fullchain.pem mosquitto/certs/server.crt
sudo cp /etc/letsencrypt/live/mqtt.your-domain.com/privkey.pem mosquitto/certs/server.key
sudo cp /etc/letsencrypt/live/mqtt.your-domain.com/chain.pem mosquitto/certs/ca.crt

# Fix permissions
sudo chown -R 1883:1883 mosquitto/certs
chmod 644 mosquitto/certs/*.crt
chmod 600 mosquitto/certs/*.key
```

### 3. Update docker-compose.yml

Add volume mount for certificates:

```yaml
mosquitto:
  volumes:
    - ./mosquitto/certs:/mosquitto/certs:ro
```

### 4. Enable TLS in mosquitto.conf

Uncomment the TLS listener section in `mosquitto/config/mosquitto.conf`

### 5. Update Environment Variables

```env
LED_MQTT_PORT=8883
LED_MQTT_TLS=true
```

### 6. Update ESP32 Configuration

```cpp
#define MQTT_PORT 8883
#define USE_TLS 1
```

## Firewall Configuration

Make sure your firewall allows incoming connections on the MQTT ports:

```bash
# For plain MQTT
sudo ufw allow 1883/tcp

# For MQTT over TLS
sudo ufw allow 8883/tcp

# For WebSockets (optional)
sudo ufw allow 9001/tcp
```

## Monitoring

### Check Broker Status

```bash
# View real-time logs
docker-compose logs -f mosquitto

# Check connected clients
docker exec mosquitto mosquitto_sub -t '$SYS/broker/clients/connected' -C 1 -u leduser -P yourpassword
```

### System Topics

Mosquitto publishes system information to `$SYS/` topics:

- `$SYS/broker/version` - Broker version
- `$SYS/broker/clients/connected` - Number of connected clients
- `$SYS/broker/messages/received` - Total messages received
- `$SYS/broker/uptime` - Broker uptime

Subscribe to view:

```bash
mosquitto_sub -h localhost -p 1883 -t '$SYS/#' -u leduser -P yourpassword
```

## Troubleshooting

### Cannot Connect to Broker

1. Check if mosquitto is running: `docker-compose ps mosquitto`
2. Check logs: `docker-compose logs mosquitto`
3. Verify password file exists: `ls -la mosquitto/config/passwords`
4. Test locally: `mosquitto_sub -h localhost -p 1883 -t test -u leduser -P yourpassword`

### Authentication Failed

1. Verify username/password are correct
2. Check password file format: `cat mosquitto/config/passwords` (should show username:hash)
3. Recreate password file if corrupted

### Permission Denied Errors

```bash
# Fix permissions
chmod -R 755 mosquitto/
chown -R 1883:1883 mosquitto/data mosquitto/log
```

### ESP32 Cannot Connect

1. Verify server's public IP/domain is correct in ESP32 `secrets.h`
2. Check firewall allows port 1883/8883
3. Verify StorageCore and Mosquitto are on same network
4. Test with mosquitto_pub from another machine:
   ```bash
   mosquitto_pub -h your-server.example.com -p 1883 -t test -m "hello" -u leduser -P yourpassword
   ```

## Security Best Practices

1. **Always use strong passwords** for MQTT users
2. **Enable TLS in production** (port 8883)
3. **Restrict network access** using firewall rules
4. **Use ACL files** to limit topic access per user (optional but recommended)
5. **Never expose plain MQTT (1883)** to the internet - use TLS only
6. **Regularly update** the Mosquitto Docker image

## ACL Configuration (Optional)

For fine-grained access control, create `mosquitto/config/acl.conf`:

```conf
# ACL file example

# leduser can publish/subscribe to warehouse topics
user leduser
topic readwrite weidelbach/#

# statususer can only subscribe to status
user statususer
topic read weidelbach/+/status
```

Then uncomment the ACL line in `mosquitto.conf`:

```conf
acl_file /mosquitto/config/acl.conf
```

## Backup

Important files to backup:

- `mosquitto/config/passwords` - User credentials
- `mosquitto/data/` - Persistent message store
- `mosquitto/config/mosquitto.conf` - Configuration
- `mosquitto/config/acl.conf` - Access control (if used)
