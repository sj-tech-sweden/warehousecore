# XIAO ESP32-C6 Setup Guide for WarehouseCore

Complete setup instructions for flashing the WarehouseCore LED controller firmware on Seeed Studio XIAO ESP32-C6 boards.

---

## Hardware Overview

**Board:** Seeed Studio XIAO ESP32-C6
**Chip:** ESP32-C6 (RISC-V, 160MHz, WiFi 6)
**Flash:** 4MB
**RAM:** 512KB SRAM
**Dimensions:** 21×17.5mm (ultra-compact)

### Pin Layout (XIAO ESP32-C6)

```
        USB-C
         ___
    D0  |   | 5V
    D1  |   | GND
    D2  |   | 3V3
    D3  |   | D10
    D4  |   | D9
    D5  |   | D8
    D6  |   | D7
        |___|
```

**Recommended LED Pin:** D0 (GPIO 0) - Default in firmware
**Alternative Pins:** D1-D5 (GPIO 1-5)

---

## Software Requirements

### 1. Arduino IDE Setup

**Required Arduino IDE Version:** 2.0 or higher
**Required ESP32 Core:** 3.0.0 or higher (for ESP32-C6 support)

#### Install ESP32 Board Support

1. Open Arduino IDE
2. Go to **File → Preferences**
3. Add to "Additional Board Manager URLs":
   ```
   https://espressif.github.io/arduino-esp32/package_esp32_index.json
   ```
4. Go to **Tools → Board → Boards Manager**
5. Search for "esp32"
6. Install **"esp32 by Espressif Systems"** version **3.0.0+**
7. Select **Tools → Board → ESP32 Arduino → XIAO_ESP32C6**

### 2. Required Libraries

Install via **Sketch → Include Library → Manage Libraries**:

| Library | Version | Purpose |
|---------|---------|---------|
| PubSubClient | 2.8+ | MQTT communication |
| ArduinoJson | 7.x | JSON parsing |
| Adafruit NeoPixel | 1.12+ | LED strip control |

---

## Firmware Configuration

### Step 1: Create secrets.h

```bash
cd firmware/esp32_sk6812_leds/
cp secrets.h.template secrets.h
```

### Step 2: Edit secrets.h

**CRITICAL:** Use your server's **external domain or IP**, not "mosquitto"!

```cpp
// WiFi
#define WIFI_SSID "YourWiFiNetwork"
#define WIFI_PASS "YourWiFiPassword"

// MQTT - Use external server address!
#define MQTT_HOST "mqtt.example.com"   // ← Your server domain/IP
#define MQTT_PORT 1883
#define MQTT_USER "leduser"             // Default from docker-compose
#define MQTT_PASS "ledpassword123"      // Default from docker-compose

// WarehouseCore Settings
#define TOPIC_PREFIX "weidelbach"       // Must match WarehouseCore .env
#define WAREHOUSE_ID "WDL"              // Must match WarehouseCore .env

// LED Configuration (XIAO uses GPIO 0 by default - no need to change)
#define LED_LENGTH 60                   // Your actual LED count
```

### Step 3: Verify WarehouseCore MQTT Broker

Ensure your MQTT broker is accessible externally:

```bash
# Test from another machine on your network
mosquitto_sub -h mqtt.example.com -p 1883 -u leduser -P ledpassword123 -t '#' -v
```

If this fails, check firewall rules:
```bash
sudo ufw allow 1883/tcp
```

---

## Flashing the Firmware

### 1. Board Configuration

In Arduino IDE:
- **Board:** "XIAO_ESP32C6"
- **USB CDC On Boot:** "Enabled"
- **Flash Mode:** "QIO 80MHz"
- **Flash Size:** "4MB (32Mb)"
- **Partition Scheme:** "Default 4MB with spiffs"
- **Upload Speed:** "921600"
- **Port:** Select your XIAO's COM/serial port

### 2. Compile and Upload

1. Open `esp32_sk6812_leds.ino` in Arduino IDE
2. Click **Verify** (✓) to compile
3. Click **Upload** (→) to flash

**Expected Output:**
```
Sketch uses 1234567 bytes (37%) of program storage space.
...
Writing at 0x00010000... (100%)
Wrote 1234567 bytes at 0x00010000 in 12.3 seconds
Hard resetting via RTS pin...
```

### 3. Monitor Serial Output

Open **Tools → Serial Monitor** (115200 baud):

```
=================================
ESP32 SK6812 LED Controller
WarehouseCore Warehouse Highlighting
Firmware: v1.2.0
Board: XIAO-ESP32-C6
=================================

[BOARD] Chip: ESP32-C6
[BOARD] Cores: 1
[BOARD] CPU Freq: 160 MHz
[BOARD] Flash: 4096 KB
[BOARD] Free Heap: 385 KB
[BOARD] SDK: v5.1.0

[ID] Controller ID: esp-a1b2c3
[ID] Topic suffix: esp-a1b2c3
[ID] Warehouse ID: WDL

[MQTT] Broker: mqtt.example.com:1883
[MQTT] User: leduser
[MQTT] Topic prefix: weidelbach
[MQTT] Command topic: weidelbach/esp-a1b2c3/cmd
[MQTT] Status topic: weidelbach/esp-a1b2c3/status

[LED] Pin: GPIO 0
[LED] Count: 60 pixels
[LED] Strip initialized

[WiFi] Connecting to: YourWiFi
[WiFi] Connected!
[WiFi] IP: 192.168.1.123
[WiFi] RSSI: -45 dBm
[WiFi] MAC (eFuse): a1b2c3d4e5f6

[MQTT] Connecting to mqtt.example.com:1883 as 'leduser'...
[MQTT] ✓ Connected successfully!
[MQTT] ✓ Subscribed to: weidelbach/esp-a1b2c3/cmd
[MQTT] Sending initial heartbeat...
[HEARTBEAT] ✓ Sent successfully (uptime: 5 s, RSSI: -45 dBm)

[INIT] Setup complete, entering main loop
```

