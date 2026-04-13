import { useState, useEffect } from 'react';
import { User, Mail, Shield, Save } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { api } from '../lib/api';

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
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [displayName, setDisplayName] = useState('');
  const [avatarURL, setAvatarURL] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    loadProfile();
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
      </div>
    </div>
  );
}
