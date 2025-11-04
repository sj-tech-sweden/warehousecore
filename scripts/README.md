# Utility Scripts

## MQTT Retained Message Cleanup

### Problem
ESP32-Controller sendeten Heartbeats mit dem MQTT `retained=true` Flag, was dazu führte, dass alte Controller-Daten im MQTT-Broker gespeichert blieben und immer wieder ausgeliefert wurden, selbst wenn die ESPs längst offline waren.

### Quick Fix

**Auf dem Server mit dem MQTT-Broker:**

```bash
cd /opt/dev/cores/warehousecore
./scripts/cleanup_mqtt_retained.sh
```

**Mit Docker Compose (wenn MQTT in Docker läuft):**

```bash
# Credentials aus .env oder docker-compose.yml übernehmen
export MQTT_USER="leduser"
export MQTT_PASS="ledpassword123"
export MQTT_HOST="localhost"
export TOPIC_PREFIX="weidelbach"

./scripts/cleanup_mqtt_retained.sh
```

**Direkt im mosquitto Container:**

```bash
docker exec -it mosquitto sh -c 'timeout 5 mosquitto_sub -h localhost -u leduser -P ledpassword123 -t "weidelbach/+/status" -v -R | while read topic msg; do mosquitto_pub -h localhost -u leduser -P ledpassword123 -t "$topic" -n -r; echo "Cleared: $topic"; done || true'
```

### Verfügbare Scripts

| Script | Beschreibung | Voraussetzungen |
|--------|--------------|-----------------|
| `cleanup_mqtt_retained.sh` | Bash-basiertes Cleanup | `mosquitto_sub`, `mosquitto_pub` |
| `cleanup_mqtt_retained.py` | Python-basiertes Cleanup | Python 3, `paho-mqtt` |

### Nach dem Cleanup

1. **ESP32-Firmware updaten** auf v1.5.1 oder neuer
2. **Admin-Dashboard überprüfen** - nur noch aktive Controller sollten sichtbar sein
3. **"Offline löschen"** Button nutzen, um verbleibende alte Einträge zu entfernen

### Verifikation

Prüfe, ob noch retained messages vorhanden sind:

```bash
timeout 10s mosquitto_sub -h localhost -u leduser -P ledpassword123 -t 'weidelbach/+/status' -v -R
```

Keine Ausgabe = Erfolgreich! ✅

### Technische Details

Siehe [MQTT_RETAINED_FIX.md](../MQTT_RETAINED_FIX.md) für eine ausführliche Erklärung des Problems und der Lösung.
