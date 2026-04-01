import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';

interface RoleGuardProps {
  requiredRoles: string[];
  children: ReactNode;
}

export function RoleGuard({ requiredRoles, children }: RoleGuardProps) {
  const { user, loading } = useAuth();
  const { t } = useTranslation();

  if (loading) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center text-gray-400">
        {t('roleGuard.loading')}
      </div>
    );
  }

  const roles = (user?.Roles ?? user?.roles ?? []) as any[];
  const hasRequiredRole = roles.some((role) => {
    const name = (role?.name || role?.Name || '').toString().toLowerCase();
    return requiredRoles.some((required) => required.toLowerCase() === name);
  });

  if (!hasRequiredRole) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center text-center space-y-3">
        <div className="text-5xl font-bold text-accent-red">403</div>
        <p className="text-lg text-gray-300 font-semibold">{t('roleGuard.accessDenied')}</p>
        <p className="text-sm text-gray-500 max-w-md">
          {t('roleGuard.accessDeniedDescription')}
        </p>
      </div>
    );
  }

  return <>{children}</>;
}
