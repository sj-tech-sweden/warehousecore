# ESP32 SK6812 LED Controller Firmware

Firmware for ESP32 microcontroller to control SK6812 GRBW LED strips for warehouse bin highlighting via MQTT.

## Features

- WiFi connectivity with auto-reconnect
- MQTT client with TLS support (optional)
- JSON command parsing from StorageCore
- Multiple LEDs per storage bin
- Animation patterns: solid, blink, breathe
- Automatic controller ID + topic suffix generation (shared firmware for unlimited ESP32s)
- Dual heartbeat (MQTT + REST) with IP/hostname/RSSI/uptime telemetry
- Watchdog timer for robust operation
- Supports SK6812 GRBW LED chipset

## Hardware Requirements

- ESP32 Development Board (e.g., ESP32-DevKitC, ESP32-WROOM-32)
- SK6812 GRBW LED Strip (addressable)
- 5V Power Supply (sufficient amperage for LED count)
- Level shifter (3.3V → 5V) for data line (recommended)
- Capacitor (1000µF, 6.3V+) across power supply

## Hardware Wiring

```
ESP32          SK6812 Strip
-----          ------------
GPIO 5   --->  DIN (via level shifter)
GND      --->  GND (common ground!)
5V PSU   --->  5V+ (power LEDs separately if > 50 LEDs)
```

**Important:**
- Use a level shifter (e.g., 74HCT245) between ESP32 (3.3V) and SK6812 (5V data)
- Connect ESP32 GND to LED strip GND (common ground is critical!)
- For strips > 50 LEDs, use separate 5V power supply with adequate amperage
- Add 1000µF capacitor across power supply near LED strip
- Inject power every 100-150 LEDs for long strips

## Software Setup

### 1. Install Arduino IDE & ESP32 Support

