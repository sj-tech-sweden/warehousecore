-- Backfill helpers for WarehouseCore (placeholder)
BEGIN;

CREATE TABLE IF NOT EXISTS backfill_queue (
  id serial PRIMARY KEY,
  entity_type text NOT NULL,
  entity_id integer NOT NULL,
  enqueued_at timestamptz DEFAULT now()
);

DELETE FROM backfill_queue q
USING (
  SELECT ctid
  FROM (
    SELECT ctid,
           row_number() OVER (PARTITION BY entity_type, entity_id ORDER BY id) AS rn
    FROM backfill_queue
  ) dedupe
  WHERE dedupe.rn > 1
) d
WHERE q.ctid = d.ctid;

CREATE UNIQUE INDEX IF NOT EXISTS idx_backfill_queue_entity_unique
  ON backfill_queue (entity_type, entity_id);

CREATE OR REPLACE FUNCTION enqueue_entity_backfill(entity text, id integer) RETURNS void AS $$
BEGIN
  INSERT INTO backfill_queue (entity_type, entity_id)
  VALUES (entity, id)
  ON CONFLICT (entity_type, entity_id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

COMMIT;
