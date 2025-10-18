import { useEffect, useState } from 'react';
import { Package, Search } from 'lucide-react';
import { devicesApi } from '../lib/api';
import type { Device } from '../lib/api';
import { getStatusColor, formatStatus } from '../lib/utils';
import { DeviceDetailModal } from '../components/DeviceDetailModal';

export function DevicesPage() {
  const [devices, setDevices] = useState<Device[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);

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

  const handleDeviceClick = (device: Device) => {
    setSelectedDevice(device);
    setIsDetailModalOpen(true);
  };

  const filteredDevices = devices.filter((device) =>
    device.device_id.toLowerCase().includes(search.toLowerCase()) ||
    device.product_name?.toLowerCase().includes(search.toLowerCase())
  );

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
          <h2 className="text-2xl sm:text-3xl font-bold text-white mb-1 sm:mb-2">Geräte</h2>
          <p className="text-sm sm:text-base text-gray-400">{filteredDevices.length} Geräte gefunden</p>
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
            placeholder="Geräte suchen..."
            className="w-full pl-10 sm:pl-12 pr-3 sm:pr-4 py-2.5 sm:py-3 bg-white/10 backdrop-blur-md border border-white/20 rounded-lg sm:rounded-xl text-sm sm:text-base text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
        </div>
      </div>

      {/* Devices Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 sm:gap-4">
        {filteredDevices.map((device) => (
          <div
            key={device.device_id}
            onClick={() => handleDeviceClick(device)}
            className="glass-dark rounded-lg sm:rounded-xl p-4 sm:p-5 hover:bg-white/10 transition-all cursor-pointer group"
          >
            <div className="flex items-start gap-3 sm:gap-4">
              <div className="p-2 sm:p-3 rounded-lg bg-gradient-to-br from-accent-red/20 to-red-700/20 group-hover:from-accent-red/30 group-hover:to-red-700/30 transition-colors flex-shrink-0">
                <Package className="w-5 h-5 sm:w-6 sm:h-6 text-accent-red" />
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-sm sm:text-base font-bold text-white truncate mb-0.5 sm:mb-1">
                  {device.product_name || 'Unbekannt'}
                </h3>
                <p className="text-xs sm:text-sm text-gray-400 mb-1.5 sm:mb-2 truncate">{device.device_id}</p>
                <div className="flex items-center gap-1.5 sm:gap-2 flex-wrap">
                  <span className={`text-[10px] sm:text-xs font-semibold px-1.5 sm:px-2 py-0.5 sm:py-1 rounded-full ${
                    getStatusColor(device.status)
                  } bg-white/10`}>
                    {formatStatus(device.status)}
                  </span>
                  {device.zone_name && (
                    <span className="text-[10px] sm:text-xs text-gray-500 truncate">📍 {device.zone_name}</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {filteredDevices.length === 0 && (
        <div className="text-center py-8 sm:py-12">
          <Package className="w-12 h-12 sm:w-16 sm:h-16 text-gray-600 mx-auto mb-3 sm:mb-4" />
          <p className="text-sm sm:text-base text-gray-400">Keine Geräte gefunden</p>
        </div>
      )}

      {/* Device Detail Modal */}
      <DeviceDetailModal
        device={selectedDevice}
        isOpen={isDetailModalOpen}
        onClose={() => setIsDetailModalOpen(false)}
      />
    </div>
  );
}
