/*
 * ESP32 SK6812 LED Controller for StorageCore
 *
 * Subscribes to MQTT commands from StorageCore server and controls
 * SK6812 GRBW LED strips to highlight storage bins.
 *
 * Features:
 * - WiFi connection with auto-reconnect
 * - MQTT client with TLS support (optional)
 * - JSON command parsing
 * - Multiple LEDs per bin support
 * - Patterns: solid, blink, breathe
 * - Heartbeat status reporting
 * - Watchdog timer for robustness
 */

#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>
#include <Adafruit_NeoPixel.h>

// Include secrets (copy secrets.h.template to secrets.h and fill in your credentials)
#include "secrets.h"

#ifndef CONTROLLER_ID_PREFIX
#define CONTROLLER_ID_PREFIX "esp"
#endif

#ifndef FIRMWARE_VERSION
#define FIRMWARE_VERSION "1.1.0"
#endif

// Pin configuration
#ifndef LED_PIN
#define LED_PIN 5
#endif

#ifndef LED_LENGTH
#define LED_LENGTH 600
#endif

// LED strip setup (SK6812 GRBW)
Adafruit_NeoPixel strip(LED_LENGTH, LED_PIN, NEO_GRBW + NEO_KHZ800);

// WiFi and MQTT clients
#ifdef USE_TLS
WiFiClientSecure espClient;
#else
WiFiClient espClient;
#endif

PubSubClient mqttClient(espClient);

// Controller identity & MQTT topics
String controllerId;
String controllerTopic;
String cmdTopic;
String statusTopic;

// LED state management
struct BinLED {
  int pixel;
  uint32_t color;
  String pattern;
  uint8_t intensity;
  unsigned long lastUpdate;
  uint8_t phaseOffset;
};

std::vector<BinLED> activeLEDs;
bool ledsActive = false;

// Timing
unsigned long lastHeartbeat = 0;
unsigned long lastReconnect = 0;
unsigned long lastCommandTime = 0;
const unsigned long HEARTBEAT_INTERVAL = 15000;
const unsigned long RECONNECT_INTERVAL = 5000;
const unsigned long COMMAND_DEBOUNCE = 500;

// Watchdog
hw_timer_t *watchdogTimer = NULL;

void IRAM_ATTR resetModule() {
  esp_restart();
}

String makeShortMAC() {
  String mac = WiFi.macAddress();
  mac.replace(":", "");
  mac.toLowerCase();
  if (mac.length() > 6) {
    mac = mac.substring(mac.length() - 6);
  }
  return mac;
}

String determineControllerId() {
#ifdef CONTROLLER_ID
  return String(CONTROLLER_ID);
#else
  return String(CONTROLLER_ID_PREFIX) + "-" + makeShortMAC();
#endif
}

String determineTopicSuffix(const String& id) {
#ifdef TOPIC_SUFFIX
  return String(TOPIC_SUFFIX);
#else
  return id;
#endif
}

void sendHTTPHeartbeat(const String& payload);

void setup() {
  Serial.begin(115200);
  delay(1000);

  Serial.println("\n\n=================================");
  Serial.println("ESP32 SK6812 LED Controller");
  Serial.println("StorageCore Warehouse Highlighting");
  Serial.println("=================================\n");

  controllerId = determineControllerId();
  controllerTopic = determineTopicSuffix(controllerId);

  cmdTopic = String(TOPIC_PREFIX) + "/" + controllerTopic + "/cmd";
  statusTopic = String(TOPIC_PREFIX) + "/" + controllerTopic + "/status";

  Serial.printf("[ID] Controller ID: %s\n", controllerId.c_str());
  Serial.printf("[MQTT] Topic suffix: %s\n", controllerTopic.c_str());
  Serial.printf("[MQTT] Command topic: %s\n", cmdTopic.c_str());
  Serial.printf("[MQTT] Status topic: %s\n", statusTopic.c_str());
  Serial.printf("[MQTT] Warehouse ID: %s\n", WAREHOUSE_ID);

  // Initialize LED strip
  strip.begin();
  strip.clear();
  strip.show();
  strip.setBrightness(255);
  Serial.println("[LED] Strip initialized");

  // Connect to WiFi
  connectWiFi();

  // Setup MQTT
  mqttClient.setServer(MQTT_HOST, MQTT_PORT);
  mqttClient.setCallback(mqttCallback);
  mqttClient.setBufferSize(4096); // Increased buffer for JSON commands

  // Setup watchdog (10 seconds)
  // ESP32 Core 3.x API: timerBegin(frequency_in_hz)
  watchdogTimer = timerBegin(1000000); // 1 MHz = 1 tick per microsecond
  timerAttachInterrupt(watchdogTimer, &resetModule);
  timerAlarm(watchdogTimer, 10000000, true, 0); // 10 seconds in microseconds, autoreload=true
  Serial.println("[WDT] Watchdog enabled (10s)");

  // Initial connection
  connectMQTT();

  Serial.println("\n[INIT] Setup complete, entering main loop\n");
}

