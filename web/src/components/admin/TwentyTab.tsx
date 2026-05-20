import { useEffect, useMemo, useState } from 'react';
import { CheckCircle2, AlertCircle, Save, RefreshCcw, Clock3, Link2, KeyRound } from 'lucide-react';
import { twentyApi, type TwentySettingsPayload } from '../../lib/api';
import { useTranslation } from 'react-i18next';

const SYNC_INTERVAL_OPTIONS = [
  { value: 0, labelKey: 'admin.twenty.syncIntervalDisabled' },
  { value: 15, labelKey: 'admin.twenty.syncInterval15m' },
  { value: 30, labelKey: 'admin.twenty.syncInterval30m' },
  { value: 60, labelKey: 'admin.twenty.syncInterval1h' },
  { value: 120, labelKey: 'admin.twenty.syncInterval2h' },
  { value: 240, labelKey: 'admin.twenty.syncInterval4h' },
  { value: 480, labelKey: 'admin.twenty.syncInterval8h' },
  { value: 1440, labelKey: 'admin.twenty.syncInterval24h' },
];

export function TwentyTab() {
  const { t } = useTranslation();

  const [baseUrl, setBaseUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [apiKeyConfigured, setApiKeyConfigured] = useState(false);
  const [apiKeyMasked, setApiKeyMasked] = useState('');
  const [baseUrlLocked, setBaseUrlLocked] = useState(false);
  const [apiKeyLocked, setApiKeyLocked] = useState(false);
  const [clearApiKey, setClearApiKey] = useState(false);
  const [syncInterval, setSyncInterval] = useState(0);
  const [enableJobBootstrap, setEnableJobBootstrap] = useState(true);

  const [saved, setSaved] = useState<{
    baseUrl: string;
    syncInterval: number;
    enableJobBootstrap: boolean;
  } | null>(null);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const [message, setMessage] = useState<{ type: 'success' | 'error' | 'info'; text: string } | null>(null);

  const hasUnsavedChanges = useMemo(() => {
    if (!saved) return false;
    return (
      baseUrl !== saved.baseUrl ||
      syncInterval !== saved.syncInterval ||
      enableJobBootstrap !== saved.enableJobBootstrap ||
      apiKey !== '' ||
      clearApiKey
    );
  }, [saved, baseUrl, syncInterval, enableJobBootstrap, apiKey, clearApiKey]);

  const loadSettings = async () => {
    setLoading(true);
    setMessage(null);
    try {
      const resp = await twentyApi.getSettings();
      const d = resp.data;
      setBaseUrl(d.base_url || '');
      setApiKey('');
      setApiKeyConfigured(!!d.api_key_configured);
      setApiKeyMasked(d.api_key_masked || '');
      setBaseUrlLocked(Boolean(d.base_url_locked));
      setApiKeyLocked(Boolean(d.api_key_locked));
      setClearApiKey(false);
      setSyncInterval(Number(d.sync_interval_minutes || 0));
      setEnableJobBootstrap(Boolean(d.enable_job_bootstrap));
      setSaved({
        baseUrl: d.base_url || '',
        syncInterval: Number(d.sync_interval_minutes || 0),
        enableJobBootstrap: Boolean(d.enable_job_bootstrap),
      });
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || t('admin.twenty.loadError') });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadSettings();
  }, []);

  const saveSettings = async () => {
    setSaving(true);
    setMessage(null);
    try {
      const payload: TwentySettingsPayload = {
        base_url: baseUrl.trim(),
        api_key: apiKey.trim() || undefined,
        clear_api_key: clearApiKey,
        sync_interval_minutes: syncInterval,
        enable_job_bootstrap: enableJobBootstrap,
      };
      const resp = await twentyApi.updateSettings(payload);
      const d = resp.data;
      setApiKey('');
      setClearApiKey(false);
      setApiKeyConfigured(!!d.api_key_configured);
      setApiKeyMasked(d.api_key_masked || '');
      setBaseUrlLocked(Boolean(d.base_url_locked));
      setApiKeyLocked(Boolean(d.api_key_locked));
      setSaved({
        baseUrl: d.base_url || baseUrl,
        syncInterval: Number(d.sync_interval_minutes || 0),
        enableJobBootstrap: Boolean(d.enable_job_bootstrap),
      });
      setMessage({ type: 'success', text: d.message || t('admin.twenty.settingsSaved') });
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || t('admin.twenty.saveError') });
    } finally {
      setSaving(false);
    }
  };

  const syncNow = async () => {
    setSyncing(true);
    setMessage(null);
    try {
      const resp = await twentyApi.sync();
      setMessage({
        type: 'success',
        text: t('admin.twenty.syncResultDetailed', {
          created: resp.data.created,
          updated: resp.data.updated,
          productCreated: resp.data.productCreated,
          productUpdated: resp.data.productUpdated,
          packageCreated: resp.data.packageCreated,
          packageUpdated: resp.data.packageUpdated,
        }),
      });
    } catch (err: any) {
      setMessage({ type: 'error', text: err?.response?.data?.error || t('admin.twenty.syncError') });
    } finally {
      setSyncing(false);
    }
  };

  if (loading) {
    return <div className="p-4 text-sm text-gray-500 dark:text-gray-400">{t('common.loading')}</div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('admin.twenty.title')}</h3>
        <p className="text-sm text-gray-600 dark:text-gray-300 mt-1">{t('admin.twenty.subtitle')}</p>
      </div>

      {message && (
        <div
          className={`rounded-lg border p-3 flex items-start gap-2 ${
            message.type === 'success'
              ? 'border-green-200 bg-green-50 text-green-800'
              : message.type === 'error'
                ? 'border-red-200 bg-red-50 text-red-800'
                : 'border-blue-200 bg-blue-50 text-blue-800'
          }`}
        >
          {message.type === 'success' ? <CheckCircle2 className="w-4 h-4 mt-0.5" /> : <AlertCircle className="w-4 h-4 mt-0.5" />}
          <span className="text-sm">{message.text}</span>
        </div>
      )}

      <div className="rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200">
          <Link2 className="w-4 h-4" />
          {t('admin.twenty.connectionSettings')}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">{t('admin.twenty.baseUrl')}</label>
          <input
            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 disabled:bg-gray-100 dark:disabled:bg-gray-700 disabled:text-gray-500 dark:disabled:text-gray-400"
            placeholder="http://localhost:2020"
            value={baseUrl}
            onChange={(e) => setBaseUrl(e.target.value)}
            disabled={baseUrlLocked}
          />
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t('admin.twenty.baseUrlDesc')}</p>
          {baseUrlLocked && (
            <p className="text-xs text-amber-700 mt-1">{t('admin.twenty.baseUrlLocked')}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">{t('admin.twenty.apiKey')}</label>
          <input
            type="password"
            className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-3 py-2 text-sm text-gray-900 dark:text-gray-100 disabled:bg-gray-100 dark:disabled:bg-gray-700 disabled:text-gray-500 dark:disabled:text-gray-400"
            placeholder={apiKeyConfigured ? t('admin.twenty.keyPlaceholderUpdate') : t('admin.twenty.keyPlaceholder')}
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            disabled={apiKeyLocked}
          />
          <div className="mt-2 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 text-xs min-w-0">
            <span className="text-gray-600 dark:text-gray-300 break-all min-w-0">
              {apiKeyConfigured
                ? `${t('admin.twenty.keyConfigured')}${apiKeyMasked ? ` (${apiKeyMasked})` : ''}`
                : t('admin.twenty.keyNotConfigured')}
            </span>
            <label className="inline-flex items-center gap-2 shrink-0">
              <input type="checkbox" checked={clearApiKey} onChange={(e) => setClearApiKey(e.target.checked)} disabled={apiKeyLocked} />
              {t('admin.twenty.clearKey')}
            </label>
          </div>
          {apiKeyLocked && (
            <p className="text-xs text-amber-700 mt-1">{t('admin.twenty.apiKeyLocked')}</p>
          )}
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">
              <Clock3 className="inline w-4 h-4 mr-1" />
              {t('admin.twenty.syncInterval')}
            </label>
            <select
              className="w-full rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 px-3 py-2 text-sm"
              value={syncInterval}
              onChange={(e) => setSyncInterval(Number(e.target.value))}
              title={t('admin.twenty.syncInterval')}
            >
              {SYNC_INTERVAL_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value} className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
                  {t(opt.labelKey)}
                </option>
              ))}
            </select>
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">{t('admin.twenty.syncIntervalDesc')}</p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-200 mb-1">{t('admin.twenty.jobBootstrap')}</label>
            <label className="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-200 mt-2">
              <input
                type="checkbox"
                checked={enableJobBootstrap}
                onChange={(e) => setEnableJobBootstrap(e.target.checked)}
              />
              {t('admin.twenty.jobBootstrapDesc')}
            </label>
          </div>
        </div>

        <div className="flex flex-wrap gap-3 pt-2">
          <button
            className="inline-flex items-center gap-2 rounded-md bg-blue-600 text-white px-4 py-2 text-sm hover:bg-blue-700 disabled:opacity-60"
            onClick={saveSettings}
            disabled={saving || !hasUnsavedChanges}
          >
            <Save className="w-4 h-4" />
            {saving ? t('admin.twenty.saving') : t('admin.twenty.saveSettings')}
          </button>

          <button
            className="inline-flex items-center gap-2 rounded-md bg-emerald-600 text-white px-4 py-2 text-sm hover:bg-emerald-700 disabled:opacity-60"
            onClick={syncNow}
            disabled={syncing || hasUnsavedChanges}
          >
            <RefreshCcw className="w-4 h-4" />
            {syncing ? t('admin.twenty.syncing') : t('admin.twenty.syncNow')}
          </button>

          <button
            className="inline-flex items-center gap-2 rounded-md border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 px-4 py-2 text-sm hover:bg-gray-50 dark:hover:bg-gray-800"
            onClick={() => void loadSettings()}
            disabled={saving || syncing}
          >
            <KeyRound className="w-4 h-4" />
            {t('admin.twenty.reload')}
          </button>
        </div>

        {hasUnsavedChanges && (
          <p className="text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-md p-2">
            {t('admin.twenty.actionsUnsavedHint')}
          </p>
        )}
      </div>
    </div>
  );
}
