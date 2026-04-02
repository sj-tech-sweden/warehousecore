import { Download, Pencil, QrCode, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { Device } from '../lib/api';
import { devicesAdminApi } from '../lib/api';
import { ModalPortal } from './ModalPortal';
import { useBlockBodyScroll } from '../hooks/useBlockBodyScroll';
import { useEffect } from 'react';

interface DeviceInfoModalProps {
  device: Device | null;
  isOpen: boolean;
  onClose: () => void;
  /** Called when the Edit button is clicked. If omitted, the Edit button is hidden. */
  onEdit?: (device: Device) => void;
}

function statusLabel(t: (key: string, fallback: string) => string, status?: string) {
  if (!status) return '-';
  return t(`admin.devices.statuses.${status}`, status);
}

export function DeviceInfoModal({ device, isOpen, onClose, onEdit }: DeviceInfoModalProps) {
  const { t } = useTranslation();

  useBlockBodyScroll(isOpen);

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) onClose();
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, onClose]);

  if (!isOpen || !device) return null;

  const downloadQR = () => window.open(devicesAdminApi.downloadQR(device.device_id), '_blank', 'noopener,noreferrer');
  const downloadBarcode = () => window.open(devicesAdminApi.downloadBarcode(device.device_id), '_blank', 'noopener,noreferrer');
  const openLabel = () => { if (device.label_path) window.open(device.label_path, '_blank', 'noopener,noreferrer'); };

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
        <div className="glass-dark rounded-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-2xl font-bold text-white">{t('modals.deviceDetail.title')}</h3>
            <button
              onClick={onClose}
              className="p-2 hover:bg-white/10 rounded-lg transition-colors"
              aria-label={t('common.close')}
              title={t('common.close')}
            >
              <X className="w-6 h-6 text-gray-400" />
            </button>
          </div>

          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <p className="text-sm text-gray-400">{t('devices.deviceId')}</p>
                <p className="text-white font-semibold">{device.device_id}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('zoneDetail.columns.product')}</p>
                <p className="text-white font-semibold">{device.product_name || '-'}</p>
              </div>
              {device.product_category && (
                <div>
                  <p className="text-sm text-gray-400">{t('admin.tabs.categories')}</p>
                  <p className="text-white font-semibold">{device.product_category}</p>
                </div>
              )}
              <div>
                <p className="text-sm text-gray-400">{t('devices.status')}</p>
                <p className="text-white font-semibold">{statusLabel((key, fb) => t(key, fb), device.status)}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('devices.serialNumber')}</p>
                <p className="text-white font-semibold">{device.serial_number || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('devices.rfid')}</p>
                <p className="text-white font-semibold">{device.rfid || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('devices.barcode')}</p>
                <p className="text-white font-semibold">{device.barcode || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('labels.qrCode')}</p>
                <p className="text-white font-semibold">{device.qr_code || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('devices.zone')}</p>
                <p className="text-white font-semibold">
                  {device.zone_code ? `${device.zone_code} - ${device.zone_name}` : '-'}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('admin.devices.location')}</p>
                <p className="text-white font-semibold">{device.current_location || '-'}</p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('devices.condition')}</p>
                <p className="text-white font-semibold">
                  {device.condition_rating ? `${device.condition_rating}/10` : '-'}
                </p>
              </div>
              <div>
                <p className="text-sm text-gray-400">{t('devices.usageHours')}</p>
                <p className="text-white font-semibold">
                  {device.usage_hours ? `${device.usage_hours}h` : '-'}
                </p>
              </div>
              {device.purchase_date && (
                <div>
                  <p className="text-sm text-gray-400">{t('admin.devices.purchaseDate')}</p>
                  <p className="text-white font-semibold">{device.purchase_date}</p>
                </div>
              )}
              {device.warranty_end_date && (
                <div>
                  <p className="text-sm text-gray-400">{t('admin.devices.warrantyEndDate')}</p>
                  <p className="text-white font-semibold">{device.warranty_end_date}</p>
                </div>
              )}
              {device.retire_date && (
                <div>
                  <p className="text-sm text-gray-400">{t('admin.devices.retireDate')}</p>
                  <p className="text-white font-semibold">{device.retire_date}</p>
                </div>
              )}
              {device.last_maintenance && (
                <div>
                  <p className="text-sm text-gray-400">{t('admin.devices.lastMaintenance')}</p>
                  <p className="text-white font-semibold">{device.last_maintenance}</p>
                </div>
              )}
              {device.next_maintenance && (
                <div>
                  <p className="text-sm text-gray-400">{t('admin.devices.nextMaintenance')}</p>
                  <p className="text-white font-semibold">{device.next_maintenance}</p>
                </div>
              )}
              {device.case_name && (
                <div>
                  <p className="text-sm text-gray-400">{t('devices.case')}</p>
                  <p className="text-white font-semibold">{device.case_name}</p>
                </div>
              )}
              {device.job_number && (
                <div>
                  <p className="text-sm text-gray-400">{t('devices.job')}</p>
                  <p className="text-white font-semibold">{device.job_number}</p>
                </div>
              )}
            </div>

            {device.notes && (
              <div className="border border-white/10 rounded-xl p-4 text-sm text-gray-300 bg-white/5">
                <p className="font-semibold text-white mb-1">{t('modals.productDependencies.notes')}</p>
                <p className="whitespace-pre-line">{device.notes}</p>
              </div>
            )}

            <div className="flex flex-wrap gap-3 pt-4">
              <button
                onClick={downloadQR}
                className="flex-1 px-4 py-3 bg-white/5 hover:bg-white/10 rounded-lg font-semibold text-gray-300 transition-colors flex items-center justify-center gap-2"
              >
                <QrCode className="w-5 h-5" />
                {t('labels.qrCode')}
              </button>
              <button
                onClick={downloadBarcode}
                className="flex-1 px-4 py-3 bg-white/5 hover:bg-white/10 rounded-lg font-semibold text-gray-300 transition-colors flex items-center justify-center gap-2"
              >
                <Download className="w-5 h-5" />
                {t('devices.barcode')}
              </button>
              {device.label_path && (
                <button
                  onClick={openLabel}
                  className="flex-1 px-4 py-3 bg-white/5 hover:bg-white/10 rounded-lg font-semibold text-gray-300 transition-colors flex items-center justify-center gap-2"
                >
                  <Download className="w-5 h-5" />
                  {t('nav.labels')}
                </button>
              )}
              {onEdit && (
                <button
                  onClick={() => { onClose(); onEdit(device); }}
                  className="flex-1 btn-primary flex items-center justify-center gap-2"
                >
                  <Pencil className="w-5 h-5" />
                  {t('common.edit')}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