---

## Troubleshooting

### ❌ MQTT Connection Failed (Error -2 or -4)

**Problem:** MQTT broker not reachable

**Solutions:**
1. Verify MQTT_HOST is the **external** IP/domain (not "mosquitto")
2. Check server firewall: `sudo ufw status`
3. Test connectivity: `ping mqtt.example.com`
4. Ensure port 1883 is open on your router (port forwarding if needed)

### ❌ MQTT Error 4: Bad Credentials

**Problem:** Incorrect username/password

**Solutions:**
1. Check MQTT_USER and MQTT_PASS in secrets.h
2. Verify credentials match WarehouseCore `.env` (leduser / ledpassword123)
3. Check Mosquitto password file inside Docker container

### ❌ LEDs Not Lighting

**Problem:** Wrong GPIO pin or wiring issue

**Solutions:**
1. Verify LED_PIN matches your wiring (default GPIO 0 for XIAO)
2. Check LED strip power supply (SK6812 needs 5V)
3. Check data line connection (LED strip DIN → GPIO 0)
4. Try a different GPIO pin: uncomment `#define LED_PIN 1` in secrets.h

### ❌ ESP Not Appearing in Admin Panel

**Problem:** Heartbeats not being received by WarehouseCore

**Solutions:**
1. Check Serial Monitor for heartbeat confirmations
2. Verify TOPIC_PREFIX and WAREHOUSE_ID match WarehouseCore `.env`
3. Check WarehouseCore logs: `docker-compose logs -f warehousecore`
4. Manually subscribe to MQTT topic to verify messages:
   ```bash
   mosquitto_sub -h mqtt.example.com -p 1883 -u leduser -P ledpassword123 \
     -t 'weidelbach/+/status' -v
   ```

### ❌ Watchdog Reset Loop

**Problem:** ESP keeps restarting

**Solutions:**
1. Check WiFi credentials (failed WiFi triggers restart)
2. Increase power supply capacity (brownout detection)
3. Update Arduino-ESP32 core to latest 3.x version
4. Check Serial Monitor for error messages before reset

---

## Verify Registration in WarehouseCore

### Check via Admin Panel

1. Login to WarehouseCore
2. Navigate to **Admin → LED Controllers**
3. Your ESP should appear with:
   - Controller ID: `esp-xxxxxx` (MAC-based)
   - Status: Online
   - Last Seen: Recent timestamp
   - IP Address, RSSI, Firmware Version

### Check via MQTT

Subscribe to status messages:
```bash
mosquitto_sub -h mqtt.example.com -p 1883 \
  -u leduser -P ledpassword123 \
  -t 'weidelbach/+/status' -v
```

You should see heartbeats every 15 seconds:
```json
{
  "status": "online",
  "controller_id": "esp-a1b2c3",
  "topic_suffix": "esp-a1b2c3",
  "warehouse_id": "WDL",
  "active_leds": 0,
  "wifi_rssi": -45,
  "uptime_seconds": 123,
  "ip_address": "192.168.1.123",
  "hostname": "ESP32-esp-a1b2c3",
  "firmware_version": "1.2.0",
  "mac_address": "a1b2c3d4e5f6",
  "led_count": 60
}
```

---

## Hardware Wiring

### SK6812 GRBW LED Strip

```
ESP32-C6 XIAO          SK6812 Strip
┌─────────────┐        ┌─────────┐
│             │        │         │
│ 5V (VBUS) ──┼────────┼─ VCC   │
│             │        │         │
│ GND ────────┼────────┼─ GND   │
│             │        │         │
│ D0 (GPIO0) ─┼────────┼─ DIN   │
│             │        │         │
└─────────────┘        └─────────┘
```

**Important:**
- Use a **5V power supply** adequate for your LED count (60mA per LED @ full white)
- Connect ESP GND to power supply GND (common ground)
- For 60 LEDs: 3.6A @ 5V recommended
- For 600 LEDs: 36A @ 5V required (use external PSU!)

---

## Next Steps

After successful flashing and registration:

1. **Assign to Zone Types** (Admin Panel → LED Controllers → Edit)
2. **Test LEDs** using the "Identify" button in admin panel
3. **Configure LED Mapping** for specific bins/shelves
4. **Deploy additional ESPs** using the same firmware (each gets unique MAC-based ID)

---

## Firmware Updates

To update firmware on already-deployed ESPs:

1. Make changes to `esp32_sk6812_leds.ino`
2. Increment `FIRMWARE_VERSION` (e.g., "1.2.0" → "1.3.0")
3. Compile and upload via USB
4. Verify new version in Serial Monitor and Admin Panel

---

## Support

For issues or questions:
- Check WarehouseCore logs: `docker-compose logs -f`
- Monitor MQTT traffic: `mosquitto_sub -h HOST -u USER -P PASS -t '#' -v`
- Review ESP Serial Monitor output at 115200 baud
- Ensure Arduino-ESP32 core is version 3.0.0+

**Hardware Specs:** [XIAO ESP32-C6 Wiki](https://wiki.seeedstudio.com/xiao_esp32c6_getting_started/)