void loop() {
  // Reset watchdog (ESP32 Core 3.x API)
  timerRestart(watchdogTimer);

  // Maintain connections
  if (!mqttClient.connected()) {
    if (millis() - lastReconnect > RECONNECT_INTERVAL) {
      connectMQTT();
      lastReconnect = millis();
    }
  } else {
    mqttClient.loop();
  }

  // Update LED animations
  if (ledsActive) {
    updateLEDPatterns();
  }

  // Send heartbeat
  if (millis() - lastHeartbeat > HEARTBEAT_INTERVAL) {
    sendHeartbeat();
    lastHeartbeat = millis();
  }

  delay(10);
}

void connectWiFi() {
  Serial.printf("[WiFi] Connecting to: %s\n", WIFI_SSID);
  WiFi.mode(WIFI_STA);
  WiFi.begin(WIFI_SSID, WIFI_PASS);

  int attempts = 0;
  while (WiFi.status() != WL_CONNECTED && attempts < 20) {
    delay(500);
    Serial.print(".");
    attempts++;
  }

  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("\n[WiFi] Connected!");
    Serial.printf("[WiFi] IP: %s\n", WiFi.localIP().toString().c_str());
    Serial.printf("[WiFi] RSSI: %d dBm\n", WiFi.RSSI());
  } else {
    Serial.println("\n[WiFi] Connection failed! Restarting...");
    delay(5000);
    ESP.restart();
  }
}

void connectMQTT() {
  if (WiFi.status() != WL_CONNECTED) {
    return;
  }

  Serial.printf("[MQTT] Connecting to %s:%d\n", MQTT_HOST, MQTT_PORT);

  #ifdef USE_TLS
  espClient.setInsecure(); // For testing; use proper certificates in production
  Serial.println("[MQTT] TLS enabled");
  #endif

  // Set Last Will
  String lwt = "{\"status\":\"offline\",\"controller_id\":\"" + controllerId + "\",\"warehouse_id\":\"" + String(WAREHOUSE_ID) + "\"}";

  String clientId = "ESP32-" + controllerId + "-" + String(random(0xffff), HEX);

  if (mqttClient.connect(clientId.c_str(), MQTT_USER, MQTT_PASS,
                         statusTopic.c_str(), 1, true, lwt.c_str())) {
    Serial.println("[MQTT] Connected!");

    // Subscribe to command topic
    if (mqttClient.subscribe(cmdTopic.c_str(), 1)) {
      Serial.printf("[MQTT] Subscribed to: %s\n", cmdTopic.c_str());
    } else {
      Serial.println("[MQTT] Subscription failed!");
    }

    // Send online status
    sendHeartbeat();
  } else {
    Serial.printf("[MQTT] Connection failed, rc=%d\n", mqttClient.state());
  }
}

void mqttCallback(char* topic, byte* payload, unsigned int length) {
  // Debounce repeated commands
  if (millis() - lastCommandTime < COMMAND_DEBOUNCE) {
    return;
  }
  lastCommandTime = millis();

  Serial.printf("\n[MQTT] Message received on %s (%u bytes)\n", topic, length);

  // Parse JSON
  StaticJsonDocument<4096> doc;
  DeserializationError error = deserializeJson(doc, payload, length);

  if (error) {
    Serial.printf("[JSON] Parse error: %s\n", error.c_str());
    return;
  }

  // Print command for debugging
  serializeJsonPretty(doc, Serial);
  Serial.println();

  // Process command
  const char* op = doc["op"];
  if (!op) {
    Serial.println("[CMD] Missing 'op' field");
    return;
  }

  if (strcmp(op, "highlight") == 0) {
    handleHighlightCommand(doc);
  } else if (strcmp(op, "clear") == 0) {
    handleClearCommand();
  } else if (strcmp(op, "identify") == 0) {
    handleIdentifyCommand();
  } else {
    Serial.printf("[CMD] Unknown operation: %s\n", op);
  }
}

