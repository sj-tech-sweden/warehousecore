-- warehousecore seed data (idempotent)
BEGIN;

CREATE TABLE IF NOT EXISTS public.seed_marker (name text PRIMARY KEY, applied_at timestamptz DEFAULT now());
INSERT INTO public.seed_marker (name) VALUES ('initial_seed') ON CONFLICT DO NOTHING;

-- Inter-service API keys are configured via SERVICE_API_KEY (env), not seeded.

-- Products (handle optional `created_at` column)
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'products' AND column_name = 'created_at') THEN
		EXECUTE $q$
			INSERT INTO products (productID, name, created_at)
			VALUES (1, 'Widget A', NOW()),
			       (2, 'Cable 1m', NOW())
			ON CONFLICT DO NOTHING;
		$q$;
	ELSE
		EXECUTE $q$
			INSERT INTO products (productID, name)
			VALUES (1, 'Widget A'),
			       (2, 'Cable 1m')
			ON CONFLICT DO NOTHING;
		$q$;
	END IF;
END$$;

-- Devices / Zones
INSERT INTO zones (id, name) VALUES (1, 'Main Warehouse') ON CONFLICT DO NOTHING;
-- Insert device, handle optional `created_at` column
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'devices' AND column_name = 'created_at') THEN
		EXECUTE $q$
			INSERT INTO devices (deviceID, serialnumber, productID, zone_id, created_at)
			VALUES ('DEV-0001', 'DEV-0001', 1, 1, NOW()) ON CONFLICT DO NOTHING;
		$q$;
	ELSE
		EXECUTE $q$
			INSERT INTO devices (deviceID, serialnumber, productID, zone_id)
			VALUES ('DEV-0001', 'DEV-0001', 1, 1) ON CONFLICT DO NOTHING;
		$q$;
	END IF;
END$$;

COMMIT;
