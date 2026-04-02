import { useState, useEffect } from 'react';
import { Save, DollarSign, AlertCircle } from 'lucide-react';
import { adminSettingsApi } from '../../lib/api';
import { invalidateCurrencyCache } from '../../hooks/useCurrencySymbol';
import { useTranslation } from 'react-i18next';

export function CurrencySettingsTab() {
  const { t } = useTranslation();
  const [symbol, setSymbol] = useState('€');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  useEffect(() => {
    loadCurrency();
  }, []);

  const loadCurrency = async () => {
    try {
      const { data } = await adminSettingsApi.getCurrency();
      setSymbol(data.currency_symbol || '€');
    } catch (error) {
      console.error('Failed to load currency settings:', error);
      setMessage({ type: 'error', text: t('admin.currencySettings.loadError') });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    const trimmed = symbol.trim();
    if (!trimmed) {
      setMessage({ type: 'error', text: t('admin.currencySettings.symbolRequired') });
      return;
    }
    if (trimmed.length > 8) {
      setMessage({ type: 'error', text: t('admin.currencySettings.symbolTooLong') });
      return;
    }

    setSaving(true);
    setMessage(null);

    try {
      const { data } = await adminSettingsApi.updateCurrency(trimmed);
      setSymbol(data.currency_symbol);
      invalidateCurrencyCache();
      setMessage({ type: 'success', text: t('admin.currencySettings.settingsSaved') });
    } catch (error) {
      console.error('Failed to update currency settings:', error);
      setMessage({ type: 'error', text: t('admin.currencySettings.saveError') });
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3 mb-6">
        <DollarSign className="w-6 h-6 text-accent-red" />
        <div>
          <h2 className="text-2xl font-bold text-white">{t('admin.currencySettings.title')}</h2>
          <p className="text-gray-400 text-sm">{t('admin.currencySettings.subtitle')}</p>
        </div>
      </div>

      {message && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            message.type === 'success'
              ? 'bg-green-500/10 border border-green-500/20'
              : 'bg-red-500/10 border border-red-500/20'
          }`}
        >
          <AlertCircle
            className={`w-5 h-5 ${message.type === 'success' ? 'text-green-500' : 'text-red-500'}`}
          />
          <span className={message.type === 'success' ? 'text-green-500' : 'text-red-500'}>
            {message.text}
          </span>
        </div>
      )}

      <div className="space-y-6">
        {/* Currency Symbol */}
        <div className="bg-white/5 rounded-xl p-6 border border-white/10">
          <label className="block text-sm font-medium text-gray-300 mb-3">
            {t('admin.currencySettings.currencySymbol')}
          </label>
          <p className="text-sm text-gray-400 mb-4">
            {t('admin.currencySettings.currencySymbolDesc')}
          </p>
          <div className="flex items-center gap-4">
            <input
              type="text"
              maxLength={8}
              value={symbol}
              onChange={e => setSymbol(e.target.value)}
              className="w-32 px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors text-lg text-center"
              placeholder="€"
              title={t('admin.currencySettings.currencySymbol')}
            />
            <div className="text-gray-400 text-sm">
              {t('admin.currencySettings.preview')}:{' '}
              <span className="text-white font-semibold">
                1.99 {symbol || '€'}
              </span>
            </div>
          </div>
          <div className="mt-2 text-xs text-gray-500">
            {t('admin.currencySettings.examples')}
          </div>
        </div>

        {/* Info Box */}
        <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-6">
          <div className="flex items-start gap-3">
            <AlertCircle className="w-5 h-5 text-blue-400 flex-shrink-0 mt-0.5" />
            <div className="space-y-2">
              <h3 className="text-blue-400 font-semibold">{t('admin.currencySettings.importantNotes')}</h3>
              <ul className="text-sm text-blue-300 space-y-1 list-disc list-inside">
                <li>{t('admin.currencySettings.note1')}</li>
                <li>{t('admin.currencySettings.note2')}</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      {/* Save Button */}
      <div className="flex justify-end pt-4">
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-6 py-3 bg-accent-red hover:bg-accent-red/80 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg transition-colors"
        >
          <Save className="w-5 h-5" />
          {saving ? t('admin.currencySettings.saving') : t('admin.currencySettings.saveSettings')}
        </button>
      </div>
    </div>
  );
}
