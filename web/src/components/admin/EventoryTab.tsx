import { useState, useEffect, useCallback } from 'react';
import { Save, RefreshCcw, AlertCircle, Link2, Package, CheckCircle2, XCircle, Clock } from 'lucide-react';
import { eventoryApi, type EventoryProduct, type EventorySettingsPayload } from '../../lib/api';
import { useTranslation } from 'react-i18next';

const SYNC_INTERVAL_OPTIONS = [
  { value: 0, labelKey: 'admin.eventory.syncIntervalDisabled' },
  { value: 15, labelKey: 'admin.eventory.syncInterval15m' },
  { value: 30, labelKey: 'admin.eventory.syncInterval30m' },
  { value: 60, labelKey: 'admin.eventory.syncInterval1h' },
  { value: 120, labelKey: 'admin.eventory.syncInterval2h' },
  { value: 240, labelKey: 'admin.eventory.syncInterval4h' },
  { value: 480, labelKey: 'admin.eventory.syncInterval8h' },
  { value: 1440, labelKey: 'admin.eventory.syncInterval24h' },
];

export function EventoryTab() {
  const { t } = useTranslation();

  // Settings state
  const [apiUrl, setApiUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [apiKeyConfigured, setApiKeyConfigured] = useState(false);
  const [apiKeyMasked, setApiKeyMasked] = useState('');
  const [clearApiKey, setClearApiKey] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfigured, setPasswordConfigured] = useState(false);
  const [clearPassword, setClearPassword] = useState(false);
  const [tokenEndpoint, setTokenEndpoint] = useState('');
  const [supplierName, setSupplierName] = useState('');
  const [syncInterval, setSyncInterval] = useState(0);

  // Snapshot of the last successfully loaded/saved non-secret fields.
  // hasUnsavedChanges is derived from this rather than from a separate boolean
  // flag so that reverting a field back to its saved value correctly re-enables
  // the Fetch / Sync actions.
  const [savedSettings, setSavedSettings] = useState<{
    apiUrl: string;
    username: string;
    tokenEndpoint: string;
    supplierName: string;
    syncInterval: number;
  } | null>(null);

  // Secrets (api_key, password) are never returned by the server, so we cannot
  // compare them to a snapshot. Instead, any typed value or explicit clear flag
  // counts as a pending change.
  const hasUnsavedChanges =
    savedSettings !== null &&
    (apiUrl !== savedSettings.apiUrl ||
      username !== savedSettings.username ||
      tokenEndpoint !== savedSettings.tokenEndpoint ||
      supplierName !== savedSettings.supplierName ||
      syncInterval !== savedSettings.syncInterval ||
      apiKey !== '' ||
      clearApiKey ||
      password !== '' ||
      clearPassword);

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const [products, setProducts] = useState<EventoryProduct[] | null>(null);
  const [productCount, setProductCount] = useState<number | null>(null);

  const [message, setMessage] = useState<{ type: 'success' | 'error' | 'info'; text: string } | null>(null);

  const loadSettings = useCallback(async () => {
    try {
      const { data } = await eventoryApi.getSettings();
      setApiUrl(data.api_url || '');
      setApiKeyConfigured(data.api_key_configured);
      setApiKeyMasked(data.api_key_masked || '');
      setClearApiKey(false);
      setUsername(data.username || '');
      setPasswordConfigured(data.password_configured);
      setClearPassword(false);
      setTokenEndpoint(data.token_endpoint || '');
      setSupplierName(data.supplier_name || '');
      setSyncInterval(data.sync_interval_minutes ?? 0);
      setSavedSettings({
        apiUrl: data.api_url || '',
        username: data.username || '',
        tokenEndpoint: data.token_endpoint || '',
        supplierName: data.supplier_name || '',
        syncInterval: data.sync_interval_minutes ?? 0,
      });
    } catch (err) {
      console.error('Failed to load Eventory settings:', err);
      setMessage({ type: 'error', text: t('admin.eventory.loadError') });
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const handleSave = async () => {
    const trimmedUrl = apiUrl.trim();
    if (!trimmedUrl) {
      setMessage({ type: 'error', text: t('admin.eventory.urlRequired') });
      return;
    }

    setSaving(true);
    setMessage(null);

    try {
      const payload: EventorySettingsPayload = {
        api_url: trimmedUrl,
        username: username.trim(),
        token_endpoint: tokenEndpoint.trim(),
        supplier_name: supplierName.trim(),
        sync_interval_minutes: syncInterval,
      };
      if (clearApiKey) {
        payload.clear_api_key = true;
      } else if (apiKey.trim()) {
        payload.api_key = apiKey.trim();
      }
      if (clearPassword) {
        payload.clear_password = true;
      } else if (password.trim()) {
        payload.password = password.trim();
      }

      const { data } = await eventoryApi.updateSettings(payload);
      setApiUrl(data.api_url || trimmedUrl);
      setApiKeyConfigured(data.api_key_configured);
      setApiKeyMasked(data.api_key_masked || '');
      setApiKey('');
      setClearApiKey(false);
      setUsername(data.username || '');
      setPasswordConfigured(data.password_configured);
      setPassword('');
      setClearPassword(false);
      setTokenEndpoint(data.token_endpoint || '');
      setSupplierName(data.supplier_name || '');
      setSyncInterval(data.sync_interval_minutes ?? 0);
      setSavedSettings({
        apiUrl: data.api_url || trimmedUrl,
        username: data.username || '',
        tokenEndpoint: data.token_endpoint || '',
        supplierName: data.supplier_name || '',
        syncInterval: data.sync_interval_minutes ?? 0,
      });
      setMessage({ type: 'success', text: t('admin.eventory.settingsSaved') });
    } catch (err) {
      console.error('Failed to save Eventory settings:', err);
      setMessage({ type: 'error', text: t('admin.eventory.saveError') });
    } finally {
      setSaving(false);
    }
  };

  const handleFetchProducts = async () => {
    setFetching(true);
    setMessage(null);
    setProducts(null);
    setProductCount(null);

    try {
      const { data } = await eventoryApi.getProducts();
      setProducts(data.products || []);
      setProductCount(data.count);
      setMessage({
        type: 'info',
        text: t('admin.eventory.fetchedProducts', { count: data.count }),
      });
    } catch (err: any) {
      console.error('Failed to fetch Eventory products:', err);
      const detail = err?.response?.data?.error || t('admin.eventory.fetchError');
      setMessage({ type: 'error', text: detail });
    } finally {
      setFetching(false);
    }
  };

  const handleSync = async () => {
    setSyncing(true);
    setMessage(null);

    try {
      const { data } = await eventoryApi.sync();
      setMessage({
        type: 'success',
        text: t('admin.eventory.syncResult', {
          imported: data.imported,
          updated: data.updated,
          skipped: data.skipped,
        }),
      });
    } catch (err: any) {
      console.error('Failed to sync Eventory products:', err);
      const detail = err?.response?.data?.error || t('admin.eventory.syncError');
      setMessage({ type: 'error', text: detail });
    } finally {
      setSyncing(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red" />
      </div>
    );
  }

  const messageBg =
    message?.type === 'success'
      ? 'bg-green-500/10 border border-green-500/20'
      : message?.type === 'info'
        ? 'bg-blue-500/10 border border-blue-500/20'
        : 'bg-red-500/10 border border-red-500/20';
  const messageTextColor =
    message?.type === 'success'
      ? 'text-green-500'
      : message?.type === 'info'
        ? 'text-blue-400'
        : 'text-red-500';

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <Link2 className="w-6 h-6 text-accent-red" />
        <div>
          <h2 className="text-2xl font-bold text-white">{t('admin.eventory.title')}</h2>
          <p className="text-gray-400 text-sm">{t('admin.eventory.subtitle')}</p>
        </div>
      </div>

      {/* Message banner */}
      {message && (
        <div className={`p-4 rounded-lg flex items-center gap-3 ${messageBg}`}>
          <AlertCircle className={`w-5 h-5 flex-shrink-0 ${messageTextColor}`} />
          <span className={messageTextColor}>{message.text}</span>
        </div>
      )}

      {/* Settings card */}
      <div className="bg-white/5 rounded-xl p-6 border border-white/10 space-y-5">
        <h3 className="text-lg font-semibold text-white">{t('admin.eventory.connectionSettings')}</h3>

        {/* API URL */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            {t('admin.eventory.apiUrl')}
          </label>
          <input
            type="url"
            value={apiUrl}
            onChange={e => setApiUrl(e.target.value)}
            placeholder="https://api.eventory.se"
            className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
          <p className="mt-1 text-xs text-gray-500">{t('admin.eventory.apiUrlDesc')}</p>
        </div>

        {/* Username / Password (OAuth2 password grant) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              {t('admin.eventory.username')}
            </label>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder={t('admin.eventory.usernamePlaceholder')}
              autoComplete="off"
              className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              {t('admin.eventory.password')}
              {passwordConfigured && !clearPassword && (
                <span className="ml-2 inline-flex items-center gap-1 text-xs text-green-400">
                  <CheckCircle2 className="w-3 h-3" />
                  {t('admin.eventory.passwordConfigured')}
                </span>
              )}
              {clearPassword && (
                <span className="ml-2 inline-flex items-center gap-1 text-xs text-yellow-400">
                  <XCircle className="w-3 h-3" />
                  {t('admin.eventory.clearPassword')}
                </span>
              )}
            </label>
            <div className="flex gap-2">
              <input
                type="password"
                value={password}
                onChange={e => { setPassword(e.target.value); setClearPassword(false); }}
                placeholder={passwordConfigured ? t('admin.eventory.passwordPlaceholderUpdate') : t('admin.eventory.passwordPlaceholder')}
                autoComplete="new-password"
                className="flex-1 px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
              />
              {passwordConfigured && !clearPassword && (
                <button
                  type="button"
                  onClick={() => { setClearPassword(true); setPassword(''); }}
                  className="px-3 py-2 text-xs text-red-400 border border-red-400/30 rounded-lg hover:bg-red-400/10 transition-colors whitespace-nowrap"
                  title={t('admin.eventory.clearPassword')}
                  aria-label={t('admin.eventory.clearPassword')}
                >
                  <XCircle className="w-4 h-4" />
                </button>
              )}
            </div>
          </div>
        </div>
        <p className="text-xs text-gray-500">{t('admin.eventory.credentialsDesc')}</p>

        {/* API Key (alternative to username/password) */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            {t('admin.eventory.apiKey')}
            {apiKeyConfigured && !clearApiKey && (
              <span className="ml-2 inline-flex items-center gap-1 text-xs text-green-400">
                <CheckCircle2 className="w-3 h-3" />
                {t('admin.eventory.keyConfigured')} ({apiKeyMasked})
              </span>
            )}
            {clearApiKey && (
              <span className="ml-2 inline-flex items-center gap-1 text-xs text-yellow-400">
                <XCircle className="w-3 h-3" />
                {t('admin.eventory.clearKey')}
              </span>
            )}
            {!apiKeyConfigured && !clearApiKey && (
              <span className="ml-2 inline-flex items-center gap-1 text-xs text-yellow-400">
                <XCircle className="w-3 h-3" />
                {t('admin.eventory.keyNotConfigured')}
              </span>
            )}
          </label>
          <div className="flex gap-2">
            <input
              type="password"
              value={apiKey}
              onChange={e => { setApiKey(e.target.value); setClearApiKey(false); }}
              placeholder={apiKeyConfigured ? t('admin.eventory.keyPlaceholderUpdate') : t('admin.eventory.keyPlaceholder')}
              className="flex-1 px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
            />
            {apiKeyConfigured && !clearApiKey && (
              <button
                type="button"
                onClick={() => { setClearApiKey(true); setApiKey(''); }}
                className="px-3 py-2 text-xs text-red-400 border border-red-400/30 rounded-lg hover:bg-red-400/10 transition-colors whitespace-nowrap"
                title={t('admin.eventory.clearKey')}
                aria-label={t('admin.eventory.clearKey')}
              >
                <XCircle className="w-4 h-4" />
              </button>
            )}
          </div>
          <p className="mt-1 text-xs text-gray-500">{t('admin.eventory.apiKeyDesc')}</p>
        </div>

        {/* Optional: token endpoint override */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            {t('admin.eventory.tokenEndpoint')}
          </label>
          <input
            type="url"
            value={tokenEndpoint}
            onChange={e => setTokenEndpoint(e.target.value)}
            placeholder="https://api.eventory.se/oauth/token"
            className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
          <p className="mt-1 text-xs text-gray-500">{t('admin.eventory.tokenEndpointDesc')}</p>
        </div>

        {/* Supplier name */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            {t('admin.eventory.supplierName')}
          </label>
          <input
            type="text"
            value={supplierName}
            onChange={e => setSupplierName(e.target.value)}
            placeholder="Eventory"
            className="w-full sm:w-64 px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
          <p className="mt-1 text-xs text-gray-500">{t('admin.eventory.supplierNameDesc')}</p>
        </div>

        {/* Sync interval */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            <Clock className="w-4 h-4 inline mr-1" />
            {t('admin.eventory.syncInterval')}
          </label>
          <select
            value={syncInterval}
            onChange={e => setSyncInterval(Number(e.target.value))}
            className="w-full sm:w-64 px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white focus:outline-none focus:border-accent-red transition-colors"
          >
            {SYNC_INTERVAL_OPTIONS.map(opt => (
              <option key={opt.value} value={opt.value}>
                {t(opt.labelKey)}
              </option>
            ))}
          </select>
          <p className="mt-1 text-xs text-gray-500">{t('admin.eventory.syncIntervalDesc')}</p>
        </div>

        {/* Save button */}
        <div className="flex justify-end">
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-2 px-6 py-3 bg-accent-red hover:bg-accent-red/80 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
          >
            <Save className="w-5 h-5" />
            {saving ? t('admin.eventory.saving') : t('admin.eventory.saveSettings')}
          </button>
        </div>
      </div>

      {/* Actions card */}
      <div className="bg-white/5 rounded-xl p-6 border border-white/10 space-y-4">
        <h3 className="text-lg font-semibold text-white">{t('admin.eventory.actions')}</h3>
        <p className="text-sm text-gray-400">{t('admin.eventory.actionsDesc')}</p>

        {hasUnsavedChanges && (
          <p className="text-sm text-yellow-400">{t('admin.eventory.actionsUnsavedHint')}</p>
        )}

        <div className="flex flex-wrap gap-3">
          {/* Fetch / preview */}
          <button
            onClick={handleFetchProducts}
            disabled={fetching || !apiUrl.trim() || hasUnsavedChanges}
            className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
          >
            <RefreshCcw className={`w-4 h-4 ${fetching ? 'animate-spin' : ''}`} />
            {fetching ? t('admin.eventory.fetching') : t('admin.eventory.fetchProducts')}
          </button>

          {/* Sync */}
          <button
            onClick={handleSync}
            disabled={syncing || !apiUrl.trim() || hasUnsavedChanges}
            className="flex items-center gap-2 px-5 py-2.5 bg-green-700 hover:bg-green-600 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
          >
            <Package className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
            {syncing ? t('admin.eventory.syncing') : t('admin.eventory.syncProducts')}
          </button>
        </div>

        <p className="text-xs text-gray-500">
          {t('admin.eventory.syncNote', { supplier: (supplierName || '').trim() || 'Eventory' })}
        </p>
      </div>

      {/* Products preview */}
      {products !== null && (
        <div className="bg-white/5 rounded-xl p-6 border border-white/10">
          <div className="flex items-center gap-2 mb-4">
            <Package className="w-5 h-5 text-accent-red" />
            <h3 className="text-lg font-semibold text-white">
              {t('admin.eventory.previewTitle', { count: productCount ?? products.length })}
            </h3>
          </div>

          {products.length === 0 ? (
            <p className="text-gray-400 text-sm">{t('admin.eventory.noProducts')}</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-white/10 text-gray-400 text-left">
                    <th className="pb-2 pr-4">{t('admin.eventory.colName')}</th>
                    <th className="pb-2 pr-4">{t('admin.eventory.colCategory')}</th>
                    <th className="pb-2 pr-4">{t('admin.eventory.colPrice')}</th>
                    <th className="pb-2">{t('admin.eventory.colDescription')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
                  {products.slice(0, 50).map((p, i) => (
                    <tr key={p.id != null ? String(p.id) : `row-${i}`} className="hover:bg-white/5 transition-colors">
                      <td className="py-2 pr-4 text-white font-medium">{p.name || '—'}</td>
                      <td className="py-2 pr-4 text-gray-300">{p.category || '—'}</td>
                      <td className="py-2 pr-4 text-gray-300">{p.price != null ? p.price : '—'}</td>
                      <td className="py-2 text-gray-400 max-w-xs truncate">{p.description || '—'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
              {products.length > 50 && (
                <p className="mt-3 text-xs text-gray-500">
                  {t('admin.eventory.showingFirst', { shown: 50, total: products.length })}
                </p>
              )}
            </div>
          )}
        </div>
      )}

      {/* Info box */}
      <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-6">
        <div className="flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-blue-400 flex-shrink-0 mt-0.5" />
          <div className="space-y-2">
            <h3 className="text-blue-400 font-semibold">{t('admin.eventory.infoTitle')}</h3>
            <ul className="text-sm text-blue-300 space-y-1 list-disc list-inside">
              <li>{t('admin.eventory.info1')}</li>
              <li>{t('admin.eventory.info2')}</li>
              <li>{t('admin.eventory.info3')}</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}

