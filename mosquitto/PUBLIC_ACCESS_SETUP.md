# 🌐 Öffentlich erreichbarer MQTT-Server Setup

Diese Anleitung zeigt, wie du deinen Mosquitto MQTT-Server über **mqtt.server-nt.de** von überall erreichbar machst (z.B. für ESP32 im Warehouse, der sich über mobiles Internet verbindet).

---

## ✅ Voraussetzungen

- Server läuft bereits und ist über server-nt.de erreichbar
- Docker und docker-compose sind installiert
- Mosquitto läuft bereits lokal
- Du hast Zugriff auf DNS-Einstellungen (z.B. bei deinem Domain-Provider)

---

## 📋 Schritt 1: DNS-Eintrag erstellen

Erstelle einen **A-Record** oder **CNAME** bei deinem DNS-Provider:

```
Typ:   A
Name:  mqtt.server-nt.de
Wert:  [IP-Adresse deines Servers]
TTL:   3600 (oder Standard)
```

**Alternativ als CNAME:**
```
Typ:   CNAME
Name:  mqtt
Wert:  server-nt.de
```

**Test:**
```bash
# Warte 1-5 Minuten nach DNS-Änderung, dann teste:
ping mqtt.server-nt.de
# Sollte die Server-IP zeigen
```

---

## 🔒 Schritt 2: SSL-Zertifikat besorgen (Let's Encrypt)

### Option A: Mit Certbot (empfohlen)

```bash
# 1. Certbot installieren
sudo apt update
sudo apt install certbot -y

# 2. Zertifikat für mqtt.server-nt.de besorgen
# WICHTIG: Alle Dienste auf Port 80 müssen gestoppt sein!
sudo docker stop $(docker ps -q)  # Optional: Alle Container stoppen
sudo certbot certonly --standalone -d mqtt.server-nt.de

# 3. Container wieder starten
sudo docker start $(docker ps -aq)
```

**Zertifikat wird gespeichert in:**
```
/etc/letsencrypt/live/mqtt.server-nt.de/fullchain.pem
/etc/letsencrypt/live/mqtt.server-nt.de/privkey.pem
/etc/letsencrypt/live/mqtt.server-nt.de/chain.pem
```

### Option B: Bestehendes Wildcard-Zertifikat nutzen

Falls du bereits ein Wildcard-Zertifikat für `*.server-nt.de` hast, kannst du das nutzen.

---

## 📦 Schritt 3: Zertifikat für Mosquitto kopieren

```bash
# 1. Navigiere zum WarehouseCore-Verzeichnis
cd /opt/dev/lager_weidelbach/warehousecore

# 2. Erstelle Zertifikat-Ordner
mkdir -p mosquitto/certs

# 3. Kopiere Zertifikate
sudo cp /etc/letsencrypt/live/mqtt.server-nt.de/fullchain.pem mosquitto/certs/server.crt
sudo cp /etc/letsencrypt/live/mqtt.server-nt.de/privkey.pem mosquitto/certs/server.key
sudo cp /etc/letsencrypt/live/mqtt.server-nt.de/chain.pem mosquitto/certs/ca.crt

# 4. Setze Berechtigungen (wichtig!)
sudo chown -R 1883:1883 mosquitto/certs
chmod 644 mosquitto/certs/server.crt
chmod 644 mosquitto/certs/ca.crt
chmod 600 mosquitto/certs/server.key
```

---

## ⚙️ Schritt 4: Mosquitto für TLS konfigurieren

### 4.1 mosquitto.conf bearbeiten

```bash
nano mosquitto/config/mosquitto.conf
```

**Füge am Ende hinzu (oder kommentiere aus):**

```conf
# TLS Listener (Port 8883)
listener 8883
protocol mqtt
cafile /mosquitto/certs/ca.crt
certfile /mosquitto/certs/server.crt
keyfile /mosquitto/certs/server.key
require_certificate false
use_identity_as_username false

# TLS Version (nur sichere Versionen)
tls_version tlsv1.2
```

