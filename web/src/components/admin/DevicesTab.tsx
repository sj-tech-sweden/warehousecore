import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  CheckSquare,
  Download,
  Eye,
  LayoutGrid,
  List,
  MinusSquare,
  Package,
  Pencil,
  Plus,
  QrCode,
  RefreshCcw,
  ScanLine,
  Search,
  Square,
  Trash2,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { api, devicesAdminApi, labelsApi } from '../../lib/api';
import type { Device, DeviceCreateInput, DeviceUpdateInput, LabelTemplate } from '../../lib/api';
import { useBlockBodyScroll } from '../../hooks/useBlockBodyScroll';
import { ModalPortal } from '../ModalPortal';
import { DeviceInfoModal } from '../DeviceInfoModal';
import { SearchableSelect } from '../SearchableSelect';
import { ScanFieldModal } from '../ScanFieldModal';

interface Product {
  product_id: number;
  name: string;
  category_name?: string | null;
  subcategory_name?: string | null;
}

interface Zone {
  zone_id: number;
  code: string;
  name: string;
}

interface DeviceFormData {
  new_device_id: string;
  product_id?: number;
  status: string;
  serial_number: string;
  rfid: string;
  barcode: string;
  qr_code: string;
  current_location: string;
  zone_id?: number;
  condition_rating?: number;
  usage_hours?: number;
  purchase_date: string;
  retire_date: string;
  warranty_end_date: string;
  last_maintenance: string;
  next_maintenance: string;
  notes: string;
  quantity: number;
  device_prefix: string;
  increment_serial: boolean;
  auto_generate_label: boolean;
  label_template_id?: number;
  regenerate_codes: boolean;
  regenerate_label: boolean;
}

const initialFormData: DeviceFormData = {
  new_device_id: '',
  status: 'in_storage',
  serial_number: '',
  rfid: '',
  barcode: '',
  qr_code: '',
  current_location: '',
  notes: '',
  quantity: 1,
  device_prefix: '',
  increment_serial: false,
  purchase_date: '',
  retire_date: '',
  warranty_end_date: '',
  last_maintenance: '',
  next_maintenance: '',
  auto_generate_label: true,
  regenerate_codes: false,
  regenerate_label: false,
};

function useDebouncedValue<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(handle);
  }, [value, delay]);

  return debounced;
}

interface DevicesTabProps {
  initialProductFilter?: number;
  initialEditDeviceId?: string;
  onEditComplete?: () => void;
  onCancel?: () => void;
}

