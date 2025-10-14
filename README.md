# StorageCore

**Physical Warehouse Management System for Tsunami Events UG**

StorageCore is the digital twin of the Weidelbach warehouse, providing real-time tracking of devices, cases, zones, and movements with barcode/QR scan-driven workflows.

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
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
- Go 1.22+ (matching RentalCore)
- gorilla/mux (routing)
- MySQL 9.2 (shared RentalCore database)
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

- Go 1.22+
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
- `GET /defects` - List defect reports
- `POST /defects` - Create defect report
- `PUT /defects/{id}` - Update defect report
- `GET /maintenance/inspections` - Get inspection schedules

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
docker-compose up -d
```

**Check logs:**
```bash
docker-compose logs -f storagecore
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
- `1.4` - Zone creation fix (current)
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

**Version:** 1.4
**Last Updated:** 2025-10-14
**Maintainer:** Tsunami Events UG Development Team

---

## Changelog

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
