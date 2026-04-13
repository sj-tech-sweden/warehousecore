import { useEffect, useState } from 'react';
import { KeyRound, Plus, Trash2, ToggleLeft, ToggleRight, RefreshCcw, Copy, Shield } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { apiKeysAdminApi, type APIKeyItem } from '../../lib/api';
import { formatLocalDateTime } from '../../lib/utils';

export function APIKeysTab() {
  const { t } = useTranslation();
  const [keys, setKeys] = useState<APIKeyItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState('');
  const [newIsAdmin, setNewIsAdmin] = useState(false);
  const [plainKey, setPlainKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadKeys();
  }, []);

  const loadKeys = async () => {
    setLoading(true);
    setError(null);
    try {
      const { data } = await apiKeysAdminApi.list();
      setKeys(data.keys || []);
    } catch (err) {
      console.error('Failed to load API keys', err);
      setError(t('admin.apiKeys.errors.load'));
    } finally {
      setLoading(false);
    }
  };

  const createKey = async () => {
    if (!newName.trim()) {
      setError(t('admin.apiKeys.errors.nameRequired'));
      return;
    }
    setCreating(true);
    setError(null);
    setPlainKey(null);
    try {
      const { data } = await apiKeysAdminApi.create({ name: newName.trim(), is_admin: newIsAdmin });
      setPlainKey(data.api_key);
      setNewName('');
      setNewIsAdmin(false);
      await loadKeys();
    } catch (err) {
      console.error('Failed to create API key', err);
      setError(t('admin.apiKeys.errors.create'));
    } finally {
      setCreating(false);
    }
  };

  const toggleKey = async (id: number, active: boolean) => {
    try {
      await apiKeysAdminApi.updateStatus(id, !active);
      await loadKeys();
    } catch (err) {
      console.error('Failed to update API key status', err);
      setError(t('admin.apiKeys.errors.update'));
    }
  };

  const deleteKey = async (id: number) => {
    if (!confirm(t('admin.apiKeys.confirmDelete'))) return;
    try {
      await apiKeysAdminApi.delete(id);
      await loadKeys();
    } catch (err) {
      console.error('Failed to delete API key', err);
      setError(t('admin.apiKeys.errors.delete'));
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <KeyRound className="w-6 h-6 text-accent-red" />
        <div>
          <h2 className="text-2xl font-bold text-white">{t('admin.apiKeys.title')}</h2>
          <p className="text-gray-400 text-sm">{t('admin.apiKeys.subtitle')}</p>
        </div>
      </div>

      {error && <div className="bg-red-500/10 border border-red-500/30 text-red-400 px-4 py-3 rounded-lg">{error}</div>}

      <div className="glass-dark rounded-xl p-4 flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="flex-1">
          <label className="block text-sm text-gray-300 mb-1">{t('admin.apiKeys.nameDescription')}</label>
          <input
            className="w-full px-4 py-3 bg-dark-light border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red"
            placeholder={t('admin.apiKeys.namePlaceholder')}
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
          />
        </div>
        <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer select-none" title={t('admin.apiKeys.adminAccessHint')}>
          <input
            type="checkbox"
            checked={newIsAdmin}
            onChange={(e) => setNewIsAdmin(e.target.checked)}
            className="accent-accent-red w-4 h-4"
          />
          <Shield className="w-4 h-4 text-yellow-400" />
          {t('admin.apiKeys.adminAccess')}
        </label>
        <button
          onClick={createKey}
          disabled={creating}
          className="flex items-center gap-2 px-4 py-3 bg-accent-red hover:bg-accent-red/80 rounded-lg text-white font-semibold disabled:opacity-60"
        >
          <Plus className="w-4 h-4" /> {t('admin.apiKeys.newKey')}
        </button>
        <button
          onClick={loadKeys}
          className="flex items-center gap-2 px-4 py-3 bg-white/10 hover:bg-white/20 rounded-lg text-white"
        >
          <RefreshCcw className="w-4 h-4" /> {t('common.update')}
        </button>
      </div>

      {plainKey && (
        <div className="bg-green-500/10 border border-green-500/20 text-green-200 px-4 py-3 rounded-lg flex items-center justify-between gap-3">
          <div>
            <div className="text-sm font-semibold">{t('admin.apiKeys.oneTimeVisible')}</div>
            <div className="font-mono break-all">{plainKey}</div>
          </div>
          <button
            onClick={() => navigator.clipboard.writeText(plainKey)}
            className="flex items-center gap-2 px-3 py-2 bg-green-500/20 hover:bg-green-500/30 rounded-md text-green-100"
          >
            <Copy className="w-4 h-4" /> {t('admin.apiKeys.copy')}
          </button>
        </div>
      )}

      <div className="glass-dark rounded-xl overflow-hidden">
        <div className="grid grid-cols-12 px-4 py-3 text-sm font-semibold text-gray-300 border-b border-white/5">
          <div className="col-span-3">{t('cases.name')}</div>
          <div className="col-span-2">{t('devices.status')}</div>
          <div className="col-span-2">{t('admin.apiKeys.adminAccess')}</div>
          <div className="col-span-3">{t('admin.apiKeys.lastUsed')}</div>
          <div className="col-span-2 text-right">{t('labels.actions')}</div>
        </div>
        {loading ? (
          <div className="p-6 text-center text-gray-400">{t('common.loading')}</div>
        ) : keys.length === 0 ? (
          <div className="p-6 text-center text-gray-400">{t('admin.apiKeys.empty')}</div>
        ) : (
          keys.map((k) => (
            <div
              key={k.id}
              className="grid grid-cols-12 px-4 py-3 items-center border-b border-white/5 text-sm text-gray-100"
            >
              <div className="col-span-3">
                <div className="font-semibold text-white">{k.name}</div>
                <div className="text-xs text-gray-400">{t('admin.apiKeys.id', { id: k.id })}</div>
              </div>
              <div className="col-span-2 flex items-center gap-2">
                {k.is_active ? (
                  <>
                    <ToggleRight className="w-5 h-5 text-green-400" />
                    <span className="text-green-300">{t('zones.active')}</span>
                  </>
                ) : (
                  <>
                    <ToggleLeft className="w-5 h-5 text-gray-500" />
                    <span className="text-gray-400">{t('zones.inactive')}</span>
                  </>
                )}
                <button
                  onClick={() => toggleKey(k.id, k.is_active)}
                  className="ml-3 px-3 py-1 rounded-md bg-white/10 hover:bg-white/20 text-white text-xs"
                >
                  {k.is_active ? t('admin.apiKeys.deactivate') : t('admin.apiKeys.activate')}
                </button>
              </div>
              <div className="col-span-2 flex items-center gap-1">
                {k.is_admin ? (
                  <>
                    <Shield className="w-4 h-4 text-yellow-400" />
                    <span className="text-yellow-300 text-xs font-semibold">{t('admin.apiKeys.adminBadge')}</span>
                  </>
                ) : (
                  <span className="text-gray-500 text-xs">—</span>
                )}
              </div>
              <div className="col-span-3 text-gray-300 text-sm">
                {k.last_used_at ? formatLocalDateTime(k.last_used_at) : t('admin.apiKeys.neverUsed')}
              </div>
              <div className="col-span-2 flex justify-end gap-2">
                <button
                  onClick={() => deleteKey(k.id)}
                  className="p-2 rounded-md bg-red-500/10 hover:bg-red-500/20 text-red-300"
                  title={t('common.delete')}
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
