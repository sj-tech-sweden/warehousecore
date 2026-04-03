import { useState, useEffect } from 'react';
import { Save, RefreshCcw, AlertCircle, Link2, Package, CheckCircle2, XCircle } from 'lucide-react';
import { eventoryApi, type EventoryProduct } from '../../lib/api';
import { useTranslation } from 'react-i18next';

export function EventoryTab() {
  const { t } = useTranslation();

  const [apiUrl, setApiUrl] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [apiKeyConfigured, setApiKeyConfigured] = useState(false);
  const [apiKeyMasked, setApiKeyMasked] = useState('');

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const [products, setProducts] = useState<EventoryProduct[] | null>(null);
  const [productCount, setProductCount] = useState<number | null>(null);

  const [message, setMessage] = useState<{ type: 'success' | 'error' | 'info'; text: string } | null>(null);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const { data } = await eventoryApi.getSettings();
      setApiUrl(data.api_url || '');
      setApiKeyConfigured(data.api_key_configured);
      setApiKeyMasked(data.api_key_masked || '');
    } catch (err) {
      console.error('Failed to load Eventory settings:', err);
      setMessage({ type: 'error', text: t('admin.eventory.loadError') });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    const trimmedUrl = apiUrl.trim();
    if (!trimmedUrl) {
      setMessage({ type: 'error', text: t('admin.eventory.urlRequired') });
      return;
    }

    setSaving(true);
    setMessage(null);

    try {
      const payload: { api_url: string; api_key?: string } = { api_url: trimmedUrl };
      if (apiKey.trim()) {
        payload.api_key = apiKey.trim();
      }

      const { data } = await eventoryApi.updateSettings(payload);
      setApiUrl(data.api_url || trimmedUrl);
      setApiKeyConfigured(data.api_key_configured);
      setApiKey(''); // clear the input after saving
      setMessage({ type: 'success', text: t('admin.eventory.settingsSaved') });
      loadSettings();
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

        {/* API Key */}
        <div>
          <label className="block text-sm font-medium text-gray-300 mb-2">
            {t('admin.eventory.apiKey')}
            {apiKeyConfigured && (
              <span className="ml-2 inline-flex items-center gap-1 text-xs text-green-400">
                <CheckCircle2 className="w-3 h-3" />
                {t('admin.eventory.keyConfigured')} ({apiKeyMasked})
              </span>
            )}
            {!apiKeyConfigured && (
              <span className="ml-2 inline-flex items-center gap-1 text-xs text-yellow-400">
                <XCircle className="w-3 h-3" />
                {t('admin.eventory.keyNotConfigured')}
              </span>
            )}
          </label>
          <input
            type="password"
            value={apiKey}
            onChange={e => setApiKey(e.target.value)}
            placeholder={apiKeyConfigured ? t('admin.eventory.keyPlaceholderUpdate') : t('admin.eventory.keyPlaceholder')}
            className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
          />
          <p className="mt-1 text-xs text-gray-500">{t('admin.eventory.apiKeyDesc')}</p>
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

        <div className="flex flex-wrap gap-3">
          {/* Fetch / preview */}
          <button
            onClick={handleFetchProducts}
            disabled={fetching || !apiUrl.trim()}
            className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
          >
            <RefreshCcw className={`w-4 h-4 ${fetching ? 'animate-spin' : ''}`} />
            {fetching ? t('admin.eventory.fetching') : t('admin.eventory.fetchProducts')}
          </button>

          {/* Sync */}
          <button
            onClick={handleSync}
            disabled={syncing || !apiUrl.trim()}
            className="flex items-center gap-2 px-5 py-2.5 bg-green-700 hover:bg-green-600 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
          >
            <Package className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
            {syncing ? t('admin.eventory.syncing') : t('admin.eventory.syncProducts')}
          </button>
        </div>

        <p className="text-xs text-gray-500">{t('admin.eventory.syncNote')}</p>
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
                    <tr key={p.id ?? i} className="hover:bg-white/5 transition-colors">
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
