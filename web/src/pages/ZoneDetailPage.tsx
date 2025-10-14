import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { MapPin, Package, ChevronRight, ArrowLeft, Plus, Trash2 } from 'lucide-react';
import { zonesApi } from '../lib/api';

interface ZoneDetails {
  zone_id: number;
  code: string;
  name: string;
  type: string;
  description?: string;
  capacity?: number;
  device_count: number;
  is_active: boolean;
  breadcrumb: Array<{ zone_id: string; code: string; name: string }>;
  subzones: Array<{
    zone_id: number;
    code: string;
    name: string;
    type: string;
    capacity?: number;
    device_count: number;
    subzone_count: number;
  }>;
}

interface DeviceInZone {
  device_id: string;
  product_id?: number;
  product_name: string;
  manufacturer?: string;
  model?: string;
  status: string;
  barcode?: string;
  qr_code?: string;
}

export function ZoneDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [zone, setZone] = useState<ZoneDetails | null>(null);
  const [devices, setDevices] = useState<DeviceInZone[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (id) {
      loadZoneDetails();
      loadDevices();
    }
  }, [id]);

  const loadZoneDetails = async () => {
    try {
      const { data } = await zonesApi.getById(Number(id));
      setZone(data as ZoneDetails);
    } catch (error) {
      console.error('Failed to load zone details:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadDevices = async () => {
    try {
      const response = await fetch(`/api/v1/zones/${id}/devices`);
      const data = await response.json();
      setDevices(data);
    } catch (error) {
      console.error('Failed to load devices:', error);
    }
  };

  const handleZoneClick = (zoneId: number) => {
    navigate(`/zones/${zoneId}`);
  };

  const handleCreateSubzone = () => {
    navigate(`/zones?parent=${id}`);
  };

  const handleDelete = async () => {
    if (!zone) return;

    if (zone.device_count > 0) {
      alert('Diese Zone enthält noch Geräte und kann nicht gelöscht werden.');
      return;
    }

    if (zone.subzones && zone.subzones.length > 0) {
      alert('Diese Zone enthält noch Unterzonen und kann nicht gelöscht werden.');
      return;
    }

    if (!confirm(`Zone "${zone.name}" (${zone.code}) wirklich löschen?`)) {
      return;
    }

    try {
      await zonesApi.delete(zone.zone_id);
      navigate('/zones');
    } catch (error) {
      console.error('Failed to delete zone:', error);
      alert('Fehler beim Löschen der Zone');
    }
  };

  const zoneTypes = {
    warehouse: { label: 'Lager', icon: '🏭' },
    rack: { label: 'Regal', icon: '🗄️' },
    gitterbox: { label: 'Gitterbox', icon: '📦' },
    other: { label: 'Sonstige', icon: '📍' },
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
      </div>
    );
  }

  if (!zone) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-400">Zone nicht gefunden</p>
      </div>
    );
  }

  const typeInfo = zoneTypes[zone.type as keyof typeof zoneTypes] || zoneTypes.other;

  return (
    <div className="space-y-6">
      {/* Header with Breadcrumb */}
      <div>
        <button
          onClick={() => navigate('/zones')}
          className="flex items-center gap-2 text-gray-400 hover:text-white mb-4 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          Zurück zu Zonen
        </button>

        {/* Breadcrumb */}
        {zone.breadcrumb && zone.breadcrumb.length > 0 && (
          <div className="flex items-center gap-2 text-sm text-gray-400 mb-4 flex-wrap">
            {zone.breadcrumb.map((crumb, index) => (
              <div key={crumb.zone_id} className="flex items-center gap-2">
                {index > 0 && <ChevronRight className="w-4 h-4" />}
                <button
                  onClick={() => handleZoneClick(Number(crumb.zone_id))}
                  className="hover:text-white transition-colors"
                >
                  {crumb.name}
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div className="text-5xl">{typeInfo.icon}</div>
            <div>
              <h2 className="text-3xl font-bold text-white mb-1">{zone.name}</h2>
              <p className="text-gray-400 font-mono text-sm">{zone.code}</p>
              <p className="text-sm text-gray-500">{typeInfo.label}</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <button
              onClick={handleDelete}
              className="flex items-center gap-2 px-4 py-2 glass text-red-400 hover:text-red-300 hover:bg-red-900/20 font-semibold rounded-xl transition-all"
            >
              <Trash2 className="w-4 h-4" />
              Löschen
            </button>
            <button
              onClick={handleCreateSubzone}
              className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-accent-red to-red-700 text-white font-semibold rounded-xl hover:shadow-lg hover:shadow-accent-red/50 transition-all"
            >
              <Plus className="w-4 h-4" />
              Unterzone erstellen
            </button>
          </div>
        </div>

        {zone.description && (
          <p className="text-gray-400 mt-4">{zone.description}</p>
        )}

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mt-6">
          <div className="glass rounded-xl p-4">
            <div className="text-2xl font-bold text-white">{zone.device_count}</div>
            <div className="text-sm text-gray-400">Geräte</div>
          </div>
          <div className="glass rounded-xl p-4">
            <div className="text-2xl font-bold text-white">{zone.subzones?.length || 0}</div>
            <div className="text-sm text-gray-400">Unterzonen</div>
          </div>
          {zone.capacity && (
            <div className="glass rounded-xl p-4">
              <div className="text-2xl font-bold text-white">{zone.capacity}</div>
              <div className="text-sm text-gray-400">Kapazität</div>
            </div>
          )}
        </div>
      </div>

      {/* Subzones */}
      {zone.subzones && zone.subzones.length > 0 && (
        <div>
          <h3 className="text-xl font-bold text-white mb-4 flex items-center gap-2">
            <MapPin className="w-5 h-5" />
            Unterzonen
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {zone.subzones.map((subzone) => {
              const subTypeInfo = zoneTypes[subzone.type as keyof typeof zoneTypes] || zoneTypes.other;
              return (
                <button
                  key={subzone.zone_id}
                  onClick={() => handleZoneClick(subzone.zone_id)}
                  className="glass rounded-xl p-5 hover:bg-white/20 transition-all text-left group"
                >
                  <div className="flex items-start gap-4">
                    <div className="text-3xl">{subTypeInfo.icon}</div>
                    <div className="flex-1 min-w-0">
                      <h4 className="font-bold text-white truncate group-hover:text-accent-red transition-colors">
                        {subzone.name}
                      </h4>
                      <p className="text-sm text-gray-400 font-mono">{subzone.code}</p>
                      <div className="flex items-center gap-3 mt-2 text-xs text-gray-500">
                        <span>{subzone.device_count} Geräte</span>
                        {subzone.subzone_count > 0 && (
                          <span>{subzone.subzone_count} Unterzonen</span>
                        )}
                      </div>
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Devices */}
      {devices.length > 0 && (
        <div>
          <h3 className="text-xl font-bold text-white mb-4 flex items-center gap-2">
            <Package className="w-5 h-5" />
            Gelagerte Geräte ({devices.length})
          </h3>
          <div className="glass-dark rounded-2xl overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-white/10">
                    <th className="text-left p-4 text-sm font-semibold text-gray-400">Geräte-ID</th>
                    <th className="text-left p-4 text-sm font-semibold text-gray-400">Produkt</th>
                    <th className="text-left p-4 text-sm font-semibold text-gray-400">Hersteller</th>
                    <th className="text-left p-4 text-sm font-semibold text-gray-400">Modell</th>
                    <th className="text-left p-4 text-sm font-semibold text-gray-400">Barcode</th>
                  </tr>
                </thead>
                <tbody>
                  {devices.map((device) => (
                    <tr
                      key={device.device_id}
                      className="border-b border-white/5 hover:bg-white/5 transition-colors"
                    >
                      <td className="p-4 font-mono text-sm text-white">{device.device_id}</td>
                      <td className="p-4 text-white">{device.product_name || '-'}</td>
                      <td className="p-4 text-gray-400">{device.manufacturer || '-'}</td>
                      <td className="p-4 text-gray-400">{device.model || '-'}</td>
                      <td className="p-4 font-mono text-sm text-gray-400">{device.barcode || '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      )}

      {devices.length === 0 && (!zone.subzones || zone.subzones.length === 0) && (
        <div className="text-center py-12 glass rounded-2xl">
          <Package className="w-16 h-16 text-gray-600 mx-auto mb-4" />
          <p className="text-gray-400 mb-4">Diese Zone ist leer</p>
          <button
            onClick={handleCreateSubzone}
            className="text-accent-red hover:text-red-500 font-semibold"
          >
            Unterzone erstellen
          </button>
        </div>
      )}
    </div>
  );
}