**Wichtig:** Lass den Plain-Listener (1883) erstmal drin für lokale Tests!

---

### 4.2 docker-compose.yml anpassen

```bash
nano docker-compose.yml
```

**Füge unter mosquitto → volumes hinzu:**

```yaml
mosquitto:
  image: eclipse-mosquitto:2.0
  container_name: mosquitto
  ports:
    - "1883:1883"   # Plain (lokal)
    - "8883:8883"   # TLS (öffentlich)
    - "9001:9001"   # WebSocket
  volumes:
    - ./mosquitto/config/mosquitto.conf:/mosquitto/config/mosquitto.conf
    - ./mosquitto/docker-entrypoint.sh:/docker-entrypoint.sh
    - ./mosquitto/data:/mosquitto/data
    - ./mosquitto/log:/mosquitto/log
    - ./mosquitto/config:/mosquitto/config
    - ./mosquitto/certs:/mosquitto/certs:ro  # ← Diese Zeile hinzufügen
  environment:
    - MQTT_USER=${LED_MQTT_USER}
    - MQTT_PASS=${LED_MQTT_PASS}
  entrypoint: ["/docker-entrypoint.sh"]
  restart: unless-stopped
  networks:
    - warehousecore-network
  healthcheck:
    test: ["CMD", "mosquitto_sub", "-t", "$$SYS/#", "-C", "1", "-i", "healthcheck", "-W", "3"]
    interval: 30s
    timeout: 5s
    retries: 3
    start_period: 5s
```

---

### 4.3 Mosquitto neu starten

```bash
docker-compose restart mosquitto

# Logs prüfen
docker-compose logs -f mosquitto
```

**Erwartete Log-Ausgabe:**
```
mosquitto version 2.0.22 running
Opening ipv4 listen socket on port 1883.
Opening ipv4 listen socket on port 8883.
```

---

## 🧪 Schritt 5: Testen

### Test 1: Lokal testen (ohne TLS)

```bash
mosquitto_sub -h localhost -p 1883 -t test -u leduser -P ledpassword123
```

### Test 2: TLS testen (von Server aus)

```bash
mosquitto_sub -h mqtt.server-nt.de -p 8883 --cafile /etc/letsencrypt/live/mqtt.server-nt.de/chain.pem -t test -u leduser -P ledpassword123
```

### Test 3: Von außen testen

Von einem anderen Gerät (z.B. deinem PC):

```bash
# Installiere mosquitto-clients
sudo apt install mosquitto-clients

# Teste Verbindung
mosquitto_sub -h mqtt.server-nt.de -p 8883 --capath /etc/ssl/certs -t test -u leduser -P ledpassword123
```

Wenn das funktioniert, ist der Server öffentlich erreichbar! ✅

---

## 📱 Schritt 6: ESP32 konfigurieren

### 6.1 ESP32 secrets.h anpassen

```cpp
// WiFi
#define WIFI_SSID "YourWiFiSSID"
#define WIFI_PASS "YourWiFiPassword"

// MQTT Broker
#define MQTT_HOST "mqtt.server-nt.de"  // ← Deine Domain statt IP
#define MQTT_PORT 8883                  // ← TLS Port
#define MQTT_USER "leduser"
#define MQTT_PASS "ledpassword123"

// Topics & IDs
#define TOPIC_PREFIX "weidelbach"
#define WAREHOUSE_ID "WDL"

// TLS aktivieren
#define USE_TLS 1  // ← Wichtig!
```

### 6.2 ESP32 Firmware flashen

1. Öffne Arduino IDE
2. Datei öffnen: `firmware/esp32_sk6812_leds/esp32_sk6812_leds.ino`
3. Upload auf ESP32
4. Serial Monitor öffnen (115200 baud)

**Erwartete Ausgabe:**
```
[WiFi] Connecting to: YourWiFiSSID
[WiFi] Connected!
[MQTT] Connecting to mqtt.server-nt.de:8883
[MQTT] TLS enabled
[MQTT] Connected!
[MQTT] Subscribed to: weidelbach/WDL/cmd
```

