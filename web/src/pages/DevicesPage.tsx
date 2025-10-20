import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Package, Search, Lightbulb, MapPin } from 'lucide-react';
import { devicesApi, ledApi } from '../lib/api';
import type { Device } from '../lib/api';
import { getStatusColor, formatStatus } from '../lib/utils';
import { DeviceDetailModal } from '../components/DeviceDetailModal';

interface ProductGroup {
  key: string;
  productId: number | null;
  productName: string;
  devices: Device[];
}

export function DevicesPage() {
  const navigate = useNavigate();
  const [devices, setDevices] = useState<Device[]>([]);
  const [productGroups, setProductGroups] = useState<ProductGroup[]>([]);
  const [activeProductKey, setActiveProductKey] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [actionMessage, setActionMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  useEffect(() => {
    loadDevices();
  }, []);

  const loadDevices = async () => {
    try {
      const { data } = await devicesApi.getAll({ limit: 100 });
      setDevices(data);
    } catch (error) {
      console.error('Failed to load devices:', error);
    } finally {
      setLoading(false);
    }
  };

  const buildProductGroups = (items: Device[]): ProductGroup[] => {
    const map = new Map<string, ProductGroup>();

    items.forEach((device) => {
      const key =
        device.product_id !== undefined && device.product_id !== null
          ? `product-${device.product_id}`
          : `name-${device.product_name ?? 'Unbekannt'}`;

      if (!map.has(key)) {
        map.set(key, {
          key,
          productId: device.product_id ?? null,
          productName: device.product_name || 'Unbekanntes Produkt',
          devices: [],
        });
      }

      map.get(key)!.devices.push(device);
    });

    return Array.from(map.values()).sort((a, b) =>
      a.productName.localeCompare(b.productName, 'de', { sensitivity: 'base' })
    );
  };

  useEffect(() => {
    setProductGroups(buildProductGroups(devices));
  }, [devices]);

  const filteredProducts = useMemo(() => {
    const term = search.trim().toLowerCase();
    if (!term) {
      return productGroups;
    }

    return productGroups.filter((group) => {
      const matchesName = group.productName.toLowerCase().includes(term);
      const matchesDevice = group.devices.some(
        (device) =>
          device.device_id.toLowerCase().includes(term) ||
          device.serial_number?.toLowerCase().includes(term) ||
          device.zone_name?.toLowerCase().includes(term)
      );
      return matchesName || matchesDevice;
    });
  }, [productGroups, search]);

  useEffect(() => {
    if (filteredProducts.length === 0) {
      setActiveProductKey(null);
      return;
    }

    if (!activeProductKey || !filteredProducts.some((group) => group.key === activeProductKey)) {
      setActiveProductKey(filteredProducts[0].key);
    }
  }, [filteredProducts, activeProductKey]);

  const activeProduct = filteredProducts.find((group) => group.key === activeProductKey) ?? null;
  const totalDeviceCount = filteredProducts.reduce((acc, group) => acc + group.devices.length, 0);

  const handleLocateDevice = async (device: Device) => {
    if (!device.zone_code) {
      setActionMessage({ type: 'error', text: 'Kein Fachcode vorhanden – Gerät nicht im Lager.' });
      return;
    }
    try {
      await ledApi.locateBin(device.zone_code);
      setActionMessage({
        type: 'success',
        text: `Fach ${device.zone_code} wird hervorgehoben.`,
      });
      setTimeout(() => setActionMessage(null), 4000);
    } catch (error: any) {
      setActionMessage({
        type: 'error',
        text: 'LED-Befehl fehlgeschlagen: ' + (error.response?.data?.error || error.message || error.toString()),
      });
    }
  };

  const handleOpenZone = (device: Device) => {
    if (device.zone_id) {
      navigate(`/zones/${device.zone_id}`);
    } else if (device.zone_code) {
      navigate(`/zones?parent=${device.zone_code}`);
    }
  };

  const handleOpenDevice = (device: Device) => {
    setSelectedDevice(device);
    setDetailOpen(true);
  };

  const handleCloseDevice = () => {
    setSelectedDevice(null);
    setDetailOpen(false);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
      </div>
    );
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold text-white mb-1 sm:mb-2">Produkte &amp; Geräte</h2>
          <p className="text-sm sm:text-base text-gray-400">
            {filteredProducts.length} Produkte • {totalDeviceCount} Geräte
          </p>
        </div>
      </div>

      {/* Search */}
      <div className="glass rounded-xl sm:rounded-2xl p-3 sm:p-4">
        <div className="relative">
          <Search className="absolute left-3 sm:left-4 top-1/2 -translate-y-1/2 w-4 h-4 sm:w-5 sm:h-5 text-gray-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Produkte oder Geräte suchen..."
            className="w-full pl-10 sm:pl-12 pr-3 sm:pr-4 py-2.5 sm:py-3 bg-white/10 backdrop-blur-md border border-white/20 rounded-lg sm:rounded-xl text-sm sm:text-base text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
        </div>
      </div>

      {/* Products Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {filteredProducts.map((group) => {
          const inStorage = group.devices.filter((device) => device.status === 'in_storage').length;
          const onJob = group.devices.filter((device) => device.status === 'on_job' || device.status === 'rented').length;
          const defective = group.devices.filter((device) => device.status === 'defective').length;

          return (
            <div
              key={group.key}
              onClick={() => setActiveProductKey(group.key)}
              className={`glass-dark rounded-lg sm:rounded-xl p-4 sm:p-5 transition-all cursor-pointer group border border-white/10 ${
                activeProductKey === group.key ? 'ring-2 ring-accent-red bg-white/10' : 'hover:bg-white/10'
              }`}
            >
              <div className="flex items-start gap-3 sm:gap-4">
                <div className="p-2 sm:p-3 rounded-lg bg-gradient-to-br from-accent-red/20 to-red-700/20 group-hover:from-accent-red/30 group-hover:to-red-700/30 transition-colors flex-shrink-0">
                  <Package className="w-5 h-5 sm:w-6 sm:h-6 text-accent-red" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="text-sm sm:text-base font-bold text-white truncate mb-0.5 sm:mb-1">
                    {group.productName}
                  </h3>
                  <p className="text-xs sm:text-sm text-gray-400 mb-1.5 sm:mb-2 truncate">
                    {group.devices.length} Gerät{group.devices.length === 1 ? '' : 'e'}
                  </p>
                  <div className="flex items-center gap-2 text-[10px] sm:text-xs text-gray-500 flex-wrap">
                    <span>📦 Lager: {inStorage}</span>
                    <span>🚚 Auf Job: {onJob}</span>
                    <span>⚠️ Defekt: {defective}</span>
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {filteredProducts.length === 0 && (
        <div className="text-center py-8 sm:py-12">
          <Package className="w-12 h-12 sm:w-16 sm:h-16 text-gray-600 mx-auto mb-3 sm:mb-4" />
          <p className="text-sm sm:text-base text-gray-400">Keine Produkte gefunden</p>
        </div>
      )}

      {activeProduct && (
        <div className="glass-dark rounded-xl sm:rounded-2xl p-5 sm:p-6 space-y-5 border border-white/10">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div>
              <h3 className="text-lg sm:text-xl font-bold text-white mb-1">{activeProduct.productName}</h3>
              <p className="text-sm text-gray-400">
                {activeProduct.devices.length} Gerät{activeProduct.devices.length === 1 ? '' : 'e'} in diesem Produkt
              </p>
            </div>
          </div>

          {actionMessage && (
            <div
              className={`px-3 py-2 rounded-lg text-sm font-semibold ${
                actionMessage.type === 'success'
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-red-500/20 text-red-400'
              }`}
            >
              {actionMessage.text}
            </div>
          )}

          <div className="space-y-3">
            {activeProduct.devices.map((device) => (
              <div
                key={device.device_id}
                onClick={() => handleOpenDevice(device)}
                className="glass rounded-lg p-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border border-white/10 cursor-pointer hover:bg-white/10 transition-colors"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-semibold text-white text-sm sm:text-base truncate">{device.device_id}</span>
                    <span className={`text-[10px] sm:text-xs font-semibold px-2 py-1 rounded-full ${getStatusColor(device.status)} bg-white/10`}>
                      {formatStatus(device.status)}
                    </span>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 text-xs sm:text-sm text-gray-400 mt-1">
                    {device.zone_name && (
                      <span className="flex items-center gap-1">
                        <MapPin className="w-3 h-3" />
                        {device.zone_name}
                      </span>
                    )}
                    {device.zone_code && (
                      <span className="text-gray-500 font-mono">{device.zone_code}</span>
                    )}
                    {device.serial_number && (
                      <span className="text-gray-500">SN: {device.serial_number}</span>
                    )}
                  </div>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleOpenDevice(device);
                    }}
                    className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-semibold bg-white/10 hover:bg-white/20 transition-colors text-white"
                  >
                    <Package className="w-4 h-4" /> Details
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleLocateDevice(device);
                    }}
                    className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-semibold bg-white/10 hover:bg-white/20 transition-colors"
                  >
                    <Lightbulb className="w-4 h-4 text-yellow-300" /> Fach aufleuchten
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleOpenZone(device);
                    }}
                    className="flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-semibold bg-accent-red/80 hover:bg-accent-red transition-colors text-white"
                  >
                    <MapPin className="w-4 h-4" /> Zone öffnen
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <DeviceDetailModal device={selectedDevice} isOpen={detailOpen} onClose={handleCloseDevice} />
    </div>
  );
}
