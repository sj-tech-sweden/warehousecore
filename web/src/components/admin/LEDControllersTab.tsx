import { useEffect, useMemo, useState } from 'react';
import { ledApi, api, type LEDController, type LEDControllerPayload, type ZoneTypeDefinition } from '../../lib/api';
import { Plus, Save, X, RefreshCcw, Trash2, Cpu, Settings, RotateCw } from 'lucide-react';

type EditorTarget = number | 'new' | null;
type ConfigureTarget = number | null;

interface ControllerForm {
  controller_id: string;
  display_name: string;
  topic_suffix: string;
  is_active: boolean;
  metadata: string;
  zoneTypeIds: number[];
}

const defaultForm: ControllerForm = {
  controller_id: '',
  display_name: '',
  topic_suffix: '',
  is_active: true,
  metadata: '{\n  \n}',
  zoneTypeIds: [],
};

const ONLINE_THRESHOLD_MS = 5 * 60 * 1000; // 5 minutes

const formatUptime = (seconds?: number): string | null => {
  if (seconds === undefined || seconds === null || Number.isNaN(seconds)) return null;
  const totalSeconds = Math.max(0, Math.floor(seconds));
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  if (parts.length === 0) {
    const secs = totalSeconds % 60;
    parts.push(`${secs}s`);
  }
  return parts.join(' ');
};

