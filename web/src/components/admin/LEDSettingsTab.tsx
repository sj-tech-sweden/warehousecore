import { useState, useEffect } from 'react';
import { Save, Lightbulb, RefreshCcw, SlidersHorizontal, FileText } from 'lucide-react';
import { api, ledApi, type LEDAppearance, type LEDJobHighlightSettings } from '../../lib/api';

interface LEDDefault {
  color: string;
  pattern: string;
  intensity: number;
}

interface ZoneTypeLED {
  id: number;
  key: string;
  label: string;
  description?: string;
  default_led_pattern: string;
  default_led_color: string;
  default_intensity: number;
}

const defaultJobSettings: LEDJobHighlightSettings = {
  mode: 'all_bins',
  required: {
    color: '#00FF00',
    pattern: 'solid',
    intensity: 220,
    speed: 1200,
  },
  non_required: {
    color: '#FF0000',
    pattern: 'solid',
    intensity: 160,
    speed: 1200,
  },
};

export function LEDSettingsTab() {
  const [defaults, setDefaults] = useState<LEDDefault>({
    color: '#FF7A00',
    pattern: 'breathe',
    intensity: 180,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  const [jobSettings, setJobSettings] = useState<LEDJobHighlightSettings>(defaultJobSettings);
  const [jobSaving, setJobSaving] = useState(false);
  const [jobMessage, setJobMessage] = useState('');
  const [zoneTypes, setZoneTypes] = useState<ZoneTypeLED[]>([]);
  const [zoneTypeLoading, setZoneTypeLoading] = useState(true);
  const [zoneTypeSaving, setZoneTypeSaving] = useState<number | null>(null);
  const [zoneTypeMessages, setZoneTypeMessages] = useState<Record<number, string>>({});
  const [mappingJSON, setMappingJSON] = useState('');
  const [mappingLoading, setMappingLoading] = useState(true);
  const [mappingSaving, setMappingSaving] = useState(false);
  const [mappingValidating, setMappingValidating] = useState(false);
  const [mappingMessage, setMappingMessage] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewMessage, setPreviewMessage] = useState('');

  useEffect(() => {
    loadDefaults();
    loadJobSettings();
    loadZoneTypes();
    loadMapping();
  }, []);

  const loadDefaults = async () => {
    try {
      const response = await api.get('/admin/led/single-bin-default');
      setDefaults(response.data);
    } catch (error) {
      console.error('Failed to load LED defaults:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadJobSettings = async () => {
    try {
      const { data } = await ledApi.getJobSettings();
      setJobSettings(data);
    } catch (error) {
      console.error('Failed to load job highlight settings:', error);
    }
  };

  const loadZoneTypes = async () => {
    try {
      const response = await api.get('/admin/zone-types');
      setZoneTypes(response.data);
    } catch (error) {
      console.error('Failed to load zone type LED defaults:', error);
    } finally {
      setZoneTypeLoading(false);
    }
  };

  const loadMapping = async () => {
    setMappingLoading(true);
    try {
      const { data } = await ledApi.getMapping();
      setMappingJSON(JSON.stringify(data, null, 2));
      setMappingMessage('');
    } catch (error) {
      console.error('Failed to load LED mapping:', error);
      setMappingMessage('Fehler beim Laden des LED-Mappings.');
    } finally {
      setMappingLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setMessage('');

    try {
      await api.put('/admin/led/single-bin-default', defaults);
      setMessage('✓ Einstellungen gespeichert');
      setTimeout(() => setMessage(''), 3000);
    } catch (error: any) {
      setMessage('Fehler: ' + (error.response?.data?.error || error.message));
    } finally {
      setSaving(false);
    }
  };

  const updateJobAppearance = (section: 'required' | 'non_required', patch: Partial<LEDAppearance>) => {
    setJobSettings((prev) => ({
      ...prev,
      [section]: { ...prev[section], ...patch },
    }));
  };

  const handleJobSettingsSave = async () => {
    setJobSaving(true);
    setJobMessage('');

    try {
      await ledApi.updateJobSettings(jobSettings);
      setJobMessage('✓ Job-Highlight gespeichert');
      setTimeout(() => setJobMessage(''), 3000);
    } catch (error: any) {
      setJobMessage('Fehler: ' + (error.response?.data?.error || error.message));
    } finally {
      setJobSaving(false);
    }
  };

  const handleMappingValidate = async () => {
    setMappingValidating(true);
    setMappingMessage('');
    try {
      const parsed = JSON.parse(mappingJSON);
      const { data } = await ledApi.validateMapping(parsed);
      if (data.valid) {
        setMappingMessage(`✓ Mapping gültig (${data.total_bins ?? 0} Bins, ${data.total_shelves ?? 0} Regale)`);
      } else {
        const errors = Array.isArray(data.errors) ? data.errors.join(', ') : 'Unbekannter Fehler';
        setMappingMessage('⚠️ ' + errors);
      }
    } catch (error: any) {
      setMappingMessage('Fehler: ' + (error.response?.data?.error || error.message || error.toString()));
    } finally {
      setMappingValidating(false);
    }
  };

  const handleMappingSave = async () => {
    setMappingSaving(true);
    setMappingMessage('');
    try {
      const parsed = JSON.parse(mappingJSON);
      await ledApi.updateMapping(parsed);
      setMappingMessage('✓ Mapping gespeichert');
      setTimeout(() => setMappingMessage(''), 4000);
    } catch (error: any) {
      setMappingMessage('Fehler: ' + (error.response?.data?.error || error.message || error.toString()));
    } finally {
      setMappingSaving(false);
    }
  };

  const toPreviewAppearance = (color: string, pattern: string, intensity: number, speed?: number): LEDAppearance => ({
    color,
    pattern,
    intensity: Math.max(0, Math.min(255, intensity)),
    speed: speed && speed > 0 ? speed : 1200,
  });

  const triggerPreview = async (appearances: LEDAppearance[], clearBefore: boolean = true) => {
    if (appearances.length === 0) {
      return;
    }
    setPreviewLoading(true);
    setPreviewMessage('');
    try {
      await ledApi.preview(appearances, clearBefore);
      setPreviewMessage('✓ Vorschau gestartet – zum Stoppen ggf. „LEDs löschen“ verwenden.');
      setTimeout(() => setPreviewMessage(''), 4000);
    } catch (error: any) {
      setPreviewMessage('Fehler bei der Vorschau: ' + (error.response?.data?.error || error.message));
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleZoneTypeFieldChange = (id: number, field: keyof ZoneTypeLED, value: string | number) => {
    setZoneTypes((prev) =>
      prev.map((zoneType) =>
        zoneType.id === id
          ? { ...zoneType, [field]: value }
          : zoneType
      )
    );
  };

  const setZoneTypeFeedback = (id: number, text: string) => {
    setZoneTypeMessages((prev) => ({ ...prev, [id]: text }));
    if (text) {
      setTimeout(() => {
        setZoneTypeMessages((prev) => {
          const next = { ...prev };
          delete next[id];
          return next;
        });
      }, 3000);
    }
  };

  const handleZoneTypeSave = async (zoneType: ZoneTypeLED) => {
    setZoneTypeSaving(zoneType.id);
    setZoneTypeFeedback(zoneType.id, '');

    try {
      await api.put(`/admin/zone-types/${zoneType.id}`, {
        default_led_pattern: zoneType.default_led_pattern,
        default_led_color: zoneType.default_led_color,
        default_intensity: zoneType.default_intensity,
      });
      setZoneTypeFeedback(zoneType.id, '✓ Zonentyp gespeichert');
      loadZoneTypes();
    } catch (error: any) {
      setZoneTypeFeedback(
        zoneType.id,
        'Fehler: ' + (error.response?.data?.error || error.message)
      );
    } finally {
      setZoneTypeSaving(null);
    }
  };

  const applyGlobalDefaultsToZoneType = (zoneTypeId: number) => {
    setZoneTypes((prev) =>
      prev.map((zoneType) =>
        zoneType.id === zoneTypeId
          ? {
              ...zoneType,
              default_led_color: defaults.color,
              default_led_pattern: defaults.pattern,
              default_intensity: defaults.intensity,
            }
          : zoneType
      )
    );
  };

  if (loading || zoneTypeLoading) return <div className="text-white">Lädt...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Lightbulb className="w-6 h-6 text-yellow-400" />
        <div>
          <h2 className="text-xl font-bold text-white">LED-Standardverhalten (Einzelfach-Highlight)</h2>
          <p className="text-gray-400 text-sm">Diese Einstellungen gelten für die "Fach beleuchten" Funktion</p>
        </div>
      </div>

      {previewMessage && (
        <div
          className={`px-4 py-3 rounded-lg text-sm font-semibold ${
            previewMessage.startsWith('✓') ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
          }`}
        >
          {previewMessage}
        </div>
      )}

      <div className="glass rounded-xl p-6 space-y-6">
        {/* Pattern Selection */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-3">Muster</label>
          <div className="grid grid-cols-3 gap-3">
            {['solid', 'breathe', 'blink'].map(pattern => (
              <button
                key={pattern}
                onClick={() => setDefaults({ ...defaults, pattern })}
                className={`px-4 py-3 rounded-xl font-semibold transition-all ${
                  defaults.pattern === pattern
                    ? 'bg-accent-red text-white shadow-lg'
                    : 'glass text-gray-400 hover:bg-white/10'
                }`}
              >
                {pattern === 'solid' && 'Durchgehend'}
                {pattern === 'breathe' && 'Atmen'}
                {pattern === 'blink' && 'Blinken'}
              </button>
            ))}
          </div>
        </div>

        {/* Color Picker */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-3">Farbe</label>
          <div className="flex items-center gap-4">
            <input
              type="color"
              value={defaults.color}
              onChange={(e) => setDefaults({ ...defaults, color: e.target.value })}
              className="w-20 h-20 rounded-xl cursor-pointer"
            />
            <div className="flex-1">
              <input
                type="text"
                value={defaults.color}
                onChange={(e) => setDefaults({ ...defaults, color: e.target.value })}
                className="w-full px-4 py-3 rounded-xl glass text-white font-mono"
                placeholder="#FF4500"
              />
              <p className="text-gray-500 text-xs mt-1">Hex-Format (z.B. #FF4500 für Orange)</p>
            </div>
          </div>
        </div>

        {/* Intensity Slider */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-3">
            Intensität: {defaults.intensity} / 255
          </label>
          <input
            type="range"
            min="0"
            max="255"
            value={defaults.intensity}
            onChange={(e) => setDefaults({ ...defaults, intensity: parseInt(e.target.value) })}
            className="w-full h-3 rounded-lg bg-white/10 appearance-none cursor-pointer"
            style={{
              background: `linear-gradient(to right, ${defaults.color} 0%, ${defaults.color} ${(defaults.intensity / 255) * 100}%, rgba(255,255,255,0.1) ${(defaults.intensity / 255) * 100}%, rgba(255,255,255,0.1) 100%)`
            }}
          />
        </div>

        {/* Preview */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-3">Vorschau</label>
          <div className="glass rounded-xl p-8 flex items-center justify-center">
            <div
              className="w-32 h-32 rounded-2xl transition-all duration-1000"
              style={{
                backgroundColor: defaults.color,
                opacity: defaults.intensity / 255,
                animation: defaults.pattern === 'breathe' ? 'breathe 2s infinite' : defaults.pattern === 'blink' ? 'blink 1s infinite' : 'none'
              }}
            ></div>
          </div>
        </div>

        {/* Save Button */}
        <div className="pt-4 border-t border-white/10">
          <div className="flex flex-col sm:flex-row gap-3">
            <button
              onClick={handleSave}
              disabled={saving}
              className={`flex-1 py-3 px-6 rounded-xl font-semibold text-white transition-all flex items-center justify-center gap-2 ${
                saving
                  ? 'bg-gray-600 cursor-not-allowed'
                  : 'bg-gradient-to-r from-accent-red to-red-700 hover:shadow-lg hover:shadow-red-500/50 hover:scale-105 active:scale-95'
              }`}
            >
              <Save className="w-5 h-5" />
              <span>{saving ? 'Speichert...' : 'Einstellungen speichern'}</span>
            </button>
            <button
              onClick={() =>
                triggerPreview(
                  [
                    toPreviewAppearance(
                      defaults.color,
                      defaults.pattern,
                      defaults.intensity,
                      defaults.pattern === 'solid' ? 1200 : 1200
                    ),
                  ],
                  true
                )
              }
              disabled={previewLoading}
              className={`flex-1 py-3 px-6 rounded-xl font-semibold transition-all flex items-center justify-center gap-2 ${
                previewLoading
                  ? 'bg-gray-600 text-gray-300 cursor-not-allowed'
                  : 'bg-white/10 text-white hover:bg-white/20'
              }`}
            >
              <Lightbulb className="w-5 h-5 text-yellow-300" />
              <span>{previewLoading ? 'Vorschau läuft…' : 'LED Vorschau'}</span>
            </button>
          </div>

          {message && (
            <div className={`mt-3 p-3 rounded-lg text-center text-sm font-semibold ${
              message.includes('✓')
                ? 'bg-green-500/20 text-green-400'
                : 'bg-red-500/20 text-red-400'
            }`}>
              {message}
            </div>
          )}
        </div>
      </div>

      {/* Inline CSS for animations */}
      <style>{`
        @keyframes breathe {
          0%, 100% { opacity: ${defaults.intensity / 255}; }
          50% { opacity: ${(defaults.intensity / 255) * 0.3}; }
        }
        @keyframes blink {
          0%, 49% { opacity: ${defaults.intensity / 255}; }
          50%, 100% { opacity: 0; }
      }
      `}</style>

      {/* Job highlight behaviour */}
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <SlidersHorizontal className="w-5 h-5 text-accent-red" />
          <h3 className="text-lg font-semibold text-white">Job-Highlight Einstellungen</h3>
        </div>
        <p className="text-sm text-gray-400">
          Steuere, wie die Regalfächer leuchten, wenn ein Job ausgewählt ist. Du kannst zwischen
          einer kompletten Visualisierung aller Fächer und einem Fokus auf die zu packenden Fächer wählen.
        </p>

        <div className="glass rounded-xl p-5 space-y-5">
          <div className="flex flex-wrap gap-3">
            <button
              onClick={() => setJobSettings((prev) => ({ ...prev, mode: 'all_bins' }))}
              className={`px-4 py-2 rounded-lg font-semibold transition-all ${
                jobSettings.mode === 'all_bins'
                  ? 'bg-accent-red text-white shadow-lg shadow-accent-red/30'
                  : 'glass text-gray-300 hover:text-white'
              }`}
            >
              Alle Fächer markieren
            </button>
            <button
              onClick={() => setJobSettings((prev) => ({ ...prev, mode: 'required_only' }))}
              className={`px-4 py-2 rounded-lg font-semibold transition-all ${
                jobSettings.mode === 'required_only'
                  ? 'bg-accent-red text-white shadow-lg shadow-accent-red/30'
                  : 'glass text-gray-300 hover:text-white'
              }`}
            >
              Nur benötigte Fächer
            </button>
          </div>
          <p className="text-xs text-gray-500">
            {jobSettings.mode === 'all_bins'
              ? 'Alle Fächer leuchten: benötigte Fächer heben sich durch andere Farben/Pattern hervor.'
              : 'Nur Fächer mit noch zu packenden Geräten leuchten – alle anderen werden ausgeschaltet.'}
          </p>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {[{
              key: 'required' as const,
              title: 'Fächer mit fehlenden Geräten',
              description: 'Wird verwendet, wenn in einem Fach noch Geräte gepackt werden müssen.',
            }, {
              key: 'non_required' as const,
              title: 'Fächer ohne Bedarf',
              description: 'Fächer, die für den aktuellen Job nicht benötigt werden.',
            }].map((cfg) => {
              const appearance = jobSettings[cfg.key];
              return (
                <div key={cfg.key} className="glass-dark rounded-xl p-5 space-y-4 border border-white/5">
                  <div>
                    <h4 className="text-white font-semibold">{cfg.title}</h4>
                    <p className="text-xs text-gray-400 mt-1">{cfg.description}</p>
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-gray-400 mb-2">Farbe</label>
                    <div className="flex items-center gap-3">
                      <input
                        type="color"
                        value={appearance.color}
                        onChange={(e) => updateJobAppearance(cfg.key, { color: e.target.value })}
                        className="w-14 h-14 rounded-lg cursor-pointer"
                      />
                      <input
                        type="text"
                        value={appearance.color}
                        onChange={(e) => updateJobAppearance(cfg.key, { color: e.target.value })}
                        className="flex-1 px-3 py-2 rounded-lg glass text-white font-mono"
                        placeholder="#00FF00"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-gray-400 mb-2">Muster</label>
                    <select
                      value={appearance.pattern}
                      onChange={(e) => updateJobAppearance(cfg.key, { pattern: e.target.value as LEDAppearance['pattern'] })}
                      className="w-full px-3 py-2 rounded-lg glass text-white"
                    >
                      <option value="solid">Durchgehend</option>
                      <option value="breathe">Atmen</option>
                      <option value="blink">Blinken</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-gray-400 mb-2">
                      Intensität: {appearance.intensity} / 255
                    </label>
                    <input
                      type="range"
                      min={0}
                      max={255}
                      value={appearance.intensity}
                      onChange={(e) => updateJobAppearance(cfg.key, { intensity: parseInt(e.target.value, 10) })}
                      className="w-full h-3 rounded-lg bg-white/10 appearance-none cursor-pointer"
                      style={{
                        background: `linear-gradient(to right, ${appearance.color} 0%, ${appearance.color} ${(appearance.intensity / 255) * 100}%, rgba(255,255,255,0.1) ${(appearance.intensity / 255) * 100}%, rgba(255,255,255,0.1) 100%)`,
                      }}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-semibold text-gray-400 mb-2">
                      Geschwindigkeit{appearance.pattern === 'solid' ? '' : `: ${appearance.speed} ms`}
                    </label>
                    <input
                      type="range"
                      min={200}
                      max={3000}
                      step={100}
                      value={appearance.pattern === 'solid' ? 1200 : appearance.speed}
                      disabled={appearance.pattern === 'solid'}
                      onChange={(e) => updateJobAppearance(cfg.key, { speed: parseInt(e.target.value, 10) })}
                      className="w-full h-3 rounded-lg bg-white/10 appearance-none cursor-pointer disabled:opacity-40"
                    />
                    <p className="text-xs text-gray-500 mt-1">
                      {appearance.pattern === 'solid'
                        ? 'Keine Animation — keine Geschwindigkeit erforderlich.'
                        : 'Niedrige Werte = schnelleres Atmen/Blinken.'}
                    </p>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="pt-4 border-t border-white/10 space-y-3">
            <div className="flex flex-col sm:flex-row gap-3">
              <button
                onClick={handleJobSettingsSave}
                disabled={jobSaving}
                className={`flex-1 px-4 py-2 rounded-lg font-semibold text-white flex items-center justify-center gap-2 ${
                  jobSaving ? 'bg-gray-600 cursor-not-allowed' : 'bg-accent-red hover:bg-red-600 transition-colors'
                }`}
              >
                <Save className="w-4 h-4" />
                <span>{jobSaving ? 'Speichert...' : 'Job-Highlight speichern'}</span>
              </button>
              <button
                onClick={() =>
                  triggerPreview(
                    [
                      toPreviewAppearance(
                        jobSettings.required.color,
                        jobSettings.required.pattern,
                        jobSettings.required.intensity,
                        jobSettings.required.speed
                      ),
                      ...(jobSettings.mode === 'all_bins'
                        ? [
                            toPreviewAppearance(
                              jobSettings.non_required.color,
                              jobSettings.non_required.pattern,
                              jobSettings.non_required.intensity,
                              jobSettings.non_required.speed
                            ),
                          ]
                        : []),
                    ],
                    true
                  )
                }
                disabled={previewLoading}
                className={`flex-1 px-4 py-2 rounded-lg font-semibold flex items-center justify-center gap-2 ${
                  previewLoading
                    ? 'bg-gray-600 text-gray-300 cursor-not-allowed'
                    : 'bg-white/10 text-white hover:bg-white/20'
                }`}
              >
                <Lightbulb className="w-4 h-4 text-yellow-300" />
                <span>{previewLoading ? 'Vorschau läuft…' : 'Job-Highlight Vorschau'}</span>
              </button>
            </div>
            {jobMessage && (
              <div
                className={`px-3 py-2 rounded-lg text-sm font-semibold ${
                  jobMessage.startsWith('✓') ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'
                }`}
              >
                {jobMessage}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* LED mapping editor */}
      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <FileText className="w-5 h-5 text-accent-red" />
          <h3 className="text-lg font-semibold text-white">LED-Mapping (Expertenmodus)</h3>
        </div>
        <p className="text-sm text-gray-400">
          Bearbeite die vollständige Zuordnung von Regalfächern zu LED-Pixeln. Änderungen wirken sich direkt auf die Hardware aus – bitte mit Vorsicht nutzen.
        </p>

        <div className="glass rounded-xl p-5 space-y-4">
          <div className="flex justify-between items-center gap-3 flex-wrap">
            <span className="text-xs text-gray-500">JSON-Konfiguration aus <code>internal/led/config/led_mapping.json</code></span>
            <button
              onClick={loadMapping}
              disabled={mappingLoading}
              className="flex items-center gap-2 px-3 py-2 glass text-gray-300 hover:text-white rounded-lg transition-colors disabled:opacity-50"
            >
              <RefreshCcw className="w-4 h-4" /> Neu laden
            </button>
          </div>

          {mappingLoading ? (
            <div className="text-sm text-gray-400">Mapping wird geladen...</div>
          ) : (
            <textarea
              value={mappingJSON}
              onChange={(e) => setMappingJSON(e.target.value)}
              rows={16}
              className="w-full font-mono text-sm bg-black/40 text-gray-100 border border-white/10 rounded-lg p-4 focus:outline-none focus:border-accent-red"
            />
          )}

          <div className="flex flex-wrap gap-3">
            <button
              onClick={handleMappingValidate}
              disabled={mappingLoading || mappingValidating}
              className={`px-4 py-2 rounded-lg font-semibold flex items-center gap-2 ${
                mappingLoading || mappingValidating
                  ? 'bg-gray-600 cursor-not-allowed text-gray-200'
                  : 'glass text-gray-200 hover:text-white'
              }`}
            >
              {mappingValidating ? 'Validiere...' : 'Mapping prüfen'}
            </button>
            <button
              onClick={handleMappingSave}
              disabled={mappingLoading || mappingSaving}
              className={`px-4 py-2 rounded-lg font-semibold text-white flex items-center gap-2 ${
                mappingLoading || mappingSaving
                  ? 'bg-gray-600 cursor-not-allowed'
                  : 'bg-accent-red hover:bg-red-600 transition-colors'
              }`}
            >
              <Save className="w-4 h-4" />
              <span>{mappingSaving ? 'Speichert...' : 'Mapping speichern'}</span>
            </button>
          </div>

          {mappingMessage && (
            <div
              className={`px-3 py-2 rounded-lg text-sm font-semibold ${
                mappingMessage.startsWith('✓')
                  ? 'bg-green-500/20 text-green-400'
                  : mappingMessage.startsWith('⚠️')
                    ? 'bg-yellow-500/20 text-yellow-400'
                    : 'bg-red-500/20 text-red-400'
              }`}
            >
              {mappingMessage}
            </div>
          )}
        </div>
      </div>

      {/* Zone type specific defaults */}
      <div className="space-y-3">
        <h3 className="text-lg font-semibold text-white">LED-Standardwerte pro Zonentyp</h3>
        <p className="text-sm text-gray-400">
          Passe hier das LED-Verhalten für einzelne Zonentypen an. Diese Einstellungen überschreiben den globalen Standard.
        </p>

        <div className="space-y-4">
          {zoneTypes.map((zoneType) => (
            <div key={zoneType.id} className="glass rounded-xl p-5 space-y-4">
              <div className="flex flex-wrap justify-between gap-3">
                <div>
                  <h4 className="text-white font-semibold">{zoneType.label}</h4>
                  <p className="text-xs text-gray-500 font-mono">{zoneType.key}</p>
                  {zoneType.description && (
                    <p className="text-sm text-gray-400 mt-1">{zoneType.description}</p>
                  )}
                </div>
                <button
                  onClick={() => applyGlobalDefaultsToZoneType(zoneType.id)}
                  className="flex items-center gap-2 px-3 py-2 glass text-gray-300 hover:text-white rounded-lg transition-colors"
                  title="Globale LED-Standards übernehmen"
                >
                  <RefreshCcw className="w-4 h-4" />
                  <span>Global übernehmen</span>
                </button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-semibold text-gray-400 mb-2">Muster</label>
                  <select
                    value={zoneType.default_led_pattern}
                    onChange={(e) => handleZoneTypeFieldChange(zoneType.id, 'default_led_pattern', e.target.value)}
                    className="w-full px-3 py-2 rounded-lg glass text-white"
                  >
                    <option value="solid">Durchgehend</option>
                    <option value="breathe">Atmen</option>
                    <option value="blink">Blinken</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-semibold text-gray-400 mb-2">Farbe</label>
                  <div className="flex items-center gap-3">
                    <input
                      type="color"
                      value={zoneType.default_led_color || '#FF7A00'}
                      onChange={(e) => handleZoneTypeFieldChange(zoneType.id, 'default_led_color', e.target.value)}
                      className="w-14 h-14 rounded-lg cursor-pointer"
                    />
                    <input
                      type="text"
                      value={zoneType.default_led_color || ''}
                      onChange={(e) => handleZoneTypeFieldChange(zoneType.id, 'default_led_color', e.target.value)}
                      className="flex-1 px-3 py-2 rounded-lg glass text-white font-mono"
                      placeholder="#FF4500"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-semibold text-gray-400 mb-2">
                    Intensität: {zoneType.default_intensity} / 255
                  </label>
                  <input
                    type="range"
                    min="0"
                    max="255"
                    value={zoneType.default_intensity}
                    onChange={(e) =>
                      handleZoneTypeFieldChange(zoneType.id, 'default_intensity', parseInt(e.target.value, 10))
                    }
                    className="w-full h-3 rounded-lg bg-white/10 appearance-none cursor-pointer"
                    style={{
                      background: `linear-gradient(to right, ${zoneType.default_led_color || defaults.color} 0%, ${
                        zoneType.default_led_color || defaults.color
                      } ${(zoneType.default_intensity / 255) * 100}%, rgba(255,255,255,0.1) ${
                        (zoneType.default_intensity / 255) * 100
                      }%, rgba(255,255,255,0.1) 100%)`,
                    }}
                  />
                </div>
              </div>

              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                <div className="flex flex-col sm:flex-row gap-3 flex-1">
                  <button
                    onClick={() => handleZoneTypeSave(zoneType)}
                    disabled={zoneTypeSaving === zoneType.id}
                    className={`flex-1 px-4 py-2 rounded-lg font-semibold text-white flex items-center justify-center gap-2 ${
                      zoneTypeSaving === zoneType.id
                        ? 'bg-gray-600 cursor-not-allowed'
                        : 'bg-accent-red hover:bg-red-600 transition-colors'
                    }`}
                  >
                    <Save className="w-4 h-4" />
                    <span>{zoneTypeSaving === zoneType.id ? 'Speichert...' : 'Zonentyp speichern'}</span>
                  </button>
                  <button
                    onClick={() =>
                      triggerPreview(
                        [
                          toPreviewAppearance(
                            zoneType.default_led_color || defaults.color,
                            zoneType.default_led_pattern || 'solid',
                            zoneType.default_intensity ?? 180,
                            zoneType.default_led_pattern === 'solid' ? 1200 : 1200
                          ),
                        ],
                        true
                      )
                    }
                    disabled={previewLoading}
                    className={`flex-1 px-4 py-2 rounded-lg font-semibold flex items-center justify-center gap-2 ${
                      previewLoading
                        ? 'bg-gray-600 text-gray-300 cursor-not-allowed'
                        : 'bg-white/10 text-white hover:bg-white/20'
                    }`}
                  >
                    <Lightbulb className="w-4 h-4 text-yellow-300" />
                    <span>{previewLoading ? 'Vorschau läuft…' : 'LED Vorschau'}</span>
                  </button>
                </div>
                {zoneTypeMessages[zoneType.id] && (
                  <div
                    className={`px-3 py-2 rounded-lg text-sm font-semibold ${
                      zoneTypeMessages[zoneType.id].startsWith('✓')
                        ? 'bg-green-500/20 text-green-400'
                        : 'bg-red-500/20 text-red-400'
                    }`}
                  >
                    {zoneTypeMessages[zoneType.id]}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
