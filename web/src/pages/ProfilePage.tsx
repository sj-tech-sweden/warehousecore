import { useState, useEffect, useRef } from 'react';
import { User, Mail, Shield, Save, Key } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { api } from '../lib/api';
import { useAuth } from '../contexts/AuthContext';

interface UserProfile {
  profile: {
    id: number;
    user_id: number;
    display_name: string;
    avatar_url: string;
    prefs: Record<string, unknown>;
    user?: {
      userID: number;
      username: string;
      email: string;
      first_name: string;
      last_name: string;
    }
  };
  roles: Array<{
    id: number;
    name: string;
    description: string;
  }>;
}

export function ProfilePage() {
  const { t } = useTranslation();
  const { changePassword } = useAuth();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [displayName, setDisplayName] = useState('');
  const [avatarURL, setAvatarURL] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  const [pwCurrent, setPwCurrent] = useState('');
  const [pwNew, setPwNew] = useState('');
  const [pwConfirm, setPwConfirm] = useState('');
  const [pwSaving, setPwSaving] = useState(false);
  const [pwMessage, setPwMessage] = useState('');
  const [pwMessageIsSuccess, setPwMessageIsSuccess] = useState<boolean | null>(null);
  const pwMessageTimeoutRef = useRef<number | null>(null);

  useEffect(() => {
    loadProfile();
  }, []);

  useEffect(() => {
    return () => {
      if (pwMessageTimeoutRef.current !== null) {
        window.clearTimeout(pwMessageTimeoutRef.current);
      }
    };
  }, []);

  const loadProfile = async () => {
    try {
      const response = await api.get('/profile/me');
      setProfile(response.data);
      setDisplayName(response.data.profile.display_name || '');
      setAvatarURL(response.data.profile.avatar_url || '');
    } catch (error) {
      console.error('Failed to load profile:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    setSaving(true);
    setMessage('');

    try {
      await api.put('/profile/me', {
        display_name: displayName,
        avatar_url: avatarURL,
        prefs: profile?.profile.prefs || {},
      });

      setMessage(t('profilePage.saveSuccess'));
      setTimeout(() => setMessage(''), 3000);
      loadProfile();
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      setMessage(t('profilePage.saveError', { error: errorMessage }));
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async () => {
    setPwSaving(true);
    setPwMessage('');
    setPwMessageIsSuccess(null);
    try {
      if (pwNew !== pwConfirm) throw new Error(t('profilePage.passwordMismatchError'));
      if (pwNew.length < 6) throw new Error(t('profilePage.passwordMinLengthError'));
      if (!pwCurrent.trim()) throw new Error(t('profilePage.currentPasswordRequiredError'));
      if (pwNew === pwCurrent) throw new Error(t('profilePage.passwordSameAsCurrentError'));
      await changePassword(pwCurrent, pwNew);
      setPwMessage(t('profilePage.passwordChangeSuccess'));
      setPwMessageIsSuccess(true);
      setPwCurrent(''); setPwNew(''); setPwConfirm('');
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('profilePage.passwordChangeFailedError');
      setPwMessage(msg);
      setPwMessageIsSuccess(false);
    } finally {
      setPwSaving(false);
      if (pwMessageTimeoutRef.current !== null) {
        window.clearTimeout(pwMessageTimeoutRef.current);
      }
      pwMessageTimeoutRef.current = window.setTimeout(() => {
        setPwMessage('');
        setPwMessageIsSuccess(null);
        pwMessageTimeoutRef.current = null;
      }, 3000);
    }
  };

  if (loading) {
    return <div className="text-white">{t('common.loading')}</div>;
  }

  return (
    <div className="space-y-6 max-w-3xl">
      {/* Header */}
      <div className="flex items-center gap-3">
        <User className="w-8 h-8 text-accent-red" />
        <div>
          <h1 className="text-3xl font-bold text-white">{t('profilePage.title')}</h1>
          <p className="text-gray-400">{t('profilePage.subtitle')}</p>
        </div>
      </div>

      {/* Profile Card */}
      <div className="glass-dark rounded-2xl p-6 space-y-6">
        {/* User Info */}
        <div className="flex items-center gap-4 pb-6 border-b border-white/10">
          <div className="w-20 h-20 rounded-full bg-accent-red/20 flex items-center justify-center">
            <User className="w-10 h-10 text-accent-red" />
          </div>
          <div>
            <h2 className="text-2xl font-bold text-white">
              {profile?.profile.user?.first_name} {profile?.profile.user?.last_name}
            </h2>
            <p className="text-gray-400 flex items-center gap-2">
              <Mail className="w-4 h-4" />
              {profile?.profile.user?.email}
            </p>
          </div>
        </div>

        {/* Roles */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-2 flex items-center gap-2">
            <Shield className="w-4 h-4" />
            {t('profilePage.roles')}
          </label>
          <div className="flex flex-wrap gap-2">
            {profile?.roles.map(role => (
              <span
                key={role.id}
                className="px-3 py-1 rounded-full bg-accent-red/20 text-accent-red text-sm font-semibold"
              >
                {role.name}
              </span>
            ))}
          </div>
        </div>

        {/* Display Name */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-2">
            {t('profilePage.displayName')}
          </label>
          <input
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder={t('profilePage.displayNamePlaceholder')}
            className="w-full px-4 py-3 rounded-xl glass text-white placeholder-gray-500 focus:ring-2 focus:ring-accent-red outline-none"
          />
        </div>

        {/* Avatar URL */}
        <div>
          <label className="block text-sm font-semibold text-gray-400 mb-2">
            {t('profilePage.avatarUrl')}
          </label>
          <input
            type="url"
            value={avatarURL}
            onChange={(e) => setAvatarURL(e.target.value)}
            placeholder={t('profilePage.avatarUrlPlaceholder')}
            className="w-full px-4 py-3 rounded-xl glass text-white placeholder-gray-500 focus:ring-2 focus:ring-accent-red outline-none"
          />
        </div>

        {/* Save Button */}
        <div className="pt-4 border-t border-white/10">
          <button
            onClick={handleSave}
            disabled={saving}
            className={`w-full py-3 px-6 rounded-xl font-semibold text-white transition-all flex items-center justify-center gap-2 ${
              saving
                ? 'bg-gray-600 cursor-not-allowed'
                : 'bg-gradient-to-r from-accent-red to-red-700 hover:shadow-lg hover:shadow-red-500/50 hover:scale-105 active:scale-95'
            }`}
          >
            <Save className="w-5 h-5" />
            <span>{saving ? t('common.saving') : t('common.save')}</span>
          </button>

          {message && (
            <div className={`mt-3 p-3 rounded-lg text-center text-sm font-semibold ${
              message === t('profilePage.saveSuccess')
                ? 'bg-green-500/20 text-green-400'
                : 'bg-red-500/20 text-red-400'
            }`}>
              {message}
            </div>
          )}
        </div>

          {/* Change Password */}
          <div className="glass-dark rounded-2xl p-6 space-y-4 mt-6">
            <h3 className="text-lg font-semibold text-white flex items-center gap-2"><Key className="w-5 h-5 text-accent-red" />{t('profilePage.changePasswordTitle')}</h3>
            <p className="text-gray-400 text-sm">{t('profilePage.changePasswordHelp')}</p>

            <div>
              <label htmlFor="profile-current-password" className="block text-sm font-semibold text-gray-400 mb-2">{t('profilePage.currentPassword')}</label>
              <input id="profile-current-password" type="password" autoComplete="current-password" required value={pwCurrent} onChange={(e)=>setPwCurrent(e.target.value)} className="w-full px-4 py-3 rounded-xl glass text-white" />
            </div>

            <div>
              <label htmlFor="profile-new-password" className="block text-sm font-semibold text-gray-400 mb-2">{t('profilePage.newPassword')}</label>
              <input id="profile-new-password" type="password" autoComplete="new-password" required value={pwNew} onChange={(e)=>setPwNew(e.target.value)} className="w-full px-4 py-3 rounded-xl glass text-white" />
            </div>

            <div>
              <label htmlFor="profile-confirm-password" className="block text-sm font-semibold text-gray-400 mb-2">{t('profilePage.confirmPassword')}</label>
              <input id="profile-confirm-password" type="password" autoComplete="new-password" required value={pwConfirm} onChange={(e)=>setPwConfirm(e.target.value)} className="w-full px-4 py-3 rounded-xl glass text-white" />
            </div>

            <div className="pt-2">
              <button onClick={handleChangePassword} disabled={pwSaving} className={`w-full py-3 px-6 rounded-xl font-semibold text-white ${pwSaving ? 'bg-gray-600' : 'bg-gradient-to-r from-accent-red to-red-700'}`}>
                {pwSaving ? t('common.saving') : t('profilePage.changePasswordButton')}
              </button>

              {pwMessage && (
                <div className={`mt-3 p-3 rounded-lg text-center text-sm font-semibold ${pwMessageIsSuccess ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'}`}>
                  {pwMessage}
                </div>
              )}
            </div>
          </div>
      </div>
    </div>
  );
}
