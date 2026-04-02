import { useEffect, useMemo, useState } from 'react';
import { X, ChevronRight, Package, Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import { ModalPortal } from './ModalPortal';
import { useBlockBodyScroll } from '../hooks/useBlockBodyScroll';

interface Device {
  device_id: string;
  product_name: string;
  status: string;
  barcode?: string;
  serial_number?: string;
  zone_id?: number;
  zone_code?: string;
}

interface Subbiercategory {
  id: number;
  name: string;
  devices: Device[];
  device_count: number;
}

interface Subcategory {
  id: number;
  name: string;
  subbiercategories: Subbiercategory[];
  direct_devices: Device[];
  device_count: number;
}

interface Category {
  id: number;
  name: string;
  subcategories: Subcategory[];
  direct_devices: Device[];
  device_count: number;
}

interface DeviceTreeModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (selectedDevices: string[]) => void;
  zoneId: number;
}

export function DeviceTreeModal({ isOpen, onClose, onConfirm, zoneId }: DeviceTreeModalProps) {
  const { t } = useTranslation();
  const [treeData, setTreeData] = useState<Category[]>([]);
  const [selectedDevices, setSelectedDevices] = useState<Set<string>>(new Set());
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');

  // Block body scroll when modal is open
  useBlockBodyScroll(isOpen);

  useEffect(() => {
    if (isOpen) {
      loadDeviceTree();
    }
  }, [isOpen]);

  const loadDeviceTree = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/devices/tree');
      const data = await response.json();
      setTreeData(data.treeData || []);
    } catch (error) {
      console.error('Failed to load device tree:', error);
    } finally {
      setLoading(false);
    }
  };

  // Collect all devices matching search query
  const matchingDeviceIds = useMemo<Set<string>>(() => {
    const q = search.trim().toLowerCase();
    if (!q) return new Set();
    const ids = new Set<string>();
    const checkDevice = (d: Device) => {
      if (
        d.device_id.toLowerCase().includes(q) ||
        d.product_name.toLowerCase().includes(q) ||
        (d.serial_number && d.serial_number.toLowerCase().includes(q)) ||
        (d.barcode && d.barcode.toLowerCase().includes(q))
      ) {
        ids.add(d.device_id);
      }
    };
    for (const cat of treeData) {
      cat.direct_devices.forEach(checkDevice);
      for (const sub of cat.subcategories) {
        sub.direct_devices.forEach(checkDevice);
        for (const subbier of sub.subbiercategories) {
          subbier.devices.forEach(checkDevice);
        }
      }
    }
    return ids;
  }, [search, treeData]);

  // Auto-expand nodes that contain matching devices
  const searchExpandedNodes = useMemo<Set<string>>(() => {
    if (!search.trim()) return new Set<string>();
    const expanded = new Set<string>();
    for (const cat of treeData) {
      let catHasMatch = cat.direct_devices.some(d => matchingDeviceIds.has(d.device_id));
      for (const sub of cat.subcategories) {
        let subHasMatch = sub.direct_devices.some(d => matchingDeviceIds.has(d.device_id));
        for (const subbier of sub.subbiercategories) {
          if (subbier.devices.some(d => matchingDeviceIds.has(d.device_id))) {
            expanded.add(`subbier-${subbier.id}`);
            subHasMatch = true;
          }
        }
        if (subHasMatch) {
          expanded.add(`subcat-${sub.id}`);
          catHasMatch = true;
        }
      }
      if (catHasMatch) expanded.add(`cat-${cat.id}`);
    }
    return expanded;
  }, [search, treeData, matchingDeviceIds]);

  const effectiveExpanded = useMemo(
    () => search.trim() ? new Set([...expandedNodes, ...searchExpandedNodes]) : expandedNodes,
    [expandedNodes, searchExpandedNodes, search]
  );

  const toggleNode = (nodeId: string) => {
    const newExpanded = new Set(expandedNodes);
    if (newExpanded.has(nodeId)) {
      newExpanded.delete(nodeId);
    } else {
      newExpanded.add(nodeId);
    }
    setExpandedNodes(newExpanded);
  };

  const toggleDeviceSelection = (deviceId: string) => {
    const newSelected = new Set(selectedDevices);
    if (newSelected.has(deviceId)) {
      newSelected.delete(deviceId);
    } else {
      newSelected.add(deviceId);
    }
    setSelectedDevices(newSelected);
  };

  const handleConfirm = () => {
    onConfirm(Array.from(selectedDevices));
    setSelectedDevices(new Set());
    setSearch('');
    onClose();
  };

  const handleClose = () => {
    setSelectedDevices(new Set());
    setSearch('');
    onClose();
  };

  if (!isOpen) return null;

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
        <div className="glass-dark rounded-2xl w-full max-w-4xl flex flex-col shadow-2xl max-h-[85vh]">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-white/10">
          <h2 className="text-2xl font-bold text-white flex items-center gap-2">
            <Package className="w-6 h-6 text-accent-red" />
            {t('modals.deviceTree.addDevices')}
          </h2>
          <button
            onClick={handleClose}
            className="p-2 hover:bg-white/10 rounded-lg transition-colors"
            aria-label={t('common.close')}
            title={t('common.close')}
          >
            <X className="w-5 h-5 text-gray-400" />
          </button>
        </div>
        {/* Selected Devices Count */}
        <div className="px-6 py-3 bg-white/5 border-b border-white/10">
          <p className="text-sm text-gray-400">
            {t('modals.deviceTree.selectedCount', { count: selectedDevices.size })}
          </p>
        </div>

        {/* Search */}
        <div className="px-6 py-3 border-b border-white/10">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('modals.deviceTree.searchPlaceholder')}
              aria-label={t('modals.deviceTree.searchLabel')}
              className="w-full pl-9 pr-4 py-2 bg-white/10 border border-white/20 rounded-lg text-sm text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
            />
            {search && (
              <button
                type="button"
                onClick={() => setSearch('')}
                aria-label={t('modals.deviceTree.clearSearch')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white transition-colors"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>
        </div>

        {/* Device Tree */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
            </div>
          ) : treeData.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              {t('modals.deviceTree.noDevices')}
            </div>
          ) : search.trim() && matchingDeviceIds.size === 0 ? (
            <div className="text-center py-12 text-gray-400">
              {t('modals.deviceTree.noSearchResults')}
            </div>
          ) : (
            <div className="space-y-2">
              {treeData.map((category) => (
                <CategoryNode
                  key={category.id}
                  category={category}
                  expandedNodes={effectiveExpanded}
                  selectedDevices={selectedDevices}
                  onToggleNode={toggleNode}
                  onToggleDevice={toggleDeviceSelection}
                  currentZoneId={zoneId}
                  matchingDeviceIds={search.trim() ? matchingDeviceIds : null}
                  t={t}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 p-6 border-t border-white/10">
          <button
            onClick={handleClose}
            className="px-6 py-2.5 glass hover:bg-white/10 text-white font-semibold rounded-xl transition-all"
          >
            {t('common.cancel')}
          </button>
          <button
            onClick={handleConfirm}
            disabled={selectedDevices.size === 0}
            className="px-6 py-2.5 bg-gradient-to-r from-accent-red to-red-700 text-white font-semibold rounded-xl hover:shadow-lg hover:shadow-accent-red/50 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {t('modals.deviceTree.addSelected', { count: selectedDevices.size })}
          </button>
        </div>
        </div>
      </div>
    </ModalPortal>
  );
}

// Category Node Component
function CategoryNode({
  category,
  expandedNodes,
  selectedDevices,
  onToggleNode,
  onToggleDevice,
  currentZoneId,
  matchingDeviceIds,
  t,
}: {
  category: Category;
  expandedNodes: Set<string>;
  selectedDevices: Set<string>;
  onToggleNode: (id: string) => void;
  onToggleDevice: (id: string) => void;
  currentZoneId: number;
  matchingDeviceIds: Set<string> | null;
  t: TFunction;
}) {
  const nodeId = `cat-${category.id}`;
  const isExpanded = expandedNodes.has(nodeId);

  // When filtering, only show this category if it has matching devices
  const visibleSubcategories = matchingDeviceIds
    ? category.subcategories.filter(sub =>
        sub.direct_devices.some(d => matchingDeviceIds.has(d.device_id)) ||
        sub.subbiercategories.some(sb => sb.devices.some(d => matchingDeviceIds.has(d.device_id)))
      )
    : category.subcategories;
  const visibleDirectDevices = matchingDeviceIds
    ? category.direct_devices.filter(d => matchingDeviceIds.has(d.device_id))
    : category.direct_devices;

  if (matchingDeviceIds && visibleSubcategories.length === 0 && visibleDirectDevices.length === 0) {
    return null;
  }

  return (
    <div className="border border-white/10 rounded-lg bg-surface-1">
      <div
        className="flex items-center gap-2 p-3 cursor-pointer hover:bg-white/5 transition-colors rounded-lg"
        onClick={() => onToggleNode(nodeId)}
      >
        <ChevronRight
          className={`w-4 h-4 text-gray-400 transition-transform ${
            isExpanded ? 'rotate-90' : ''
          }`}
        />
        <span className="text-lg">📁</span>
        <span className="font-semibold text-white">{category.name}</span>
        <span className="text-sm text-gray-400">({t('modals.deviceTree.countShort', { count: category.device_count })})</span>
      </div>

      {isExpanded && (
        <div className="pl-6 pb-2 pr-2">
          {visibleSubcategories.map((subcategory) => (
            <SubcategoryNode
              key={subcategory.id}
              subcategory={subcategory}
              expandedNodes={expandedNodes}
              selectedDevices={selectedDevices}
              onToggleNode={onToggleNode}
              onToggleDevice={onToggleDevice}
              currentZoneId={currentZoneId}
              matchingDeviceIds={matchingDeviceIds}
              t={t}
            />
          ))}
          {visibleDirectDevices.length > 0 && (
            <div className="mt-2 pl-2 space-y-1">
              {visibleDirectDevices.map((device) => (
                <DeviceNode
                  key={device.device_id}
                  device={device}
                  isSelected={selectedDevices.has(device.device_id)}
                  onToggle={onToggleDevice}
                  isInCurrentZone={device.zone_id === currentZoneId}
                  t={t}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Subcategory Node Component
function SubcategoryNode({
  subcategory,
  expandedNodes,
  selectedDevices,
  onToggleNode,
  onToggleDevice,
  currentZoneId,
  matchingDeviceIds,
  t,
}: {
  subcategory: Subcategory;
  expandedNodes: Set<string>;
  selectedDevices: Set<string>;
  onToggleNode: (id: string) => void;
  onToggleDevice: (id: string) => void;
  currentZoneId: number;
  matchingDeviceIds: Set<string> | null;
  t: TFunction;
}) {
  const nodeId = `subcat-${subcategory.id}`;
  const isExpanded = expandedNodes.has(nodeId);

  const visibleSubbiercategories = matchingDeviceIds
    ? subcategory.subbiercategories.filter(sb => sb.devices.some(d => matchingDeviceIds.has(d.device_id)))
    : subcategory.subbiercategories;
  const visibleDirectDevices = matchingDeviceIds
    ? subcategory.direct_devices.filter(d => matchingDeviceIds.has(d.device_id))
    : subcategory.direct_devices;

  return (
    <div className="mt-2">
      <div
        className="flex items-center gap-2 p-2 cursor-pointer hover:bg-white/5 transition-colors rounded-lg"
        onClick={() => onToggleNode(nodeId)}
      >
        <ChevronRight
          className={`w-3.5 h-3.5 text-gray-400 transition-transform ${
            isExpanded ? 'rotate-90' : ''
          }`}
        />
        <span className="text-base">📂</span>
        <span className="font-medium text-white text-sm">{subcategory.name}</span>
        <span className="text-xs text-gray-400">({subcategory.device_count})</span>
      </div>

      {isExpanded && (
        <div className="pl-6 pt-1">
          {visibleSubbiercategories.map((subbiercategory) => (
            <SubbiercategoryNode
              key={subbiercategory.id}
              subbiercategory={subbiercategory}
              expandedNodes={expandedNodes}
              selectedDevices={selectedDevices}
              onToggleNode={onToggleNode}
              onToggleDevice={onToggleDevice}
              currentZoneId={currentZoneId}
              matchingDeviceIds={matchingDeviceIds}
              t={t}
            />
          ))}
          {visibleDirectDevices.length > 0 && (
            <div className="mt-1 space-y-1">
              {visibleDirectDevices.map((device) => (
                <DeviceNode
                  key={device.device_id}
                  device={device}
                  isSelected={selectedDevices.has(device.device_id)}
                  onToggle={onToggleDevice}
                  isInCurrentZone={device.zone_id === currentZoneId}
                  t={t}
                />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Subbiercategory Node Component
function SubbiercategoryNode({
  subbiercategory,
  expandedNodes,
  selectedDevices,
  onToggleNode,
  onToggleDevice,
  currentZoneId,
  matchingDeviceIds,
  t,
}: {
  subbiercategory: Subbiercategory;
  expandedNodes: Set<string>;
  selectedDevices: Set<string>;
  onToggleNode: (id: string) => void;
  onToggleDevice: (id: string) => void;
  currentZoneId: number;
  matchingDeviceIds: Set<string> | null;
  t: TFunction;
}) {
  const nodeId = `subbier-${subbiercategory.id}`;
  const isExpanded = expandedNodes.has(nodeId);

  const visibleDevices = matchingDeviceIds
    ? subbiercategory.devices.filter(d => matchingDeviceIds.has(d.device_id))
    : subbiercategory.devices;

  return (
    <div className="mt-2">
      <div
        className="flex items-center gap-2 p-2 cursor-pointer hover:bg-white/5 transition-colors rounded-lg"
        onClick={() => onToggleNode(nodeId)}
      >
        <ChevronRight
          className={`w-3 h-3 text-gray-400 transition-transform ${
            isExpanded ? 'rotate-90' : ''
          }`}
        />
        <span className="text-sm">📄</span>
        <span className="font-medium text-white text-sm">{subbiercategory.name}</span>
        <span className="text-xs text-gray-400">({subbiercategory.device_count})</span>
      </div>

      {isExpanded && (
        <div className="pl-6 pt-1 space-y-1">
          {visibleDevices.map((device) => (
            <DeviceNode
              key={device.device_id}
              device={device}
              isSelected={selectedDevices.has(device.device_id)}
              onToggle={onToggleDevice}
              isInCurrentZone={device.zone_id === currentZoneId}
              t={t}
            />
          ))}
        </div>
      )}
    </div>
  );
}

// Device Node Component
function DeviceNode({
  device,
  isSelected,
  onToggle,
  isInCurrentZone,
  t,
}: {
  device: Device;
  isSelected: boolean;
  onToggle: (id: string) => void;
  isInCurrentZone: boolean;
  t: TFunction;
}) {
  const statusColors: Record<string, string> = {
    in_storage: 'text-green-400',
    on_job: 'text-yellow-400',
    rented: 'text-yellow-400',
    defective: 'text-red-400',
    in_transit: 'text-blue-400',
  };

  return (
    <div
      className={`flex items-center justify-between p-2 rounded-lg cursor-pointer transition-all ${
        isSelected
          ? 'bg-accent-red/20 border border-accent-red'
          : 'hover:bg-white/5 border border-transparent'
      } ${isInCurrentZone ? 'opacity-50 cursor-not-allowed' : ''}`}
      onClick={() => !isInCurrentZone && onToggle(device.device_id)}
    >
      <div className="flex items-center gap-2 flex-1 min-w-0">
        <Package className={`w-4 h-4 flex-shrink-0 ${isSelected ? 'text-accent-red' : 'text-gray-400'}`} />
        <div className="flex-1 min-w-0">
          <div className="font-medium text-white text-sm truncate">{device.product_name}</div>
          <div className="text-xs text-gray-400 font-mono">
            {device.device_id}
            {device.serial_number && <span> • {device.serial_number}</span>}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        {isInCurrentZone && (
          <span className="text-xs text-gray-500 italic">{t('modals.deviceTree.alreadyInZone')}</span>
        )}
        {device.zone_code && !isInCurrentZone && (
          <span className="text-xs text-gray-500">📍 {device.zone_code}</span>
        )}
        <span className={`text-xs font-medium ${statusColors[device.status] || 'text-gray-400'}`}>
          {device.status}
        </span>
      </div>
    </div>
  );
}
