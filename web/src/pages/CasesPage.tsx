import { useEffect, useMemo, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { Package, Search, MapPin, Box, RefreshCw } from 'lucide-react';
import {
  casesApi,
  devicesApi,
  ledApi,
  type CaseSummary,
  type CaseDetail,
  type CaseDevice,
  type Device,
} from '../lib/api';
import { formatStatus, getStatusColor } from '../lib/utils';
import { CaseDetailModal } from '../components/CaseDetailModal';
import { DeviceDetailModal } from '../components/DeviceDetailModal';

type StatusFilter = 'all' | 'free' | 'rented' | 'maintance';

interface ActionMessage {
  type: 'success' | 'error';
  text: string;
}

export function CasesPage() {
  const navigate = useNavigate();
  const [cases, setCases] = useState<CaseSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [selectedCase, setSelectedCase] = useState<CaseDetail | null>(null);
  const [caseDevices, setCaseDevices] = useState<CaseDevice[]>([]);
  const [caseModalOpen, setCaseModalOpen] = useState(false);
  const [caseModalLoading, setCaseModalLoading] = useState(false);
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [deviceModalOpen, setDeviceModalOpen] = useState(false);
  const [actionMessage, setActionMessage] = useState<ActionMessage | null>(null);

  useEffect(() => {
    void loadCases();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter]);

  const totalDevices = useMemo(
    () => cases.reduce((sum, current) => sum + current.device_count, 0),
    [cases]
  );

  const statusCounts = useMemo(() => {
    return cases.reduce<Record<string, number>>((acc, current) => {
      acc[current.status] = (acc[current.status] ?? 0) + 1;
      return acc;
    }, {});
  }, [cases]);

  const loadCases = async (overrideSearch?: string) => {
    setLoading(true);
    try {
      const params: { search?: string; status?: string } = {};
      const effectiveSearch = overrideSearch ?? search.trim();
      if (effectiveSearch) {
        params.search = effectiveSearch;
      }
      if (statusFilter !== 'all') {
        params.status = statusFilter;
      }
      const { data } = await casesApi.list(params);
      setCases(data.cases ?? []);
    } catch (error) {
      console.error('Failed to load cases:', error);
      setCases([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSearchSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await loadCases();
  };

  const handleResetFilters = async () => {
    setSearch('');
    setStatusFilter('all');
    await loadCases('');
  };

  const handleOpenCase = async (caseSummary: CaseSummary) => {
    setCaseModalLoading(true);
    try {
      const [detailRes, devicesRes] = await Promise.all([
        casesApi.getById(caseSummary.case_id),
        casesApi.getDevices(caseSummary.case_id),
      ]);
      setSelectedCase(detailRes.data);
      setCaseDevices(devicesRes.data);
      setCaseModalOpen(true);
    } catch (error) {
      console.error('Failed to load case:', error);
    } finally {
      setCaseModalLoading(false);
    }
  };

  const handleLocateDevice = async (device: CaseDevice) => {
    if (!device.zone_code) {
      setActionMessage({ type: 'error', text: 'Kein Fachcode vorhanden – Gerät nicht im Lager.' });
      clearActionMessage();
      return;
    }

    try {
      await ledApi.locateBin(device.zone_code);
      setActionMessage({
        type: 'success',
        text: `Fach ${device.zone_code} wird hervorgehoben.`,
      });
    } catch (error: any) {
      setActionMessage({
        type: 'error',
        text: 'LED-Befehl fehlgeschlagen: ' + (error.response?.data?.error || error.message || error.toString()),
      });
    } finally {
      clearActionMessage();
    }
  };

  const handleOpenZone = (info: { zone_id?: number; zone_code?: string }) => {
    if (info.zone_id) {
      navigate(`/zones/${info.zone_id}`);
      return;
    }
    if (info.zone_code) {
      navigate(`/zones?parent=${info.zone_code}`);
    }
  };

  const handleOpenDevice = async (deviceId: string) => {
    try {
      const { data } = await devicesApi.getById(deviceId);
      setSelectedDevice(data);
      setDeviceModalOpen(true);
    } catch (error) {
      console.error('Failed to load device:', error);
      setActionMessage({
        type: 'error',
        text: 'Gerätedetails konnten nicht geladen werden.',
      });
      clearActionMessage();
    }
  };

  const clearActionMessage = () => {
    setTimeout(() => setActionMessage(null), 4000);
  };

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl sm:text-3xl font-bold text-white mb-1 sm:mb-2">Cases</h2>
          <p className="text-sm sm:text-base text-gray-400">
            {cases.length} Cases • {totalDevices} Geräte gesamt
          </p>
        </div>
        <button
          onClick={() => loadCases()}
          className="self-start sm:self-auto px-4 py-2 rounded-lg text-sm font-semibold bg-white/10 hover:bg-white/20 transition-colors text-white flex items-center gap-2"
        >
          <RefreshCw className="w-4 h-4" />
          Aktualisieren
        </button>
      </div>

      <form onSubmit={handleSearchSubmit} className="glass rounded-xl sm:rounded-2xl p-3 sm:p-4 space-y-3 sm:space-y-0 sm:flex sm:items-center sm:gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Cases suchen…"
            className="w-full pl-10 pr-3 py-2.5 bg-white/10 backdrop-blur-md border border-white/20 rounded-lg text-sm text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
        </div>
        <div className="flex items-center gap-2">
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-sm text-white focus:outline-none focus:border-accent-red"
          >
            <option value="all">Alle Status</option>
            <option value="free">Frei</option>
            <option value="rented">Vermietet</option>
            <option value="maintance">Wartung</option>
          </select>
          <button
            type="submit"
            className="px-4 py-2 bg-accent-red/80 hover:bg-accent-red rounded-lg text-sm font-semibold text-white transition-colors"
          >
            Suchen
          </button>
          <button
            type="button"
            onClick={handleResetFilters}
            className="px-4 py-2 bg-white/10 hover:bg-white/20 rounded-lg text-sm font-semibold text-white transition-colors"
          >
            Zurücksetzen
          </button>
        </div>
      </form>

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

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
        <SummaryCard label="Frei" value={statusCounts['free'] ?? 0} tone="green" />
        <SummaryCard label="Vermietet" value={statusCounts['rented'] ?? 0} tone="red" />
        <SummaryCard label="Wartung" value={statusCounts['maintance'] ?? 0} tone="yellow" />
        <SummaryCard label="Geräte in Cases" value={totalDevices} tone="blue" />
      </div>

      <div className="glass-dark rounded-xl sm:rounded-2xl p-4 sm:p-6 border border-white/10 min-h-[320px]">
        {loading ? (
          <div className="flex flex-col items-center justify-center h-60 gap-3 text-gray-400">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red" />
            <p className="text-sm">Cases werden geladen…</p>
          </div>
        ) : cases.length === 0 ? (
          <div className="text-center py-12 text-gray-400 text-sm">
            Keine Cases gefunden.
          </div>
        ) : (
          <div className="space-y-3">
            {cases.map((caseItem) => (
              <div
                key={caseItem.case_id}
                className="rounded-xl border border-white/10 bg-white/[0.02] p-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 transition-colors hover:bg-white/[0.04]"
              >
                <div className="flex items-start gap-3 sm:gap-4">
                  <div className="p-3 rounded-lg bg-gradient-to-br from-accent-red/20 to-red-700/20 flex-shrink-0">
                    <Box className="w-5 h-5 text-accent-red" />
                  </div>
                  <div>
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="text-base sm:text-lg font-bold text-white">{caseItem.name}</h3>
                      <span
                        className={`text-[10px] sm:text-xs font-semibold px-2 py-1 rounded-full bg-white/10 uppercase tracking-wide ${getStatusColor(caseItem.status)}`}
                      >
                        {formatStatus(caseItem.status)}
                      </span>
                    </div>
                    {caseItem.description && (
                      <p className="text-sm text-gray-400 mt-1">{caseItem.description}</p>
                    )}
                    <div className="flex flex-wrap items-center gap-3 text-xs text-gray-400 mt-2">
                      <span className="flex items-center gap-1">
                        <Package className="w-3 h-3" />
                        {caseItem.device_count} Gerät{caseItem.device_count === 1 ? '' : 'e'}
                      </span>
                      {caseItem.zone_name && (
                        <span className="flex items-center gap-1">
                          <MapPin className="w-3 h-3" />
                          {caseItem.zone_name}
                          {caseItem.zone_code && (
                            <span className="font-mono text-gray-500 ml-1">{caseItem.zone_code}</span>
                          )}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleOpenCase(caseItem)}
                    className="px-4 py-2 rounded-lg text-sm font-semibold bg-white/10 hover:bg-white/20 transition-colors text-white"
                  >
                    Details
                  </button>
                  <button
                    onClick={() => handleOpenZone({ zone_id: caseItem.zone_id, zone_code: caseItem.zone_code })}
                    disabled={!caseItem.zone_id && !caseItem.zone_code}
                    className="flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold bg-accent-red/80 hover:bg-accent-red transition-colors text-white disabled:opacity-50"
                  >
                    <MapPin className="w-4 h-4" /> Zone
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <CaseDetailModal
        caseInfo={selectedCase}
        devices={caseDevices}
        isOpen={caseModalOpen}
        loading={caseModalLoading}
        onClose={() => setCaseModalOpen(false)}
        onLocateDevice={handleLocateDevice}
        onOpenDevice={handleOpenDevice}
        onOpenZone={handleOpenZone}
      />

      <DeviceDetailModal
        device={selectedDevice}
        isOpen={deviceModalOpen}
        onClose={() => setDeviceModalOpen(false)}
      />
    </div>
  );
}

interface SummaryCardProps {
  label: string;
  value: number;
  tone: 'green' | 'red' | 'yellow' | 'blue';
}

function SummaryCard({ label, value, tone }: SummaryCardProps) {
  const toneClass = {
    green: 'from-green-500/20 to-emerald-500/20 text-green-300',
    red: 'from-red-500/20 to-rose-500/20 text-red-300',
    yellow: 'from-yellow-500/20 to-amber-500/20 text-yellow-300',
    blue: 'from-sky-500/20 to-blue-600/20 text-sky-300',
  }[tone];

  return (
    <div className={`rounded-xl border border-white/10 bg-gradient-to-br ${toneClass} p-4 sm:p-5`}>
      <p className="text-xs uppercase tracking-wide text-gray-300">{label}</p>
      <p className="text-2xl font-bold text-white mt-2">{value}</p>
    </div>
  );
}
