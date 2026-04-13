# ESPHome LED Controller for WarehouseCore

ESPHome-based firmware for controlling addressable LED strips in the WarehouseCore warehouse highlighting system. This is a drop-in alternative to the [Arduino firmware](../esp32_sk6812_leds/README.md) that adds Home Assistant integration, OTA updates, and a web-based configuration portal.

## Why ESPHome?

| Feature | Arduino Firmware | ESPHome Firmware |
|---------|-----------------|-----------------|
| WarehouseCore MQTT control | ✅ | ✅ |
| Home Assistant integration | ❌ | ✅ Native entities |
| OTA updates | ❌ | ✅ Via ESPHome dashboard / HA |
| Web configuration portal | ❌ | ✅ Captive portal + fallback AP |
| YAML-based config | ❌ | ✅ No C++ needed |
| Auto-discovery (MQTT) | ✅ | ✅ |
| LED patterns (solid/blink/breathe) | ✅ | ✅ |
| Heartbeat telemetry | ✅ | ✅ |
| Identify / test command | ✅ | ✅ |
| Restart via MQTT | ✅ | ✅ |

## Prerequisites

- [ESPHome](https://esphome.io/guides/installing_esphome.html) installed (`pip install esphome`)
- ESP32 development board (ESP32, ESP32-S2, ESP32-S3, ESP32-C3, ESP32-C6)
- SK6812 GRBW or WS2812B LED strip
- WiFi network reachable by both ESP32 and MQTT broker
- MQTT broker (same as used by WarehouseCore)

## Quick Start

### 1. Copy secrets template

```bash
cd firmware/esphome
cp secrets.yaml.template secrets.yaml
```

### 2. Edit secrets.yaml

Fill in your WiFi, MQTT, and OTA credentials:

```yaml
wifi_ssid: "YourWiFiNetwork"
wifi_password: "YourWiFiPassword"
mqtt_host: "mqtt.example.com"
mqtt_port: 1883
mqtt_user: "leduser"
mqtt_password: "ledpassword123"
ota_password: "a_secure_password"
ap_password: "fallback_password"
```

### 3. Adjust configuration

Edit `warehousecore_led_controller.yaml` and modify the `substitutions` section:

```yaml
substitutions:
  controller_id_prefix: "esp"       # Prefix for auto-generated IDs
  topic_prefix: "weidelbach"        # Must match LED_MQTT_TOPIC_PREFIX
  warehouse_id: "WDL"               # Must match LED_WAREHOUSE_ID
  led_count: "600"                  # Number of LEDs in your strip
  led_pin: "GPIO5"                  # Data pin (GPIO5 for ESP32, GPIO0 for XIAO-C6)
  heartbeat_interval: "15"          # Heartbeat interval in seconds
  firmware_version: "1.0.0"         # Your firmware version
```

> **Note:** The LED chipset (SK6812 GRBW) is configured in the `light` section of the YAML.
> To change chipset, edit the `type` and `variant` under the `neopixelbus` light platform.

### 4. Flash the firmware

```bash
# First-time flash via USB
esphome run warehousecore_led_controller.yaml

# Subsequent updates can be done over-the-air (OTA)
esphome run warehousecore_led_controller.yaml --device <ip_address>
```

### 5. Verify in WarehouseCore

1. Open **Admin → ESP Controllers** in WarehouseCore
2. Your controller should appear with firmware type badge **ESPHome**
3. Assign a friendly display name and zone types

## Board-Specific Configuration

### Standard ESP32 (ESP32-DevKitC, ESP32-WROOM-32)

```yaml
esp32:
  board: esp32dev

substitutions:
  led_pin: "GPIO5"
```

### XIAO ESP32-C6

```yaml
esp32:
  board: esp32-c6-devkitc-1
  framework:
    type: esp-idf

substitutions:
  led_pin: "GPIO0"
```

### ESP32-S3

```yaml
esp32:
  board: esp32-s3-devkitc-1

substitutions:
  led_pin: "GPIO5"
```

### ESP32-C3

```yaml
esp32:
  board: esp32-c3-devkitm-1

substitutions:
  led_pin: "GPIO2"
```

## MQTT Protocol Compatibility

This ESPHome firmware is fully compatible with the existing WarehouseCore MQTT protocol:

### Topics

| Topic | Direction | Description |
|-------|-----------|-------------|
| `<prefix>/<controller_id>/cmd` | Subscribe | Receives commands from WarehouseCore |
| `<prefix>/<controller_id>/status` | Publish | Sends heartbeat telemetry |

### Commands (same as Arduino firmware)

- **highlight** — Light up specific bins with colors and patterns
- **clear** — Turn off all LEDs
- **identify** — Flash all LEDs white (3 blinks) for identification
- **config** — Logged but handled at compile time in ESPHome
- **restart** — Restart the ESP32

### Heartbeat Format

The ESPHome firmware sends heartbeats identical to the Arduino firmware, with an additional `firmware_type` field:

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
  "hostname": "warehousecore-led",
  "firmware_version": "1.0.0",
  "firmware_type": "esphome",
  "mac_address": "246f28a1b2c3",
  "led_count": 600,
  "chipset": "SK6812_GRBW"
}
```

## Home Assistant Integration

When `discovery: true` is set in the MQTT config (the default), the ESPHome controller automatically appears in Home Assistant with:

- **Light entity** — Full control of the LED strip (on/off, brightness, color)
- **WiFi Signal sensor** — Signal strength in dBm
- **Uptime sensor** — How long the controller has been running
- **IP Address sensor** — Current IP address
- **MAC Address sensor** — Hardware MAC address
- **Restart button** — Reboot the controller from HA

> **Note:** When WarehouseCore sends a highlight command, it takes priority. You can use Home Assistant for manual control or automation during non-highlighting periods.

## Troubleshooting

### ESPHome won't connect to MQTT

- Ensure `mqtt_host` in `secrets.yaml` uses the **external** IP/domain, not `localhost` or `mosquitto`
- Check that the MQTT port (default 1883) is accessible from the ESP32's network
- Verify MQTT username/password match your broker configuration

### Controller doesn't appear in WarehouseCore

- Check the ESPHome logs: `esphome logs warehousecore_led_controller.yaml`
- Verify the `topic_prefix` matches `LED_MQTT_TOPIC_PREFIX` in your WarehouseCore `.env`
- Ensure the MQTT broker is the same one WarehouseCore connects to

### LEDs don't respond to highlight commands

- Verify the controller shows as **Online** in WarehouseCore admin panel
- Check that zone types are assigned to the controller
- Review ESPHome logs for any JSON parsing errors

### Fallback AP mode

If WiFi credentials are wrong or the network is unavailable, the ESP32 creates a fallback access point:
- **SSID:** `WC-LED-Fallback`
- **Password:** Value from `ap_password` in `secrets.yaml`
- Connect and navigate to `192.168.4.1` to reconfigure WiFi
