import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { zoneTypesApi, type ZoneTypeDefinition } from './api';

export function useZoneTypes() {
  const { t } = useTranslation();
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
      setError(t('zonesPage.loadError'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadZoneTypes();
  }, [loadZoneTypes]);

  return { zoneTypes, loading, error, reload: loadZoneTypes };
}
