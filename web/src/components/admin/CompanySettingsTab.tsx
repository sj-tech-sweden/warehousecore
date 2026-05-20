import { useState, useEffect } from 'react';
import { Save, Building, AlertCircle } from 'lucide-react';
import { adminSettingsApi } from '../../lib/api';
import { useTranslation } from 'react-i18next';

export function CompanySettingsTab() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const [name, setName] = useState('');
  const [addressLine1, setAddressLine1] = useState('');
  const [addressLine2, setAddressLine2] = useState('');
  const [city, setCity] = useState('');
  const [stateVal, setStateVal] = useState('');
  const [postalCode, setPostalCode] = useState('');
  const [country, setCountry] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [website, setWebsite] = useState('');

  // Invoicing
  const [taxNumber, setTaxNumber] = useState('');
  const [vatNumber, setVatNumber] = useState('');
  const [invoicePrefix, setInvoicePrefix] = useState('');
  const [invoiceFooter, setInvoiceFooter] = useState('');
  const [defaultTaxRate, setDefaultTaxRate] = useState('');
  const [currency, setCurrency] = useState('');

  // Banking
  const [bankName, setBankName] = useState('');
  const [iban, setIban] = useState('');
  const [bic, setBic] = useState('');
  const [accountHolder, setAccountHolder] = useState('');

  // Company details
  const [ceoName, setCeoName] = useState('');
  const [registerCourt, setRegisterCourt] = useState('');
  const [registerNumber, setRegisterNumber] = useState('');

  // Branding
  const [brandPrimaryColor, setBrandPrimaryColor] = useState('');
  const [brandAccentColor, setBrandAccentColor] = useState('');
  const [brandDarkMode, setBrandDarkMode] = useState(false);
  const [brandLogoUrl, setBrandLogoUrl] = useState('');

  // SMTP
  const [smtpHost, setSmtpHost] = useState('');
  const [smtpPort, setSmtpPort] = useState('');
  const [smtpUsername, setSmtpUsername] = useState('');
  const [smtpFromEmail, setSmtpFromEmail] = useState('');
  const [smtpFromName, setSmtpFromName] = useState('');
  const [smtpUseTLS, setSmtpUseTLS] = useState(true);

  useEffect(() => {
    load();
  }, []);

  const load = async () => {
    try {
      const { data } = await adminSettingsApi.getCompany();
      setName(data.name || '');
      setAddressLine1(data.address_line1 || '');
      setAddressLine2(data.address_line2 || '');
      setCity(data.city || '');
      setStateVal(data.state || '');
      setPostalCode(data.postal_code || '');
      setCountry(data.country || '');
      setEmail(data.email || '');
      setPhone(data.phone || '');
      setWebsite(data.website || '');

      setTaxNumber(data.tax_number || '');
      setVatNumber(data.vat_number || '');
      setInvoicePrefix(data.invoice_prefix || '');
      setInvoiceFooter(data.invoice_footer || '');
      setDefaultTaxRate(data.default_tax_rate ? String(data.default_tax_rate) : '');
      setCurrency(data.currency || '');

      setBankName(data.bank_name || '');
      setIban(data.iban || '');
      setBic(data.bic || '');
      setAccountHolder(data.account_holder || '');

      setCeoName(data.ceo_name || '');
      setRegisterCourt(data.register_court || '');
      setRegisterNumber(data.register_number || '');

      setBrandPrimaryColor(data.brand_primary_color || '');
      setBrandAccentColor(data.brand_accent_color || '');
      setBrandDarkMode(Boolean(data.brand_dark_mode));
      setBrandLogoUrl(data.brand_logo_url || '');

      setSmtpHost(data.smtp_host || '');
      setSmtpPort(data.smtp_port ? String(data.smtp_port) : '');
      setSmtpUsername(data.smtp_username || '');
      setSmtpFromEmail(data.smtp_from_email || '');
      setSmtpFromName(data.smtp_from_name || '');
      setSmtpUseTLS(data.smtp_use_tls === undefined ? true : Boolean(data.smtp_use_tls));
    } catch (err) {
      console.error('Failed to load company settings', err);
      setMessage({ type: 'error', text: t('admin.companySettings.loadError') });
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!name.trim()) {
      setMessage({ type: 'error', text: t('admin.companySettings.nameRequired') });
      return;
    }
    setSaving(true);
    setMessage(null);
    try {
      const payload = {
        name: name.trim(),
        address_line1: addressLine1.trim(),
        address_line2: addressLine2.trim(),
        city: city.trim(),
        state: stateVal.trim(),
        postal_code: postalCode.trim(),
        country: country.trim(),
        email: email.trim(),
        phone: phone.trim(),
        website: website.trim(),

        tax_number: taxNumber.trim(),
        vat_number: vatNumber.trim(),
        invoice_prefix: invoicePrefix.trim(),
        invoice_footer: invoiceFooter.trim(),
        default_tax_rate: defaultTaxRate ? Number(defaultTaxRate) : undefined,
        currency: currency.trim(),

        bank_name: bankName.trim(),
        iban: iban.trim(),
        bic: bic.trim(),
        account_holder: accountHolder.trim(),

        ceo_name: ceoName.trim(),
        register_court: registerCourt.trim(),
        register_number: registerNumber.trim(),

        brand_primary_color: brandPrimaryColor.trim(),
        brand_accent_color: brandAccentColor.trim(),
        brand_dark_mode: brandDarkMode,
        brand_logo_url: brandLogoUrl.trim(),

        smtp_host: smtpHost.trim(),
        smtp_port: smtpPort ? Number(smtpPort) : undefined,
        smtp_username: smtpUsername.trim(),
        smtp_from_email: smtpFromEmail.trim(),
        smtp_from_name: smtpFromName.trim(),
        smtp_use_tls: smtpUseTLS,
      };
      await adminSettingsApi.updateCompany(payload);
      setMessage({ type: 'success', text: t('admin.companySettings.settingsSaved') });
    } catch (err) {
      console.error('Failed to save company settings', err);
      setMessage({ type: 'error', text: t('admin.companySettings.saveError') });
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
        <Building className="w-6 h-6 text-accent-red" />
        <div>
          <h2 className="text-2xl font-bold text-white">{t('admin.companySettings.title')}</h2>
          <p className="text-gray-400 text-sm">{t('admin.companySettings.subtitle')}</p>
        </div>
      </div>

      {message && (
        <div
          className={`p-4 rounded-lg flex items-center gap-3 ${
            message.type === 'success' ? 'bg-green-500/10 border border-green-500/20' : 'bg-red-500/10 border border-red-500/20'
          }`}
        >
          <AlertCircle className={`w-5 h-5 ${message.type === 'success' ? 'text-green-500' : 'text-red-500'}`} />
          <span className={message.type === 'success' ? 'text-green-500' : 'text-red-500'}>{message.text}</span>
        </div>
      )}

      <div className="space-y-6">
        <div className="bg-white/5 rounded-xl p-6 border border-white/10 space-y-4">
          <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.name')}</label>
          <input value={name} onChange={e => setName(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />

          <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.addressLine1')}</label>
          <input value={addressLine1} onChange={e => setAddressLine1(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />

          <label className="block text-sm font-medium text-gray-300 mt-3">{t('admin.companySettings.addressLine2')}</label>
          <input value={addressLine2} onChange={e => setAddressLine2(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />

          <div className="grid grid-cols-1 sm:grid-cols-4 gap-4 mt-3">
            <div>
              <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.city')}</label>
              <input value={city} onChange={e => setCity(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.state')}</label>
              <input value={stateVal} onChange={e => setStateVal(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.postalCode')}</label>
              <input value={postalCode} onChange={e => setPostalCode(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.country')}</label>
              <input value={country} onChange={e => setCountry(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <div>
              <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.email')}</label>
              <input value={email} onChange={e => setEmail(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-300">{t('admin.companySettings.phone')}</label>
              <input value={phone} onChange={e => setPhone(e.target.value)} className="w-full px-4 py-3 bg-dark-light border border-white/20 rounded-lg text-white" />
            </div>
          </div>
        </div>
      </div>

      <div className="flex justify-end pt-4">
        <button onClick={handleSave} disabled={saving} className="flex items-center gap-2 px-6 py-3 bg-accent-red hover:bg-accent-red/80 disabled:bg-gray-600 disabled:cursor-not-allowed text-white font-semibold rounded-lg">
          <Save className="w-5 h-5" />
          {saving ? t('admin.companySettings.saving') : t('admin.companySettings.saveSettings')}
        </button>
      </div>
    </div>
  );
}