---

## 🛡️ Schritt 7: Firewall verifizieren

### Prüfen, ob Port 8883 offen ist:

```bash
sudo iptables -L -n | grep 8883
```

**Sollte zeigen:**
```
ACCEPT     tcp  --  0.0.0.0/0            0.0.0.0/0            tcp dpt:8883
```

Falls nicht, öffne den Port manuell (Docker macht das normalerweise automatisch).

---

## 🔄 Schritt 8: WarehouseCore anpassen (optional)

Wenn du möchtest, dass WarehouseCore auch über die öffentliche Domain mit MQTT spricht:

```bash
nano .env
```

**Ändere:**
```env
LED_MQTT_HOST=mqtt.server-nt.de  # Statt "mosquitto"
LED_MQTT_PORT=8883
LED_MQTT_TLS=true
```

**Dann:**
```bash
docker-compose restart warehousecore
```

---

## ✅ Fertig!

Dein MQTT-Server ist jetzt von überall erreichbar über:

```
mqtt.server-nt.de:8883 (mit TLS verschlüsselt)
```

### Zusammenfassung:

- ✅ DNS-Eintrag: `mqtt.server-nt.de` → Server-IP
- ✅ SSL-Zertifikat: Let's Encrypt
- ✅ Mosquitto: TLS auf Port 8883 aktiviert
- ✅ Firewall: Port 8883 offen
- ✅ ESP32: Konfiguriert für mqtt.server-nt.de:8883 mit TLS

---

## 🔒 Sicherheitshinweise

1. **Nutze IMMER Port 8883 (TLS)** für öffentlichen Zugriff
2. **Port 1883 (Plain)** nur für lokales Netzwerk
3. **Starke Passwörter** für MQTT-User
4. **Zertifikat erneuern** alle 90 Tage (Let's Encrypt):
   ```bash
   sudo certbot renew
   # Dann Zertifikate neu kopieren und mosquitto neu starten
   ```

---

## 🐛 Troubleshooting

### ESP32 verbindet sich nicht

**Problem:** `SSL handshake failed`

**Lösung 1:** Nutze `setInsecure()` im ESP32-Code (nur für Tests!):
```cpp
espClient.setInsecure(); // Im Setup nach WiFiClientSecure espClient;
```

**Lösung 2:** ESP32 muss der CA vertrauen. Lade das CA-Zertifikat herunter:
```bash
curl https://letsencrypt.org/certs/isrgrootx1.pem.txt > ca.pem
```
Dann im ESP32-Code einbinden.

### Port 8883 nicht erreichbar

```bash
# Von außen testen mit telnet
telnet mqtt.server-nt.de 8883

# Sollte sich verbinden. Wenn nicht:
sudo iptables -A INPUT -p tcp --dport 8883 -j ACCEPT
```

### Zertifikat abgelaufen

Let's Encrypt Zertifikate laufen nach 90 Tagen ab. Erneuere sie:

```bash
sudo certbot renew
sudo cp /etc/letsencrypt/live/mqtt.server-nt.de/* mosquitto/certs/
sudo chown -R 1883:1883 mosquitto/certs
docker-compose restart mosquitto
```

**Automatische Erneuerung einrichten:**
```bash
sudo crontab -e
```
Füge hinzu:
```
0 0 1 * * certbot renew --quiet && cp /etc/letsencrypt/live/mqtt.server-nt.de/fullchain.pem /opt/dev/lager_weidelbach/warehousecore/mosquitto/certs/server.crt && cp /etc/letsencrypt/live/mqtt.server-nt.de/privkey.pem /opt/dev/lager_weidelbach/warehousecore/mosquitto/certs/server.key && docker restart mosquitto
```

---

## 📚 Weitere Informationen

- [Mosquitto TLS Documentation](https://mosquitto.org/man/mosquitto-tls-7.html)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [ESP32 MQTT over TLS](https://github.com/espressif/arduino-esp32/tree/master/libraries/WiFiClientSecure)