1. Download and install [Arduino IDE](https://www.arduino.cc/en/software)
2. Add ESP32 board support:
   - Open Arduino IDE → File → Preferences
   - Add to "Additional Board Manager URLs":
     ```
     https://dl.espressif.com/dl/package_esp32_index.json
     ```
   - Go to Tools → Board → Boards Manager
   - Search for "esp32" and install "ESP32 by Espressif Systems"
   - **Compatible with ESP32 Core 3.x** (uses new timer API)

### 2. Install Required Libraries

Go to Sketch → Include Library → Manage Libraries, then install:

- **PubSubClient** by Nick O'Leary (MQTT client)
- **ArduinoJson** by Benoit Blanchon (v6.x)
- **Adafruit NeoPixel** (LED strip driver)

### 3. Configure Secrets

1. Copy `secrets.h.template` to `secrets.h`:
   ```bash
   cp secrets.h.template secrets.h
   ```

2. Edit `secrets.h` with your credentials:
   ```cpp
   // WiFi
   #define WIFI_SSID "YourWiFiSSID"
   #define WIFI_PASS "YourWiFiPassword"

   // MQTT Broker
   #define MQTT_HOST "broker.example.com"
   #define MQTT_PORT 8883  // 1883 for non-TLS, 8883 for TLS
   #define MQTT_USER "your_mqtt_user"
   #define MQTT_PASS "your_mqtt_password"

   // Topics & IDs
   #define TOPIC_PREFIX "weidelbach"
   #define WAREHOUSE_ID "weidelbach"

   // LED Configuration
   #define LED_PIN 5
   #define LED_LENGTH 600

   // Optional: Enable TLS
   // #define USE_TLS 1
   ```

**Note:** `secrets.h` is in `.gitignore` - never commit it!

### 4. Upload Firmware

1. Connect ESP32 via USB
2. Select your board:
   - Tools → Board → ESP32 Arduino → ESP32 Dev Module
3. Select COM port: Tools → Port → (your ESP32 port)
4. Click Upload button (→)
5. Monitor serial output: Tools → Serial Monitor (115200 baud)

## MQTT Topics

### Command Topic (Subscribe)
```
{TOPIC_PREFIX}/{TOPIC_SUFFIX}/cmd
Example: weidelbach/esp-a1b2c3/cmd
```

> `TOPIC_SUFFIX` = `controller_id` (auto) unless overridden in `secrets.h`. Each ESP32 listens on its own topic so that WarehouseCore can route zone-specific commands.

Receives JSON commands from WarehouseCore:

**Highlight Command:**
```json
{
  "op": "highlight",
  "warehouse_id": "weidelbach",
  "shelves": [
    {
      "shelf_id": "A",
      "bins": [
        {
          "bin_id": "A-01",
          "pixels": [0, 1, 2, 3],
          "color": "#FF0000",
          "pattern": "breathe",
          "intensity": 180
        }
      ]
    }
  ]
}
```

**Clear Command:**
```json
{
  "op": "clear",
  "warehouse_id": "weidelbach"
}
```

**Identify Command (test all LEDs):**
```json
{
  "op": "identify",
  "warehouse_id": "weidelbach"
}
```

### Status Topic (Publish)
```
{TOPIC_PREFIX}/{TOPIC_SUFFIX}/status
Example: weidelbach/esp-a1b2c3/status
```

Firmware publishes MQTT heartbeat every 15 s **and** mirrors the payload to the WarehouseCore REST API (optional, see below):
```json
{
  "status": "online",
  "controller_id": "esp-a1b2c3",
  "topic_suffix": "esp-a1b2c3",
  "warehouse_id": "weidelbach",
  "active_leds": 12,
  "wifi_rssi": -45,
  "uptime_seconds": 3600,
  "ip_address": "192.168.10.25",
  "hostname": "esp-a1b2c3",
  "firmware_version": "1.1.0",
  "mac_address": "24:6F:28:A1:B2:C3",
  "led_count": 600
}
```

Last Will Testament (offline):
```json
{
  "status": "offline",
  "controller_id": "esp-a1b2c3",
  "warehouse_id": "weidelbach"
}
```

### Controller Heartbeat

Each ESP32 publishes its status exclusively via MQTT to

```
{TOPIC_PREFIX}/{TOPIC_SUFFIX}/status
```

(where `TOPIC_SUFFIX` defaults to `controller_id` unless overridden in `secrets.h`)

The JSON payload matches the example shown above (`status`, `wifi_rssi`, `uptime_seconds`, …). WarehouseCore listens on these topics, automatically creates missing controllers, and updates `last_seen`, `is_active`, IP, firmware version, etc.

## LED Patterns

- **solid**: Constant color at specified intensity
- **blink**: On for 500ms, off for 500ms
- **breathe**: Sine wave brightness variation (2-second cycle)

## Configuration

### Controller Identity & Topics

`secrets.h` controls how the controller identifies itself:

- `CONTROLLER_ID_PREFIX`: Prefix for auto-generated IDs (`esp-<macsuffix>`). The same firmware can thus be flashed to any number of boards.
- Optional `CONTROLLER_ID`: Forces a fixed ID (e.g., `esp-shelf-1`).
- Optional `TOPIC_SUFFIX`: Overrides the topic (default = controller ID). Commands are sent to `{TOPIC_PREFIX}/{TOPIC_SUFFIX}/cmd`.

### LED Strip Settings

Adjust in `secrets.h`:

- `LED_PIN`: GPIO pin for data line (default: 5)
- `LED_LENGTH`: Total number of LEDs (e.g., 600)

### Chipset Support

Current: SK6812 GRBW (NEO_GRBW + NEO_KHZ800)

To change chipset, edit in `.ino` file:
```cpp
Adafruit_NeoPixel strip(LED_LENGTH, LED_PIN, NEO_GRBW + NEO_KHZ800);
```

Options:
- WS2812B: `NEO_GRB + NEO_KHZ800`
- SK6812 RGB: `NEO_GRB + NEO_KHZ800`
- SK6812 GRBW: `NEO_GRBW + NEO_KHZ800`

## Troubleshooting

### LEDs don't light up
- Check power supply (5V, sufficient amperage)
- Verify data pin connection (GPIO 5 by default)
- Check common ground between ESP32 and LED strip
- Use level shifter (3.3V → 5V)
- Try different GPIO pin if pin 5 has issues

### WiFi won't connect
- Verify SSID and password in `secrets.h`
- Check WiFi signal strength (ESP32 should be close to AP)
- ESP32 only supports 2.4GHz WiFi (not 5GHz)
- Check serial monitor for error messages

### MQTT connection fails
- Verify broker hostname, port, username, password
- Check if broker is reachable from ESP32's network
- For TLS: ensure `USE_TLS` is defined and port is 8883
- Check serial monitor for MQTT error codes

### LEDs flicker or show wrong colors
- Insufficient power supply (check amperage)
- Long wires causing voltage drop (shorten data wire)
- No level shifter (3.3V signal too weak for 5V logic)
- Electromagnetic interference (route data wire away from power)

### ESP32 crashes or reboots
- Watchdog timeout (increase timeout value in `timerAlarm()` call)
- Insufficient power to ESP32 (use separate regulated 3.3V)
- Memory issues (reduce `LED_LENGTH` or JSON buffer size)

## Power Consumption

Estimate power requirements:

- SK6812 GRBW per LED at full white: ~80mA
- Example: 100 LEDs × 80mA = 8A at 5V
- At typical 50% brightness: ~4A at 5V

**Always use a power supply with 20% overhead!**

## Performance

- Maximum LED count: Limited by RAM (~2000 LEDs typical)
- Update rate: ~50-100 Hz depending on LED count
- MQTT reconnect: Automatic with 5-second retry
- Watchdog timeout: 10 seconds

## Security Notes

- Use TLS for production deployments (`#define USE_TLS`)
- Use strong MQTT passwords
- Consider certificate pinning for TLS
- Never commit `secrets.h` to version control

## License

Part of StorageCore project. Internal use only.
