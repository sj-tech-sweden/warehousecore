import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Cable,
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
import { cablesAdminApi } from '../../lib/api';
import type { Cable as CableType, CableConnector, CableType as CableTypeData, CableCreateInput, CableUpdateInput } from '../../lib/api';

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
  const [cables, setCables] = useState<CableType[]>([]);
  const [connectors, setConnectors] = useState<CableConnector[]>([]);
  const [cableTypes, setCableTypes] = useState<CableTypeData[]>([]);
  const [loadingCables, setLoadingCables] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewCable, setViewCable] = useState<CableType | null>(null);
  const [editingCable, setEditingCable] = useState<number | null>(null);
  const [formData, setFormData] = useState<CableFormData>(initialFormData);
  const [submitting, setSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'cards'>('table');
  const [searchTerm, setSearchTerm] = useState('');
  const [connector1Filter, setConnector1Filter] = useState<number | ''>('');
  const [connector2Filter, setConnector2Filter] = useState<number | ''>('');
  const [typeFilter, setTypeFilter] = useState<number | ''>('');
  const [lengthMinFilter, setLengthMinFilter] = useState<number | ''>('');
  const [lengthMaxFilter, setLengthMaxFilter] = useState<number | ''>('');
  const [refreshing, setRefreshing] = useState(false);

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
    if (!window.confirm('Möchten Sie dieses Kabel wirklich löschen?')) {
      return;
    }

    try {
      await cablesAdminApi.delete(cableId);
      await fetchCables();
    } catch (error: unknown) {
      console.error('Failed to delete cable:', error);
      alert('Fehler beim Löschen des Kabels');
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!formData.connector1 || !formData.connector2 || !formData.typ) {
      alert('Bitte füllen Sie alle Pflichtfelder aus.');
      return;
    }

    if (formData.length <= 0) {
      alert('Die Länge muss größer als 0 sein.');
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
      alert('Fehler beim Speichern des Kabels');
    } finally {
      setSubmitting(false);
    }
  };

  const handleViewCable = async (cable: CableType) => {
    setViewCable(cable);
  };

  const formatGenderText = (gender?: string | null) => {
    if (!gender) return '';
    return gender === 'male' ? 'male' : 'female';
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

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Cable className="w-6 h-6 text-accent-red" />
          <h2 className="text-2xl font-bold text-white">Kabel-Verwaltung</h2>
        </div>
        <button
          onClick={openCreateModal}
          className="btn-primary flex items-center gap-2"
        >
          <Plus className="w-5 h-5" />
          Neues Kabel
        </button>
      </div>

      {/* Filters */}
      <div className="glass-dark rounded-xl p-4 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-6 gap-4">
          {/* Search */}
          <div className="relative lg:col-span-2">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Suchen (Name)..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="input-field pl-10 w-full"
            />
          </div>

          {/* Connector 1 Filter */}
          <select
            value={connector1Filter}
            onChange={(e) => setConnector1Filter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
          >
            <option value="">Alle Stecker 1</option>
            {connectors.map((connector) => (
              <option key={connector.connector_id} value={connector.connector_id}>
                {formatConnectorLabel(connector)}
              </option>
            ))}
          </select>

          {/* Connector 2 Filter */}
          <select
            value={connector2Filter}
            onChange={(e) => setConnector2Filter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
          >
            <option value="">Alle Stecker 2</option>
            {connector2FilterOptions.map((connector) => (
              <option key={connector.connector_id} value={connector.connector_id}>
                {formatConnectorLabel(connector)}
              </option>
            ))}
          </select>

          {/* Type Filter */}
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
          >
            <option value="">Alle Typen</option>
            {cableTypes.map((type) => (
              <option key={type.cable_type_id} value={type.cable_type_id}>
                {type.name}
              </option>
            ))}
          </select>

          {/* Length Min */}
          <input
            type="number"
            placeholder="Min Länge (m)"
            min="0"
            step="0.1"
            value={lengthMinFilter}
            onChange={(e) => setLengthMinFilter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
          />

          {/* Length Max */}
          <input
            type="number"
            placeholder="Max Länge (m)"
            min="0"
            step="0.1"
            value={lengthMaxFilter}
            onChange={(e) => setLengthMaxFilter(e.target.value ? Number(e.target.value) : '')}
            className="input-field"
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
              Filter löschen
            </button>
            <button
              onClick={handleRefresh}
              disabled={refreshing}
              className="px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors disabled:opacity-50"
            >
              <RefreshCcw className={`w-4 h-4 inline mr-1 ${refreshing ? 'animate-spin' : ''}`} />
              Aktualisieren
            </button>
          </div>

          {/* View Mode Toggle */}
          <div className="flex gap-2">
            <button
              onClick={() => setViewMode('table')}
              className={`p-2 rounded-lg transition-colors ${
                viewMode === 'table' ? 'bg-accent-red text-white' : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
            >
              <List className="w-5 h-5" />
            </button>
            <button
              onClick={() => setViewMode('cards')}
              className={`p-2 rounded-lg transition-colors ${
                viewMode === 'cards' ? 'bg-accent-red text-white' : 'bg-white/5 text-gray-400 hover:bg-white/10'
              }`}
            >
              <LayoutGrid className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>

      {/* Cable List */}
      {loadingCables ? (
        <div className="text-center py-12 text-gray-400">Lädt Kabel...</div>
      ) : cables.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          {debouncedSearch || connector1Filter || connector2Filter || typeFilter || lengthMinFilter || lengthMaxFilter
            ? 'Keine Kabel gefunden mit den aktuellen Filtern'
            : 'Noch keine Kabel vorhanden'}
        </div>
      ) : viewMode === 'table' ? (
        <div className="glass-dark rounded-xl overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-white/5">
                <tr>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">ID</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Name</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Stecker 1</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Stecker 2</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Typ</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">Länge</th>
                  <th className="px-4 py-3 text-left text-sm font-semibold text-gray-300">mm²</th>
                  <th className="px-4 py-3 text-right text-sm font-semibold text-gray-300">Aktionen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {cables.map((cable) => (
                  <tr key={cable.cable_id} className="hover:bg-white/5 transition-colors">
                    <td className="px-4 py-3 text-sm text-gray-300">{cable.cable_id}</td>
                    <td className="px-4 py-3 text-sm text-white">{cable.name || '-'}</td>
                    <td className="px-4 py-3 text-sm text-gray-300">
                      {getConnectorDisplay(cable.connector1, cable.connector1_gender)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">
                      {getConnectorDisplay(cable.connector2, cable.connector2_gender)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">
                      {getCableTypeName(cable.typ)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-300">{cable.length}m</td>
                    <td className="px-4 py-3 text-sm text-gray-300">{cable.mm2 || '-'}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleViewCable(cable)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-gray-400 hover:text-white"
                          title="Details anzeigen"
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => openEditModal(cable)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-blue-400 hover:text-blue-300"
                          title="Bearbeiten"
                        >
                          <Pencil className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(cable.cable_id)}
                          className="p-1.5 hover:bg-white/10 rounded-lg transition-colors text-red-400 hover:text-red-300"
                          title="Löschen"
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
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {cables.map((cable) => (
            <div key={cable.cable_id} className="glass-dark rounded-xl p-4 space-y-3">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="font-bold text-white">{cable.name || `Kabel #${cable.cable_id}`}</h3>
                  <p className="text-sm text-gray-400">{getCableTypeName(cable.typ)}</p>
                </div>
                <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded-full text-xs font-semibold">
                  {cable.length}m
                </span>
              </div>

              <div className="space-y-1 text-sm">
                <p className="text-gray-300">
                  <span className="text-gray-500">Stecker 1:</span> {getConnectorDisplay(cable.connector1, cable.connector1_gender)}
                </p>
                <p className="text-gray-300">
                  <span className="text-gray-500">Stecker 2:</span> {getConnectorDisplay(cable.connector2, cable.connector2_gender)}
                </p>
                {cable.mm2 && (
                  <p className="text-gray-300">
                    <span className="text-gray-500">Querschnitt:</span> {cable.mm2}mm²
                  </p>
                )}
              </div>

              <div className="flex items-center gap-2 pt-2 border-t border-white/10">
                <button
                  onClick={() => handleViewCable(cable)}
                  className="flex-1 px-3 py-2 bg-white/5 hover:bg-white/10 rounded-lg text-sm text-gray-300 transition-colors flex items-center justify-center gap-2"
                >
                  <Eye className="w-4 h-4" />
                  Details
                </button>
                <button
                  onClick={() => openEditModal(cable)}
                  className="flex-1 px-3 py-2 bg-blue-500/20 hover:bg-blue-500/30 rounded-lg text-sm text-blue-400 transition-colors flex items-center justify-center gap-2"
                >
                  <Pencil className="w-4 h-4" />
                  Bearbeiten
                </button>
                <button
                  onClick={() => handleDelete(cable.cable_id)}
                  className="px-3 py-2 bg-red-500/20 hover:bg-red-500/30 rounded-lg text-sm text-red-400 transition-colors"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      {modalOpen && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="glass-dark rounded-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-white">
                {editingCable ? 'Kabel bearbeiten' : 'Neues Kabel erstellen'}
              </h3>
              <button
                onClick={() => setModalOpen(false)}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors"
              >
                <X className="w-6 h-6 text-gray-400" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Name (optional)
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="input-field w-full"
                  placeholder="z.B. Haupt-Stromkabel"
                />
              </div>

              {/* Connectors */}
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Stecker 1 *
                  </label>
                  <select
                    value={formData.connector1 || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, connector1: e.target.value ? Number(e.target.value) : undefined })
                    }
                    className="input-field w-full"
                    required
                  >
                    <option value="">Stecker auswählen...</option>
                    {connectors.map((connector) => (
                      <option key={connector.connector_id} value={connector.connector_id}>
                        {formatConnectorLabel(connector)}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Stecker 2 *
                  </label>
                  <select
                    value={formData.connector2 || ''}
                    onChange={(e) =>
                      setFormData({ ...formData, connector2: e.target.value ? Number(e.target.value) : undefined })
                    }
                    className="input-field w-full"
                    required
                  >
                    <option value="">Stecker auswählen...</option>
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
                  Kabeltyp *
                </label>
                <select
                  value={formData.typ || ''}
                  onChange={(e) =>
                    setFormData({ ...formData, typ: e.target.value ? Number(e.target.value) : undefined })
                  }
                  className="input-field w-full"
                  required
                >
                  <option value="">Typ auswählen...</option>
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
                    Länge (Meter) *
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
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Querschnitt (mm²)
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
                  Abbrechen
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="flex-1 btn-primary disabled:opacity-50"
                >
                  {submitting
                    ? 'Speichert...'
                    : editingCable
                    ? 'Aktualisieren'
                    : 'Erstellen'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* View Cable Modal */}
      {viewCable && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-4">
          <div className="glass-dark rounded-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-6">
              <h3 className="text-2xl font-bold text-white">Kabel-Details</h3>
              <button
                onClick={() => setViewCable(null)}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors"
              >
                <X className="w-6 h-6 text-gray-400" />
              </button>
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-400">Kabel-ID</p>
                  <p className="text-white font-semibold">{viewCable.cable_id}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Name</p>
                  <p className="text-white font-semibold">{viewCable.name || '-'}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Stecker 1</p>
                  <p className="text-white font-semibold">
                    {getConnectorDisplay(viewCable.connector1, viewCable.connector1_gender)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Stecker 2</p>
                  <p className="text-white font-semibold">
                    {getConnectorDisplay(viewCable.connector2, viewCable.connector2_gender)}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Kabeltyp</p>
                  <p className="text-white font-semibold">{getCableTypeName(viewCable.typ)}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Länge</p>
                  <p className="text-white font-semibold">{viewCable.length}m</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Querschnitt</p>
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
                  Bearbeiten
                </button>
                <button
                  onClick={() => {
                    setViewCable(null);
                    handleDelete(viewCable.cable_id);
                  }}
                  className="px-4 py-3 bg-red-500/20 hover:bg-red-500/30 rounded-lg font-semibold text-red-400 transition-colors flex items-center justify-center gap-2"
                >
                  <Trash2 className="w-5 h-5" />
                  Löschen
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
