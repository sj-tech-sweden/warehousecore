import { useEffect, useMemo, useState } from 'react';
import { ledApi, api, type LEDController, type LEDControllerPayload, type ZoneTypeDefinition } from '../../lib/api';
import { Plus, Save, X, RefreshCcw, Trash2, Cpu } from 'lucide-react';

type EditorTarget = number | 'new' | null;

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

export function LEDControllersTab() {
  const [controllers, setControllers] = useState<LEDController[]>([]);
  const [zoneTypes, setZoneTypes] = useState<ZoneTypeDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editor, setEditor] = useState<EditorTarget>(null);
  const [form, setForm] = useState<ControllerForm>(defaultForm);
  const [message, setMessage] = useState<string>('');

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

  const handleCheckboxToggle = (id: number) => {
    setForm((prev) => {
      const exists = prev.zoneTypeIds.includes(id);
      return {
        ...prev,
        zoneTypeIds: exists ? prev.zoneTypeIds.filter((zid) => zid !== id) : [...prev.zoneTypeIds, id],
      };
    });
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

  const controllerStatus = (controller: LEDController) => {
    if (!controller.last_seen) return { label: 'Offline', className: 'text-gray-400' };
    const last = new Date(controller.last_seen).getTime();
    const diff = Date.now() - last;
    if (diff < ONLINE_THRESHOLD_MS) {
      return { label: 'Online', className: 'text-green-400' };
    }
    return { label: 'Offline', className: 'text-gray-400' };
  };

  const sortedZoneTypes = useMemo(() => zoneTypes.slice().sort((a, b) => a.label.localeCompare(b.label, 'de')), [zoneTypes]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-white flex items-center gap-2">
            <Cpu className="w-5 h-5 text-accent-red" /> Mikrocontroller verwalten
          </h2>
          <p className="text-gray-400 text-sm">Verwalte verbaute Controller, deren Topics und zugehörige Zonenarten.</p>
        </div>
        <div className="flex gap-2">
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

          <div>
            <label className="block text-sm font-semibold text-gray-300 mb-2">Zugeordnete Zonentypen</label>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2">
              {sortedZoneTypes.map((zone) => (
                <label key={zone.id} className="flex items-center gap-2 text-sm text-gray-200 bg-white/5 rounded-lg px-3 py-2">
                  <input
                    type="checkbox"
                    className="w-4 h-4"
                    checked={form.zoneTypeIds.includes(zone.id)}
                    onChange={() => handleCheckboxToggle(zone.id)}
                  />
                  <span>{zone.label}</span>
                </label>
              ))}
            </div>
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
                          Zonenarten: {controller.zone_types.map((zt) => zt.label).join(', ')}
                        </p>
                      )}
                      {controller.last_seen && (
                        <p className="text-xs text-gray-500">Letzter Kontakt: {new Date(controller.last_seen).toLocaleString()}</p>
                      )}
                    </div>
                  </div>
                </div>
                <div className="flex gap-2">
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
