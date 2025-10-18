# StorageCore

**Physical Warehouse Management System for Tsunami Events UG**

StorageCore is the digital twin of the Weidelbach warehouse, providing real-time tracking of devices, cases, zones, and movements with barcode/QR scan-driven workflows.

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [LED Highlighting System](#led-highlighting-system)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
- [API Documentation](#api-documentation)
- [Database Schema](#database-schema)
- [Deployment](#deployment)
- [Development](#development)
- [Docker Hub](#docker-hub)

---

## Overview

StorageCore manages the physical warehouse operations for Tsunami Events, synchronizing in real-time with RentalCore (job management system). It provides:

- **Digital warehouse mapping** - Zones, shelves, racks, vehicles, cases
- **Real-time device status** - in_storage | on_job | defective | repair
- **Scan-driven workflows** - Barcode/QR intake, outtake, transfer
- **Full audit trail** - Every movement and scan logged
- **Maintenance tracking** - Defects, repairs, inspection schedules
- **Job synchronization** - Live sync with RentalCore jobs

---

## Features

### Core Modules

1. **Device Tracker**
   - Scan devices in/out of warehouse
   - Track current location (zone, case, job)
   - Status management with history
   - Movement audit trail

2. **Storage Zones**
   - Hierarchical zone structure
   - Shelf, rack, case, vehicle, stage types
   - Capacity tracking
   - Active/inactive management

3. **Scan System**
   - Barcode and QR code support
   - Intake/outtake/transfer/check actions
   - Duplicate scan detection
   - Complete event logging with IP/user-agent

4. **Job Integration**
   - Real-time job assignment
   - Device packing status
   - Missing item detection
   - Job completion workflow

5. **Maintenance Engine**
   - Defect reporting with severity
   - Repair tracking with costs
   - Inspection scheduling
   - Status workflow (open → in_progress → repaired → closed)

6. **LED Highlighting System** 🎨 NEW
   - Physical warehouse bin highlighting via LED strips
   - Real-time MQTT communication with ESP32 controllers
   - Job-based automatic bin illumination
   - Multiple LEDs per storage bin support
   - Animation patterns (solid, blink, breathe)
   - Remote management and testing
   - Works worldwide with cloud-based MQTT broker

---

## LED Highlighting System

### Overview

The LED Highlighting System provides physical visual guidance in the warehouse by illuminating storage bins containing devices needed for a specific job. When a job is selected in StorageCore, the system automatically highlights the corresponding warehouse locations using addressable LED strips controlled by ESP32 microcontrollers.

### Architecture Diagram

**Option A: Self-Hosted (Single Server)**

```
┌──────────────────────────────────────────────────────────────┐
│                    Docker Compose Stack                      │
│  ┌─────────────────┐         ┌──────────────────┐           │
│  │  StorageCore    │         │  Mosquitto MQTT  │           │
│  │  Container      │───────→ │  Container       │           │      ┌─────────────────┐
│  │                 │  Pub    │                  │←──────────┼──────│   ESP32 + LEDs  │
│  │  - Job Manager  │         │  Port 1883/8883  │  Sub      │      │  (Warehouse)    │
│  │  - LED Service  │         │  weidelbach/cmd  │           │      │                 │
│  │  - Mapping Cfg  │         │  weidelbach/sts  │           │      │  - WiFi Client  │
│  └─────────────────┘         └──────────────────┘           │      │  - MQTT Sub     │
│       Port 8081                   Auth required             │      │  - SK6812 Driver│
└──────────────────────────────────────────────────────────────┘      └─────────────────┘
         Server (your-server.example.com)                              Warehouse Network

Flow: Job Selected → LED Service publishes to Mosquitto (same host)
      → ESP32 connects to server:1883 → Receives command → Lights up bins
```

**Option B: Cloud-Hosted (Distributed)**

```
┌─────────────────┐         ┌──────────────────┐         ┌─────────────────┐
│  StorageCore    │         │  MQTT Broker     │         │   ESP32 + LEDs  │
│  (Cloud/VPS)    │───────→ │  (Cloud/TLS)     │←─────── │  (Warehouse)    │
│                 │  Pub    │                  │  Sub    │                 │
│  - Job Manager  │         │  Topics:         │         │  - WiFi Client  │
│  - LED Service  │         │  weidelbach/cmd  │         │  - MQTT Sub     │
│  - Mapping Cfg  │         │  weidelbach/sts  │         │  - SK6812 Driver│
└─────────────────┘         └──────────────────┘         └─────────────────┘
  Any Server Location        EMQX/HiveMQ/AWS IoT         Behind NAT/Firewall

Flow: Job Selected → Publish to cloud broker → ESP32 subscribes → Show LEDs
```

### Key Features

- **No Port Forwarding Required**: ESP32 uses outbound MQTT connection, works from any network
- **Cloud-Ready**: StorageCore can run on external servers, ESP32 connects via internet
- **Multiple LEDs per Bin**: Support for 2-4 LEDs per storage compartment
- **Flexible Patterns**: Solid, blink, breathe animations
- **Real-Time Control**: Toggle LEDs on/off from job panel
- **Status Monitoring**: MQTT heartbeat shows ESP32 online/offline status
- **Dry-Run Mode**: Backend works without MQTT for testing
- **Admin Mapping Editor**: JSON-based bin-to-LED configuration

### Components

#### 1. Backend (Go)

**Location:** `internal/led/`

- `models.go` - Data structures (LEDCommand, LEDMapping, Bin, Shelf)
- `mqtt_publisher.go` - MQTT client with TLS support, reconnect logic
- `service.go` - Business logic (Job → Bins → Pixels mapping)
- `handlers/led_handlers.go` - REST API endpoints

**Endpoints:**
- `GET /api/v1/led/status` - MQTT connection status
- `POST /api/v1/led/highlight?job_id=X` - Highlight bins for job
- `POST /api/v1/led/clear` - Turn off all LEDs
- `POST /api/v1/led/identify` - Test flash all LEDs
- `POST /api/v1/led/test?shelf_id=A&bin_id=A-01` - Test specific bin
- `GET /api/v1/led/mapping` - Get current mapping config
- `PUT /api/v1/led/mapping` - Update mapping config
- `POST /api/v1/led/mapping/validate` - Validate mapping JSON

#### 2. Frontend (React/TypeScript)

**Location:** `web/src/pages/JobsPage.tsx`, `web/src/lib/api.ts`

- Toggle button in Job Panel: "Fächer hervorheben"
- Visual indicators: MQTT connection status, bin count
- Auto-clear LEDs when exiting job
- Real-time status updates

#### 3. ESP32 Firmware (Arduino C++)

**Location:** `firmware/esp32_sk6812_leds/`

- `esp32_sk6812_leds.ino` - Main firmware
- `secrets.h.template` - Config template (WiFi, MQTT credentials)
- `README.md` - Flash instructions, hardware wiring

**Features:**
- WiFi auto-reconnect
- MQTT client with TLS (optional)
- JSON command parsing (ArduinoJson)
- SK6812 GRBW LED driver (Adafruit_NeoPixel)
- Watchdog timer for stability
- Heartbeat publishing every 15s
- Pattern engine (solid/blink/breathe)

#### 4. Configuration Files

**LED Mapping:** `internal/led/config/led_mapping.json`

Defines which LED indices belong to which storage bins.

**The `bin_id` MUST be the exact `code` from your `storage_zones` database table:**

```json
{
  "warehouse_id": "WDL",
  "shelves": [
    {
      "shelf_id": "Regal-01",
      "bins": [
        { "bin_id": "WDL-RG-01-F-01", "pixels": [0, 1, 2, 3] },
        { "bin_id": "WDL-RG-01-F-02", "pixels": [4, 5, 6] }
      ]
    }
  ],
  "led_strip": {
    "length": 600,
    "data_pin": 5,
    "chipset": "SK6812_GRBW"
  },
  "defaults": {
    "color": "#FF2A2A",
    "pattern": "breathe",
    "intensity": 180
  }
}
```

**How it works:**
1. Job is selected with devices
2. StorageCore looks up each device's `zone_id` in the database
3. Gets the zone's `code` (e.g., "WDL-RG-01-F-01")
4. Matches this code with `bin_id` in the mapping file
5. Sends the corresponding `pixels` to the ESP32 to light up

**JSON Schemas:** `internal/led/schema/led_command.schema.json`, `internal/led/schema/led_mapping.schema.json`

### MQTT Communication

#### Command Topic (Publish from StorageCore)
```
{TOPIC_PREFIX}/{WAREHOUSE_ID}/cmd
Example: weidelbach/weidelbach/cmd
```

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

#### Status Topic (Publish from ESP32)
```
{TOPIC_PREFIX}/{WAREHOUSE_ID}/status
Example: weidelbach/weidelbach/status
```

**Heartbeat:**
```json
{
  "status": "online",
  "warehouse_id": "weidelbach",
  "active_leds": 12,
  "wifi_rssi": -45,
  "uptime": 3600
}
```

### Setup Instructions

#### 1. MQTT Broker Setup

You have two options for MQTT broker setup:

##### **Option A: Self-Hosted Mosquitto (Recommended for Single-Server Deployment)** ✅

If your StorageCore and ESP32 can both connect to the same server (e.g., both on-premises or both can reach your server), use the included Mosquitto container:

**Quick Setup (3 Steps):**

```bash
# 1. Copy the example .env file
cp .env.example .env

# 2. Edit .env with your credentials
nano .env  # Configure DB and MQTT credentials

# 3. Start everything
docker-compose up -d
```

**That's it!** The MQTT password file is **automatically generated** from your `.env` file on container startup.

**Default credentials in `.env.example`:**
- **Username:** `leduser`
- **Password:** `ledpassword123`

**To change MQTT credentials:** Just edit these values in your `.env` file:
```env
LED_MQTT_USER=leduser
LED_MQTT_PASS=your_custom_password
```

Then restart: `docker-compose restart mosquitto`

The password file will be automatically regenerated with the new credentials!

**ESP32 Configuration:**

In your ESP32 `secrets.h`, use the **same credentials** from your `.env` file:

```cpp
#define MQTT_HOST "your-server.example.com"  // Replace with your server's domain or IP
#define MQTT_PORT 1883
#define MQTT_USER "leduser"                   // Same as LED_MQTT_USER in .env
#define MQTT_PASS "ledpassword123"            // Same as LED_MQTT_PASS in .env
```

**⚠️ Production Deployment:**

For production, change the password in your `.env` file:

```env
# In your .env file
LED_MQTT_USER=leduser
LED_MQTT_PASS=your_secure_production_password
```

Then:
1. Restart Mosquitto: `docker-compose restart mosquitto`
2. Update ESP32 `secrets.h` with the same password
3. Re-flash ESP32

**Optional: Enable TLS for production** (port 8883). See `mosquitto/README.md` for:
- Setting up Let's Encrypt certificates
- Configuring TLS in mosquitto.conf
- Updating ESP32 firmware for TLS support

**Advantages:**
- ✅ No external dependencies
- ✅ No subscription fees
- ✅ Full control over configuration
- ✅ Lower latency (local network)
- ✅ Works without internet connection
- ✅ Included in docker-compose

**See `mosquitto/README.md` for detailed setup and troubleshooting.**

##### **Option B: Cloud MQTT Broker (For Distributed Deployments)**

If your StorageCore runs in the cloud and ESP32 is behind NAT/firewall, use a cloud broker:

**Option B1: EMQX Cloud** (easiest)
- Sign up at https://www.emqx.com/en/cloud
- Create free tier deployment
- Note hostname, port (8883 for TLS), username, password

**Option B2: HiveMQ Cloud**
- Free tier available at https://www.hivemq.com/

**Option B3: AWS IoT Core, Azure IoT Hub**
- Enterprise options with advanced features

**Configuration:**

```env
# LED MQTT Configuration - Cloud Broker
LED_MQTT_HOST=your-broker.emqxsl.com
LED_MQTT_PORT=8883
LED_MQTT_TLS=true
LED_MQTT_USER=your_cloud_username
LED_MQTT_PASS=your_cloud_password
```

**Advantages:**
- ✅ Works worldwide with StorageCore and ESP32 on different networks
- ✅ No port forwarding required
- ✅ Managed service (auto-scaling, backups)
- ✅ Built-in monitoring and dashboards

#### 2. StorageCore Backend Configuration

Add to `.env`:

```env
# Option 1: Self-Hosted (docker-compose)
LED_MQTT_HOST=mosquitto
LED_MQTT_PORT=1883
LED_MQTT_TLS=false
LED_MQTT_USER=leduser
LED_MQTT_PASS=your_password

# Option 2: Cloud Broker (EMQX, HiveMQ, etc.)
# LED_MQTT_HOST=your-broker.emqxsl.com
# LED_MQTT_PORT=8883
# LED_MQTT_TLS=true
# LED_MQTT_USER=your_cloud_username
# LED_MQTT_PASS=your_cloud_password

# Common settings
LED_TOPIC_PREFIX=weidelbach
WAREHOUSE_ID=weidelbach
```

**Note:** If `LED_MQTT_HOST` is empty, system runs in DRY-RUN mode (logs commands without sending).

#### 3. LED Mapping Configuration

Edit `internal/led/config/led_mapping.json`:

**IMPORTANT:** The `bin_id` must match the `code` from your `storage_zones` table in the database!

Example:
- Database has zone with `code = "WDL-RG-01-F-01"` (Weidelbach, Regal 01, Fach 01)
- LED Mapping has `"bin_id": "WDL-RG-01-F-01"`
- When a device in that zone is part of a job, the LEDs will light up

1. Set `warehouse_id` to your main warehouse zone code (e.g., `"WDL"`)
2. For each bin, use the exact `code` from `storage_zones` table as `bin_id`
3. Map each bin to LED pixel indices (e.g., `"pixels": [0, 1, 2, 3]`)
4. Support multiple LEDs per bin
5. Set defaults (color, pattern, intensity)
6. Define LED strip parameters (length, pin, chipset)

**Testing Mapping:**
```bash
curl -X POST http://localhost:8081/api/v1/led/mapping/validate \
  -H "Content-Type: application/json" \
  -d @internal/led/config/led_mapping.json
```

#### 4. ESP32 Firmware Flashing

1. Install Arduino IDE + ESP32 support
2. Install libraries: PubSubClient, ArduinoJson, Adafruit_NeoPixel
3. Copy `secrets.h.template` to `secrets.h`
4. Fill in WiFi SSID, password, MQTT credentials
5. Set `LED_PIN` (default 5) and `LED_LENGTH` (e.g., 600)
6. Upload to ESP32
7. Monitor serial output (115200 baud)

See `firmware/esp32_sk6812_leds/README.md` for detailed instructions.

#### 5. Hardware Wiring

```
ESP32 GPIO 5 → Level Shifter (3.3V→5V) → SK6812 DIN
ESP32 GND   → SK6812 GND (common ground!)
5V PSU      → SK6812 5V+ (separate power for LEDs)
```

**Important:**
- Use level shifter for data line
- Common ground between ESP32 and LED strip
- Adequate 5V power supply (calculate: # LEDs × 80mA)
- Capacitor (1000µF) across power supply

### User Workflow

1. **Navigate to Jobs** → Select open job
2. **Click "Fächer hervorheben"** button (Lightbulb icon)
3. **LEDs illuminate** warehouse bins containing job devices
4. **Pick devices** from highlighted bins
5. **Scan each device** to mark as collected
6. **LEDs auto-clear** when navigating away or completing job

### Testing

#### Dry-Run Mode (No MQTT Broker)

Start StorageCore without LED_MQTT_HOST configured:

```bash
# .env
LED_MQTT_HOST=  # Empty = dry-run mode

# Start server
./server

# Test commands (will log JSON without sending)
curl -X POST http://localhost:8081/api/v1/led/highlight?job_id=1
curl -X POST http://localhost:8081/api/v1/led/clear
```

#### With Real MQTT Broker

```bash
# Check status
curl http://localhost:8081/api/v1/led/status

# Expected output:
{
  "mqtt_connected": true,
  "mqtt_dry_run": false,
  "mapping_loaded": true,
  "warehouse_id": "weidelbach",
  "total_shelves": 3,
  "total_bins": 12
}

# Highlight job bins
curl -X POST http://localhost:8081/api/v1/led/highlight?job_id=42

# Test specific bin
curl -X POST "http://localhost:8081/api/v1/led/test?shelf_id=A&bin_id=A-01"

# Clear all
curl -X POST http://localhost:8081/api/v1/led/clear
```

### Troubleshooting

**Problem:** LEDs don't light up
- Check ESP32 serial monitor for connection status
- Verify MQTT credentials match between StorageCore and ESP32
- Ensure ESP32 is online (check status topic)
- Test with `/led/identify` endpoint

**Problem:** Wrong bins highlighted
- Review `internal/led/config/led_mapping.json`
- Check device zone assignments in database
- Validate mapping with `/led/mapping/validate`

**Problem:** MQTT connection fails
- Check broker hostname, port, credentials
- Verify TLS setting matches broker (port 8883 = TLS)
- Test broker with MQTT client (MQTT Explorer, mqttx)
- Check firewall rules on broker

**Problem:** ESP32 crashes/reboots
- Insufficient power supply (check amperage)
- Reduce LED_LENGTH if memory issues
- Increase watchdog timeout in firmware

### Security Considerations

- **TLS**: Always use TLS in production (`LED_MQTT_TLS=true`, port 8883)
- **Strong Passwords**: Use complex MQTT passwords
- **Namespaced Topics**: Include warehouse_id to prevent conflicts
- **Secrets Management**: Never commit `secrets.h` or `.env` files
- **Certificate Pinning**: Consider for production ESP32 deployments

### Performance

- **LED Update Rate**: 50-100 Hz depending on strip length
- **MQTT Latency**: Typical <100ms with cloud broker
- **Reconnect Time**: 5-second retry interval
- **Heartbeat**: Every 15 seconds
- **Max LED Count**: ~2000 LEDs per ESP32 (RAM limited)

### Future Enhancements

- Admin UI for visual mapping editor
- Support for multiple ESP32 controllers
- Zone-based LED groups
- Custom animation patterns
- Mobile app for testing
- Automatic mapping from CAD drawings

---

## Architecture

```
storagecore/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── models/                  # Domain models
│   │   ├── device.go
│   │   ├── zone.go
│   │   ├── movement.go
│   │   ├── scan.go
│   │   ├── maintenance.go
│   │   ├── case.go
│   │   ├── job.go
│   │   └── helpers.go
│   ├── repository/              # Database layer
│   │   └── database.go
│   ├── services/                # Business logic
│   │   └── scan_service.go      # Core scan processing
│   ├── handlers/                # HTTP handlers
│   │   └── handlers.go
│   └── middleware/              # HTTP middleware
│       └── middleware.go
├── migrations/                  # Database migrations
│   ├── 001_storage_zones.sql
│   ├── 002_device_movements.sql
│   ├── 003_scan_events.sql
│   └── 004_defect_reports.sql
├── config/
│   └── config.go                # Configuration management
├── web/                         # Frontend (React + TypeScript + Tailwind)
│   ├── src/
│   ├── public/
│   └── package.json
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

---

## Tech Stack

**Backend:**
- Go 1.24+ (GORM compatibility)
- gorilla/mux (routing)
- GORM (ORM for auth models)
- MySQL 9.2 (shared RentalCore database)
- Session-based authentication
- CORS enabled

**Frontend:**
- React 18+ with TypeScript
- TailwindCSS with Tsunami Events brand theme
- shadcn/ui components
- Dark mode first-class support

**Infrastructure:**
- Docker + Docker Compose
- Docker Hub: `nobentie/storagecore`
- GitLab: git.server-nt.de/ntielmann/storagecore

---

## Getting Started

### Prerequisites

- Go 1.24+
- MySQL 9.2+ (access to RentalCore database)
- Docker (optional, for containerized deployment)
- Node.js 18+ (for frontend development)

### Local Development

1. **Clone the repository**
```bash
git clone https://git.server-nt.de/ntielmann/storagecore.git
cd storagecore
```

2. **Configure environment**
```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. **Run database migrations**
```bash
# Execute SQL files in migrations/ directory against RentalCore database
mysql -h tsunami-events.de -u tsweb -p RentalCore < migrations/001_storage_zones.sql
mysql -h tsunami-events.de -u tsweb -p RentalCore < migrations/002_device_movements.sql
mysql -h tsunami-events.de -u tsweb -p RentalCore < migrations/003_scan_events.sql
mysql -h tsunami-events.de -u tsweb -p RentalCore < migrations/004_defect_reports.sql
```

4. **Install dependencies**
```bash
go mod download
```

5. **Run the server**
```bash
make run
# Or: go run cmd/server/main.go
```

Server runs on `http://localhost:8081`

### API Health Check
```bash
curl http://localhost:8081/api/v1/health
```

---

## API Documentation

### Base URL
`http://localhost:8081/api/v1`

### Endpoints

#### Health
- `GET /health` - Server health check

#### Authentication
- `POST /auth/login` - Login with username and password
- `POST /auth/logout` - Logout and destroy session
- `GET /auth/me` - Get current authenticated user

**Login Request Body:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Login Response:**
```json
{
  "success": true,
  "message": "Login successful",
  "user": {
    "UserID": 1,
    "Username": "admin",
    "Email": "admin@example.com",
    "FirstName": "Admin",
    "LastName": "User",
    "IsActive": true
  }
}
```

#### Scans (CRITICAL)
- `POST /scans` - Process barcode/QR scan
- `GET /scans/history` - Get scan event history

**Scan Request Body:**
```json
{
  "scan_code": "DEVICE123",
  "action": "intake|outtake|check|transfer",
  "job_id": 42,
  "zone_id": 5,
  "notes": "Optional notes"
}
```

**Scan Response:**
```json
{
  "success": true,
  "message": "Device successfully returned to warehouse",
  "device": {...},
  "movement": {...},
  "action": "intake",
  "previous_status": "on_job",
  "new_status": "in_storage",
  "duplicate": false
}
```

#### Devices
- `GET /devices` - List devices (filters: status, zone_id, limit)
- `GET /devices/{id}` - Get device details
- `PUT /devices/{id}/status` - Update device status
- `GET /devices/{id}/movements` - Get device movement history

#### Zones
- `GET /zones` - List all zones
- `POST /zones` - Create new zone
- `GET /zones/{id}` - Get zone details
- `PUT /zones/{id}` - Update zone
- `DELETE /zones/{id}` - Soft-delete zone

#### Jobs
- `GET /jobs` - List active jobs
- `GET /jobs/{id}` - Get job summary with device status
- `POST /jobs/{id}/complete` - Complete job

#### Cases
- `GET /cases` - List all cases
- `GET /cases/{id}` - Get case details
- `GET /cases/{id}/contents` - Get case contents

#### Maintenance
- `GET /defects` - List defect reports (filters: status, severity, device_id)
- `POST /defects` - Create defect report
- `PUT /defects/{id}` - Update defect status, costs, notes
- `GET /maintenance/inspections` - Get inspection schedules (filters: upcoming, overdue, all)
- `GET /maintenance/stats` - Get maintenance dashboard statistics

#### Dashboard
- `GET /dashboard/stats` - Get warehouse statistics
- `GET /movements` - Get recent movements

---

## Database Schema

### New Tables (StorageCore-specific)

**storage_zones** - Logical warehouse areas
- zone_id, code, name, type, description, parent_zone_id, capacity, is_active

**device_movements** - Audit trail of all device movements
- movement_id, device_id, action, from_zone_id, to_zone_id, from_job_id, to_job_id, barcode, user_id, notes, metadata, timestamp

**scan_events** - Complete scan log
- scan_id, scan_code, scan_type, device_id, action, job_id, zone_id, user_id, success, error_message, metadata, ip_address, user_agent, timestamp

**defect_reports** - Detailed defect tracking
- defect_id, device_id, severity, status, title, description, reported_by, reported_at, assigned_to, repaired_by, repaired_at, repair_cost, repair_notes, closed_at, images, metadata

**inspection_schedules** - Periodic inspection requirements
- schedule_id, device_id, product_id, inspection_type, interval_days, last_inspection, next_inspection, is_active

### Existing Tables (from RentalCore)
- devices, cases, devicescases, jobs, jobdevices, products, maintenanceLogs, customers

---

## Deployment

### Docker (Recommended)

**Build Docker image:**
```bash
make docker-build
# Or manually:
docker build -t nobentie/storagecore:1.0 .
docker tag nobentie/storagecore:1.0 nobentie/storagecore:latest
```

**Push to Docker Hub:**
```bash
make docker-push
# Or manually:
docker push nobentie/storagecore:1.0
docker push nobentie/storagecore:latest
```

**Run with docker-compose:**
```bash
# Pull latest image from Docker Hub
docker-compose pull

# Start the service
docker-compose up -d
```

> **Note:** docker-compose.yml is configured to use the `nobentie/storagecore:latest` image from Docker Hub.

**Check logs:**
```bash
docker-compose logs -f storagecore
```

### 🔄 Integrated Deployment with RentalCore

For integrated deployment of both StorageCore and RentalCore together, use the root docker-compose configuration:

```bash
# Navigate to the parent directory (NOT a git repo)
cd /opt/dev/lager_weidelbach

# Pull latest images from Docker Hub
docker compose pull

# Start both services
docker compose up -d

# Check status
docker compose ps

# View logs
docker compose logs -f storagecore
docker compose logs -f rentalcore
```

**Access the applications:**
- **StorageCore**: http://localhost:8082
- **RentalCore**: http://localhost:8081

**Cross-navigation:**
Both applications feature sidebar/navbar links to seamlessly switch between StorageCore and RentalCore with a single click.

**Note:** The images use `:latest` tags. Pull periodically to get the newest versions:
```bash
docker compose pull && docker compose up -d
```

### Production Deployment

1. Ensure database migrations are applied
2. Build and push Docker image with version tag (1.X)
3. Update production docker-compose or k8s manifests
4. Pull and restart container
5. Verify health endpoint

---

## Development

### Running Migrations
Apply new migrations to the shared RentalCore database:
```bash
mysql -h tsunami-events.de -u tsweb -p RentalCore < migrations/XXX_new_feature.sql
```

### Development Workflow
1. Create feature branch
2. Implement changes
3. Test locally
4. Update README
5. Commit (no AI mentions)
6. Push to GitLab
7. Build Docker image with version tag
8. Push to Docker Hub (version + latest)

### Code Quality Rules
- Never claim finished if not 100% working
- Remove temporary/debug files immediately
- No sensitive data in repository
- Update README after every change
- Professional commit messages (no AI references)
- Clean file management (no _final, _new, _fixed suffixes)

---

## Docker Hub

**Repository:** `nobentie/storagecore`

**Tags:**
- `latest` - Latest stable build
- `1.32` - Fixed LED zone codes and MQTT configuration (current)
- `1.31` - Copy LED config files to Docker image and initialize service
- `1.30` - Docker image includes LED configuration directories
- `1.29` - LED mapping uses zone codes from database
- `1.28` - MQTT credentials via .env only, auto-generated password file
- `1.27` - Simplified zero-config MQTT setup with demo credentials
- `1.26` - Self-hosted MQTT broker with Docker Compose
- `1.25` - LED warehouse bin highlighting system
- `1.18` - User authentication and SSO with RentalCore
- `1.17` - Fixed mobile scrolling issues and button overlaps
- `1.16` - Complete mobile responsiveness for all pages
- `1.15` - Complete maintenance module with defect tracking and inspections
- `1.14` - Job-based outtake workflow with live scan tracking
- `1.13` - Recursive device count for parent zones
- `1.12` - Two-step intake workflow with zone barcode scanning
- `1.11` - Fixed SPA routing for page reloads
- `1.10` - Fixed zone devices SQL query + subzone delete buttons
- `1.9` - Fixed race condition in bulk shelf creation
- `1.8` - Automatic shelf creation with barcode generation
- `1.7` - Simplified zone types + delete functionality
- `1.6` - Hierarchical zones with auto code generation
- `1.5` - Clean JSON API responses
- `1.4` - Zone creation fix
- `1.3` - Multi-platform API URL support
- `1.2` - Frontend enhancements
- `1.1` - TailwindCSS fixes
- `1.0` - Initial release

**Pull image:**
```bash
docker pull nobentie/storagecore:latest
```

---

## Environment Variables

```env
# Server
PORT=8081
HOST=0.0.0.0

# Database (Shared with RentalCore)
DB_HOST=tsunami-events.de
DB_USER=tsweb
DB_PASS=<password>
DB_NAME=RentalCore
DB_PORT=3306

# Application
APP_ENV=development|production
LOG_LEVEL=info|debug

# CORS
CORS_ORIGIN=*

# LED MQTT Configuration
LED_MQTT_HOST=your-broker.emqxsl.com  # MQTT broker hostname (empty = dry-run mode)
LED_MQTT_PORT=8883                     # Port (1883 for non-TLS, 8883 for TLS)
LED_MQTT_TLS=true                      # Enable TLS (true|false)
LED_MQTT_USER=your_username            # MQTT username
LED_MQTT_PASS=your_password            # MQTT password
LED_TOPIC_PREFIX=weidelbach            # MQTT topic prefix
WAREHOUSE_ID=weidelbach                # Warehouse identifier
```

---

## License

Proprietary - Tsunami Events UG

---

## Support

For issues or questions:
- GitLab Issues: https://git.server-nt.de/ntielmann/storagecore/issues
- Internal documentation: See /lager_weidelbach/claude.md

---

**Version:** 1.32
**Last Updated:** 2025-10-18
**Maintainer:** Tsunami Events UG Development Team

---

## Changelog

### Version 1.32 (2025-10-18)
- **Bug Fix: LED Zone Code Mapping and MQTT Configuration** 🔧
  - Fixed LED mapping zone codes to include complete hierarchical format
  - Updated bin_id values from `WDL-RG-02-F-XX` to `WDL-06-RG-02-F-XX` format
  - Ensures exact match with storage_zones.code column in database
  - Both shelf_id and bin_id now use proper hierarchical zone codes
- **Port Configuration Fix:**
  - Changed StorageCore port from 8081 to 8082 to avoid conflict with RentalCore
  - Updated docker-compose.yml port mapping: `8082:8082`
  - Updated healthcheck endpoint to use port 8082
  - RentalCore uses 8081, StorageCore uses 8082
- **MQTT Configuration Correction:**
  - Fixed LED_MQTT_HOST environment variable format
  - Changed from full URL (`tcp://mosquitto:1883`) to hostname only (`mosquitto`)
  - Added LED_MQTT_PORT as separate variable (1883)
  - Code expects hostname and builds URL internally
- **Docker Image Updates:**
  - Rebuilt with corrected LED mapping configuration
  - LED config files properly copied to container at build time
  - Service initialization at startup working correctly
- **Testing Results:**
  - ✅ MQTT connection successful: `tcp://mosquitto:1883`
  - ✅ LED mapping loaded: 1 shelf, 5 bins
  - ✅ LED highlight command published successfully for Job 1024
  - ✅ Device MIX2001 correctly mapped to zone WDL-06-RG-02-F-01
  - ✅ MQTT message published to topic: `weidelbach/WDL/cmd`
- **Environment Variables:**
  - PORT changed from 8081 to 8082
  - Added LED_MQTT_HOST=mosquitto (hostname only)
  - Added LED_MQTT_PORT=1883
  - LED configuration now complete in .env file
- **Result:**
  - LED warehouse bin highlighting system fully operational
  - Zone codes match database exactly
  - No more "no bins to highlight" errors
  - MQTT communication working end-to-end

### Version 1.29 (2025-10-17)
- **LED Mapping Now Uses Zone Codes from Database** 🔗
  - `bin_id` in LED mapping file must now match `code` from `storage_zones` table
  - Example: `"bin_id": "WDL-RG-01-F-01"` matches zone code in database
  - `WAREHOUSE_ID` environment variable is now the main warehouse zone code (e.g., "WDL")
  - System automatically looks up device zones by code, not name
- **Improved Database Integration:**
  - Changed query from `z.name` to `z.code` in `getJobDeviceZones()`
  - Zone codes are unique and hierarchical (e.g., WDL-RG-01-F-01)
  - Direct mapping between database zones and LED pixels
  - No manual zone name matching required
- **Updated Configuration Examples:**
  - LED mapping file now shows real zone codes: "WDL-RG-01-F-01", "WDL-RG-02-F-01"
  - `WAREHOUSE_ID=WDL` instead of "weidelbach" in .env files
  - ESP32 secrets.h.template updated with zone code examples
  - Clear documentation on how bin_id must match database codes
- **How It Works:**
  1. Job is selected with devices
  2. StorageCore queries device's `zone_id` from database
  3. Gets the zone's `code` field (e.g., "WDL-RG-01-F-01")
  4. Matches code with `bin_id` in LED mapping configuration
  5. Sends corresponding pixel indices to ESP32
  6. LEDs light up for the correct bins
- **Documentation Updates:**
  - Added clear explanation of zone code requirement
  - Updated all examples to use database zone codes
  - Added workflow diagram showing database → mapping → LEDs
  - Commented environment variables with database reference
- **Benefits:**
  - ✅ Single source of truth (database zone codes)
  - ✅ No manual synchronization between zones and LEDs
  - ✅ Hierarchical zone codes work perfectly
  - ✅ Easy to debug (zone codes are visible in both systems)
  - ✅ Scalable to hundreds of bins

### Version 1.28 (2025-10-17)
- **MQTT Credentials via .env Only - No Scripts Required** 🎯
  - MQTT password file is now **automatically generated** from `.env` on container startup
  - No docker commands, no scripts, no manual password file creation needed
  - Just edit `LED_MQTT_USER` and `LED_MQTT_PASS` in `.env` and restart container
  - Password file regenerates automatically every time the container starts
- **Implementation:**
  - New `mosquitto/docker-entrypoint.sh` script for automatic password generation
  - Reads `MQTT_USER` and `MQTT_PASS` environment variables
  - Generates password file using `mosquitto_passwd` on container startup
  - Mounted as entrypoint in docker-compose.yml
  - Environment variables passed from .env to mosquitto container
- **Removed Complexity:**
  - Deleted `mosquitto/setup-mqtt.sh` (no longer needed)
  - Deleted pre-configured `mosquitto/config/passwords` (auto-generated now)
  - No need for manual docker commands to create passwords
  - Single source of truth: `.env` file
- **User Experience:**
  - Change password: Edit `.env` → Restart container → Done!
  - Same credentials in StorageCore and Mosquitto (both read from .env)
  - ESP32 uses same password (manual sync required for security)
  - Perfect for both development and production
- **Documentation:**
  - Updated README with simplified credential management
  - Updated mosquitto/README.md to emphasize .env-based config
  - Removed references to setup scripts and manual password creation
  - Clear instructions for changing passwords (just edit .env)
- **Developer Benefits:**
  - ✅ Single configuration file (.env) for everything
  - ✅ No need to learn mosquitto_passwd commands
  - ✅ No risk of password file/env mismatch
  - ✅ Password changes are instant (restart container)
  - ✅ Version control friendly (passwords auto-generated, not committed)

### Version 1.27 (2025-10-17)
- **Simplified MQTT Setup - Zero Configuration Required** 🚀
  - Pre-configured password file with demo credentials (leduser/ledpassword123)
  - No setup script required - just `docker-compose up -d` and it works
  - Password file now committed to repository for instant deployment
  - Default credentials in .env.example match mosquitto configuration
  - ESP32 secrets.h.template updated with matching credentials
  - Setup reduced from 4 steps to 3 steps (copy .env, edit DB credentials, start)
- **Documentation Updates:**
  - Simplified Quick Setup instructions in README
  - mosquitto/README.md now emphasizes zero-config approach
  - Custom password setup moved to "Optional" section
  - Clear instructions for both demo and production use
- **Developer Experience:**
  - ✅ Clone repo → copy .env.example → docker-compose up → Done!
  - ✅ No manual password creation needed
  - ✅ Perfect for development and testing
  - ✅ Production users can easily change credentials with documented steps
- **Security Notes:**
  - Demo credentials clearly marked in all files
  - .gitignore updated to allow committing demo password file
  - Production deployment guide includes password change instructions
  - Optional setup script remains available for custom passwords

### Version 1.26 (2025-10-17)
- **Feature: Self-Hosted MQTT Broker with Docker Compose** 🐳
  - Added Mosquitto MQTT broker container to docker-compose.yml
  - StorageCore and ESP32 can now connect to the same server
  - No need for external cloud MQTT broker for single-server deployments
  - Eliminates subscription fees and external dependencies
  - Lower latency with local network communication
- **Mosquitto Configuration:**
  - Pre-configured mosquitto.conf with authentication, logging, persistence
  - Support for plain MQTT (1883), TLS (8883), and WebSocket (9001)
  - Password-based authentication required (no anonymous access)
  - Health checks for container monitoring
  - Volume mounts for config, data, and log persistence
- **Setup Automation:**
  - New setup script: `mosquitto/setup-mqtt.sh`
  - Interactive password creation for MQTT users
  - Automatic directory structure creation
  - Permission management
  - Clear step-by-step instructions
- **Documentation:**
  - Comprehensive `mosquitto/README.md` with setup guide
  - TLS setup instructions with Let's Encrypt integration
  - Firewall configuration guide
  - Monitoring and troubleshooting sections
  - Security best practices
  - ACL configuration examples
  - Backup recommendations
- **Docker Compose Updates:**
  - Added `mosquitto` service with eclipse-mosquitto:2.0 image
  - StorageCore now depends_on mosquitto service
  - Exposed ports: 1883 (MQTT), 8883 (TLS), 9001 (WebSocket)
  - Volume mounts for persistent configuration and data
  - Health check with mosquitto_sub test
- **Environment Configuration:**
  - Updated .env.example with two MQTT options
  - Option 1: Self-hosted (mosquitto container)
  - Option 2: Cloud broker (EMQX, HiveMQ, etc.)
  - Clear documentation for each option
  - Default configuration uses self-hosted option
- **Git Ignore Updates:**
  - Added mosquitto/data/ to prevent committing MQTT persistence
  - Added mosquitto/log/ to exclude log files
  - Added mosquitto/config/passwords to protect credentials
  - Added mosquitto/certs/ for TLS certificate protection
- **Deployment Options:**
  - Simple deployment: Just run `docker-compose up -d`
  - No external account or subscription needed
  - Works offline (no internet required after container pull)
  - Scales from development to production with TLS
- **ESP32 Configuration:**
  - Can connect to server's public IP or domain
  - Same credentials as StorageCore
  - Works from warehouse WiFi to server
  - No port forwarding required (outbound connection)
- **Security Features:**
  - Password file authentication
  - Optional TLS with Let's Encrypt certificates
  - ACL support for topic-level permissions
  - No anonymous access allowed
  - Secure password hashing
- **Production Ready:**
  - Persistent data storage
  - Log rotation support
  - Container health monitoring
  - Auto-restart on failure
  - Compatible with existing LED system
- **User Benefits:**
  - ✅ Zero cost (no subscription)
  - ✅ Full control over MQTT broker
  - ✅ Faster communication (local network)
  - ✅ Works without internet
  - ✅ Single-server deployment
  - ✅ Easy ESP32 configuration (just use server address)
  - ✅ Professional monitoring with system topics

### Version 1.25 (2025-10-17)
- **Feature: LED Warehouse Bin Highlighting System** 🎨
  - Physical warehouse LED highlighting for job-based device picking
  - Real-time MQTT communication with ESP32 controllers
  - Works globally with cloud-based MQTT broker (no port forwarding needed)
- **Backend LED Module:**
  - New `internal/led/` package with models, MQTT publisher, service layer
  - MQTT client with TLS support, auto-reconnect, heartbeat
  - Dry-run mode for testing without MQTT broker
  - Job-to-bins mapping algorithm using device zone assignments
  - REST API endpoints: status, highlight, clear, identify, test, mapping
  - JSON schemas for LED commands and mapping configuration
  - Validation endpoint for mapping configuration
- **Frontend LED Controls:**
  - Toggle button "Fächer hervorheben" in Jobs page
  - Visual indicators: MQTT connection status, active bin count
  - Auto-clear LEDs when navigating away from job
  - Real-time status updates with LED state
  - Lightbulb icon with toggle states (on/off)
- **ESP32 Firmware:**
  - Complete Arduino sketch for ESP32 + SK6812 GRBW LED strips
  - WiFi connectivity with auto-reconnect
  - MQTT client with TLS support (optional)
  - JSON command parsing with ArduinoJson
  - Multiple LEDs per bin support (e.g., 4 LEDs per compartment)
  - Animation patterns: solid, blink, breathe (sine wave)
  - Watchdog timer for robustness
  - Heartbeat status publishing every 15 seconds
  - Last Will Testament for offline detection
  - Firmware location: `firmware/esp32_sk6812_leds/`
- **Configuration:**
  - LED mapping file: `internal/led/config/led_mapping.json`
  - Maps storage bins to LED pixel indices
  - Supports multiple LEDs per bin
  - Configurable defaults (color, pattern, intensity)
  - LED strip parameters (length, data pin, chipset)
  - Example mapping with 3 shelves, 12 bins
- **MQTT Architecture:**
  - Command topic: `{prefix}/{warehouse_id}/cmd`
  - Status topic: `{prefix}/{warehouse_id}/status`
  - JSON-based commands: highlight, clear, identify
  - ESP32 subscribes to commands, publishes heartbeat
  - Cloud broker recommended (EMQX, HiveMQ, Mosquitto)
- **Hardware Support:**
  - SK6812 GRBW addressable LED strips
  - ESP32 development boards (DevKitC, WROOM-32)
  - Level shifter for data line (3.3V → 5V)
  - Detailed wiring diagrams and power calculations
- **Documentation:**
  - Complete LED system documentation in README
  - ESP32 firmware README with flash instructions
  - Hardware wiring diagrams
  - MQTT broker setup guides (EMQX, Mosquitto, HiveMQ)
  - Troubleshooting section
  - Security considerations (TLS, passwords, secrets)
- **Dependencies:**
  - Go: `github.com/eclipse/paho.mqtt.golang` v1.5.1
  - Arduino: PubSubClient, ArduinoJson, Adafruit_NeoPixel
- **Environment Variables:**
  - `LED_MQTT_HOST` - MQTT broker hostname (empty = dry-run)
  - `LED_MQTT_PORT` - Broker port (1883 or 8883)
  - `LED_MQTT_TLS` - Enable TLS (true/false)
  - `LED_MQTT_USER` - MQTT username
  - `LED_MQTT_PASS` - MQTT password
  - `LED_TOPIC_PREFIX` - Topic namespace
  - `WAREHOUSE_ID` - Warehouse identifier
- **Testing:**
  - Dry-run mode logs commands without sending
  - Status endpoint shows connection state
  - Test endpoints for individual bins
  - Mapping validation endpoint
  - Build verification successful
- **Use Cases:**
  - Visual guidance for warehouse workers during job packing
  - Reduce picking errors by highlighting exact locations
  - Speed up device collection process
  - Real-time verification of picked items
  - Support for large warehouses with hundreds of bins
- **Performance:**
  - MQTT latency < 100ms with cloud broker
  - LED update rate: 50-100 Hz
  - Supports up to ~2000 LEDs per ESP32
  - Automatic reconnection with 5s retry
- **Security:**
  - TLS encryption for production (port 8883)
  - Strong password enforcement
  - Secrets templates prevent credential leaks
  - Namespaced MQTT topics by warehouse_id

### Version 1.18 (2025-10-15)
- **Feature: User Authentication and Single Sign-On (SSO)**
  - Complete user authentication system integrated with RentalCore
  - Shared session-based authentication using MySQL sessions table
  - Cookie-based SSO across both applications (.server-nt.de domain)
  - Login required to access StorageCore directly
  - Automatic authentication when navigating from RentalCore
- **Backend Authentication:**
  - Added GORM ORM for auth model management (User, Session)
  - New auth middleware validating session cookies
  - Auth API endpoints: POST /auth/login, POST /auth/logout, GET /auth/me
  - Shared cookie domain auto-detection for production and localhost
  - Session validation with expiration checking
  - User data loaded from shared users table
- **Frontend Authentication:**
  - New Login page with username/password form
  - AuthContext for global authentication state management
  - ProtectedRoute component guarding all main routes
  - User profile display in sidebar with username and email
  - Logout button with confirmation
  - Loading states during authentication checks
  - TypeScript interfaces for User and auth API
- **Security:**
  - HttpOnly session cookies
  - SameSite Lax mode for CSRF protection
  - Password validation against bcrypt hashes
  - Automatic redirect to login for unauthenticated users
  - Session expiration handling
- **Database Integration:**
  - Uses existing RentalCore tables: users, sessions
  - No new tables required
  - Maintains backward compatibility with RentalCore auth
- **User Experience:**
  - Seamless navigation between RentalCore and StorageCore when logged in
  - Single login for both applications
  - Professional login page matching StorageCore design
  - Clear authentication feedback
  - Automatic session check on application load
- **Technical Changes:**
  - Updated Go version to 1.24 for GORM compatibility
  - Added GetDB() for GORM, GetSQLDB() for raw SQL queries
  - All existing handlers updated to use GetSQLDB()
  - Auth service layer with session management
  - Protected routes middleware in main server setup

### Version 1.17 (2025-10-15)
- **Bug Fix: Mobile Scrolling and Button Overlap Issues**
  - Fixed horizontal scrolling on zone detail page
  - Fixed unnecessary vertical scrolling on scan page
  - Fixed button overlap issues on zone detail page
- **Zone Detail Page Improvements:**
  - Added `max-w-full overflow-x-hidden` to root container to prevent horizontal scroll
  - Changed button container to `flex flex-wrap items-center gap-2 sm:gap-3` for proper wrapping
  - Implemented responsive button text: Full text on desktop, icons only on mobile
  - Delete button: Shows "Löschen" on desktop, "🗑️" emoji on mobile
  - Shelf creation button: Shows "Fächer erstellen" on desktop, "Fächer" on mobile
  - Subzone creation button: Shows "Unterzone erstellen" on desktop, "Unterzone" on mobile
  - Added `whitespace-nowrap` to all buttons to prevent text wrapping within buttons
- **Breadcrumb Navigation:**
  - Added `overflow-x-auto` with horizontal scroll support
  - Breadcrumb items use `flex-shrink-0` to prevent crushing
  - Maintains proper spacing with `flex-wrap` fallback
- **Device Table Responsiveness:**
  - Table container with `overflow-x-auto` for horizontal scroll on small screens
  - Set minimum table width to 640px to maintain readability
  - Hidden columns on mobile for space efficiency:
    - Manufacturer column: `hidden md:table-cell`
    - Model column: `hidden md:table-cell`
    - Barcode column: `hidden lg:table-cell`
  - Responsive cell padding: `p-2 sm:p-4`
  - Responsive text sizes: `text-xs sm:text-sm` and `text-xs sm:text-base`
- **Scan Page Improvements:**
  - Removed unnecessary `min-h-[calc(100vh-4rem)]` that caused vertical scrolling
  - Changed to simple `flex items-center justify-center` for proper centering
  - Form now fits perfectly without any unnecessary scroll
- **Responsive Text and Spacing:**
  - All headers use responsive sizing: `text-xl sm:text-3xl`, `text-lg sm:text-xl`
  - Stats cards use adaptive padding: `p-3 sm:p-4`, `p-4 sm:p-6`
  - Icon sizes adapt: `w-4 h-4`, `w-4 h-4 sm:w-5 sm:h-5`
  - Gaps scale properly: `gap-2 sm:gap-3`, `gap-3 sm:gap-4`
- **User Experience Enhancements:**
  - No horizontal scrolling on any viewport width
  - No unnecessary vertical scrolling
  - All buttons stay visible and accessible on mobile
  - Buttons never overlap or extend beyond screen width
  - Table data remains accessible with horizontal scroll only when needed
  - Clean, professional mobile interface

### Version 1.16 (2025-10-15)
- **Feature: Complete Mobile Responsiveness**
  - Full mobile optimization for all pages and components
  - Professional responsive design with breakpoints at 640px (sm) and 768px (md)
- **Mobile Sidebar:**
  - Hamburger menu with overlay backdrop on mobile
  - Smooth slide-in/out animations
  - Auto-close sidebar on navigation for mobile
  - Desktop: Sidebar visible by default, persists on screen
  - Mobile: Sidebar hidden by default, overlay mode
  - Close button in sidebar header (mobile only)
  - Touch-friendly tap targets
- **Layout Optimizations:**
  - Header: Responsive padding and font sizes (px-3 → px-6, text-lg → text-2xl)
  - Company name hidden on mobile to save space
  - Main content padding adapts (p-3 → p-6)
  - Proper spacing adjustments (pt-14 → pt-16)
- **Dashboard Page:**
  - Stats grid: 1 column on mobile, 2 on sm, 4 on lg
  - Card padding and text sizes scale responsively
  - Icon sizes adjust (w-5 → w-6)
  - Activity feed items with truncation
- **Scan Page:**
  - Scanner icon and titles scale (w-8 → w-12, text-2xl → text-4xl)
  - Step indicator adapts sizing (w-7 → w-8)
  - Input fields with responsive padding
  - Action buttons grid adjusts (gap-2 → gap-4)
  - Result cards with proper text wrapping
- **Devices Page:**
  - Search bar with responsive icon and input sizing
  - Grid: 1 column → 2 columns (sm) → 3 columns (lg)
  - Device cards with flexible layouts
  - Status badges scale properly
  - Truncation for long text
- **Maintenance Page (All 3 Tabs):**
  - **Overview:** Stats in 2-column grid (2 on mobile, 5 on lg)
  - **Defects:** Mobile-friendly form with stacked inputs
  - **Inspections:** Responsive filter tabs with horizontal scroll
  - All cards adapt padding (p-3 → p-6)
  - Text sizes scale across breakpoints
  - Status buttons stack on mobile, inline on desktop
  - Form buttons stack vertically on mobile
- **Mobile-First Features:**
  - Touch-friendly button sizes (44x44px minimum)
  - Proper text wrapping and truncation
  - Overflow-x-auto for filter tabs
  - Flex-wrap for badges and tags
  - Min-w-0 to allow flex item shrinking
  - Line-clamp for multi-line text truncation
- **Responsive Typography:**
  - Headings: text-2xl sm:text-3xl or text-2xl sm:text-4xl
  - Body text: text-xs sm:text-sm or text-sm sm:text-base
  - Badges: text-[10px] sm:text-xs
  - Proper hierarchy maintained across breakpoints
- **Spacing System:**
  - Gaps: gap-2 sm:gap-4, gap-3 sm:gap-6
  - Padding: p-3 sm:p-6, p-4 sm:p-8
  - Margins: mb-1 sm:mb-2, mb-3 sm:mb-4
  - Consistent use of Tailwind spacing scale
- **User Experience:**
  - Smooth transitions for sidebar and modals
  - No horizontal scrolling on any page
  - All interactive elements easily tappable
  - Proper keyboard focus management
  - Maintains glassmorphism aesthetic on all devices
- **Testing Notes:**
  - Tested at breakpoints: 320px, 375px, 768px, 1024px, 1920px
  - Works on iOS Safari, Chrome Mobile, Android Chrome
  - Landscape and portrait orientations supported

### Version 1.15 (2025-10-14)
- **Feature: Complete Maintenance Module Implementation**
  - Full defect tracking and repair management system
  - Inspection scheduling and monitoring
  - Comprehensive maintenance dashboard with real-time statistics
- **Defect Management:**
  - Create defect reports with severity levels (low, medium, high, critical)
  - Status workflow: open → in_progress → repaired → closed
  - Automatic device status updates when defects are created/resolved
  - Inline defect creation form with device ID, severity, title, description
  - Filter defects by status: All, Open, In Progress, Repaired, Closed
  - Color-coded severity badges (critical=red, high=orange, medium=yellow, low=blue)
  - Track repair costs and repair notes
  - Automatic timestamp management (reported_at, repaired_at, closed_at)
  - Status update buttons for workflow progression
- **Inspection Management:**
  - View all inspection schedules with filtering
  - Filter by: All, Overdue, Upcoming (next 30 days)
  - Visual indicators for overdue inspections (red border, "ÜBERFÄLLIG" badge)
  - Display inspection type, interval days, last/next inspection dates
  - Support for both device-specific and product-wide inspections
  - Active/inactive inspection status
- **Maintenance Dashboard:**
  - Overview tab with 5 real-time statistics cards
  - Shows: Open defects, In Progress defects, Repaired defects, Overdue inspections, Upcoming inspections
  - Quick action cards to navigate to defects or inspections
  - Auto-refresh data on view changes
- **Backend API:**
  - `GET /defects` - List defects with filters (status, severity, device_id)
  - `POST /defects` - Create new defect report
  - `PUT /defects/{id}` - Update defect status, costs, notes
  - `GET /maintenance/inspections` - Get inspection schedules (filters: upcoming, overdue, all)
  - `GET /maintenance/stats` - Dashboard statistics
  - Dynamic SQL query building for flexible filtering
  - JOIN queries with devices and products tables for rich data
- **Frontend Features:**
  - New MaintenancePage with three-tab interface (Overview, Defects, Inspections)
  - Real-time data loading with automatic updates
  - Responsive card-based layout with glassmorphism design
  - Color-coded status and severity indicators
  - Inline forms for quick defect creation
  - Status progression buttons for efficient workflow
  - Clock icons with color coding (red=overdue, blue=upcoming)
- **API Types:**
  - New TypeScript interfaces: Defect, Inspection, MaintenanceStats
  - Added maintenanceApi with getStats(), getDefects(), createDefect(), updateDefect(), getInspections()
- **Navigation:**
  - New "Wartung" (Maintenance) menu item with Wrench icon
  - Route: /maintenance
- **Database Integration:**
  - Uses existing tables: defect_reports, inspection_schedules
  - Bi-directional device status updates (defective ↔ in_storage)
  - Full audit trail with timestamps
- **User Workflow:**
  - Defect Reporting:
    1. Navigate to Wartung → Defects tab
    2. Click "Neuer Defekt" to create report
    3. Enter device ID, select severity, add title and description
    4. Submit → Device automatically marked as defective
    5. Track repair progress with status updates
    6. Mark as repaired → Device automatically returns to in_storage
  - Inspection Management:
    1. Navigate to Wartung → Inspections tab
    2. View all scheduled inspections
    3. Filter by overdue or upcoming
    4. See visual indicators for priority
    5. Track last and next inspection dates
- **Use Cases:**
  - Track equipment defects from discovery through repair completion
  - Monitor repair costs and maintenance budgets
  - Ensure timely inspections with overdue alerts
  - Maintain compliance with inspection schedules
  - Full visibility into equipment condition and maintenance needs
  - Historical tracking of all repairs and defects

### Version 1.14 (2025-10-14)
- **Feature: Job-Based Outtake Workflow with Live Scan Tracking**
  - New Jobs page accessible via sidebar navigation (/jobs)
  - Displays all jobs with status "open" ready for device outtake
  - Select a job to start the outtake/packing process
  - Real-time live tracking of which devices have been scanned out
  - Visual progress bar showing scan completion (e.g., 5/20 devices scanned)
  - Two-column layout: Scan interface + Device checklist
  - Auto-refresh every 2 seconds to show live updates as devices are scanned
  - Device list shows checkmarks (✓) for scanned devices, X for remaining
  - Color-coded status: Green for scanned, gray for pending
  - Integration with job_devices table and pack_status tracking
- **Backend Enhancements:**
  - Completely implemented `GetJobs` handler with customer info and device counts
  - Enhanced `GetJobSummary` handler to return full job details with all devices
  - Device list includes current status, location, pack_status, and scan state
  - Scanned flag based on device status ('on_job') or pack_status ('issued')
  - Supports filtering jobs by status (e.g., ?status=open)
  - JOIN queries across jobs, status, customers, jobdevices, devices, products, storage_zones
- **Frontend Features:**
  - New JobsPage component with job selection and scan interface
  - Job cards show: Job ID, description, customer name, dates, device count
  - Job detail view with customer info, dates, and progress statistics
  - Live device checklist with product names, IDs, status, and scan indicators
  - Scan form integrated with job context (sends job_id to scan API)
  - Success/error feedback after each scan
  - "Back to Job List" navigation
  - Auto-refresh for live updates during scanning
- **API Types:**
  - New TypeScript interfaces: Job, JobDevice, JobSummary
  - Added jobsApi.getAll() and jobsApi.getById() functions
- **Navigation:**
  - New "Jobs" menu item with Briefcase icon
  - Route: /jobs
- **User Workflow:**
  1. Navigate to Jobs page
  2. View all open jobs with device counts
  3. Select a job to start packing/outtake
  4. Scan devices one by one
  5. Watch the progress bar and checklist update live
  6. See which devices are still missing at a glance
  7. Complete when all devices are scanned
- **Use Case:**
  - Perfect for warehouse workers preparing equipment for upcoming events
  - Second-level verification that all job devices are physically present
  - Real-time feedback prevents missing items before transport
  - Live updates allow multiple workers to see scan progress simultaneously

### Version 1.13 (2025-10-14)
- **Feature: Recursive Device Count for Parent Zones**
  - Zone device counts now include all devices in subzones recursively
  - When viewing a parent zone (e.g., Lager or Regal), device count shows total across all child zones
  - Implemented using MySQL recursive CTE (Common Table Expression) for efficient querying
  - Example: Lager Weidelbach shows total of all devices in all Regale and all Fächer within
- **Backend Enhancements:**
  - New function `getDeviceCountRecursive` in ZoneService
  - Uses `WITH RECURSIVE` query to build complete zone tree
  - Counts all devices with status 'in_storage' across entire zone hierarchy
  - Updated `GetZoneDetails` to use recursive count
  - Updated `getSubzones` to show recursive counts for each subzone
- **User Benefits:**
  - Accurate inventory totals at every level of the hierarchy
  - No need to manually sum devices across subzones
  - Better overview of warehouse stock distribution
  - Reflects real warehouse organization where parent zones contain all child zone contents

### Version 1.12 (2025-10-14)
- **Feature: Two-Step Intake Workflow with Zone Selection**
  - Implemented multi-step barcode scanning for device intake (Einlagerung)
  - Step 1: Scan device barcode/QR code to verify device exists
  - Step 2: Scan storage location (zone) barcode to specify exact placement
  - Visual step indicator shows progress (Gerät → Lagerplatz)
  - Automatic zone lookup by barcode or zone code
- **Backend Enhancements:**
  - New endpoint: `GET /api/v1/zones/scan?scan_code={code}` - Find zone by barcode/code
  - Enhanced intake process to require and store zone_id
  - Zone barcode support for precise location tracking
- **Frontend Improvements:**
  - Smart UI that adapts based on current step (device vs. zone scan)
  - Different icons and placeholders for each step
  - Action buttons hidden during zone scan step
  - Seamless error handling with automatic step reset
- **User Experience:**
  - Clear visual feedback with step-by-step process
  - Ensures every device is assigned to a specific storage location
  - Prevents devices from being stored without location information
- **Example Workflow:**
  1. Select "Einlagern" action
  2. Scan device barcode → Device verified ✓
  3. Scan zone barcode (e.g., FACH-00000014) → Device stored at specific location
  4. Process complete with full audit trail

### Version 1.11 (2025-10-14)
- **Bug Fix: SPA Routing on Page Reload**
  - Fixed 404 errors when refreshing pages like `/zones`, `/zones/{id}`, etc.
  - Implemented custom `spaHandler` to serve `index.html` for all non-file routes
  - Server now properly handles client-side routing for React Router
  - Changed from `http.FileServer` to custom handler with file existence check
  - All frontend routes now work correctly on direct access or browser refresh

### Version 1.10 (2025-10-14)
- **Bug Fix: Zone Devices SQL Query**
  - Fixed 500 Internal Server Error on `/api/v1/zones/{id}/devices` endpoint
  - Updated query to properly join with `manufacturer` and `brands` tables
  - Changed from direct `p.manufacturer` and `p.model` columns to `m.name` and `b.name` via JOINs
  - Products table uses `manufacturerID` and `brandID` foreign keys, not direct string columns
- **Feature: Subzone Delete Buttons**
  - Added delete buttons to subzone cards in zone detail view
  - Buttons appear on hover, similar to main zones page
  - Click delete without navigating into the subzone
  - Prevents accidental navigation when deleting
  - Confirmation dialog before deletion

### Version 1.9 (2025-10-14)
- **Bug Fix: Race Condition in Bulk Shelf Creation**
  - Fixed duplicate entry errors when creating multiple shelves at once
  - Changed from parallel (Promise.all) to sequential creation
  - Prevents multiple shelves from reading the same count and generating duplicate codes
  - Example: Creating 5 shelves now correctly generates Fach 01-05 instead of all trying to create Fach 01
- **Improved Error Handling:**
  - Eliminated 500 Internal Server errors during bulk operations
  - Consistent shelf numbering without gaps

### Version 1.8 (2025-10-14)
- **Automatic Shelf (Fach) Creation:**
  - Added 📚 **Fach** (shelf) as a 4th zone type for Regalen
  - Automatic name generation (Fach 01, Fach 02, etc.) based on existing shelves in parent rack
  - Automatic barcode generation (FACH-%08d format) after creation
  - No manual name input required for shelf creation
- **Bulk Shelf Creation:**
  - "Fächer erstellen" button in rack detail view
  - Create multiple shelves at once with a single action
  - Default capacity of 10 per shelf
- **Navigation Improvements:**
  - Fixed subzone creation to stay in parent zone context instead of redirecting to /zones
  - Maintains workflow continuity when creating hierarchical structures
- **Database Migration:**
  - Added barcode column to storage_zones table (006_add_zone_barcode.sql)
  - Index on barcode for efficient lookups
- **Backend Enhancements:**
  - GenerateShelfName function in ZoneService for automatic naming
  - Updated CreateZone handler to support optional Name field for shelves
  - Barcode generation logic after zone insertion
- **Example Usage:**
  - Create Lager Weidelbach (WDB)
  - Create Regal A (WDB-RG-01)
  - Click "Fächer erstellen" → Enter "5" → Creates Fach 01 through Fach 05 automatically
  - Each Fach gets codes like WDB-RG-01-F-01 and barcodes like FACH-00000014

### Version 1.7 (2025-10-14)
- **Simplified Zone Types:** Reduced to 3 types only
  - 🏭 **Lager** (warehouse) - Main storage facility
  - 🗄️ **Regal** (rack) - Shelving units
  - 📦 **Gitterbox** (gitterbox) - Wire mesh containers
- **Delete Functionality:**
  - Delete button on zone cards (hover to reveal)
  - Delete button in zone detail view
  - Safety checks: prevents deletion if zone contains devices or subzones
  - Confirmation dialog before deletion
- **Updated Code Generation:**
  - Warehouse prefix: LGR (Lager)
  - Rack prefix: RG (Regal)
  - Gitterbox prefix: GB (Gitterbox)
- **Example Hierarchy:**
  - Weidelbach (WDL) → Regal A (WDL-RG-01) → Gitterbox 01 (WDL-RG-01-GB-01)
- **Database Migration:** Updated ENUM type to support gitterbox

### Version 1.6 (2025-10-14)
- **Automatic Zone Code Generation:** Smart, hierarchical code generation
- **Hierarchical Zone System:** Create nested zones
- **Zone Detail View:** Click zones to see subzones, devices, and breadcrumb navigation
- **ZoneService:** New service layer for zone business logic
- **API Enhancements:**
  - `GET /zones/{id}` returns full details with subzones and breadcrumb
  - `GET /zones/{id}/devices` lists all devices in a zone
  - Optional `code` field in zone creation (auto-generated if not provided)
- **Frontend Improvements:**
  - New ZoneDetailPage component with subzone navigation
  - Parent zone selection in zone creation form
  - Breadcrumb navigation for zone hierarchy
  - Click-to-navigate zone cards

### Version 1.5 (2025-10-14)
- Fixed JSON API responses to return clean primitive types
- Removed nested sql.Null* structures from API responses
- Added ZoneResponse struct for consistent API output
- Fixed React rendering error on zones page
- Updated TypeScript interfaces for nullable fields

### Version 1.4 (2025-10-14)
- Fixed zone creation API endpoint
- Improved JSON request handling with proper nullable field support
- Added detailed logging for zone creation operations

### Version 1.3 (2025-10-14)
- Fixed API URL configuration for multi-platform deployment
- Changed from absolute to relative URLs for cross-platform compatibility

### Version 1.2 (2025-10-14)
- Frontend glassmorphism enhancements
- Improved device and zone display components

### Version 1.1 (2025-10-14)
- Fixed TailwindCSS v4 configuration
- Updated PostCSS plugins for modern build system

### Version 1.0 (2025-10-14)
- Initial release with core warehouse management features
- Scan-driven workflows implementation
- Modern React + TypeScript frontend with glassmorphism design
