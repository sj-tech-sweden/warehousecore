# MQTT Retained Messages Fix

## Problem

ESP32-Controller sendeten Heartbeats mit dem MQTT `retained=true` Flag. Dies führte dazu, dass:

1. **Der MQTT-Broker alle Heartbeats dauerhaft speicherte**
2. **Alte Controller-Daten wurden immer wieder ausgeliefert**, selbst wenn die ESPs längst offline waren
3. **Das Admin-Dashboard zeigte Controller an, die gar nicht mehr existierten**

## Ursache

In der ESP32-Firmware v1.5.0 und früher:

```cpp
// firmware/esp32_sk6812_leds/esp32_sk6812_leds.ino:663
mqttClient.publish(statusTopic.c_str(), payload.c_str(), true);  // ❌ retained=true
```

Das dritte Argument `true` ist das **retained flag**. Der MQTT-Broker speichert solche Messages persistent und liefert sie bei jedem neuen Subscriber sofort aus.

## Lösung

### 1. Firmware-Fix (v1.5.1)

**Geändert:**
```cpp
mqttClient.publish(statusTopic.c_str(), payload.c_str(), false);  // ✅ retained=false
```

**Status:** ✅ Fixed in ESP32 Firmware v1.5.1

### 2. Cleanup-Script für bestehende retained messages

Zwei Scripts stehen zur Verfügung:

#### Option A: Bash-Script (empfohlen)

```bash
cd /opt/dev/cores/warehousecore
./scripts/cleanup_mqtt_retained.sh
```

**Voraussetzungen:** `mosquitto_pub` und `mosquitto_sub` müssen installiert sein

#### Option B: Python-Script

```bash
cd /opt/dev/cores/warehousecore
pip3 install paho-mqtt
python3 scripts/cleanup_mqtt_retained.py
```

**Voraussetzungen:** Python 3 und `paho-mqtt` package

### 3. Docker-basierte Cleanup (wenn Scripts nicht lokal laufen)

Wenn die mosquitto-Tools nicht lokal verfügbar sind, kannst du das Script im mosquitto-Container ausführen:

```bash
# In den mosquitto Container gehen
docker exec -it mosquitto sh

# Im Container:
mosquitto_sub -h localhost -u leduser -P ledpassword123 -t 'weidelbach/+/status' -v -R -W 5 | while read topic message; do
    echo "Clearing $topic"
    mosquitto_pub -h localhost -u leduser -P ledpassword123 -t "$topic" -n -r
done

exit
```

## Deployment-Schritte

### Schritt 1: Cleanup durchführen

Wähle eine der obigen Methoden und führe das Cleanup aus:

```bash
cd /opt/dev/cores/warehousecore
./scripts/cleanup_mqtt_retained.sh
```

**Erwartete Ausgabe:**
```
============================================================
MQTT Retained Message Cleanup
============================================================
Broker: localhost:1883
Topic:  weidelbach/+/status
============================================================

🔍 Scanning for retained messages (timeout: 5 seconds)...

  Found retained: weidelbach/esp-abc123/status
  Found retained: weidelbach/esp-def456/status
  Found retained: weidelbach/esp-ghi789/status

🧹 Cleaning up 3 retained message(s)...

  ✓ Cleared: weidelbach/esp-abc123/status
  ✓ Cleared: weidelbach/esp-def456/status
  ✓ Cleared: weidelbach/esp-ghi789/status

✅ Cleanup complete! Removed 3 retained message(s).
```

### Schritt 2: ESP32-Firmware updaten

Flashe die neue Firmware v1.5.1 auf alle ESP32-Controller:

```bash
# Firmware-Datei:
/opt/dev/cores/warehousecore/firmware/esp32_sk6812_leds/esp32_sk6812_leds.ino
```

**Wichtig:** Ohne Firmware-Update werden weiterhin retained messages erstellt!

### Schritt 3: Admin-Dashboard überprüfen

1. Öffne das Admin-Dashboard → "Mikrocontroller verwalten"
2. Nur noch aktive Controller sollten angezeigt werden
3. Offline-Controller verschwinden nach 5 Minuten automatisch

Optional: Nutze den "Offline löschen" Button, um alle inaktiven Controller auf einmal zu entfernen.

## Verifikation

Nach dem Cleanup sollten keine retained messages mehr vorhanden sein:

```bash
# Test: Nur aktive Heartbeats empfangen (keine alten retained messages)
timeout 10s mosquitto_sub -h localhost -u leduser -P ledpassword123 -t 'weidelbach/+/status' -v -R
```

**Erwartung:** Keine Ausgabe, oder nur Messages von aktuell online ESPs.

## Langfristige Lösung

- ✅ ESP32 Firmware v1.5.1+ sendet keine retained messages mehr
- ✅ Cleanup-Scripts entfernen bestehende retained messages
- ✅ Admin-Dashboard zeigt "Offline löschen" Button für manuelle Bereinigung

## Technische Details

### Was ist MQTT Retained?

MQTT retained messages sind persistent:
- Broker speichert die **letzte** Message auf einem Topic
- Neue Subscriber bekommen die retained message **sofort**
- Bleibt gespeichert, auch wenn der Publisher offline geht

### Warum war das ein Problem?

1. ESP32 sendet Heartbeat alle 15 Sekunden mit `retained=true`
2. ESP32 geht offline (z.B. ausgeschaltet, umprogrammiert, zerstört)
3. **Letzte Heartbeat bleibt im Broker gespeichert**
4. Bei Server-Neustart: WarehouseCore subscribt → bekommt alte Heartbeats
5. WarehouseCore erstellt Controller-Eintrag für längst offline ESP32

### Wie löscht man retained messages?

Um eine retained message zu löschen:
```bash
mosquitto_pub -t "topic/path" -n -r
```

- `-n`: null payload (leere message)
- `-r`: retained flag
- Broker ersetzt alte retained message mit leerer → Message wird gelöscht
