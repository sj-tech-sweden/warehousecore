import { useCallback, useEffect, useState } from 'react';
import { zoneTypesApi, type ZoneTypeDefinition } from './api';

export function useZoneTypes() {
  const [zoneTypes, setZoneTypes] = useState<ZoneTypeDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadZoneTypes = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const { data } = await zoneTypesApi.getAll();
      setZoneTypes(data);
    } catch (err) {
      console.error('Failed to load zone types:', err);
      setError('Lagertypen konnten nicht geladen werden.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadZoneTypes();
  }, [loadZoneTypes]);

  return { zoneTypes, loading, error, reload: loadZoneTypes };
}
