# Multi-ESP32 Setup Guide für WarehouseCore

**Komplette Anleitung für die Einrichtung und Verwaltung mehrerer ESP32-Controller im Lager**

---

## 📋 Inhaltsverzeichnis

1. [Übersicht](#übersicht)
2. [Hardware-Anforderungen](#hardware-anforderungen)
3. [Schritt 1: Hardware vorbereiten](#schritt-1-hardware-vorbereiten)
4. [Schritt 2: Arduino IDE einrichten](#schritt-2-arduino-ide-einrichten)
5. [Schritt 3: Firmware konfigurieren](#schritt-3-firmware-konfigurieren)
6. [Schritt 4: Firmware auf ESP32 flashen](#schritt-4-firmware-auf-esp32-flashen)
7. [Schritt 5: ESP32s im Admin-Panel verwalten](#schritt-5-esp32s-im-admin-panel-verwalten)
8. [Schritt 6: Zonentypen zuweisen](#schritt-6-zonentypen-zuweisen)
9. [Schritt 7: System testen](#schritt-7-system-testen)
10. [Troubleshooting](#troubleshooting)

---

## Übersicht

Mit dem Multi-ESP32 System kannst du beliebig viele ESP32-Controller im Lager einsetzen, um verschiedene Bereiche (Regale, Racks, Fahrzeuge, etc.) mit LED-Beleuchtung auszustatten. Jeder Controller:

- 💡 Steuert eigene LED-Streifen (bis zu 600 LEDs pro ESP32)
- 📡 Verbindet sich automatisch mit WiFi und MQTT
- 🔄 Registriert sich selbstständig im System
- 🎯 Leuchtet nur Bereiche auf, die ihm zugewiesen sind
- 📊 Sendet Telemetriedaten (Online-Status, WiFi-Stärke, Uptime)

**Wichtig:** Dieselbe Firmware kann auf alle ESP32s geflasht werden! Jeder Controller generiert automatisch eine eindeutige ID basierend auf seiner MAC-Adresse.

---

## Hardware-Anforderungen

### Pro ESP32-Controller brauchst du:

#### Elektronik:
- ✅ **ESP32 Development Board** (z.B. ESP32-DevKitC, ESP32-WROOM-32)
  - Preis: ~5-10€
  - Bezugsquelle: Amazon, AliExpress, Reichelt, Conrad

- ✅ **SK6812 GRBW LED-Streifen** (adressierbar, mit weißem Kanal)
  - Länge: je nach Regal (z.B. 5m = 300 LEDs @ 60 LEDs/m)
  - Preis: ~15-30€ pro 5m
  - Bezugsquelle: Amazon, AliExpress, Reichelt
  - Alternative: WS2812B (RGB ohne Weiß)

- ✅ **5V Netzteil** (ausreichend Ampere!)
  - Berechnung: 300 LEDs × 60mA (bei 100% Helligkeit) = 18A
  - Typisch (50% Helligkeit): ~10A ausreichend
  - Empfehlung: 5V 10A Netzteil (50W)
  - Preis: ~15-25€

- ✅ **Level Shifter** (3.3V → 5V)
  - Typ: 74HCT245 oder ähnlich
  - Preis: ~2-5€
  - Wichtig: ESP32 hat 3.3V Logik, LED-Strip braucht 5V Datensignal

- ✅ **Kondensator** 1000µF (6.3V oder höher)
  - Für Spannungsstabilisierung
  - Preis: ~1€

#### Kabel & Zubehör:
- Jumper-Kabel (Breadboard-Kabel)
- USB-Kabel (für ESP32 Programmierung)
- Stromkabel & Lüsterklemmen
- Optional: Gehäuse für ESP32

---

## Schritt 1: Hardware vorbereiten

### 1.1 Verkabelung

**Wichtig: Alle Komponenten müssen eine gemeinsame Masse (GND) haben!**

```
┌──────────────┐
│   ESP32      │
│              │
│  GPIO 5  ────┼──────┐
│  GND     ────┼───┐  │
│  3.3V    ────┼─┐ │  │
└──────────────┘ │ │  │
                 │ │  │
         ┌───────┘ │  │
         │         │  │
    ┌────▼────┐    │  │
    │ Level   │    │  │
    │ Shifter │    │  │
    │ 74HCT245│    │  │
    │         │    │  │
    │  VCC────┼────┘  │        ┌─────────────┐
    │  GND────┼───────┼────────│ GND         │
    │  IN ────┼───────┘        │             │
    │  OUT────┼────────────────│ DIN         │
    └─────────┘                │             │
                               │ SK6812 GRBW │
    ┌──────────┐               │ LED Strip   │
    │ 5V 10A   │               │             │
    │ Netzteil │───────────────│ 5V+         │
    │          │───────────────│ GND         │
    └──────────┘               └─────────────┘
```

**Verbindungen:**

| Von | Nach | Beschreibung |
|-----|------|--------------|
| ESP32 GPIO 5 | Level Shifter IN | Datensignal (3.3V) |
| ESP32 GND | Level Shifter GND + LED GND + Netzteil GND | **Gemeinsame Masse!** |
| ESP32 3.3V | Level Shifter VCC | Versorgung für Shifter |
| Level Shifter OUT | LED Strip DIN | Datensignal (5V) |
| Netzteil 5V+ | LED Strip 5V+ | Stromversorgung |
| Netzteil GND | LED Strip GND + ESP32 GND | **Gemeinsame Masse!** |

### 1.2 Kondensator platzieren

- Kondensator (1000µF) zwischen 5V+ und GND des LED-Strips löten
- So nah wie möglich am LED-Strip Anfang
- Polarität beachten! (Minus-Pol an GND)

### 1.3 Power Injection bei langen Strips

Bei Strips > 150 LEDs zusätzlich Strom einspeiesen:
- Alle 100-150 LEDs 5V+ und GND zum Strip hinzufügen
- Verhindert Spannungsabfall und Farbverfälschungen

---

## Schritt 2: Arduino IDE einrichten

### 2.1 Arduino IDE installieren

1. Download von [https://www.arduino.cc/en/software](https://www.arduino.cc/en/software)
2. Version 2.x empfohlen
3. Installation durchführen

### 2.2 ESP32 Board Support hinzufügen

1. Arduino IDE öffnen
2. **Datei** → **Voreinstellungen**
3. Bei "Zusätzliche Boardverwalter-URLs" eintragen:
   ```
   https://dl.espressif.com/dl/package_esp32_index.json
   ```
4. Klick **OK**
5. **Werkzeuge** → **Board** → **Boardverwalter**
6. Suche nach "**esp32**"
7. Installiere "**ESP32 by Espressif Systems**" (Version 3.x)
8. Warte auf Download & Installation

### 2.3 Benötigte Bibliotheken installieren

1. **Sketch** → **Bibliothek einbinden** → **Bibliotheken verwalten**
2. Installiere folgende Bibliotheken:

| Bibliothek | Autor | Version |
|------------|-------|---------|
| **PubSubClient** | Nick O'Leary | neueste |
| **ArduinoJson** | Benoit Blanchon | 6.x |
| **Adafruit NeoPixel** | Adafruit | neueste |

3. Bei jeder Bibliothek: Suchen → **Installieren**

---

## Schritt 3: Firmware konfigurieren

### 3.1 Firmware herunterladen

Die Firmware liegt bereits im Repository:

```bash
cd /opt/dev/lager_weidelbach/warehousecore/firmware/esp32_sk6812_leds/
ls -la
```

Du solltest sehen:
- `esp32_sk6812_leds.ino` - Hauptprogramm
- `secrets.h.template` - Konfigurationsvorlage

### 3.2 Secrets-Datei erstellen

**WICHTIG:** Die `secrets.h` Datei enthält alle Zugangsdaten und wird NICHT ins Git committed!

```bash
cp secrets.h.template secrets.h
```

### 3.3 Secrets-Datei bearbeiten

Öffne `secrets.h` mit einem Editor:

```bash
nano secrets.h
```

Trage deine Daten ein:

```cpp
// ==========================================
// WiFi-Konfiguration
// ==========================================
#define WIFI_SSID "DeinWiFiName"           // ← ÄNDERN!
#define WIFI_PASS "DeinWiFiPasswort"       // ← ÄNDERN!

// ==========================================
// MQTT Broker-Konfiguration
// ==========================================
#define MQTT_HOST "tsunami-events.de"      // ← ÄNDERN falls anderer Server
#define MQTT_PORT 1883                     // 1883 = ohne TLS, 8883 = mit TLS
#define MQTT_USER "mqtt_user"              // ← ÄNDERN!
#define MQTT_PASS "mqtt_password"          // ← ÄNDERN!

// ==========================================
// API & Warehouse Konfiguration
// ==========================================
#define API_BASE_URL "http://tsunami-events.de:8081/api/v1"  // ← ÄNDERN falls nötig
#define WAREHOUSE_ID "weidelbach"
#define TOPIC_PREFIX "weidelbach"

// ==========================================
// LED-Konfiguration
// ==========================================
#define LED_PIN 5                          // GPIO Pin für LED-Datenleitung
#define LED_LENGTH 300                     // ← ÄNDERN! Anzahl deiner LEDs

// ==========================================
// Controller-Konfiguration (OPTIONAL)
// ==========================================
// Automatische ID-Generierung (Standard):
#define CONTROLLER_ID_PREFIX "esp"         // Präfix für Auto-ID

// Alternativ: Feste ID vergeben (dann auskommentieren):
// #define CONTROLLER_ID "esp-regal-1"

// Optional: Eigenes Topic-Suffix (sonst = controller_id):
// #define TOPIC_SUFFIX "shelf-a"

// ==========================================
// TLS/SSL (OPTIONAL)
// ==========================================
// Für verschlüsselte Verbindung auskommentieren:
// #define USE_TLS 1
// Dann auch MQTT_PORT auf 8883 ändern!
```

**Wichtige Parameter:**

| Parameter | Beschreibung | Beispiel |
|-----------|--------------|----------|
| `WIFI_SSID` | Dein WiFi-Name | "LagerWiFi" |
| `WIFI_PASS` | Dein WiFi-Passwort | "GeheimesPasswort123" |
| `MQTT_HOST` | Server-Adresse | "tsunami-events.de" |
| `MQTT_USER` | MQTT-Benutzername | "warehouse_mqtt" |
| `MQTT_PASS` | MQTT-Passwort | "mqtt_secret_pw" |
| `API_BASE_URL` | WarehouseCore API URL | "http://server:8081/api/v1" |
| `LED_LENGTH` | Anzahl LEDs am Strip | 300 (für 5m @ 60 LEDs/m) |

> Hinweis: `API_BASE_URL` muss genau den aus dem ESP32-Netz erreichbaren Basis-Pfad deiner WarehouseCore-Instanz enthalten – inklusive `/api/v1`. Verwende HTTP oder HTTPS passend zu deiner Installation (z. B. `https://warehousecore.example.com/api/v1` für Produktion oder `http://192.168.10.5:8081/api/v1` im lokalen Netz). Die Firmware nutzt diesen Wert ausschließlich für die Heartbeat-POSTs und kürzt einen abschließenden Slash automatisch.

Speichern: **STRG+O**, **Enter**, **STRG+X**

---

## Schritt 4: Firmware auf ESP32 flashen

### 4.1 ESP32 vorbereiten

1. **ESP32 per USB-Kabel** an deinen Computer anschließen
2. Warte auf Treiber-Installation (Windows: automatisch)
3. Überprüfe COM-Port:
   - **Windows:** Geräte-Manager → Anschlüsse → "CP210x" oder "CH340"
   - **Linux:** `ls /dev/ttyUSB*` oder `ls /dev/ttyACM*`
   - **Mac:** `ls /dev/cu.usbserial*`

### 4.2 Arduino IDE konfigurieren

1. Öffne `esp32_sk6812_leds.ino` in Arduino IDE
2. **Werkzeuge** → Einstellungen vornehmen:

| Einstellung | Wert |
|-------------|------|
| Board | "ESP32 Dev Module" |
| Upload Speed | 115200 |
| CPU Frequency | 240 MHz |
| Flash Frequency | 80 MHz |
| Flash Mode | QIO |
| Flash Size | 4MB |
| Partition Scheme | Default 4MB |
| Port | Dein COM-Port (z.B. COM3, /dev/ttyUSB0) |

### 4.3 Code kompilieren (Testen)

1. Klick auf **Verify** (✓ Symbol)
2. Warte auf "Kompilierung abgeschlossen"
3. Überprüfe Ausgabe:
   ```
   Sketch verwendet ... Bytes (..%) des Programmspeicherplatzes.
   Global variables use ... bytes (..%) of dynamic memory.
   ```

**Keine Fehler?** → Weiter zu 4.4

**Fehler?** → Siehe [Troubleshooting](#troubleshooting)

### 4.4 Firmware hochladen

1. Klick auf **Upload** (→ Symbol)
2. Warte auf:
   ```
   Connecting........_____.....
   Chip is ESP32-D0WDQ6 (revision 1)
   ...
   Writing at 0x00010000... (100%)
   Wrote ... bytes (... compressed)
   Hash of data verified.

   Leaving...
   Hard resetting via RTS pin...
   ```
3. **Erfolg!** ESP32 startet neu

**Tipp:** Bei Verbindungsproblemen:
- Boot-Button am ESP32 drücken während "Connecting..." angezeigt wird
- USB-Kabel tauschen (manche sind nur zum Laden!)

### 4.5 Serial Monitor öffnen (wichtig!)

1. **Werkzeuge** → **Serieller Monitor**
2. Baudrate auf **115200** stellen
3. Du siehst nun Debug-Ausgaben:

```
[BOOT] WarehouseCore ESP32 LED Controller v1.1.0
[BOOT] MAC: 24:6F:28:A1:B2:C3
[BOOT] Controller ID: esp-a1b2c3
[BOOT] Topic Suffix: esp-a1b2c3

[WiFi] Connecting to 'LagerWiFi'...
[WiFi] Connected! IP: 192.168.1.45
[WiFi] RSSI: -51 dBm

[MQTT] Connecting to tsunami-events.de:1883...
[MQTT] Connected!
[MQTT] Subscribed to: weidelbach/esp-a1b2c3/cmd

[LED] Strip initialized: 300 pixels
[LED] Test pattern: Rainbow

[HEARTBEAT] Sent (MQTT)
[HEARTBEAT] Sent (HTTP) → 200 OK
```

**✅ Alles grün?** → Controller läuft!

**Notiere dir:**
- Controller ID (z.B. `esp-a1b2c3`)
- IP-Adresse (z.B. `192.168.1.45`)

### 4.6 Weitere ESP32s flashen

Wiederhole Schritte 4.1 - 4.5 für jeden weiteren ESP32:
- **Gleiche Firmware verwenden!**
- **Gleiche `secrets.h` nutzen!**
- Jeder ESP32 generiert automatisch eine **eigene ID** basierend auf seiner MAC-Adresse

**Beispiel:**
- ESP32 #1: MAC `24:6F:28:A1:B2:C3` → ID `esp-a1b2c3`
- ESP32 #2: MAC `24:6F:28:D4:E5:F6` → ID `esp-d4e5f6`
- ESP32 #3: MAC `24:6F:28:11:22:33` → ID `esp-112233`

---

## Schritt 5: ESP32s im Admin-Panel verwalten

### 5.1 Admin-Panel öffnen

1. Öffne WarehouseCore im Browser
   ```
   http://tsunami-events.de:8081
   ```
2. Login mit Admin-Account
3. Navigiere zu: **Admin** → **ESP-Controller**

### 5.2 Controller überprüfen

Du siehst nun eine Liste aller ESP32-Controller. **Sobald ein frisch geflashter Controller per MQTT sein Status-Topic (`…/status`) veröffentlicht, legt WarehouseCore automatisch einen Eintrag an – du musst keine IDs mehr per Hand anlegen.** Alles was noch fehlt, ist ein sprechender Anzeigename und die Zuordnung zu den passenden Lagerbereichen.

> Tipp: Du kannst wirklich auf jedem ESP32 denselben build flashen. Die Auto-ID (MAC-basiert) stellt sicher, dass WarehouseCore jeden Controller eindeutig erkennt und ohne manuelle Vorarbeit im Admin-Panel anlegt.

```
┌─────────────────────────────────────────────────────────┐
│ 🟢 esp-a1b2c3                                Online     │
│    ID: esp-a1b2c3 • Topic: esp-a1b2c3                   │
│    Online                                                │
│    Zonenarten: (keine)                                   │
│    Letzter Kontakt: 23.10.2025 18:45:32                │
│    Hostname: esp-a1b2c3 • IP: 192.168.1.45              │
│    MAC: 24:6F:28:A1:B2:C3 • Firmware: 1.1.0             │
│    LEDs: 300 • WiFi RSSI: -51 dBm • Uptime: 2h 15m      │
│                                                          │
│    [Bearbeiten]  [🗑]                                    │
└─────────────────────────────────────────────────────────┘
```

**Status-Indikator:**
- 🟢 **Grün** = Online (Heartbeat < 5 Min)
- ⚫ **Grau** = Offline (Heartbeat > 5 Min)

### 5.3 Display-Namen vergeben

Für bessere Übersicht kannst du jedem Controller einen freundlichen Namen geben:

1. Klick **Bearbeiten** bei einem Controller
2. Ändere **Anzeigename**:
   - Vorher: `esp-a1b2c3`
   - Nachher: `Regal A - Links`
3. Klick **Speichern**

### 5.4 Lagerzonen zuweisen

Jeder Controller darf eine oder mehrere **Lagertypen/Zonenarten** (z. B. Regal, Eurobox, Fahrzeug) bedienen. Die Zuordnung geschieht jetzt bequem über ein Multi-Select-Dropdown:

1. Klicke erneut auf **Bearbeiten**
2. Öffne das Feld **„Zuständige Lagerzonen“**
3. Wähle alle passenden Zonenarten (Tipp: halte `Strg`/`Cmd` zum Mehrfachauswählen)
4. Speichern – fertig!

Im Dashboard erscheinen die gewählten Zonen als kleine Badges, sodass sofort ersichtlich ist, welcher Controller welchen Bereich beleuchtet.

**Beispiel-Namen:**
- `Regal A - Untere Etage`
- `Rack B - LEDs 1-300`
- `Fahrzeug Sprinter`
- `Shelf C - Obergeschoss`

### 5.4 Topic-Suffix anpassen (optional)

Normalerweise = Controller ID. Nur ändern wenn du eigene Topic-Struktur brauchst:

```
Controller ID:   esp-a1b2c3
Topic Suffix:    regal-a-links    ← Custom
MQTT Topic:      weidelbach/regal-a-links/cmd
```

---

## Schritt 6: Zonentypen zuweisen

**Wichtig:** Damit ein ESP32 die richtigen Bereiche beleuchtet, musst du ihm Zonentypen zuweisen!

### 6.1 Zonentypen verstehen

WarehouseCore kennt verschiedene Lagerbereiche (Zonentypen):

| Zonentyp | Beispiel | Beschreibung |
|----------|----------|--------------|
| **shelf** | "Shelf A", "Regal 1" | Feste Regale |
| **rack** | "Rack B", "Gestell 2" | Mobile Racks |
| **vehicle** | "Sprinter", "LKW" | Fahrzeuge |
| **case** | "Case 1", "Flightcase 3" | Transport-Cases |
| **stage** | "Bühne Nord" | Bühnenaufbauten |

### 6.2 Zonentypen erstellen

Falls noch nicht vorhanden:

1. **Admin** → **Zonentypen**
2. Klick **Neuer Zonentyp**
3. Trage ein:
   - **Key:** `shelf` (technischer Name)
   - **Label:** `Regal` (Anzeigename)
   - **Beschreibung:** `Feste Lagerregale`
4. **Speichern**

Erstelle alle benötigten Typen (shelf, rack, vehicle, etc.)

### 6.3 ESP32 zu Zonentypen zuordnen

Jetzt kommt die wichtige Zuordnung:

1. **Admin** → **ESP-Controller**
2. Klick **Bearbeiten** bei "Regal A - Links"
3. Abschnitt **Zugeordnete Zonentypen**:
   - ☑ **Regal** ← Ankreuzen!
   - ☐ Rack
   - ☐ Fahrzeug
   - ☐ Case
   - ☐ Bühne
4. **Speichern**

**Was passiert jetzt?**

Wenn WarehouseCore ein Fach vom Typ "Regal" (shelf) beleuchten soll:
→ Befehl wird an **alle ESP32s** gesendet, die "Regal" zugewiesen haben
→ Nur der ESP32 "Regal A - Links" reagiert!

### 6.4 Beispiel-Setup: Lager mit 3 ESP32s

**Szenario:**
- 2 Regale (Regal A + B)
- 1 Fahrzeug (Sprinter)

**Zuordnung:**

| Controller | Display-Name | Zonentypen | LED-Count |
|------------|--------------|------------|-----------|
| esp-a1b2c3 | Regal A - Komplett | ☑ Regal | 300 |
| esp-d4e5f6 | Regal B - Komplett | ☑ Regal | 450 |
| esp-112233 | Sprinter LED-System | ☑ Fahrzeug | 150 |

**Ablauf beim Job-Highlight:**

1. User wählt Job mit Geräten in "Regal A, Fach 3"
2. WarehouseCore prüft: Zone "Regal A, Fach 3" hat Typ = `shelf`
3. System sendet MQTT-Befehl an **alle** ESP32s mit Zonentyp "Regal"
4. **esp-a1b2c3** und **esp-d4e5f6** empfangen Befehl
5. Beide prüfen ihre LED-Mapping (Fach 3 = Pixel 45-60)
6. **esp-a1b2c3** hat Mapping für "Regal A" → leuchtet Pixel 45-60 auf!
7. **esp-d4e5f6** hat kein Mapping für "Regal A" → ignoriert Befehl

---

## Schritt 7: System testen

### 7.1 Identify-Test (alle LEDs)

Test ob alle ESP32s funktionieren:

1. **Admin** → **LED-Verhalten**
2. Abschnitt **System-Tests**
3. Klick **Alle LEDs testen (Identify)**
4. **Ergebnis:**
   - Alle verbundenen ESP32s leuchten **komplett Weiß** für 3 Sekunden
   - Danach wieder aus

**Funktioniert nicht?** → Siehe [Troubleshooting](#troubleshooting)

### 7.2 Einzelnes Fach testen

Teste ein spezifisches Fach:

1. **Admin** → **LED-Verhalten**
2. Abschnitt **Fach-Test**
3. Eingaben:
   - **Shelf ID:** `A` (Regal-Kennung)
   - **Bin ID:** `A-01` (Fach-Kennung)
4. Klick **Fach testen**
5. **Ergebnis:**
   - LEDs für Fach A-01 leuchten **Orange** (breathe-Effekt)

**Funktioniert nicht?**
- Prüfe LED-Mapping (Admin → LED-Verhalten → Mapping-Editor)
- Siehe [Troubleshooting](#troubleshooting)

### 7.3 Job-Highlight testen

Echter Anwendungsfall:

1. **Jobs** (Hauptmenü)
2. Wähle einen Job mit Geräten
3. Klick **Fächer hervorheben**
4. **Ergebnis:**
   - Alle Fächer mit Geräten für diesen Job leuchten **Orange**
   - Status: "12 Fächer markiert"

### 7.4 Device-Locate testen

Einzelnes Gerät finden:

1. **Geräte** (Hauptmenü)
2. Suche ein Gerät (z.B. "MAC Quantum")
3. Klick **Fach aufleuchten** (💡)
4. **Ergebnis:**
   - Das Fach mit diesem Gerät leuchtet **Orange**

---

## Troubleshooting

### ❌ ESP32 verbindet sich nicht mit WiFi

**Symptom:**
```
[WiFi] Connecting to 'LagerWiFi'...
[WiFi] Connecting...
[WiFi] Connecting...
[WiFi] Failed! Retrying...
```

**Lösungen:**

1. **SSID & Passwort prüfen:**
   - Groß-/Kleinschreibung beachten!
   - Sonderzeichen korrekt escaped?
   - Leerzeichen am Anfang/Ende?

2. **WiFi-Band prüfen:**
   - ESP32 unterstützt **NUR 2.4 GHz**
   - **NICHT** 5 GHz!
   - Router-Einstellung: 2.4 GHz aktiviert?

3. **Signal-Stärke:**
   - ESP32 näher an Access Point bringen
   - RSSI sollte besser als -75 dBm sein

4. **WiFi-Sicherheit:**
   - WPA2-PSK unterstützt
   - WPA3 oft problematisch → auf WPA2 umstellen

5. **MAC-Filter:**
   - Ist MAC-Adresse im Router freigegeben?
   - MAC anzeigen: Serial Monitor → "MAC: 24:6F:28:..."

---

### ❌ MQTT-Verbindung schlägt fehl

**Symptom:**
```
[MQTT] Connecting to tsunami-events.de:1883...
[MQTT] Failed! State: -2
[MQTT] Retrying in 5s...
```

**MQTT-Status-Codes:**

| Code | Bedeutung | Lösung |
|------|-----------|--------|
| -4 | Timeout | Server erreichbar? Port korrekt? |
| -3 | Verbindung verloren | Netzwerkproblem, neustart ESP32 |
| -2 | Verbindung fehlgeschlagen | Host/Port falsch |
| 1 | Protokoll-Version | PubSubClient aktualisieren |
| 2 | Client ID rejected | MQTT_USER/PASS falsch |
| 4 | Bad credentials | Username/Passwort falsch |
| 5 | Not authorized | User hat keine Rechte |

**Lösungen:**

1. **Server erreichbar?**
   ```bash
   ping tsunami-events.de
   telnet tsunami-events.de 1883
   ```

2. **Port korrekt?**
   - Ohne TLS: `1883`
   - Mit TLS: `8883`
   - `secrets.h` prüfen!

3. **Credentials testen:**
   - MQTT Explorer Tool nutzen
   - Gleiche User/Pass wie ESP32 eingeben
   - Verbindung testen

4. **Firewall:**
   - Port 1883 offen?
   - ESP32-IP freigegeben?

---

### ❌ LEDs leuchten nicht

**Symptom:** ESP32 verbunden, aber LEDs bleiben dunkel

**Checkliste:**

1. **Stromversorgung:**
   - ✅ 5V Netzteil angeschlossen?
   - ✅ Netzteil eingeschaltet?
   - ✅ Spannung messen (Multimeter: 4.8-5.2V)
   - ✅ Ampere ausreichend? (300 LEDs × 60mA = 18A!)

2. **Verkabelung:**
   - ✅ Gemeinsame Masse? (GND ESP32 + GND Netzteil + GND LEDs)
   - ✅ Level Shifter korrekt verbunden?
   - ✅ Datenleitung an richtigen GPIO? (Standard: GPIO 5)
   - ✅ DIN am LED-Strip (nicht DOUT!)

3. **LED-Strip:**
   - ✅ Richtung korrekt? (Pfeil auf Strip beachten!)
   - ✅ Erster LED defekt? → 2. LED testen (DIN direkt an 2. LED)

4. **Code:**
   - ✅ `LED_LENGTH` in `secrets.h` korrekt?
   - ✅ Serial Monitor: "LED Strip initialized: 300 pixels"

5. **Test:**
   ```cpp
   // Im Serial Monitor sollte stehen:
   [LED] Strip initialized: 300 pixels
   [LED] Test pattern: Rainbow  ← Sollte Rainbow zeigen!
   ```

---

### ❌ Controller erscheint nicht im Admin-Panel

**Symptom:** ESP32 läuft, aber nicht in Controller-Liste sichtbar

**Lösungen:**

1. **Heartbeat prüfen** (Serial Monitor):
   ```
   [HEARTBEAT] Sent (MQTT)
   [HEARTBEAT] Sent (HTTP) → 200 OK    ← Wichtig!
   ```

2. **HTTP-Heartbeat fehlgeschlagen?**
   ```
   [HEARTBEAT] Sent (HTTP) → Error: -1
   ```
   → `API_BASE_URL` in `secrets.h` prüfen!
   → Format: `http://server:8081/api/v1` (ohne "/" am Ende!)

3. **Auto-Registration aktiviert?**
   - Backend muss unbekannte Controller automatisch anlegen
   - Log prüfen: `docker logs warehousecore`

4. **Manuell anlegen:**
   - Admin → ESP-Controller
   - **Neuer Controller**
   - Controller ID eingeben (z.B. `esp-a1b2c3`)
   - Speichern

---

### ❌ Falsche LEDs leuchten auf

**Symptom:** Fach A-01 soll leuchten, aber A-05 leuchtet

**Problem:** LED-Mapping falsch konfiguriert

**Lösung:**

1. **Admin** → **LED-Verhalten** → **Mapping-Editor**

2. Prüfe Fach-zu-Pixel Zuordnung:
   ```json
   {
     "warehouse_id": "weidelbach",
     "shelves": [
       {
         "shelf_id": "A",
         "bins": [
           {
             "bin_id": "A-01",
             "leds": [0, 1, 2, 3]     ← Stimmen die Pixel?
           },
           {
             "bin_id": "A-02",
             "leds": [4, 5, 6, 7]
           }
         ]
       }
     ]
   }
   ```

3. **Pixel-Nummern herausfinden:**
   - Fach A-01 physisch zählen: LED 0 = erste LED am Strip
   - Mit Testbefehl prüfen (Admin → Fach-Test)
   - Mapping korrigieren

4. **Validieren & Speichern**
   - Klick **Validieren**
   - Keine Fehler? → **Mapping speichern**

---

### ❌ ESP32 stürzt ab / rebootet

**Symptom:**
```
[BOOT] WarehouseCore ESP32 LED Controller v1.1.0
[BOOT] MAC: 24:6F:28:A1:B2:C3
Guru Meditation Error: Core 1 panic'ed (LoadProhibited)
```

**Ursachen & Lösungen:**

1. **Zu viele LEDs:**
   - RAM voll!
   - `LED_LENGTH` reduzieren (max. ~1500 LEDs)

2. **Watchdog Timeout:**
   - Task dauert zu lange
   - In `.ino`: Watchdog-Timeout erhöhen
   ```cpp
   timer = timerBegin(0, 80, true);
   timerAttachInterrupt(timer, &resetModule, true);
   timerAlarmWrite(timer, 15000000, true);  // 10s → 15s
   ```

3. **Stromversorgung ESP32:**
   - ESP32 braucht stabilen 3.3V
   - Kondensator an 3.3V Pin (100µF)
   - Separates 3.3V Netzteil nutzen

4. **JSON-Buffer zu klein:**
   - Große MQTT-Nachrichten
   - In `.ino`: `DynamicJsonDocument` vergrößern

---

### ❌ LEDs flackern / falsche Farben

**Symptom:** LEDs zeigen random Farben, flackern, oder "regnen"

**Ursachen:**

1. **Kein Level Shifter:**
   - ESP32: 3.3V Signal zu schwach für 5V LEDs
   - **Lösung:** 74HCT245 einbauen!

2. **Spannungsabfall:**
   - Bei langen Strips sinkt Spannung
   - **Lösung:** Power Injection (alle 100-150 LEDs 5V+ einspießen)

3. **Keine gemeinsame Masse:**
   - GND nicht verbunden!
   - **Lösung:** ESP32-GND + LED-GND + Netzteil-GND verbinden

4. **Elektromagnetische Störung:**
   - Datenleitung zu nah an Stromkabel
   - **Lösung:** Abstand vergrößern, verdrillte Kabel nutzen

5. **Kondensator fehlt:**
   - Inrush-Current beim Einschalten
   - **Lösung:** 1000µF Kondensator über 5V+/GND

---

## 📚 Weiterführende Dokumentation

- **Firmware Details:** [firmware/esp32_sk6812_leds/README.md](firmware/esp32_sk6812_leds/README.md)
- **LED Mapping:** [LED-Mapping Guide](https://docs.google.com/document/d/LED-Mapping)
- **MQTT Topics:** Siehe `firmware/esp32_sk6812_leds/README.md` → Abschnitt "MQTT Topics"

---

## ✅ Checkliste: ESP32 erfolgreich eingerichtet

- [ ] Hardware verkabelt (Level Shifter, gemeinsame Masse)
- [ ] Arduino IDE installiert + ESP32 Support
- [ ] Bibliotheken installiert (PubSubClient, ArduinoJson, NeoPixel)
- [ ] `secrets.h` erstellt und konfiguriert
- [ ] Firmware kompiliert ohne Fehler
- [ ] Firmware auf ESP32 geflasht
- [ ] Serial Monitor zeigt erfolgreiche WiFi-Verbindung
- [ ] Serial Monitor zeigt erfolgreiche MQTT-Verbindung
- [ ] Serial Monitor zeigt HTTP-Heartbeat: "200 OK"
- [ ] Controller im Admin-Panel sichtbar
- [ ] Display-Name vergeben
- [ ] Zonentypen zugewiesen
- [ ] Identify-Test erfolgreich (alle LEDs leuchten weiß)
- [ ] Fach-Test erfolgreich (richtiges Fach leuchtet)

**Alle Punkte ✅?** → Dein ESP32 ist einsatzbereit! 🎉

---

**Version:** 1.0
**Letzte Aktualisierung:** 23.10.2025
**Autor:** Tsunami Events UG Development Team
