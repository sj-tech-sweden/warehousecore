-- 080_create_integration_links_and_event_receipts.sql
-- Foundation tables for bidirectional integration sync with idempotent event ingest.

BEGIN;

CREATE TABLE IF NOT EXISTS integration_links (
  link_id BIGSERIAL PRIMARY KEY,
  system VARCHAR(64) NOT NULL,
  entity_type VARCHAR(64) NOT NULL,
  warehouse_id TEXT NULL,
  twenty_id TEXT NULL,
  last_source VARCHAR(64) NULL,
  last_event_id TEXT NULL,
  last_synced_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_integration_links_system CHECK (system IN ('twenty')),
  CONSTRAINT chk_integration_links_entity_type CHECK (entity_type IN ('customer', 'job', 'requirement', 'product')),
  CONSTRAINT chk_integration_links_one_side_present CHECK (warehouse_id IS NOT NULL OR twenty_id IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_integration_links_system_entity_twenty
  ON integration_links(system, entity_type, twenty_id)
  WHERE twenty_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS ux_integration_links_system_entity_warehouse
  ON integration_links(system, entity_type, warehouse_id)
  WHERE warehouse_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_integration_links_last_synced_at
  ON integration_links(last_synced_at);

CREATE TABLE IF NOT EXISTS integration_event_receipts (
  receipt_id BIGSERIAL PRIMARY KEY,
  idempotency_key TEXT NOT NULL,
  event_id TEXT NOT NULL,
  source VARCHAR(64) NOT NULL,
  entity_type VARCHAR(64) NOT NULL,
  action VARCHAR(32) NOT NULL,
  correlation_id TEXT NULL,
  payload_json JSONB NOT NULL,
  received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  processed_at TIMESTAMP NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'accepted',
  status_reason TEXT NULL,
  CONSTRAINT chk_integration_event_receipts_source CHECK (source IN ('twenty', 'warehousecore')),
  CONSTRAINT chk_integration_event_receipts_entity_type CHECK (entity_type IN ('customer', 'job', 'requirement', 'product')),
  CONSTRAINT chk_integration_event_receipts_action CHECK (action IN ('upsert', 'delete'))
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_integration_event_receipts_idempotency_key
  ON integration_event_receipts(idempotency_key);

CREATE INDEX IF NOT EXISTS idx_integration_event_receipts_event_id
  ON integration_event_receipts(event_id);

CREATE INDEX IF NOT EXISTS idx_integration_event_receipts_received_at
  ON integration_event_receipts(received_at);

COMMIT;
