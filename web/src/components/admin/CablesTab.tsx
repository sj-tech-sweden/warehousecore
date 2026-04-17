import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box,
  Cable,
  ChevronDown,
  ChevronRight,
  Eye,
  LayoutGrid,
  List,
  Pencil,
  Plus,
  RefreshCcw,
  Search,
  Trash2,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cablesAdminApi } from '../../lib/api';
import type { Cable as CableType, CableConnector, CableType as CableTypeData, CableCreateInput, CableUpdateInput, Device } from '../../lib/api';
import { useBlockBodyScroll } from '../../hooks/useBlockBodyScroll';
import { SearchableSelect } from '../SearchableSelect';
import { ModalPortal } from '../ModalPortal';
import { formatStatus, getStatusColor } from '../../lib/utils';

interface CableFormData {
  name: string;
  connector1?: number;
  connector2?: number;
  typ?: number;
  length: number;
  mm2: number;
}

const initialFormData: CableFormData = {
  name: '',
  length: 1,
  mm2: 0,
};

function useDebouncedValue<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(handle);
  }, [value, delay]);

  return debounced;
}

export function CablesTab() {
  const { t } = useTranslation();
  const [cables, setCables] = useState<CableType[]>([]);
  const [connectors, setConnectors] = useState<CableConnector[]>([]);
  const [cableTypes, setCableTypes] = useState<CableTypeData[]>([]);
  const [loadingCables, setLoadingCables] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewCable, setViewCable] = useState<CableType | null>(null);
  const [editingCable, setEditingCable] = useState<number | null>(null);
  const [formData, setFormData] = useState<CableFormData>(initialFormData);
  const [submitting, setSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'cards'>(() => {
    // Set default based on screen width: mobile (<768px) = cards, desktop = table
    return typeof window !== 'undefined' && window.innerWidth < 768 ? 'cards' : 'table';
  });
  const [searchTerm, setSearchTerm] = useState('');
  const [connector1Filter, setConnector1Filter] = useState<number | ''>('');
  const [connector2Filter, setConnector2Filter] = useState<number | ''>('');
  const [typeFilter, setTypeFilter] = useState<number | ''>('');
  const [lengthMinFilter, setLengthMinFilter] = useState<number | ''>('');
  const [lengthMaxFilter, setLengthMaxFilter] = useState<number | ''>('');
  const [refreshing, setRefreshing] = useState(false);

  // Cable devices modal state
  const [devicesModal, setDevicesModal] = useState<{ cableId: number } | null>(null);
  const [cableDevices, setCableDevices] = useState<Device[]>([]);
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [devicePrefix, setDevicePrefix] = useState('');
  const [deviceQuantity, setDeviceQuantity] = useState(1);
  const [creatingDevices, setCreatingDevices] = useState(false);

  // Block body scroll when any modal is open
  useBlockBodyScroll(modalOpen || viewCable !== null || devicesModal !== null);

  const connectorIndex = useMemo(() => {
    return new Map(connectors.map((connector) => [connector.connector_id, connector]));
  }, [connectors]);

  const connectorCompatibility = useMemo(() => {
    const map = new Map<number, Set<number>>();
    cables.forEach((cable) => {
      if (!map.has(cable.connector1)) {
        map.set(cable.connector1, new Set());
      }
      map.get(cable.connector1)!.add(cable.connector2);

      if (!map.has(cable.connector2)) {
        map.set(cable.connector2, new Set());
      }
      map.get(cable.connector2)!.add(cable.connector1);
    });
    return map;
  }, [cables]);

  const getCompatibleConnectors = useCallback(
    (selected: number | '' | undefined) => {
      if (!selected || connectorCompatibility.size === 0) {
        return connectors;
      }
      const compatible = connectorCompatibility.get(selected);
      if (!compatible || compatible.size === 0) {
        return connectors.filter((connector) => connector.connector_id !== selected);
      }
      return connectors.filter((connector) => compatible.has(connector.connector_id));
    },
    [connectors, connectorCompatibility]
  );

  const connector2FilterOptions = useMemo(
    () => getCompatibleConnectors(connector1Filter === '' ? undefined : connector1Filter),
    [connector1Filter, getCompatibleConnectors]
  );

  useEffect(() => {
    if (connector1Filter === '' || connector2Filter === '') {
      return;
    }
    const compatible = connectorCompatibility.get(connector1Filter);
    if (!compatible || !compatible.has(connector2Filter)) {
      setConnector2Filter('');
    }
  }, [connector1Filter, connector2Filter, connectorCompatibility]);

  const debouncedSearch = useDebouncedValue(searchTerm, 300);

  // Update view mode based on screen size
  useEffect(() => {
    const handleResize = () => {
      const isMobile = window.innerWidth < 768;
      setViewMode(isMobile ? 'cards' : 'table');
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const fetchCables = useCallback(async () => {
    setLoadingCables(true);
    try {
      const params: {
        search?: string;
        connector1?: number;
        connector2?: number;
        type?: number;
        length_min?: number;
        length_max?: number;
      } = {};

      if (debouncedSearch) params.search = debouncedSearch;
      if (connector1Filter !== '') params.connector1 = connector1Filter;
      if (connector2Filter !== '') params.connector2 = connector2Filter;
      if (typeFilter !== '') params.type = typeFilter;
      if (lengthMinFilter !== '') params.length_min = lengthMinFilter;
      if (lengthMaxFilter !== '') params.length_max = lengthMaxFilter;

      const { data } = await cablesAdminApi.getAll(params);
      setCables(data || []);
    } catch (error) {
      console.error('Failed to load cables:', error);
      setCables([]);
    } finally {
      setLoadingCables(false);
    }
  }, [debouncedSearch, connector1Filter, connector2Filter, typeFilter, lengthMinFilter, lengthMaxFilter]);

  const loadMetadata = useCallback(async () => {
    try {
      const [connectorsRes, typesRes] = await Promise.all([
        cablesAdminApi.getConnectors(),
        cablesAdminApi.getTypes(),
      ]);

      setConnectors(connectorsRes.data || []);
      setCableTypes(typesRes.data || []);
    } catch (error) {
      console.error('Failed to load metadata:', error);
    }
  }, []);

  useEffect(() => {
    fetchCables();
  }, [fetchCables]);

  useEffect(() => {
    loadMetadata();
  }, [loadMetadata]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    await fetchCables();
    setRefreshing(false);
  }, [fetchCables]);

  const clearFilters = () => {
    setSearchTerm('');
    setConnector1Filter('');
    setConnector2Filter('');
    setTypeFilter('');
    setLengthMinFilter('');
    setLengthMaxFilter('');
    setExpandedCombos(new Set());
  };

  const openCreateModal = () => {
    setEditingCable(null);
    setFormData(initialFormData);
    setModalOpen(true);
  };

  const openEditModal = (cable: CableType) => {
    setEditingCable(cable.cable_id);
    setFormData({
      name: cable.name || '',
      connector1: cable.connector1,
      connector2: cable.connector2,
      typ: cable.typ,
      length: cable.length,
      mm2: cable.mm2 || 0,
    });
    setModalOpen(true);
  };

  const handleDelete = async (cableId: number) => {
    if (!window.confirm(t('admin.cables.confirmDelete'))) {
      return;
    }

    try {
      await cablesAdminApi.delete(cableId);
      await fetchCables();
    } catch (error: unknown) {
      console.error('Failed to delete cable:', error);
      alert(t('admin.cables.errors.delete'));
    }
  };

  const handleOpenDevicesModal = useCallback(async (cableId: number) => {
    setDevicesModal({ cableId });
    setCableDevices([]);
    setLoadingDevices(true);
    setDevicePrefix('');
    setDeviceQuantity(1);
    try {
      const { data } = await cablesAdminApi.getDevices(cableId);
      setCableDevices(data || []);
    } catch (error) {
      console.error('Failed to load cable devices:', error);
      setCableDevices([]);
      setDevicesModal(null);
      alert(t('admin.cables.errors.loadDevices'));
    } finally {
      setLoadingDevices(false);
    }
  }, [t]);

  const handleCreateCableDevices = async () => {
    if (!devicesModal || !devicePrefix.trim() || deviceQuantity <= 0) return;
    setCreatingDevices(true);
    try {
      const { data } = await cablesAdminApi.createDevices(devicesModal.cableId, {
        quantity: deviceQuantity,
        prefix: devicePrefix.trim(),
      });
      alert(t('admin.cables.devicesCreated', { count: data.created_count }));
    } catch (error) {
      console.error('Failed to create cable devices:', error);
      alert(t('admin.cables.errors.createDevices'));
      return;
    } finally {
      setCreatingDevices(false);
    }
    // Refresh device list separately so a refresh failure doesn't mask a successful create
    try {
      const { data: devices } = await cablesAdminApi.getDevices(devicesModal.cableId);
      setCableDevices(devices || []);
    } catch (refreshError) {
      console.error('Failed to refresh cable devices after creation:', refreshError);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!formData.connector1 || !formData.connector2 || !formData.typ) {
      alert(t('admin.cables.errors.requiredFields'));
      return;
    }

    if (formData.length <= 0) {
      alert(t('admin.cables.errors.lengthPositive'));
      return;
    }

    setSubmitting(true);

    try {
      if (editingCable) {
        // Update existing cable
        const updateData: CableUpdateInput = {
          name: formData.name || undefined,
          connector1: formData.connector1,
          connector2: formData.connector2,
          typ: formData.typ,
          length: formData.length,
          mm2: formData.mm2 || undefined,
        };

        await cablesAdminApi.update(editingCable, updateData);
      } else {
        // Create new cable
        const createData: CableCreateInput = {
          name: formData.name || undefined,
          connector1: formData.connector1,
          connector2: formData.connector2,
          typ: formData.typ,
          length: formData.length,
          mm2: formData.mm2 || undefined,
        };

        await cablesAdminApi.create(createData);
      }

      setModalOpen(false);
      setFormData(initialFormData);
      await fetchCables();
    } catch (error: unknown) {
      console.error('Failed to save cable:', error);
      alert(t('admin.cables.errors.save'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleViewCable = async (cable: CableType) => {
    setViewCable(cable);
  };

  const formatGenderText = (gender?: string | null) => {
    if (!gender) return '';
    return gender === 'male' ? t('admin.cables.gender.male') : t('admin.cables.gender.female');
  };

  const formatConnectorLabel = (connector: CableConnector, overrideGender?: string | null) => {
    const abbr = connector.abbreviation ? ` (${connector.abbreviation})` : '';
    const genderText = formatGenderText(overrideGender ?? connector.gender);
    return `${connector.name}${abbr}${genderText ? ` • ${genderText}` : ''}`;
  };

  const getConnectorDisplay = (connectorId: number, gender?: string | null) => {
    const connector = connectorIndex.get(connectorId);
    if (!connector) return '-';
    return formatConnectorLabel(connector, gender);
  };

  const getCableTypeName = (typeId: number) => {
    const type = cableTypes.find(t => t.cable_type_id === typeId);
    return type?.name || '-';
  };

  const buildCombinationKey = (cable: CableType) =>
    `${cable.typ}|${cable.connector1}|${cable.connector2}|${cable.length}`;

  const cableCombinationSummary = useMemo(() => {
    const summaryMap = new Map<string, {
      key: string;
      typeId: number;
      connector1: number;
      connector2: number;
      length: number;
      cables: CableType[];
    }>();

    cables.forEach((cable) => {
      const key = buildCombinationKey(cable);
      if (!summaryMap.has(key)) {
        summaryMap.set(key, {
          key,
          typeId: cable.typ,
          connector1: cable.connector1,
          connector2: cable.connector2,
          length: cable.length,
          cables: [],
        });
      }
      summaryMap.get(key)!.cables.push(cable);
    });

    return Array.from(summaryMap.values())
      .map((entry) => ({
        ...entry,
        typeName: getCableTypeName(entry.typeId),
        connector1Label: getConnectorDisplay(entry.connector1),
        connector2Label: getConnectorDisplay(entry.connector2),
        lengthLabel: `${entry.length.toFixed(2)} m`,
        count: entry.cables.length,
      }))
      .sort((a, b) => a.typeName.localeCompare(b.typeName, 'de', { sensitivity: 'base' }));
  }, [cables, getCableTypeName, getConnectorDisplay]);

  const totalCableCount = useMemo(
    () => cableCombinationSummary.reduce((sum, entry) => sum + entry.count, 0),
    [cableCombinationSummary]
  );

  const [expandedCombos, setExpandedCombos] = useState<Set<string>>(new Set());

  const toggleComboExpanded = (key: string) => {
    setExpandedCombos((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };


  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Cable className="w-6 h-6 text-accent-red" />
          <h2 className="text-2xl font-bold text-white">{t('admin.cables.title')}</h2>
        </div>
        <button
          onClick={openCreateModal}
          className="btn-primary flex items-center gap-2"
        >
          <Plus className="w-5 h-5" />
          {t('admin.cables.newCable')}
        </button>
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between text-sm text-gray-400">
        <span>{t('admin.cables.combinationsCount', { count: cableCombinationSummary.length })}</span>
        <span>{t('admin.cables.totalStock')}: <span className="text-white font-semibold">{totalCableCount}</span></span>
      </div>

      {/* Filters */}
      <div className="glass-dark rounded-xl p-4 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
          {/* Search */}
          <div className="relative lg:col-span-2">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 sm:w-5 sm:h-5 text-gray-400 pointer-events-none" />
            <input
              type="text"
              placeholder={t('admin.cables.searchPlaceholder')}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="input-field pl-10 w-full"
              title={t('common.search')}
            />
          </div>

          {/* Connector 1 Filter */}
          <SearchableSelect
            value={connector1Filter === '' ? '' : String(connector1Filter)}
            onChange={(v) => setConnector1Filter(v ? Number(v) : '')}
            options={[
              { value: '', label: t('admin.cables.filters.allConnector1') },
              ...connectors.map((connector) => ({
                value: String(connector.connector_id),
                label: formatConnectorLabel(connector),
              })),
            ]}
            className="input-field p-0 border-0"
            title={t('admin.cables.connector1')}
          />

          {/* Connector 2 Filter */}
          <SearchableSelect
            value={connector2Filter === '' ? '' : String(connector2Filter)}
            onChange={(v) => setConnector2Filter(v ? Number(v) : '')}
            options={[
              { value: '', label: t('admin.cables.filters.allConnector2') },
              ...connector2FilterOptions.map((connector) => ({
                value: String(connector.connector_id),
                label: formatConnectorLabel(connector),
              })),
            ]}
            className="input-field p-0 border-0"
            title={t('admin.cables.connector2')}
          />

          {/* Type Filter */}
          <SearchableSelect
            value={typeFilter === '' ? '' : String(typeFilter)}
            onChange={(v) => setTypeFilter(v ? Number(v) : '')}
            options={[
              { value: '', label: t('admin.cables.filters.allTypes') },
              ...cableTypes.map((type) => ({
                value: String(type.cable_type_id),
                label: type.name,
              })),
            ]}
            className="input-field p-0 border-0"
            title={t('admin.cables.type')}
          />

          {/* Length Min */}
          <input
            type="number"
            placeholder={t('admin.cables.filters.minLength')}
            min="0"
            step="0.1"
            value={lengthMinFilter}
            onChange={(e) => setLengthMinFilter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
            title={t('admin.cables.filters.minLength')}
          />

          {/* Length Max */}
          <input
            type="number"
            placeholder={t('admin.cables.filters.maxLength')}
            min="0"
            step="0.1"
            value={lengthMaxFilter}
            onChange={(e) => setLengthMaxFilter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
            title={t('admin.cables.filters.maxLength')}
          />
        </div>

        {/* Action Buttons */}
        <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
          <div className="flex flex-wrap gap-2">
            <button
              onClick={clearFilters}
              className="px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors flex items-center gap-1"
            >
              <X className="w-4 h-4" />
              <span className="hidden sm:inline">{t('admin.devices.clearFilters')}</span>
              <span className="sm:hidden">{t('common.delete')}</span>
            </button>
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors disabled:opacity-50 flex items-center gap-1"
            >
              <RefreshCcw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
              <span className="hidden sm:inline">{t('common.update')}</span>
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

      {/* Cable List */}
      {loadingCables ? (
        <div className="text-center py-12 text-gray-400">{t('admin.cables.loading')}</div>
      ) : cableCombinationSummary.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          {debouncedSearch || connector1Filter || connector2Filter || typeFilter || lengthMinFilter || lengthMaxFilter
            ? t('admin.cables.emptyFiltered')
            : t('admin.cables.empty')}
        </div>
      ) : viewMode === 'table' ? (
        <div className="glass-dark rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-white/5">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">{t('admin.cables.combination')}</th>
                  <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">{t('admin.devices.quantity')}</th>
                  <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">{t('labels.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {cableCombinationSummary.map((combo) => {
                  const isExpanded = expandedCombos.has(combo.key);
                  return (
                    <Fragment key={combo.key}>
                      <tr>
                        <td className="px-4 py-3 text-sm text-white">
                          <button
                            onClick={() => toggleComboExpanded(combo.key)}
                            className="inline-flex items-center gap-2 text-left font-semibold hover:text-accent-red transition-colors"
                          >
                            {isExpanded ? (
                              <ChevronDown className="w-4 h-4 text-accent-red" />
                            ) : (
                              <ChevronRight className="w-4 h-4 text-gray-400" />
                            )}
                            <span>
                              {combo.typeName || t('admin.cables.unknown')} ({combo.connector1Label} - {combo.connector2Label}) • {combo.lengthLabel}
                            </span>
                          </button>
                        </td>
                        <td className="px-4 py-3 text-right text-sm text-gray-200 font-semibold">{combo.count}</td>
                        <td className="px-4 py-3 text-right">
                          <button
                            onClick={() => toggleComboExpanded(combo.key)}
                            className="px-3 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors"
                          >
                            {isExpanded ? t('common.close') : t('casesPage.details')}
                          </button>
                        </td>
                      </tr>
                      {isExpanded && (
                        <tr>
                          <td colSpan={3} className="bg-white/5 px-4 pb-4">
                            <div className="overflow-x-auto">
                              <table className="w-full text-sm">
                                <thead>
                                  <tr className="text-left text-gray-400">
                                    <th className="py-2 pr-2">ID</th>
                                    <th className="py-2 pr-2">{t('common.name')}</th>
                                    <th className="py-2 pr-2">{t('admin.cables.connector1')}</th>
                                    <th className="py-2 pr-2">{t('admin.cables.connector2')}</th>
                                    <th className="py-2 pr-2">{t('admin.cables.type')}</th>
                                    <th className="py-2 pr-2">{t('admin.cables.length')}</th>
                                    <th className="py-2 pr-2">mm²</th>
                                    <th className="py-2 text-right">{t('labels.actions')}</th>
                                  </tr>
                                </thead>
                                <tbody className="divide-y divide-white/10">
                                  {combo.cables.map((cable) => (
                                    <tr key={cable.cable_id}>
                                      <td className="py-2 pr-2 text-gray-400">#{cable.cable_id}</td>
                                      <td className="py-2 pr-2 text-white">{cable.name || '-'}</td>
                                      <td className="py-2 pr-2 text-gray-300">
                                        {getConnectorDisplay(cable.connector1, cable.connector1_gender)}
                                      </td>
                                      <td className="py-2 pr-2 text-gray-300">
                                        {getConnectorDisplay(cable.connector2, cable.connector2_gender)}
                                      </td>
                                      <td className="py-2 pr-2 text-gray-300">{getCableTypeName(cable.typ)}</td>
                                      <td className="py-2 pr-2 text-gray-300">{cable.length} m</td>
                                      <td className="py-2 pr-2 text-gray-300">{cable.mm2 ? `${cable.mm2} mm²` : '-'}</td>
                                      <td className="py-2 text-right">
                                        <div className="flex justify-end gap-2">
                                          <button
                                            onClick={() => handleViewCable(cable)}
                                            className="p-2 bg-white/5 hover:bg-white/10 rounded-lg text-gray-300 hover:text-white transition-colors"
                                            title={t('casesPage.details')}
                                            aria-label={t('casesPage.details')}
                                          >
                                            <Eye className="w-4 h-4" />
                                          </button>
                                          <button
                                            onClick={() => openEditModal(cable)}
                                            className="p-2 bg-white/5 hover:bg-white/10 rounded-lg text-gray-300 hover:text-white transition-colors"
                                            title={t('common.edit')}
                                            aria-label={t('common.edit')}
                                          >
                                            <Pencil className="w-4 h-4" />
                                          </button>
                                          <button
                                            onClick={() => handleOpenDevicesModal(cable.cable_id)}
                                            className="p-2 bg-cyan-500/10 hover:bg-cyan-500/20 rounded-lg text-cyan-400 transition-colors"
                                            title={t('admin.cables.manageDevices')}
                                            aria-label={t('admin.cables.manageDevices')}
                                          >
                                            <Box className="w-4 h-4" />
                                          </button>
                                          <button
                                            onClick={() => handleDelete(cable.cable_id)}
                                            className="p-2 bg-red-500/10 hover:bg-red-500/20 rounded-lg text-red-400 transition-colors"
                                            title={t('common.delete')}
                                            aria-label={t('common.delete')}
                                          >
                                            <Trash2 className="w-4 h-4" />
                                          </button>
                                        </div>
                                      </td>
                                    </tr>
                                  ))}
                                </tbody>
                              </table>
                            </div>
                          </td>
                        </tr>
                      )}
                    </Fragment>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {cableCombinationSummary.map((combo) => (
            <div key={combo.key} className="glass-dark rounded-xl p-4 space-y-3">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-bold text-white">{combo.typeName}</h3>
                  <p className="text-sm text-gray-400">
                    {combo.connector1Label} → {combo.connector2Label} • {combo.lengthLabel}
                  </p>
                </div>
                <span className="px-2 py-1 bg-white/10 text-white rounded-full text-xs font-semibold">
                  {combo.count}
                </span>
              </div>

              <div className="space-y-3">
                {combo.cables.map((cable) => (
                  <div key={cable.cable_id} className="rounded-xl border border-white/10 p-3 space-y-2">
                    <div className="flex items-center justify-between">
                      <p className="text-sm text-white font-semibold">{cable.name || t('admin.cables.cableWithId', { id: cable.cable_id })}</p>
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleViewCable(cable)}
                          className="p-2 bg-white/5 hover:bg-white/10 rounded-lg text-gray-300 hover:text-white transition-colors"
                          title={t('casesPage.details')}
                          aria-label={t('casesPage.details')}
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => openEditModal(cable)}
                          className="p-2 bg-white/5 hover:bg-white/10 rounded-lg text-gray-300 hover:text-white transition-colors"
                          title={t('common.edit')}
                          aria-label={t('common.edit')}
                        >
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleOpenDevicesModal(cable.cable_id)}
                          className="p-2 bg-cyan-500/10 hover:bg-cyan-500/20 rounded-lg text-cyan-400 transition-colors"
                          title={t('admin.cables.manageDevices')}
                          aria-label={t('admin.cables.manageDevices')}
                        >
                          <Box className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(cable.cable_id)}
                          className="p-2 bg-red-500/10 hover:bg-red-500/20 rounded-lg text-red-400 transition-colors"
                          title={t('common.delete')}
                          aria-label={t('common.delete')}
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                    <div className="text-xs text-gray-400 flex flex-wrap gap-3">
                      <span>#{cable.cable_id}</span>
                      <span>{cable.mm2 ? `${cable.mm2} mm²` : t('admin.cables.mm2na')}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
      {/* Create/Edit Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
          <div className="glass-dark rounded-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-white">
                {editingCable ? t('admin.cables.editCable') : t('admin.cables.createCable')}
              </h3>
              <button
                onClick={() => setModalOpen(false)}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors"
                title={t('common.close')}
                aria-label={t('common.close')}
              >
                <X className="w-6 h-6 text-gray-400" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {t('common.name')} ({t('modals.productDependencies.optional')})
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="input-field w-full"
                  placeholder={t('admin.cables.namePlaceholder')}
                  title={t('common.name')}
                />
              </div>

              {/* Connectors */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.cables.connector1')} *
                  </label>
                  <select
                    value={formData.connector1 || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, connector1: e.target.value ? Number(e.target.value) : undefined })
                    }
                    className="input-field w-full"
                    required
                    title={t('admin.cables.connector1')}
                  >
                    <option value="">{t('admin.cables.selectConnector')}</option>
                    {connectors.map((connector) => (
                      <option key={connector.connector_id} value={connector.connector_id}>
                        {formatConnectorLabel(connector)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.cables.connector2')} *
                  </label>
                  <select
                    value={formData.connector2 || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, connector2: e.target.value ? Number(e.target.value) : undefined })
                    }
                    className="input-field w-full"
                    required
                    title={t('admin.cables.connector2')}
                  >
                    <option value="">{t('admin.cables.selectConnector')}</option>
                    {connectors.map((connector) => (
                      <option key={connector.connector_id} value={connector.connector_id}>
                        {formatConnectorLabel(connector)}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Cable Type */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  {t('admin.cables.type')} *
                </label>
                <select
                  value={formData.typ || ''}
                  onChange={(e) =>
                    setFormData({ ...formData, typ: e.target.value ? Number(e.target.value) : undefined })
                  }
                  className="input-field w-full"
                  required
                  title={t('admin.cables.type')}
                >
                  <option value="">{t('admin.cables.selectType')}</option>
                  {cableTypes.map((type) => (
                    <option key={type.cable_type_id} value={type.cable_type_id}>
                      {type.name}
                    </option>
                  ))}
                </select>
              </div>

              {/* Length and mm² */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.cables.lengthMeters')} *
                  </label>
                  <input
                    type="number"
                    min="0.1"
                    step="0.1"
                    value={formData.length}
                    onChange={(e) =>
                      setFormData({ ...formData, length: Number(e.target.value) || 1 })
                    }
                    className="input-field w-full"
                    required
                    title={t('admin.cables.lengthMeters')}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    {t('admin.cables.crossSection')}
                  </label>
                  <input
                    type="number"
                    min="0"
                    step="0.1"
                    value={formData.mm2}
                    onChange={(e) =>
                      setFormData({ ...formData, mm2: Number(e.target.value) || 0 })
                    }
                    className="input-field w-full"
                    title={t('admin.cables.crossSection')}
                  />
                </div>
              </div>

              {/* Submit Buttons */}
              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
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
                    : editingCable
                    ? t('common.update')
                    : t('common.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* View Cable Modal */}
      {viewCable && (
        <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
          <div className="glass-dark rounded-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-white">{t('admin.cables.detailsTitle')}</h3>
              <button
                onClick={() => setViewCable(null)}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors"
                title={t('common.close')}
                aria-label={t('common.close')}
              >
                <X className="w-6 h-6 text-gray-400" />
              </button>
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-400">{t('admin.cables.cableId')}</p>
                  <p className="text-white font-semibold">{viewCable.cable_id}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">{t('common.name')}</p>
                  <p className="text-white font-semibold">{viewCable.name || '-'}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">{t('admin.cables.connector1')}</p>
                  <p className="text-white font-semibold">
                    {getConnectorDisplay(viewCable.connector1, viewCable.connector1_gender)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">{t('admin.cables.connector2')}</p>
                  <p className="text-white font-semibold">
                    {getConnectorDisplay(viewCable.connector2, viewCable.connector2_gender)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">{t('admin.cables.type')}</p>
                  <p className="text-white font-semibold">{getCableTypeName(viewCable.typ)}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">{t('admin.cables.length')}</p>
                  <p className="text-white font-semibold">{viewCable.length}m</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">{t('admin.cables.crossSectionShort')}</p>
                  <p className="text-white font-semibold">{viewCable.mm2 ? `${viewCable.mm2}mm²` : '-'}</p>
                </div>
              </div>

              <div className="flex gap-3 pt-4">
                <button
                  onClick={() => {
                    setViewCable(null);
                    openEditModal(viewCable);
                  }}
                  className="flex-1 btn-primary flex items-center justify-center gap-2"
                >
                  <Pencil className="w-5 h-5" />
                  {t('common.edit')}
                </button>
                <button
                  onClick={() => {
                    setViewCable(null);
                    handleDelete(viewCable.cable_id);
                  }}
                  className="px-4 py-3 bg-red-500/20 hover:bg-red-500/30 rounded-lg font-semibold text-red-400 transition-colors flex items-center justify-center gap-2"
                >
                  <Trash2 className="w-5 h-5" />
                  {t('common.delete')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Cable Devices Modal */}
      {devicesModal && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark w-full max-w-4xl rounded-2xl shadow-2xl border border-white/10 flex flex-col max-h-[85vh]">
              <div className="flex items-center justify-between p-6 border-b border-white/10">
                <div>
                  <h2 className="text-2xl font-bold text-white flex items-center gap-2">
                    <Box className="w-6 h-6 text-cyan-400" />
                    {t('admin.cables.devicesTitle', { id: devicesModal.cableId })}
                  </h2>
                  <p className="text-sm text-gray-400 mt-1">
                    {t('admin.cables.deviceCount', { count: cableDevices.length })}
                  </p>
                </div>
                <button
                  onClick={() => { setDevicesModal(null); setCableDevices([]); }}
                  className="px-4 py-2 rounded-lg text-sm font-semibold bg-white/10 text-white hover:bg-white/20 transition-colors"
                  aria-label={t('common.close')}
                  title={t('common.close')}
                >
                  {t('common.close')}
                </button>
              </div>

              {/* Add devices form */}
              <div className="p-6 border-b border-white/10">
                <h3 className="text-sm font-semibold text-gray-300 mb-3">{t('admin.cables.addDevices')}</h3>
                <div className="flex flex-wrap gap-3 items-end">
                  <div>
                    <label className="block text-xs text-gray-400 mb-1">{t('admin.cables.devicePrefix')}</label>
                    <input
                      type="text"
                      value={devicePrefix}
                      onChange={e => setDevicePrefix(e.target.value.toUpperCase())}
                      maxLength={20}
                      placeholder={t('admin.cables.devicePrefixPlaceholder')}
                      className="rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-cyan-400 text-sm w-32"
                      title={t('admin.cables.devicePrefix')}
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-gray-400 mb-1">{t('admin.cables.deviceQuantity')}</label>
                    <input
                      type="number"
                      min="1"
                      max="100"
                      value={deviceQuantity}
                      onChange={e => setDeviceQuantity(Math.max(1, Math.min(100, parseInt(e.target.value) || 1)))}
                      placeholder={t('admin.cables.deviceQuantityPlaceholder')}
                      className="rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-cyan-400 text-sm w-24"
                      title={t('admin.cables.deviceQuantity')}
                    />
                  </div>
                  <button
                    onClick={handleCreateCableDevices}
                    disabled={creatingDevices || !devicePrefix.trim() || deviceQuantity <= 0}
                    className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-semibold text-white hover:bg-cyan-700 transition disabled:opacity-50 disabled:cursor-not-allowed"
                    aria-busy={creatingDevices}
                  >
                    {creatingDevices ? <RefreshCcw className="h-4 w-4 animate-spin" aria-hidden="true" /> : t('admin.cables.createDevices')}
                  </button>
                </div>
              </div>

              {/* Device list */}
              <div className="overflow-y-auto flex-1">
                {loadingDevices ? (
                  <div className="p-6 text-center text-gray-400 text-sm">
                    {t('admin.cables.loadingDevices')}
                  </div>
                ) : cableDevices.length === 0 ? (
                  <div className="p-6 text-center text-gray-400 text-sm">
                    {t('admin.cables.noDevices')}
                  </div>
                ) : (
                  <div className="p-6 space-y-3">
                    {cableDevices.map(device => (
                      <div
                        key={device.device_id}
                        className="glass rounded-lg p-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border border-white/10"
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-semibold text-white text-sm sm:text-base truncate">
                              {device.device_id}
                            </span>
                            {device.status && (
                              <span
                                className={`text-[10px] sm:text-xs font-semibold px-2 py-1 rounded-full bg-white/10 uppercase tracking-wide ${getStatusColor(device.status)}`}
                              >
                                {formatStatus(device.status)}
                              </span>
                            )}
                          </div>
                          <div className="flex flex-wrap items-center gap-2 text-xs sm:text-sm text-gray-400 mt-1">
                            {device.zone_name && <span>{device.zone_name}</span>}
                            {device.zone_code && <span className="text-gray-500 font-mono">({device.zone_code})</span>}
                            {device.serial_number && <span className="text-gray-500">SN: {device.serial_number}</span>}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        </ModalPortal>
      )}
    </div>
  );
}