void handleHighlightCommand(JsonDocument& doc) {
  Serial.println("[CMD] Processing HIGHLIGHT command");

  // Clear existing LEDs
  activeLEDs.clear();
  strip.clear();

  // Parse defaults
  uint32_t defaultColor = parseColor(doc["shelves"][0]["bins"][0]["color"] | "#FF2A2A");
  String defaultPattern = doc["shelves"][0]["bins"][0]["pattern"] | "breathe";
  uint8_t defaultIntensity = doc["shelves"][0]["bins"][0]["intensity"] | 180;

  int totalBins = 0;
  int totalPixels = 0;

  // Process shelves
  JsonArray shelves = doc["shelves"];
  for (JsonObject shelf : shelves) {
    String shelfId = shelf["shelf_id"].as<String>();
    JsonArray bins = shelf["bins"];

    for (JsonObject bin : bins) {
      String binId = bin["bin_id"].as<String>();
      JsonArray pixels = bin["pixels"];
      String color = bin["color"] | "#FF2A2A";
      String pattern = bin["pattern"] | defaultPattern;
      uint8_t intensity = bin["intensity"] | defaultIntensity;

      uint32_t ledColor = parseColor(color.c_str());

      // Add each pixel
      for (JsonVariant pixel : pixels) {
        int pixelIndex = pixel.as<int>();
        if (pixelIndex >= 0 && pixelIndex < LED_LENGTH) {
          BinLED led;
          led.pixel = pixelIndex;
          led.color = ledColor;
          led.pattern = pattern;
          led.intensity = intensity;
          led.lastUpdate = millis();
          led.phaseOffset = 0; // Synchronous breathe for all LEDs in same bin
          activeLEDs.push_back(led);
          totalPixels++;
        }
      }
      totalBins++;
    }
  }

  ledsActive = true;
  Serial.printf("[CMD] Highlighted %d bins (%d pixels)\n", totalBins, totalPixels);
}

void handleClearCommand() {
  Serial.println("[CMD] Processing CLEAR command");
  activeLEDs.clear();
  strip.clear();
  strip.show();
  ledsActive = false;
  Serial.println("[CMD] All LEDs cleared");
}

void handleIdentifyCommand() {
  Serial.println("[CMD] Processing IDENTIFY command");

  // Blink all LEDs white 3 times
  for (int i = 0; i < 3; i++) {
    for (int j = 0; j < LED_LENGTH; j++) {
      strip.setPixelColor(j, strip.Color(0, 0, 0, 255));
    }
    strip.show();
    delay(300);
    strip.clear();
    strip.show();
    delay(300);
  }

  Serial.println("[CMD] Identify complete");
}

void updateLEDPatterns() {
  unsigned long now = millis();

  for (auto& led : activeLEDs) {
    uint32_t color = led.color;
    uint8_t r = (color >> 16) & 0xFF;
    uint8_t g = (color >> 8) & 0xFF;
    uint8_t b = color & 0xFF;
    uint8_t brightness = led.intensity;

    if (led.pattern == "solid") {
      // Solid color
      r = (r * brightness) / 255;
      g = (g * brightness) / 255;
      b = (b * brightness) / 255;
      strip.setPixelColor(led.pixel, strip.Color(r, g, b, 0));

    } else if (led.pattern == "blink") {
      // Blink: 500ms on, 500ms off
      if ((now / 500) % 2 == 0) {
        r = (r * brightness) / 255;
        g = (g * brightness) / 255;
        b = (b * brightness) / 255;
        strip.setPixelColor(led.pixel, strip.Color(r, g, b, 0));
      } else {
        strip.setPixelColor(led.pixel, 0);
      }

    } else if (led.pattern == "breathe") {
      // Breathe: sine wave
      float phase = (now / 2000.0) * 2.0 * PI + (led.phaseOffset / 255.0) * 2.0 * PI;
      float intensity = (sin(phase) + 1.0) / 2.0; // 0.0 to 1.0
      uint8_t breatheBrightness = (uint8_t)(intensity * brightness);

      r = (r * breatheBrightness) / 255;
      g = (g * breatheBrightness) / 255;
      b = (b * breatheBrightness) / 255;
      strip.setPixelColor(led.pixel, strip.Color(r, g, b, 0));
    }
  }

  strip.show();
}

uint32_t parseColor(const char* hexColor) {
  if (!hexColor || hexColor[0] != '#') {
    return 0xFF2A2A; // Default red
  }

  long color = strtol(hexColor + 1, NULL, 16);
  return color;
}

void sendHeartbeat() {
  StaticJsonDocument<512> doc;
  doc["status"] = "online";
  doc["controller_id"] = controllerId;
  doc["topic_suffix"] = controllerTopic;
  doc["warehouse_id"] = WAREHOUSE_ID;
  doc["active_leds"] = activeLEDs.size();
  doc["wifi_rssi"] = WiFi.RSSI();
  doc["uptime_seconds"] = millis() / 1000;
  doc["ip_address"] = WiFi.localIP().toString();

  const char* host = WiFi.getHostname();
  if (host != nullptr) {
    doc["hostname"] = host;
  }

  doc["firmware_version"] = FIRMWARE_VERSION;
  doc["mac_address"] = WiFi.macAddress();
  doc["led_count"] = LED_LENGTH;

  String payload;
  serializeJson(doc, payload);

  if (mqttClient.connected()) {
    if (mqttClient.publish(statusTopic.c_str(), payload.c_str(), true)) {
      Serial.printf("[HEARTBEAT] MQTT sent (uptime: %lu s)\n", millis() / 1000);
    } else {
      Serial.println("[HEARTBEAT] MQTT publish failed");
    }
  }

}
