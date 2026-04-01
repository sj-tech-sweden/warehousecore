import { X, Package, MapPin, Barcode, Hash, Activity, Wrench, Lightbulb, LightbulbOff, Tag, Download, CalendarDays, FileText, Wifi } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ledApi } from '../lib/api';
import type { Device } from '../lib/api';
import { useState, useEffect, useMemo } from 'react';
import { ModalPortal } from './ModalPortal';
import { useBlockBodyScroll } from '../hooks/useBlockBodyScroll';

interface DeviceDetailModalProps {
  device: Device | null;
  isOpen: boolean;
  onClose: () => void;
}

export function DeviceDetailModal({ device, isOpen, onClose }: DeviceDetailModalProps) {
  const { t } = useTranslation();
  const [locating, setLocating] = useState(false);
  const [locateMessage, setLocateMessage] = useState<string | null>(null);
  const [ledActive, setLedActive] = useState(false);

  // Block body scroll when modal is open
  useBlockBodyScroll(isOpen);

  // Cleanup LEDs when modal closes
  useEffect(() => {
    if (!isOpen && ledActive) {
      ledApi.clear().catch(err => console.error('Failed to clear LEDs:', err));
      setLedActive(false);
    }
  }, [isOpen, ledActive]);

  // Handle ESC key to close modal
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  // Cache-busting for label image - regenerates URL when label_path changes
  const labelUrl = useMemo(() => {
    if (!device?.label_path) return null;
    return `${device.label_path}?t=${Date.now()}`;
  }, [device?.label_path]);

  if (!isOpen || !device) return null;

  const handleLocate = async () => {
    if (!device.zone_code) {
      setLocateMessage(t('modals.deviceDetail.noStorageLocation'));
      setTimeout(() => setLocateMessage(null), 3000);
      return;
    }

    setLocating(true);
    setLocateMessage(null);

    try {
      await ledApi.locateBin(device.zone_code);
      setLedActive(true);
      setLocateMessage(t('modals.deviceDetail.locateSuccess', { code: device.zone_code }));
      setTimeout(() => setLocateMessage(null), 5000);
    } catch (error: any) {
      console.error('Failed to locate bin:', error);
      setLocateMessage(t('modals.deviceDetail.locateError'));
      setTimeout(() => setLocateMessage(null), 3000);
    } finally {
      setLocating(false);
    }
  };

  const handleClearLED = async () => {
    setLocating(true);
    setLocateMessage(null);

    try {
      await ledApi.clear();
      setLedActive(false);
      setLocateMessage(t('modals.deviceDetail.clearSuccess'));
      setTimeout(() => setLocateMessage(null), 3000);
    } catch (error: any) {
      console.error('Failed to clear LEDs:', error);
      setLocateMessage(t('modals.deviceDetail.clearError'));
      setTimeout(() => setLocateMessage(null), 3000);
    } finally {
      setLocating(false);
    }
  };

  const statusColors: Record<string, { bg: string; text: string }> = {
    in_storage: { bg: 'bg-green-500/20', text: 'text-green-400' },
    on_job: { bg: 'bg-blue-500/20', text: 'text-blue-400' },
    rented: { bg: 'bg-yellow-500/20', text: 'text-yellow-400' },
    defective: { bg: 'bg-red-500/20', text: 'text-red-400' },
    in_transit: { bg: 'bg-purple-500/20', text: 'text-purple-400' },
  };

  const statusColor = statusColors[device.status] || { bg: 'bg-gray-500/20', text: 'text-gray-400' };

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
        <div className="glass-dark rounded-2xl w-full max-w-2xl shadow-2xl max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-white/10">
          <h2 className="text-2xl font-bold text-white flex items-center gap-2">
            <Package className="w-6 h-6 text-accent-red" />
            {t('modals.deviceDetail.title')}
          </h2>
          <button
            onClick={onClose}
            className="p-2 hover:bg-white/10 rounded-lg transition-colors"
            aria-label={t('common.close')}
            title={t('common.close')}
          >
            <X className="w-5 h-5 text-gray-400" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Product Name */}
          <div>
            <h3 className="text-3xl font-bold text-white mb-2">
              {device.product_name || t('modals.deviceDetail.unknownDevice')}
            </h3>
            <div className="flex items-center gap-2">
              <span className={`px-3 py-1 rounded-full text-sm font-semibold ${statusColor.bg} ${statusColor.text}`}>
                {device.status}
              </span>
            </div>
          </div>

          {/* Details Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Device ID */}
            <div className="glass rounded-xl p-4">
              <div className="flex items-center gap-3">
                <Hash className="w-5 h-5 text-gray-400" />
                <div>
                  <p className="text-xs text-gray-400">{t('devices.deviceId')}</p>
                  <p className="text-white font-mono font-semibold">{device.device_id}</p>
                </div>
              </div>
            </div>

            {/* Serial Number */}
            {device.serial_number && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <Hash className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('devices.serialNumber')}</p>
                    <p className="text-white font-mono font-semibold">{device.serial_number}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Barcode */}
            {device.barcode && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <Barcode className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('devices.barcode')}</p>
                    <p className="text-white font-mono font-semibold">{device.barcode}</p>
                  </div>
                </div>
              </div>
            )}

            {/* QR Code */}
            {device.qr_code && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <Barcode className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('labels.qrCode')}</p>
                    <p className="text-white font-mono font-semibold">{device.qr_code}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Location */}
            {device.zone_name && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <MapPin className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('devices.zone')}</p>
                    <p className="text-white font-semibold">{device.zone_name}</p>
                    {device.zone_code && (
                      <p className="text-xs text-gray-500 font-mono">{device.zone_code}</p>
                    )}
                  </div>
                </div>
              </div>
            )}

            {/* Case */}
            {device.case_name && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <Package className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('devices.case')}</p>
                    <p className="text-white font-semibold">{device.case_name}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Job */}
            {device.job_number && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <Activity className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('devices.job')}</p>
                    <p className="text-white font-semibold">#{device.job_number}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Condition */}
            <div className="glass rounded-xl p-4">
              <div className="flex items-center gap-3">
                <Wrench className="w-5 h-5 text-gray-400" />
                <div>
                  <p className="text-xs text-gray-400">{t('devices.condition')}</p>
                  <p className="text-white font-semibold">{device.condition_rating}/10</p>
                </div>
              </div>
            </div>

            {/* Usage Hours */}
            <div className="glass rounded-xl p-4">
              <div className="flex items-center gap-3">
                <Activity className="w-5 h-5 text-gray-400" />
                <div>
                  <p className="text-xs text-gray-400">{t('devices.usageHours')}</p>
                  <p className="text-white font-semibold">{device.usage_hours}h</p>
                </div>
              </div>
            </div>

            {/* RFID */}
            {device.rfid && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <Wifi className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('devices.rfid')}</p>
                    <p className="text-white font-mono font-semibold">{device.rfid}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Purchase Date */}
            {device.purchase_date && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <CalendarDays className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('admin.devices.purchaseDate')}</p>
                    <p className="text-white font-semibold">{device.purchase_date}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Warranty End Date */}
            {device.warranty_end_date && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <CalendarDays className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('admin.devices.warrantyEndDate')}</p>
                    <p className="text-white font-semibold">{device.warranty_end_date}</p>
                  </div>
                </div>
              </div>
            )}

            {/* Retire Date */}
            {device.retire_date && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-3">
                  <CalendarDays className="w-5 h-5 text-gray-400" />
                  <div>
                    <p className="text-xs text-gray-400">{t('admin.devices.retireDate')}</p>
                    <p className="text-white font-semibold">{device.retire_date}</p>
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Notes */}
          {device.notes && (
            <div className="glass rounded-xl p-4">
              <div className="flex items-start gap-3">
                <FileText className="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0" />
                <div>
                  <p className="text-xs text-gray-400 mb-1">{t('modals.productDependencies.notes')}</p>
                  <p className="text-white text-sm whitespace-pre-wrap">{device.notes}</p>
                </div>
              </div>
            </div>
          )}

          {/* LED Control Buttons */}
          {device.zone_code && (
            <div className="pt-4 border-t border-white/10">
              <div className="flex gap-3">
                {/* Locate Button */}
                <button
                  onClick={handleLocate}
                  disabled={locating || ledActive}
                  className={`flex-1 py-3 px-4 rounded-xl font-semibold text-white transition-all flex items-center justify-center gap-2 ${
                    locating || ledActive
                      ? 'bg-gray-600 cursor-not-allowed opacity-50'
                      : 'bg-gradient-to-r from-orange-600 to-orange-700 hover:shadow-lg hover:shadow-orange-500/50 hover:scale-105 active:scale-95'
                  }`}
                >
                  <Lightbulb className="w-5 h-5" />
                  <span>{locating ? t('modals.deviceDetail.locating') : t('modals.deviceDetail.highlightBin')}</span>
                </button>

                {/* Clear LEDs Button */}
                {ledActive && (
                  <button
                    onClick={handleClearLED}
                    disabled={locating}
                    className={`flex-1 py-3 px-4 rounded-xl font-semibold text-white transition-all flex items-center justify-center gap-2 ${
                      locating
                        ? 'bg-gray-600 cursor-not-allowed opacity-50'
                        : 'bg-gradient-to-r from-red-600 to-red-700 hover:shadow-lg hover:shadow-red-500/50 hover:scale-105 active:scale-95'
                    }`}
                  >
                    <LightbulbOff className="w-5 h-5" />
                    <span>{t('modals.deviceDetail.turnOff')}</span>
                  </button>
                )}
              </div>

              {/* Locate Message */}
              {locateMessage && (
                <div className={`mt-3 p-3 rounded-lg text-center text-sm font-semibold ${
                  locateMessage.includes('✓')
                    ? 'bg-green-500/20 text-green-400'
                    : 'bg-red-500/20 text-red-400'
                }`}>
                  {locateMessage}
                </div>
              )}
            </div>
          )}

          {/* Label Preview Section */}
          {labelUrl && (
            <div className="pt-4 border-t border-white/10">
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-lg font-semibold text-white flex items-center gap-2">
                  <Tag className="w-5 h-5 text-accent-red" />
                  {t('modals.deviceDetail.deviceLabel')}
                </h3>
                <a
                  href={labelUrl}
                  download={`${device.device_id}_label.png`}
                  className="px-4 py-2 rounded-xl font-semibold text-white bg-gradient-to-r from-blue-600 to-blue-700 hover:shadow-lg hover:shadow-blue-500/50 hover:scale-105 active:scale-95 transition-all flex items-center gap-2"
                >
                  <Download className="w-4 h-4" />
                  {t('modals.caseDetail.download')}
                </a>
              </div>
              <div className="flex justify-center p-4 bg-black/20 rounded-xl">
                <img
                  src={labelUrl}
                  alt={t('modals.deviceDetail.labelAlt', { id: device.device_id })}
                  className="max-w-sm h-auto border border-white/10 rounded shadow-lg"
                />
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 p-6 border-t border-white/10">
          <button
            onClick={onClose}
            className="px-6 py-2.5 glass hover:bg-white/10 text-white font-semibold rounded-xl transition-all"
          >
            {t('common.close')}
          </button>
        </div>
        </div>
      </div>
    </ModalPortal>
  );
}