export function DevicesTab({ initialProductFilter, initialEditDeviceId, onEditComplete, onCancel }: DevicesTabProps) {
  const { t } = useTranslation();
  const [devices, setDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewDevice, setViewDevice] = useState<Device | null>(null);
  const [editingDevice, setEditingDevice] = useState<string | null>(null);
  const [formData, setFormData] = useState<DeviceFormData>(initialFormData);
  const [products, setProducts] = useState<Product[]>([]);
  const [zones, setZones] = useState<Zone[]>([]);
  const [labelTemplates, setLabelTemplates] = useState<LabelTemplate[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'cards'>('table');
  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [productFilter, setProductFilter] = useState<number | ''>(initialProductFilter ?? '');
  const [zoneFilter, setZoneFilter] = useState<number | ''>('');
  const [refreshing, setRefreshing] = useState(false);
  const [scanFieldTarget, setScanFieldTarget] = useState<keyof Pick<DeviceFormData, 'rfid' | 'serial_number' | 'barcode' | 'qr_code'> | null>(null);
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set());
  const [bulkEditOpen, setBulkEditOpen] = useState(false);
  const [bulkEditData, setBulkEditData] = useState<{ status?: string; zone_id?: number }>({});

  // Block body scroll when any modal is open
  useBlockBodyScroll(modalOpen || bulkEditOpen);

  const debouncedSearch = useDebouncedValue(searchTerm, 300);

  const fetchDevices = useCallback(async () => {
    setLoadingDevices(true);
    try {
      const { data } = await api.get<Device[]>('/admin/devices-list');
      setDevices(data || []);
    } catch (error) {
      console.error('Failed to load devices:', error);
      setDevices([]);
    } finally {
      setLoadingDevices(false);
    }
  }, []);

  const loadMetadata = useCallback(async () => {
    try {
      const [productsRes, zonesRes, templatesRes] = await Promise.all([
        api.get<Product[]>('/admin/products'),
        api.get<Zone[]>('/zones'),
        labelsApi.getTemplates(),
      ]);

      setProducts(productsRes.data || []);
      setZones(zonesRes.data || []);
      setLabelTemplates(templatesRes.data || []);
    } catch (error) {
      console.error('Failed to load metadata:', error);
    }
  }, []);

  useEffect(() => {
    fetchDevices();
    loadMetadata();
    if (initialEditDeviceId) {
      // open edit modal for device navigated from scan page
      (async () => {
        try {
          const { data } = await devicesAdminApi.getById(initialEditDeviceId);
          openEditModal(data);
        } catch (err) {
          console.error('Failed to open device for editing:', err);
        }
      })();
    }
  }, [fetchDevices, loadMetadata]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    await fetchDevices();
    setRefreshing(false);
  }, [fetchDevices]);

  const clearFilters = () => {
    setSearchTerm('');
    setStatusFilter('');
    setProductFilter('');
    setZoneFilter('');
  };

  const normalizeDeviceStatus = (status: string) => {
    const normalizedStatus = status.trim().toLowerCase();
    return normalizedStatus === 'free' ? 'in_storage' : normalizedStatus;
  };

  const filteredDevices = devices.filter((device) => {
    const matchesSearch =
      !debouncedSearch ||
      device.device_id.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
      device.product_name?.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
      device.product_category?.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
      device.serial_number?.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
      device.rfid?.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
      device.barcode?.toLowerCase().includes(debouncedSearch.toLowerCase()) ||
      device.notes?.toLowerCase().includes(debouncedSearch.toLowerCase());

    const matchesStatus = !statusFilter || normalizeDeviceStatus(device.status) === statusFilter;
    const matchesProduct =
      productFilter === '' || device.product_id === productFilter;
    const matchesZone = zoneFilter === '' || device.zone_id === zoneFilter;

    return matchesSearch && matchesStatus && matchesProduct && matchesZone;
  });

  // Clear selection when filters/search change so invisible items aren't left selected
  useEffect(() => {
    setSelectedDevices(new Set());
  }, [debouncedSearch, statusFilter, productFilter, zoneFilter]);

  // Prune stale selections when the device list changes (e.g. after refresh)
  useEffect(() => {
    setSelectedDevices(prev => {
      if (prev.size === 0) return prev;
      const currentIDs = new Set(filteredDevices.map(d => d.device_id));
      const pruned = new Set([...prev].filter(id => currentIDs.has(id)));
      return pruned.size === prev.size ? prev : pruned;
    });
  }, [filteredDevices]);

  const statusLabel = (status?: string) => {
    if (!status) return '-';
    return t(`admin.devices.statuses.${normalizeDeviceStatus(status)}`, status);
  };

  const openCreateModal = () => {
    setEditingDevice(null);
    setFormData({ ...initialFormData, label_template_id: undefined });
    setModalOpen(true);
  };

  const openEditModal = (device: Device) => {
    setEditingDevice(device.device_id);
    setFormData({
      new_device_id: device.device_id,
      product_id: device.product_id,
      status: normalizeDeviceStatus(device.status || 'in_storage'),
      serial_number: device.serial_number || '',
      rfid: device.rfid || '',
      barcode: device.barcode || '',
      qr_code: device.qr_code || '',
      current_location: device.current_location || '',
      zone_id: device.zone_id,
      condition_rating: device.condition_rating,
      usage_hours: device.usage_hours,
      purchase_date: device.purchase_date || '',
      retire_date: device.retire_date || '',
      warranty_end_date: device.warranty_end_date || '',
      last_maintenance: device.last_maintenance || '',
      next_maintenance: device.next_maintenance || '',
      notes: device.notes || '',
      quantity: 1,
      device_prefix: '',
      increment_serial: false,
      auto_generate_label: true,
      regenerate_codes: false,
      regenerate_label: false,
      label_template_id: undefined,
    });
    setModalOpen(true);
  };

  const handleDelete = async (deviceId: string) => {
    if (!window.confirm(t('admin.devices.confirmDelete'))) {
      return;
    }

    try {
      await devicesAdminApi.delete(deviceId);
      await fetchDevices();
    } catch (error: unknown) {
      console.error('Failed to delete device:', error);
      alert(t('admin.devices.errors.delete'));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);

    try {
      if (editingDevice) {
        // Update existing device
        const updateData: DeviceUpdateInput = {
          product_id: formData.product_id,
          status: formData.status,
          serial_number: formData.serial_number || undefined,
          rfid: formData.rfid || undefined,
          barcode: formData.barcode || undefined,
          qr_code: formData.qr_code || undefined,
          current_location: formData.current_location || undefined,
          zone_id: formData.zone_id,
          condition_rating: formData.condition_rating,
          usage_hours: formData.usage_hours,
          notes: formData.notes || undefined,
          purchase_date: formData.purchase_date || undefined,
          retire_date: formData.retire_date || undefined,
          warranty_end_date: formData.warranty_end_date || undefined,
          last_maintenance: formData.last_maintenance || undefined,
          next_maintenance: formData.next_maintenance || undefined,
        };

        // Include new device ID if changed
        if (formData.new_device_id && formData.new_device_id !== editingDevice) {
          updateData.new_device_id = formData.new_device_id;
        }

        if (formData.regenerate_codes) {
          updateData.regenerate_codes = true;
        }
        if (formData.label_template_id) {
          updateData.label_template_id = formData.label_template_id;
        }
        if (formData.regenerate_label || formData.label_template_id) {
          updateData.regenerate_label = true;
        }

        await devicesAdminApi.update(editingDevice, updateData);
      } else {
        // Create new device(s)
        const createData: DeviceCreateInput = {
          product_id: formData.product_id!,
          status: formData.status,
          serial_number: formData.serial_number || undefined,
          rfid: formData.rfid || undefined,
          barcode: formData.barcode || undefined,
          qr_code: formData.qr_code || undefined,
          current_location: formData.current_location || undefined,
          zone_id: formData.zone_id,
          condition_rating: formData.condition_rating,
          usage_hours: formData.usage_hours,
          notes: formData.notes || undefined,
          purchase_date: formData.purchase_date || undefined,
          retire_date: formData.retire_date || undefined,
          warranty_end_date: formData.warranty_end_date || undefined,
          last_maintenance: formData.last_maintenance || undefined,
          next_maintenance: formData.next_maintenance || undefined,
          quantity: formData.quantity,
          device_prefix: formData.device_prefix || undefined,
          increment_serial: formData.increment_serial,
        };

        createData.auto_generate_label = formData.auto_generate_label;
        if (formData.label_template_id) {
          createData.label_template_id = formData.label_template_id;
        }
        if (formData.regenerate_codes) {
          createData.regenerate_codes = true;
        }

        await devicesAdminApi.create(createData);
      }

      setModalOpen(false);
      setScanFieldTarget(null);
      setFormData({ ...initialFormData, label_template_id: undefined });

      // If an onEditComplete callback is provided (navigated here from Scan), call it
      // instead of fetching devices (component may unmount after navigation).
      if (onEditComplete) {
        onEditComplete();
        return;
      }

      await fetchDevices();
    } catch (error: unknown) {
      console.error('Failed to save device:', error);
      alert(t('admin.devices.errors.save'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleViewDevice = async (device: Device) => {
    try {
      const { data } = await devicesAdminApi.getById(device.device_id);
      setViewDevice(data);
    } catch (error) {
      console.error('Failed to load device details:', error);
      setViewDevice(device);
    }
  };

  const downloadQR = (deviceId: string) => {
    window.open(devicesAdminApi.downloadQR(deviceId), '_blank');
  };

  const downloadBarcode = (deviceId: string) => {
    window.open(devicesAdminApi.downloadBarcode(deviceId), '_blank');
  };

  const toggleDeviceSelection = (deviceId: string) => {
    setSelectedDevices(prev => {
      const next = new Set(prev);
      if (next.has(deviceId)) {
        next.delete(deviceId);
      } else {
        next.add(deviceId);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (selectedDevices.size === filteredDevices.length) {
      setSelectedDevices(new Set());
    } else {
      setSelectedDevices(new Set(filteredDevices.map(d => d.device_id)));
    }
  };

  const allFilteredSelected = useMemo(
    () => filteredDevices.length > 0 && selectedDevices.size === filteredDevices.length,
    [filteredDevices, selectedDevices],
  );

  const handleBulkDelete = async () => {
    if (selectedDevices.size === 0) return;
    if (!window.confirm(t('admin.devices.confirmBulkDelete', { count: selectedDevices.size }))) return;

    try {
      const { data } = await devicesAdminApi.bulkDelete(Array.from(selectedDevices));
      if (data.failed_devices > 0) {
        // Build per-device error details from the structured failed_errors map when available
        let failedDetails = '';
        if (data.failed_errors && typeof data.failed_errors === 'object') {
          const entries = Object.entries(data.failed_errors as Record<string, string>)
            .sort(([a], [b]) => a.localeCompare(b));
          failedDetails = '\n' + entries.map(([id, reason]) => `${id}: ${reason}`).join('\n');
        } else if (data.failed_ids && data.failed_ids.length > 0) {
          failedDetails = `\n${t('admin.devices.failedIds')}: ${data.failed_ids.join(', ')}`;
        }
        if (data.deleted_devices === 0) {
          alert(data.message + failedDetails);
        } else {
          alert(t('admin.devices.bulkDeletePartial', { deleted: data.deleted_devices, failed: data.failed_devices }) + failedDetails);
        }
      } else {
        alert(t('admin.devices.bulkDeleteSuccess', { count: data.deleted_devices }));
      }
      setSelectedDevices(new Set());
      await fetchDevices();
    } catch (error) {
      console.error('Failed to bulk delete devices:', error);
      const axiosErr = error as { response?: { data?: { error?: string; message?: string } } };
      const errorMessage =
        axiosErr?.response?.data?.error ||
        axiosErr?.response?.data?.message ||
        t('admin.devices.errors.bulkDelete');
      alert(errorMessage);
    }
  };

  const handleBulkEditSubmit = async () => {
    if (selectedDevices.size === 0) return;

    const updates: { status?: string; zone_id?: number } = {};
    if (bulkEditData.status !== undefined) updates.status = bulkEditData.status;
    if (bulkEditData.zone_id !== undefined) updates.zone_id = bulkEditData.zone_id;

    if (Object.keys(updates).length === 0) {
      setBulkEditOpen(false);
      return;
    }

    try {
      await devicesAdminApi.bulkUpdate(Array.from(selectedDevices), updates);
      setBulkEditOpen(false);
      setBulkEditData({});
      setSelectedDevices(new Set());
      await fetchDevices();
    } catch (error: any) {
      console.error('Failed to bulk update devices:', error);
      const backendMsg = error?.response?.data?.error || error?.response?.data?.message;
      alert(backendMsg || t('admin.devices.errors.bulkUpdate'));
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Package className="w-6 h-6 text-accent-red" />
          <h2 className="text-2xl font-bold text-white">{t('admin.devices.title')}</h2>
        </div>
        <button
          onClick={openCreateModal}
          className="btn-primary flex items-center gap-2"
        >
          <Plus className="w-5 h-5" />
          {t('admin.devices.newDevice')}
        </button>
      </div>

      {/* Filters */}
      <div className="glass-dark rounded-xl p-4 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          {/* Search */}
          <div className="relative lg:col-span-2">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder={t('admin.devices.searchPlaceholder')}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="input-field pl-10 w-full"
            />
          </div>

          {/* Status Filter */}
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="input-field"
            title={t('devices.status')}
          >
            <option value="">{t('admin.devices.filters.allStatuses')}</option>
            <option value="in_storage">{t('admin.devices.statuses.in_storage')}</option>
            <option value="on_job">{t('admin.devices.statuses.on_job')}</option>
            <option value="defective">{t('admin.devices.statuses.defective')}</option>
            <option value="maintenance">{t('admin.devices.statuses.maintenance')}</option>
            <option value="retired">{t('admin.devices.statuses.retired')}</option>
          </select>

          {/* Product Filter */}
          <SearchableSelect
            value={productFilter === '' ? '' : String(productFilter)}
            onChange={(v) => setProductFilter(v ? Number(v) : '')}
            options={[
              { value: '', label: t('admin.devices.filters.allProducts') },
              ...products.map((product) => ({
                value: String(product.product_id),
                label: product.name,
              })),
            ]}
            className="input-field p-0 border-0"
            title={t('zoneDetail.columns.product')}
          />

          {/* Zone Filter */}
          <SearchableSelect
            value={zoneFilter === '' ? '' : String(zoneFilter)}
            onChange={(v) => setZoneFilter(v ? Number(v) : '')}
            options={[
              { value: '', label: t('admin.devices.filters.allZones') },
              ...zones.map((zone) => ({
                value: String(zone.zone_id),
                label: `${zone.code} - ${zone.name}`,
              })),
            ]}
            className="input-field p-0 border-0"
            title={t('devices.zone')}
          />
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-between">
          <div className="flex gap-2">
            <button
              onClick={clearFilters}
              className="px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors"
            >
              <X className="w-4 h-4 inline mr-1" />
              {t('admin.devices.clearFilters')}
            </button>
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors disabled:opacity-50"
            >
              <RefreshCcw className={`w-4 h-4 inline mr-1 ${refreshing ? 'animate-spin' : ''}`} />
              {t('common.update')}
            </button>
          </div>

          {/* View Mode Toggle */}
          <div className="flex gap-2">
            <button
              onClick={() => setViewMode('table')}
              className={`p-2 rounded-lg transition-colors ${
                viewMode === 'table' ? 'bg-accent-red text-white' : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
              title={t('admin.devices.tableView')}
              aria-label={t('admin.devices.tableView')}
            >
              <List className="w-5 h-5" />
            </button>
            <button
              onClick={() => setViewMode('cards')}
              className={`p-2 rounded-lg transition-colors ${
                viewMode === 'cards' ? 'bg-accent-red text-white' : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
              title={t('admin.devices.cardView')}
              aria-label={t('admin.devices.cardView')}
            >
              <LayoutGrid className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>

      {/* Bulk Action Bar */}
      {selectedDevices.size > 0 && (
        <div className="glass-dark rounded-xl p-3 flex items-center justify-between border border-accent-red/30">
          <span className="text-sm text-white font-medium">
            {t('admin.devices.selectedCount', { count: selectedDevices.size })}
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => { setBulkEditData({}); setBulkEditOpen(true); }}
              className="px-3 py-1.5 bg-blue-600/80 hover:bg-blue-600 rounded-lg text-sm text-white transition-colors flex items-center gap-1"
            >
              <Pencil className="w-3.5 h-3.5" />
              {t('admin.devices.bulkEdit')}
            </button>
            <button
              onClick={handleBulkDelete}
              className="px-3 py-1.5 bg-red-600/80 hover:bg-red-600 rounded-lg text-sm text-white transition-colors flex items-center gap-1"
            >
              <Trash2 className="w-3.5 h-3.5" />
              {t('admin.devices.bulkDelete')}
            </button>
            <button
              onClick={() => setSelectedDevices(new Set())}
              className="px-3 py-1.5 bg-white/10 hover:bg-white/20 rounded-lg text-sm text-gray-300 transition-colors"
            >
              {t('admin.devices.deselectAll')}
            </button>
          </div>
        </div>
      )}

      {/* Device List */}
      {loadingDevices ? (
        <div className="text-center py-12 text-gray-400">{t('admin.devices.loading')}</div>
      ) : filteredDevices.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          {debouncedSearch || statusFilter || productFilter || zoneFilter
            ? t('admin.devices.emptyFiltered')
            : t('admin.devices.empty')}
        </div>
      ) : viewMode === 'table' ? (
        <div className="glass-dark rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-white/5">
                <tr>
                  <th className="w-10 px-3 py-3">
                    <button
                      type="button"
                      role="checkbox"
                      aria-checked={
                        allFilteredSelected
                          ? true
                          : filteredDevices.some((device) => selectedDevices.has(device.device_id))
                            ? 'mixed'
                            : false
                      }
                      onClick={toggleSelectAll}
                      className="text-gray-400 hover:text-white"
                      aria-label={allFilteredSelected ? t('admin.devices.deselectAll') : t('admin.devices.selectAll')}
                      title={allFilteredSelected ? t('admin.devices.deselectAll') : t('admin.devices.selectAll')}
                    >
                      {(() => {
                        if (allFilteredSelected) return <CheckSquare className="w-4 h-4" />;
                        if (selectedDevices.size > 0) return <MinusSquare className="w-4 h-4" />;
                        return <Square className="w-4 h-4" />;
                      })()}
                    </button>
                  </th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">ID</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">{t('zoneDetail.columns.product')}</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">{t('admin.devices.serialShort')}</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">{t('devices.status')}</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">{t('devices.zone')}</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">{t('devices.condition')}</th>
                  <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">{t('labels.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
              {filteredDevices.map((device) => {
                  const ns = normalizeDeviceStatus(device.status);
                  return (
                  <tr key={device.device_id} className={`hover:bg-white/5 transition-colors ${selectedDevices.has(device.device_id) ? 'bg-white/5' : ''}`}>
                    <td className="w-10 px-3 py-3">
                      <button
                        type="button"
                        role="checkbox"
                        aria-checked={selectedDevices.has(device.device_id)}
                        onClick={() => toggleDeviceSelection(device.device_id)}
                        className="text-gray-400 hover:text-white"
                        aria-label={selectedDevices.has(device.device_id) ? t('admin.devices.deselectDevice', { id: device.device_id }) : t('admin.devices.selectDevice', { id: device.device_id })}
                      >
                        {selectedDevices.has(device.device_id) ? <CheckSquare className="w-4 h-4 text-accent-red" /> : <Square className="w-4 h-4" />}
                      </button>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">{device.device_id}</td>
                    <td className="px-4 py-3 text-sm">
                      <div className="flex flex-col">
                        <span className="text-white">{device.product_name || '-'}</span>
                        {device.product_category && (
                          <span className="text-xs text-gray-400">{device.product_category}</span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">{device.serial_number || '-'}</td>
                    <td className="px-4 py-3">
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-semibold ${
                          ns === 'in_storage'
                            ? 'bg-green-500/20 text-green-400'
                            : ns === 'on_job'
                            ? 'bg-blue-500/20 text-blue-400'
                            : ns === 'defective'
                            ? 'bg-red-500/20 text-red-400'
                            : 'bg-yellow-500/20 text-yellow-400'
                        }`}
                      >
                        {statusLabel(device.status)}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">
                      {device.zone_code ? `${device.zone_code} - ${device.zone_name}` : '-'}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">
                      {device.condition_rating ? `${device.condition_rating}/10` : '-'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleViewDevice(device)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                          title={t('casesPage.details')}
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => downloadQR(device.device_id)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                          title={t('admin.devices.downloadQr')}
                        >
                          <QrCode className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => downloadBarcode(device.device_id)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                          title={t('admin.devices.downloadBarcode')}
                        >
                          <Download className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => openEditModal(device)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-blue-400 hover:text-blue-300"
                          title={t('common.edit')}
                        >
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(device.device_id)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-red-400 hover:text-red-300"
                          title={t('common.delete')}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                   </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredDevices.map((device) => {
            const ns = normalizeDeviceStatus(device.status);
            return (
            <div key={device.device_id} className="glass-dark rounded-xl p-4 space-y-3">
            <div className="flex items-start justify-between">
              <div>
                <h3 className="font-bold text-white">{device.device_id}</h3>
                <p className="text-sm text-gray-400">{device.product_name || t('admin.devices.unknownProduct')}</p>
                {device.product_category && (
                  <p className="text-xs text-gray-500">{device.product_category}</p>
                )}
              </div>
                <span
                  className={`px-2 py-1 rounded-full text-xs font-semibold ${
                    ns === 'in_storage'
                      ? 'bg-green-500/20 text-green-400'
                      : ns === 'on_job'
                      ? 'bg-blue-500/20 text-blue-400'
                      : ns === 'defective'
                      ? 'bg-red-500/20 text-red-400'
                      : 'bg-yellow-500/20 text-yellow-400'
                  }`}
                >
                  {statusLabel(device.status)}
                </span>
              </div>

              <div className="space-y-1 text-sm">
                {device.serial_number && (
                  <p className="text-gray-300">
                    <span className="text-gray-500">{t('admin.devices.serialShort')}:</span> {device.serial_number}
                  </p>
                )}
                {device.zone_code && (
                  <p className="text-gray-300">
                    <span className="text-gray-500">{t('devices.zone')}:</span> {device.zone_code} - {device.zone_name}
                  </p>
                )}
                {device.condition_rating && (
                  <p className="text-gray-300">
                    <span className="text-gray-500">{t('devices.condition')}:</span> {device.condition_rating}/10
                  </p>
                )}
                {device.purchase_date && (
                  <p className="text-gray-300">
                    <span className="text-gray-500">{t('admin.devices.purchaseShort')}:</span> {device.purchase_date}
                  </p>
                )}
                {device.usage_hours && (
                  <p className="text-gray-300">
                    <span className="text-gray-500">{t('admin.devices.hoursShort')}:</span> {device.usage_hours}h
                  </p>
                )}
              </div>

              <div className="flex items-center gap-2 pt-2 border-t border-white/10">
                <button
                  onClick={() => handleViewDevice(device)}
                  className="flex-1 px-3 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors flex items-center justify-center gap-2"
                >
                  <Eye className="w-4 h-4" />
                  {t('casesPage.details')}
                </button>
                <button
                  onClick={() => openEditModal(device)}
                  className="flex-1 px-3 py-2 bg-blue-500/20 hover:bg-blue-500/30 rounded-lg text-sm text-blue-400 transition-colors flex items-center justify-center gap-2"
                >
                  <Pencil className="w-4 h-4" />
                  {t('common.edit')}
                </button>
                <button
                  onClick={() => handleDelete(device.device_id)}
                  className="px-3 py-2 bg-red-500/20 hover:bg-red-500/30 rounded-lg text-sm text-red-400 transition-colors"
                  title={t('common.delete')}
                  aria-label={t('common.delete')}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
            );
          })}
        </div>
      )}

      {/* Create/Edit Modal */}
      {modalOpen && (
        <ModalPortal>
        <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
          <div className="glass-dark rounded-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-white">
                {editingDevice ? t('admin.devices.editDevice') : t('admin.devices.createDevice')}
              </h3>
              <button
                onClick={() => {
                  setModalOpen(false);
                  setScanFieldTarget(null);
                  if (onCancel) {
                    onCancel();
                  }
                }}
                aria-label={t('common.close')}
                title={t('common.close')}
              >
                <X className="w-6 h-6 text-gray-400" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Product Selection */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {t('zoneDetail.columns.product')} *
                </label>
                <SearchableSelect
                  value={formData.product_id ? String(formData.product_id) : ''}
                  onChange={(v) =>
                    setFormData({ ...formData, product_id: v ? Number(v) : undefined })
                  }
                  options={products.map((product) => ({
                    value: String(product.product_id),
                    label: product.category_name
                      ? `${product.name} (${product.category_name})`
                      : product.name,
                  }))}
                  placeholder={t('admin.devices.selectProduct')}
                  required
                  disabled={!!editingDevice}
                  className="w-full"
                  title={t('zoneDetail.columns.product')}
                />
              </div>

              {/* Device ID (only when editing – allows renaming) */}
              {editingDevice && (
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('devices.deviceId')}
                  </label>
                  <input
                    type="text"
                    value={formData.new_device_id}
                    onChange={(e) => setFormData({ ...formData, new_device_id: e.target.value })}
                    className="input-field w-full"
                    title={t('devices.deviceId')}
                  />
                  <p className="text-xs text-gray-500 mt-1">{t('admin.devices.deviceIdHint')}</p>
                </div>
              )}

              {/* Quantity (only for new devices) */}
              {!editingDevice && (
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      {t('admin.devices.quantity')}
                    </label>
                    <input
                      type="number"
                      min="1"
                      max="100"
                      value={formData.quantity}
                      onChange={(e) =>
                        setFormData({ ...formData, quantity: Number(e.target.value) || 1 })
                      }
                      className="input-field w-full"
                      title={t('admin.devices.quantity')}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-300 mb-2">
                      {t('admin.devices.prefixOptional')}
                    </label>
                    <input
                      type="text"
                      value={formData.device_prefix}
                      onChange={(e) =>
                        setFormData({ ...formData, device_prefix: e.target.value })
                      }
                      className="input-field w-full"
                      placeholder={t('admin.devices.prefixPlaceholder')}
                      title={t('admin.devices.prefixOptional')}
                    />
                  </div>
                </div>
              )}

              {!editingDevice && formData.quantity > 1 && (
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="increment_serial"
                    checked={formData.increment_serial}
                    onChange={(e) =>
                      setFormData({ ...formData, increment_serial: e.target.checked })
                    }
                    className="w-4 h-4"
                  />
                  <label htmlFor="increment_serial" className="text-sm text-gray-300">
                    {t('admin.devices.incrementSerial')}
                  </label>
                </div>
              )}

              {/* Status */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('devices.status')} *
                  </label>
                  <select
                    value={formData.status}
                    onChange={(e) => setFormData({ ...formData, status: e.target.value })}
                    className="input-field w-full"
                    required
                    title={t('devices.status')}
                  >
                    <option value="in_storage">{t('admin.devices.statuses.in_storage')}</option>
                    <option value="on_job">{t('admin.devices.statuses.on_job')}</option>
                    <option value="defective">{t('admin.devices.statuses.defective')}</option>
                    <option value="maintenance">{t('admin.devices.statuses.maintenance')}</option>
                    <option value="retired">{t('admin.devices.statuses.retired')}</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('devices.zone')}
                  </label>
                  <SearchableSelect
                    value={formData.zone_id ? String(formData.zone_id) : ''}
                    onChange={(v) =>
                      setFormData({ ...formData, zone_id: v ? Number(v) : undefined })
                    }
                    options={[
                      { value: '', label: t('casesPage.noZone') },
                      ...zones.map((zone) => ({
                        value: String(zone.zone_id),
                        label: `${zone.code} - ${zone.name}`,
                      })),
                    ]}
                    className="w-full"
                    title={t('devices.zone')}
                  />
                </div>
              </div>

              {/* Serial, Barcode, QR */}
              <div className="grid grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('devices.serialNumber')}
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={formData.serial_number}
                      onChange={(e) => setFormData({ ...formData, serial_number: e.target.value })}
                      className="input-field w-full"
                      title={t('devices.serialNumber')}
                    />
                    <button
                      type="button"
                      onClick={() => setScanFieldTarget('serial_number')}
                      className="flex-shrink-0 p-2 rounded-lg bg-white/10 hover:bg-white/20 text-gray-300 hover:text-white transition-colors"
                      title={t('scanField.scan')}
                      aria-label={t('scanField.scan')}
                    >
                      <ScanLine className="w-4 h-4" />
                    </button>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('devices.barcode')}
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={formData.barcode}
                      onChange={(e) => setFormData({ ...formData, barcode: e.target.value })}
                      className="input-field w-full"
                      title={t('devices.barcode')}
                    />
                    <button
                      type="button"
                      onClick={() => setScanFieldTarget('barcode')}
                      className="flex-shrink-0 p-2 rounded-lg bg-white/10 hover:bg-white/20 text-gray-300 hover:text-white transition-colors"
                      title={t('scanField.scan')}
                      aria-label={t('scanField.scan')}
                    >
                      <ScanLine className="w-4 h-4" />
                    </button>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('labels.qrCode')}
                  </label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      value={formData.qr_code}
                      onChange={(e) => setFormData({ ...formData, qr_code: e.target.value })}
                      className="input-field w-full"
                      title={t('labels.qrCode')}
                    />
                    <button
                      type="button"
                      onClick={() => setScanFieldTarget('qr_code')}
                      className="flex-shrink-0 p-2 rounded-lg bg-white/10 hover:bg-white/20 text-gray-300 hover:text-white transition-colors"
                      title={t('scanField.scan')}
                      aria-label={t('scanField.scan')}
                    >
                      <ScanLine className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>

              {/* RFID */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {t('devices.rfid')}
                </label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={formData.rfid}
                    onChange={(e) => setFormData({ ...formData, rfid: e.target.value })}
                    className="input-field w-full"
                    title={t('devices.rfid')}
                  />
                  <button
                    type="button"
                    onClick={() => setScanFieldTarget('rfid')}
                    className="flex-shrink-0 p-2 rounded-lg bg-white/10 hover:bg-white/20 text-gray-300 hover:text-white transition-colors"
                    title={t('scanField.scan')}
                    aria-label={t('scanField.scan')}
                  >
                    <ScanLine className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Location and Condition */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.currentLocation')}
                  </label>
                  <input
                    type="text"
                    value={formData.current_location}
                    onChange={(e) => setFormData({ ...formData, current_location: e.target.value })}
                    className="input-field w-full"
                    title={t('admin.devices.currentLocation')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.conditionScale')}
                  </label>
                  <input
                    type="number"
                    min="0"
                    max="10"
                    step="0.1"
                    value={formData.condition_rating || ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        condition_rating: e.target.value ? Number(e.target.value) : undefined,
                      })
                    }
                    className="input-field w-full"
                    title={t('admin.devices.conditionScale')}
                  />
              </div>
              </div>

              {/* Dates */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.purchaseDate')}
                  </label>
                  <input
                    type="date"
                    value={formData.purchase_date}
                    onChange={(e) => setFormData({ ...formData, purchase_date: e.target.value })}
                    className="input-field w-full"
                    title={t('admin.devices.purchaseDate')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.warrantyEndDate')}
                  </label>
                  <input
                    type="date"
                    value={formData.warranty_end_date}
                    onChange={(e) => setFormData({ ...formData, warranty_end_date: e.target.value })}
                    className="input-field w-full"
                    title={t('admin.devices.warrantyEndDate')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.retireDate')}
                  </label>
                  <input
                    type="date"
                    value={formData.retire_date}
                    onChange={(e) => setFormData({ ...formData, retire_date: e.target.value })}
                    className="input-field w-full"
                    title={t('admin.devices.retireDate')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.lastMaintenance')}
                  </label>
                  <input
                    type="date"
                    value={formData.last_maintenance}
                    onChange={(e) => setFormData({ ...formData, last_maintenance: e.target.value })}
                    className="input-field w-full"
                    title={t('admin.devices.lastMaintenance')}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.devices.nextMaintenance')}
                  </label>
                  <input
                    type="date"
                    value={formData.next_maintenance}
                    onChange={(e) => setFormData({ ...formData, next_maintenance: e.target.value })}
                    className="input-field w-full"
                    title={t('admin.devices.nextMaintenance')}
                  />
                </div>
              </div>

              {/* Usage Hours */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {t('devices.usageHours')}
                </label>
                <input
                  type="number"
                  min="0"
                  step="0.1"
                  value={formData.usage_hours || ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      usage_hours: e.target.value ? Number(e.target.value) : undefined,
                    })
                  }
                  className="input-field w-full"
                  title={t('devices.usageHours')}
                />
              </div>

              {/* Notes */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {t('modals.productDependencies.notes')}
                </label>
                <textarea
                  value={formData.notes}
                  onChange={(e) => setFormData({ ...formData, notes: e.target.value })}
                  className="input-field w-full"
                  rows={3}
                  title={t('modals.productDependencies.notes')}
                />
              </div>

              {/* Label & Code Options */}
              <div className="border-t border-white/10 pt-4 space-y-3">
                {editingDevice ? (
                  <>
                    <label className="flex items-center gap-2 text-sm text-gray-300">
                      <input
                        type="checkbox"
                        checked={formData.regenerate_codes}
                        onChange={(e) => setFormData({ ...formData, regenerate_codes: e.target.checked })}
                        className="w-4 h-4"
                      />
                      {t('admin.devices.regenerateCodes')}
                    </label>
                    <label className="flex items-center gap-2 text-sm text-gray-300">
                      <input
                        type="checkbox"
                        checked={formData.regenerate_label}
                        onChange={(e) => setFormData({ ...formData, regenerate_label: e.target.checked })}
                        className="w-4 h-4"
                      />
                      {t('admin.devices.regenerateLabel')}
                    </label>
                  </>
                ) : (
                  <label className="flex items-center gap-2 text-sm text-gray-300">
                    <input
                      type="checkbox"
                      checked={formData.auto_generate_label}
                      onChange={(e) => setFormData({ ...formData, auto_generate_label: e.target.checked })}
                      className="w-4 h-4"
                    />
                    {t('admin.devices.autoGenerateLabel')}
                  </label>
                )}

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('labels.template')}
                  </label>
                  <select
                    value={formData.label_template_id ?? ''}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        label_template_id: e.target.value ? Number(e.target.value) : undefined,
                      })
                    }
                    className="input-field w-full"
                    title={t('labels.template')}
                  >
                    <option value="">{t('admin.devices.defaultTemplate')}</option>
                    {labelTemplates.map((template) => (
                      <option key={template.id} value={template.id}>
                        {template.name} {template.is_default ? '(Standard)' : ''}
                      </option>
                    ))}
                  </select>
                  <p className="text-xs text-gray-500 mt-1">
                    {t('admin.devices.templateHint')}
                  </p>
                </div>
              </div>

              {/* Submit Buttons */}
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => {
                    setModalOpen(false);
                    setScanFieldTarget(null);
                    if (onCancel) {
                      onCancel();
                    }
                  }}
                  className="flex-1 px-4 py-3 bg-white/5 hover:bg-white/10 rounded-lg font-semibold text-gray-300 transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="flex-1 btn-primary disabled:opacity-50"
                >
                  {submitting
                    ? t('common.saving')
                    : editingDevice
                    ? t('common.update')
                    : t('common.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
        </ModalPortal>
      )}

      {/* View Device Modal */}
      <DeviceInfoModal
        device={viewDevice}
        isOpen={viewDevice !== null}
        onClose={() => setViewDevice(null)}
        onEdit={(device) => openEditModal(device)}
      />

      {/* Bulk Edit Modal */}
      {bulkEditOpen && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl p-6 max-w-lg w-full">
              <div className="flex items-center justify-between mb-6">
                <h3 className="text-xl font-bold text-white">{t('admin.devices.bulkEditTitle')}</h3>
                <button onClick={() => setBulkEditOpen(false)} aria-label={t('common.close')} title={t('common.close')}>
                  <X className="w-6 h-6 text-gray-400" />
                </button>
              </div>
              <p className="text-sm text-gray-400 mb-4">
                {t('admin.devices.bulkEditDescription', { count: selectedDevices.size })}
              </p>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">{t('devices.status')}</label>
                  <select
                    value={bulkEditData.status || ''}
                    onChange={(e) => setBulkEditData({ ...bulkEditData, status: e.target.value || undefined })}
                    className="input-field w-full"
                    title={t('devices.status')}
                  >
                    <option value="">—</option>
                    <option value="in_storage">{t('admin.devices.statuses.in_storage')}</option>
                    <option value="on_job">{t('admin.devices.statuses.on_job')}</option>
                    <option value="defective">{t('admin.devices.statuses.defective')}</option>
                    <option value="maintenance">{t('admin.devices.statuses.maintenance')}</option>
                    <option value="retired">{t('admin.devices.statuses.retired')}</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">{t('devices.zone')}</label>
                  <SearchableSelect
                    value={bulkEditData.zone_id ? String(bulkEditData.zone_id) : ''}
                    onChange={(v) => setBulkEditData({ ...bulkEditData, zone_id: v ? Number(v) : undefined })}
                    options={[
                      { value: '', label: '—' },
                      ...zones.map((zone) => ({
                        value: String(zone.zone_id),
                        label: `${zone.code} - ${zone.name}`,
                      })),
                    ]}
                    className="input-field p-0 border-0 w-full"
                    title={t('devices.zone')}
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button
                  onClick={() => setBulkEditOpen(false)}
                  className="px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg text-sm text-white transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleBulkEditSubmit}
                  className="px-4 py-2 bg-accent-red hover:bg-accent-red/80 rounded-lg text-sm text-white font-semibold transition-colors"
                >
                  {t('common.save')}
                </button>
              </div>
            </div>
          </div>
        </ModalPortal>
      )}

      {/* Scan Field Modal */}
      {(() => {
        const scanFieldLabels: Record<NonNullable<typeof scanFieldTarget>, string> = {
          rfid: t('devices.rfid'),
          serial_number: t('devices.serialNumber'),
          barcode: t('devices.barcode'),
          qr_code: t('labels.qrCode'),
        };
        return (
          <ScanFieldModal
            isOpen={scanFieldTarget !== null}
            fieldLabel={scanFieldTarget ? scanFieldLabels[scanFieldTarget] : ''}
            onConfirm={(value) => {
              if (scanFieldTarget) {
                setFormData((prev) => ({ ...prev, [scanFieldTarget]: value }));
              }
            }}
            onClose={() => setScanFieldTarget(null)}
          />
        );
      })()}
    </div>
  );
}
