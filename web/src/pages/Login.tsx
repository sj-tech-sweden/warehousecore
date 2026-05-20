import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import { LogIn } from 'lucide-react';
import { fetchPublicCompanyName, getInitialCompanyName } from '../lib/publicConfig';

export function Login() {
  const { t } = useTranslation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const [companyName, setCompanyName] = useState<string>(getInitialCompanyName());

  useEffect(() => {
    fetchPublicCompanyName().then((name) => {
      if (name) {
        setCompanyName(name);
      }
    });
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(username, password);
      navigate('/'); // Redirect to dashboard after login
    } catch (err) {
      setError(err instanceof Error ? err.message : t('auth.loginError'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-dark flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Logo/Header */}
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold mb-2">
            <span className="text-accent-red">Warehouse</span>
            <span className="text-white">Core</span>
          </h1>
          <p className="text-gray-400">{companyName}</p>
        </div>

        {/* Login Card */}
        <div className="glass-dark p-8 rounded-xl border border-white/10">
          <div className="flex items-center gap-3 mb-6">
            <LogIn className="w-6 h-6 text-accent-red" />
            <h2 className="text-2xl font-bold text-white">{t('auth.login')}</h2>
          </div>

          {error && (
            <div className="mb-6 p-4 bg-red-500/10 border border-red-500/20 rounded-lg">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label htmlFor="username" className="block text-sm font-medium text-gray-300 mb-2">
                {t('auth.username')}
              </label>
              <input
                id="username"
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                disabled={loading}
                className="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-accent-red focus:border-transparent disabled:opacity-50"
                placeholder={t('login.usernamePlaceholder')}
                autoComplete="username"
              />
            </div>

            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-300 mb-2">
                {t('auth.password')}
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                disabled={loading}
                className="w-full px-4 py-3 bg-white/5 border border-white/10 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-accent-red focus:border-transparent disabled:opacity-50"
                placeholder={t('login.passwordPlaceholder')}
                autoComplete="current-password"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-accent-red hover:bg-accent-red/80 disabled:bg-accent-red/50 text-white font-semibold rounded-lg transition-colors flex items-center justify-center gap-2"
            >
              {loading ? (
                <>
                  <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                  {t('login.signing_in')}
                </>
              ) : (
                <>
                  <LogIn className="w-5 h-5" />
                  {t('auth.login')}
                </>
              )}
            </button>
          </form>

          <div className="mt-6 pt-6 border-t border-white/10">
            <p className="text-sm text-gray-400 text-center">
              {t('login.credentialsHint')}
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="mt-6 text-center">
          <p className="text-xs text-gray-500">
            WarehouseCore © 2025 {companyName}
          </p>
        </div>
      </div>
    </div>
  );
}