export function LEDControllersTab() {
  const [controllers, setControllers] = useState<LEDController[]>([]);
  const [zoneTypes, setZoneTypes] = useState<ZoneTypeDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editor, setEditor] = useState<EditorTarget>(null);
  const [form, setForm] = useState<ControllerForm>(defaultForm);
  const [message, setMessage] = useState<string>('');
  const [configureTarget, setConfigureTarget] = useState<ConfigureTarget>(null);
  const [configureLedCount, setConfigureLedCount] = useState<number>(600);
  const [configureDataPin, setConfigureDataPin] = useState<number>(0);
  const [configureChipset, setConfigureChipset] = useState<string>('SK6812_GRBW');
  const [configuring, setConfiguring] = useState(false);
  const [restarting, setRestarting] = useState<number | null>(null);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [controllerRes, zoneTypeRes] = await Promise.all([
        ledApi.getControllers(),
        api.get<ZoneTypeDefinition[]>('/admin/zone-types'),
      ]);
      setControllers(controllerRes.data);
      setZoneTypes(zoneTypeRes.data);
    } catch (error) {
      console.error('Failed to load controllers:', error);
      setMessage('Fehler beim Laden der Controller.');
    } finally {
      setLoading(false);
    }
  };

  const startNew = () => {
    setEditor('new');
    setForm({ ...defaultForm, metadata: '{\n\n}', zoneTypeIds: [] });
  };

  const startEdit = (controller: LEDController) => {
    setEditor(controller.id);
    setForm({
      controller_id: controller.controller_id,
      display_name: controller.display_name,
      topic_suffix: controller.topic_suffix,
      is_active: controller.is_active,
      metadata: controller.metadata ? JSON.stringify(controller.metadata, null, 2) : '{\n\n}',
      zoneTypeIds: controller.zone_types?.map((zt) => zt.id) ?? [],
    });
  };

  const resetEditor = () => {
    setEditor(null);
    setForm(defaultForm);
    setSaving(false);
  };

  const parseMetadata = (): Record<string, unknown> | null => {
    if (!form.metadata.trim()) return null;
    try {
      return JSON.parse(form.metadata);
    } catch (error) {
      alert('Ungültiges Metadata JSON: ' + (error as Error).message);
      throw error;
    }
  };

  const sortedZoneTypes = useMemo(() => zoneTypes.slice().sort((a, b) => a.label.localeCompare(b.label, 'de')), [zoneTypes]);

  const selectedZones = useMemo(
    () => sortedZoneTypes.filter((zone) => form.zoneTypeIds.includes(zone.id)),
    [sortedZoneTypes, form.zoneTypeIds],
  );

  const handleZoneSelectChange = (values: number[]) => {
    setForm((prev) => ({ ...prev, zoneTypeIds: values }));
  };

  const handleSave = async () => {
    try {
      setSaving(true);
      const payload: LEDControllerPayload = {
        controller_id: editor === 'new' ? form.controller_id.trim() : undefined,
        display_name: form.display_name.trim(),
        topic_suffix: form.topic_suffix.trim(),
        is_active: form.is_active,
        metadata: parseMetadata(),
        zone_type_ids: form.zoneTypeIds,
      };

      if (editor === 'new') {
        if (!payload.controller_id) {
          alert('Controller ID ist erforderlich.');
          return;
        }
        if (!payload.display_name) {
          alert('Anzeigename ist erforderlich.');
          return;
        }
        await ledApi.createController(payload);
        setMessage('✓ Controller angelegt');
      } else if (typeof editor === 'number') {
        await ledApi.updateController(editor, payload);
        setMessage('✓ Controller aktualisiert');
      }
      resetEditor();
      loadData();
    } catch (error: any) {
      const msg = error.response?.data?.error || error.message || 'Unbekannter Fehler';
      alert('Fehler: ' + msg);
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Controller wirklich löschen?')) return;
    try {
      await ledApi.deleteController(id);
      setMessage('✓ Controller gelöscht');
      if (editor === id) {
        resetEditor();
      }
      loadData();
    } catch (error: any) {
      const msg = error.response?.data?.error || error.message || 'Unbekannter Fehler';
      alert('Fehler: ' + msg);
    }
  };

  const startConfigure = (controller: LEDController) => {
    const statusData = controller.status_data as Record<string, unknown> | undefined;
    const currentLedCount = typeof statusData?.['led_count'] === 'number' ? statusData['led_count'] as number : 600;
    const currentDataPin = typeof statusData?.['data_pin'] === 'number' ? statusData['data_pin'] as number : 0;
    const currentChipset = typeof statusData?.['chipset'] === 'string' ? statusData['chipset'] as string : 'SK6812_GRBW';
    setConfigureTarget(controller.id);
    setConfigureLedCount(currentLedCount);
    setConfigureDataPin(currentDataPin);
    setConfigureChipset(currentChipset);
  };

  const handleConfigure = async () => {
    if (configureTarget === null) return;
    if (configureLedCount < 1 || configureLedCount > 1200) {
      alert('LED-Anzahl muss zwischen 1 und 1200 liegen.');
      return;
    }
    if (configureDataPin < 0 || configureDataPin > 50) {
      alert('Data Pin muss zwischen 0 und 50 liegen.');
      return;
    }
    try {
      setConfiguring(true);
      await ledApi.configureController(configureTarget, {
        led_count: configureLedCount,
        data_pin: configureDataPin,
        chipset: configureChipset,
      });
      setMessage(`✓ Konfiguration gesendet. Neustart erforderlich für Pin/Chipset-Änderungen.`);
      setConfigureTarget(null);
      // Reload after 2 seconds to show updated values
      setTimeout(() => loadData(), 2000);
    } catch (error: any) {
      const msg = error.response?.data?.error || error.message || 'Unbekannter Fehler';
      alert('Fehler: ' + msg);
    } finally {
      setConfiguring(false);
    }
  };

  const handleRestart = async (controllerId: number) => {
    if (!confirm('ESP32 wirklich neu starten? Der Controller wird für ca. 5-10 Sekunden offline sein.')) return;

    try {
      setRestarting(controllerId);
      await ledApi.restartController(controllerId);
      setMessage('✓ Neustart-Befehl gesendet. ESP32 startet in 2 Sekunden neu.');
      // Reload after 10 seconds to show controller coming back online
      setTimeout(() => {
        loadData();
        setRestarting(null);
      }, 10000);
    } catch (error: any) {
      const msg = error.response?.data?.error || error.message || 'Unbekannter Fehler';
      alert('Fehler: ' + msg);
      setRestarting(null);
    }
  };

  const handleDeleteOffline = async () => {
    const offlineControllers = controllers.filter(c => {
      if (!c.last_seen) return true;
      const diff = Date.now() - new Date(c.last_seen).getTime();
      return diff >= ONLINE_THRESHOLD_MS;
    });

    if (offlineControllers.length === 0) {
      alert('Keine offline Controller gefunden.');
      return;
    }

    if (!confirm(`${offlineControllers.length} offline Controller löschen?`)) return;

    try {
      await Promise.all(offlineControllers.map(c => ledApi.deleteController(c.id)));
      setMessage(`✓ ${offlineControllers.length} offline Controller gelöscht`);
      loadData();
    } catch (error: any) {
      const msg = error.response?.data?.error || error.message || 'Unbekannter Fehler';
      alert('Fehler: ' + msg);
    }
  };

  const controllerStatus = (controller: LEDController) => {
    if (!controller.last_seen) return { label: 'Offline', className: 'text-gray-400' };
    const last = new Date(controller.last_seen).getTime();
    const diff = Date.now() - last;
    if (diff < ONLINE_THRESHOLD_MS) {
      return { label: 'Online', className: 'text-green-400' };
    }
    return { label: 'Offline', className: 'text-gray-400' };
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-white flex items-center gap-2">
            <Cpu className="w-5 h-5 text-accent-red" /> Mikrocontroller verwalten
          </h2>
          <p className="text-gray-400 text-sm">Verwalte verbaute Controller, deren Topics und zugehörige Lagerarten.</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={handleDeleteOffline}
            className="px-3 py-2 bg-red-600/20 text-red-400 rounded-lg flex items-center gap-2 text-sm hover:bg-red-600/30"
            title="Alle offline Controller löschen"
          >
            <Trash2 className="w-4 h-4" /> Offline löschen
          </button>
          <button
            onClick={loadData}
            className="px-3 py-2 bg-white/10 text-white rounded-lg flex items-center gap-2 text-sm hover:bg-white/20"
          >
            <RefreshCcw className="w-4 h-4" /> Neu laden
          </button>
          <button
            onClick={startNew}
            className="px-4 py-2 bg-accent-red text-white rounded-lg flex items-center gap-2 text-sm hover:bg-accent-red/80"
          >
            <Plus className="w-4 h-4" /> Neuer Controller
          </button>
        </div>
      </div>

      {message && (
        <div className="glass rounded-xl px-4 py-2 text-sm text-green-400">{message}</div>
      )}

      {configureTarget !== null && (
        <div className="glass rounded-xl p-6 border border-blue-500 space-y-6">
          <div>
            <h3 className="text-white font-semibold mb-2 flex items-center gap-2">
              <Settings className="w-5 h-5 text-blue-400" />
              ESP32 Hardware konfigurieren
            </h3>
            <p className="text-sm text-gray-400 mb-4">
              Konfiguriere die Hardware-Einstellungen für diesen Controller.
              LED-Anzahl wird sofort übernommen, Pin und Chipset erfordern einen Neustart des ESP32.
            </p>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-semibold text-gray-300 mb-2">LED-Anzahl</label>
                <input
                  type="number"
                  min="1"
                  max="1200"
                  className="w-full px-3 py-2 rounded-lg glass text-white"
                  value={configureLedCount}
                  onChange={(e) => setConfigureLedCount(parseInt(e.target.value) || 0)}
                />
                <p className="text-xs text-gray-500 mt-1">1-1200 LEDs (sofort aktiv)</p>
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-300 mb-2">Data Pin (GPIO)</label>
                <input
                  type="number"
                  min="0"
                  max="50"
                  className="w-full px-3 py-2 rounded-lg glass text-white"
                  value={configureDataPin}
                  onChange={(e) => setConfigureDataPin(parseInt(e.target.value) || 0)}
                />
                <p className="text-xs text-gray-500 mt-1">GPIO 0-50 (Neustart nötig)</p>
              </div>
              <div>
                <label className="block text-sm font-semibold text-gray-300 mb-2">LED-Chipset</label>
                <select
                  className="w-full px-3 py-2 rounded-lg glass text-white"
                  value={configureChipset}
                  onChange={(e) => setConfigureChipset(e.target.value)}
                >
                  <option value="SK6812_GRBW">SK6812 GRBW</option>
                  <option value="SK6812_GRB">SK6812 GRB</option>
                  <option value="WS2812B">WS2812B</option>
                  <option value="WS2811">WS2811</option>
                  <option value="APA102">APA102</option>
                </select>
                <p className="text-xs text-gray-500 mt-1">LED-Typ (Neustart nötig)</p>
              </div>
            </div>
          </div>
          <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-3">
            <p className="text-yellow-300 text-xs">
              <strong>Hinweis:</strong> Änderungen an Data Pin und Chipset werden gespeichert,
              erfordern aber einen Neustart des ESP32. Nach dem Speichern kannst du den ESP direkt über den Button neu starten.
            </p>
          </div>
          <div className="flex gap-2 justify-between">
            <button
              onClick={() => configureTarget && handleRestart(configureTarget)}
              disabled={restarting !== null || configuring}
              className="px-4 py-2 rounded-lg text-sm font-semibold bg-orange-600 text-white flex items-center gap-2 disabled:opacity-50"
              title="ESP32 neu starten (für Pin/Chipset-Änderungen)"
            >
              <RotateCw className="w-4 h-4" /> {restarting === configureTarget ? 'Startet neu...' : 'ESP neu starten'}
            </button>
            <div className="flex gap-2">
              <button
                onClick={() => setConfigureTarget(null)}
                className="px-4 py-2 rounded-lg text-sm font-semibold bg-gray-600 text-white flex items-center gap-2"
              >
                <X className="w-4 h-4" /> Abbrechen
              </button>
              <button
                onClick={handleConfigure}
                disabled={configuring}
                className="px-4 py-2 rounded-lg text-sm font-semibold bg-blue-600 text-white flex items-center gap-2 disabled:opacity-50"
              >
                <Save className="w-4 h-4" /> {configuring ? 'Wird gesendet...' : 'Konfiguration senden'}
              </button>
            </div>
          </div>
        </div>
      )}

      {editor && (
        <div className="glass rounded-xl p-4 border border-accent-red space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {editor === 'new' && (
              <input
                type="text"
                className="px-3 py-2 rounded-lg glass text-white"
                placeholder="Controller ID (z.B. esp-regal-1)"
                value={form.controller_id}
                onChange={(e) => setForm((prev) => ({ ...prev, controller_id: e.target.value }))}
              />
            )}
            <input
              type="text"
              className="px-3 py-2 rounded-lg glass text-white"
              placeholder="Anzeigename"
              value={form.display_name}
              onChange={(e) => setForm((prev) => ({ ...prev, display_name: e.target.value }))}
            />
            <input
              type="text"
              className="px-3 py-2 rounded-lg glass text-white"
              placeholder="Topic-Suffix (optional)"
              value={form.topic_suffix}
              onChange={(e) => setForm((prev) => ({ ...prev, topic_suffix: e.target.value }))}
            />
            <label className="flex items-center gap-2 text-sm text-gray-300">
              <input
                type="checkbox"
                className="w-4 h-4"
                checked={form.is_active}
                onChange={(e) => setForm((prev) => ({ ...prev, is_active: e.target.checked }))}
              />
              Aktiv
            </label>
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-semibold text-gray-300">Zuständige Lagerzonen</label>
            {selectedZones.length > 0 ? (
              <div className="flex flex-wrap gap-2 text-xs">
                {selectedZones.map((zone) => (
                  <span key={zone.id} className="bg-accent-red/20 text-accent-red px-2 py-1 rounded-full">
                    {zone.label}
                  </span>
                ))}
              </div>
            ) : (
              <p className="text-xs text-gray-500">Noch keine Zonen zugewiesen.</p>
            )}
            <select
              multiple
              value={form.zoneTypeIds.map(String)}
              onChange={(e) => {
                const selected = Array.from(e.target.selectedOptions).map((opt) => Number(opt.value));
                handleZoneSelectChange(selected);
              }}
              className="w-full rounded-lg glass text-white px-3 py-2 bg-white/5 focus:outline-none focus:ring-2 focus:ring-accent-red"
            >
              {sortedZoneTypes.map((zone) => (
                <option key={zone.id} value={zone.id}>
                  {zone.label}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500">
              Tipp: Halte <span className="font-semibold">Strg / Cmd</span>, um mehrere Zonen auszuwählen.
            </p>
          </div>

          <div>
            <label className="block text-sm font-semibold text-gray-300 mb-2">Metadata (JSON)</label>
            <textarea
              className="w-full h-32 rounded-lg glass text-white font-mono text-sm px-3 py-2"
              value={form.metadata}
              onChange={(e) => setForm((prev) => ({ ...prev, metadata: e.target.value }))}
            />
          </div>

          <div className="flex gap-2 justify-end">
            <button
              onClick={resetEditor}
              className="px-4 py-2 rounded-lg text-sm font-semibold bg-gray-600 text-white flex items-center gap-2"
            >
              <X className="w-4 h-4" /> Abbrechen
            </button>
            <button
              onClick={handleSave}
              disabled={saving}
              className="px-4 py-2 rounded-lg text-sm font-semibold bg-green-600 text-white flex items-center gap-2 disabled:opacity-50"
            >
              <Save className="w-4 h-4" /> {saving ? 'Speichert...' : 'Speichern'}
            </button>
          </div>
        </div>
      )}

      <div className="space-y-2">
        {loading ? (
          <div className="glass rounded-xl p-5 text-center text-gray-400">Lade Controller...</div>
        ) : controllers.length === 0 ? (
          <div className="glass rounded-xl p-5 text-center text-gray-400">Noch keine Controller registriert.</div>
        ) : (
          controllers.map((controller) => {
            const status = controllerStatus(controller);
            const statusData = controller.status_data as Record<string, unknown> | undefined;
            const wifiRSSI =
              typeof statusData?.['wifi_rssi'] === 'number' ? (statusData['wifi_rssi'] as number) : undefined;
            const uptimeSeconds =
              typeof statusData?.['uptime_seconds'] === 'number' ? (statusData['uptime_seconds'] as number) : undefined;
            const ledCount =
              typeof statusData?.['led_count'] === 'number' ? (statusData['led_count'] as number) : undefined;
            const heartbeatAtRaw =
              typeof statusData?.['heartbeat_received_at'] === 'string'
                ? (statusData['heartbeat_received_at'] as string)
                : undefined;
            const heartbeatAt = heartbeatAtRaw ? new Date(heartbeatAtRaw) : undefined;
            const uptimeLabel = formatUptime(uptimeSeconds);
            return (
              <div key={controller.id} className="glass rounded-xl p-4 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 border border-white/10">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3">
                    <div className={`w-2 h-2 rounded-full ${status.className}`}></div>
                    <div>
                      <h3 className="text-white font-semibold text-sm sm:text-base flex items-center gap-2">
                        {controller.display_name}
                        {!controller.is_active && <span className="text-xs text-gray-500">deaktiviert</span>}
                      </h3>
                      <p className="text-xs text-gray-400">
                        ID: <span className="font-mono">{controller.controller_id}</span> • Topic: <span className="font-mono">{controller.topic_suffix}</span>
                      </p>
                      <p className={`text-xs ${status.className}`}>{status.label}</p>
                      {controller.zone_types && controller.zone_types.length > 0 && (
                        <p className="text-xs text-gray-400 mt-1">
                          Lagerarten: {controller.zone_types.map((zt) => zt.label).join(', ')}
                        </p>
                      )}
                      {controller.last_seen && (
                        <p className="text-xs text-gray-500">Letzter Kontakt: {new Date(controller.last_seen).toLocaleString()}</p>
                      )}
                      <div className="mt-2 grid grid-cols-1 sm:grid-cols-2 gap-y-1 text-xs text-gray-400">
                        {controller.hostname && <span>Hostname: <span className="font-mono text-gray-300">{controller.hostname}</span></span>}
                        {controller.ip_address && <span>IP: <span className="font-mono text-gray-300">{controller.ip_address}</span></span>}
                        {controller.mac_address && <span>MAC: <span className="font-mono text-gray-300">{controller.mac_address}</span></span>}
                        {controller.firmware_version && <span>Firmware: <span className="font-mono text-gray-300">{controller.firmware_version}</span></span>}
                        {typeof ledCount === 'number' && <span>LEDs: <span className="font-mono text-gray-300">{ledCount}</span></span>}
                        {wifiRSSI !== undefined && <span>WiFi RSSI: <span className="font-mono text-gray-300">{wifiRSSI} dBm</span></span>}
                        {uptimeLabel && <span>Uptime: <span className="font-mono text-gray-300">{uptimeLabel}</span></span>}
                        {heartbeatAt && <span>Heartbeat: <span className="font-mono text-gray-300">{heartbeatAt.toLocaleTimeString()}</span></span>}
                      </div>
                    </div>
                  </div>
                </div>
                <div className="flex gap-2">
                  {status.label === 'Online' && (
                    <>
                      <button
                        onClick={() => startConfigure(controller)}
                        className="px-3 py-2 rounded-lg text-sm font-semibold bg-blue-600/20 text-blue-400 hover:bg-blue-600/30 flex items-center gap-2"
                        title="Hardware konfigurieren"
                      >
                        <Settings className="w-4 h-4" /> Konfigurieren
                      </button>
                      <button
                        onClick={() => handleRestart(controller.id)}
                        disabled={restarting === controller.id}
                        className="px-3 py-2 rounded-lg text-sm font-semibold bg-orange-600/20 text-orange-400 hover:bg-orange-600/30 flex items-center gap-2 disabled:opacity-50"
                        title="ESP32 neu starten"
                      >
                        <RotateCw className="w-4 h-4" /> {restarting === controller.id ? 'Startet...' : 'Neustart'}
                      </button>
                    </>
                  )}
                  <button
                    onClick={() => startEdit(controller)}
                    className="px-3 py-2 rounded-lg text-sm font-semibold bg-white/10 text-white hover:bg-white/20"
                  >
                    Bearbeiten
                  </button>
                  <button
                    onClick={() => handleDelete(controller.id)}
                    className="p-2 rounded-lg text-red-400 hover:bg-white/10"
                    title="Controller löschen"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
