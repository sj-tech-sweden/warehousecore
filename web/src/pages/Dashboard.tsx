import { useEffect, useState } from 'react';
import { Package, Warehouse, AlertTriangle, TrendingUp } from 'lucide-react';
import { dashboardApi } from '../lib/api';
import type { DashboardStats, Movement } from '../lib/api';

export function Dashboard() {
  const [stats, setStats] = useState<DashboardStats>({
    in_storage: 0,
    on_job: 0,
    defective: 0,
    total: 0,
  });
  const [recentActivity, setRecentActivity] = useState<Movement[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    void loadData();
    const interval = setInterval(() => {
      void loadData();
    }, 10000); // Refresh every 10s
    return () => clearInterval(interval);
  }, []);

  const loadData = async () => {
    try {
      const { data } = await dashboardApi.getStats();
      setStats(data);
    } catch (error) {
      console.error('Failed to load stats:', error);
    }

    try {
      const { data } = await dashboardApi.getRecentMovements(10);
      setRecentActivity(data);
    } catch (error) {
      console.error('Failed to load recent activity:', error);
    }

    if (loading) {
      setLoading(false);
    }
  };

  const formatRelativeTime = (isoTimestamp: string): string => {
    const date = new Date(isoTimestamp);
    if (Number.isNaN(date.getTime())) {
      return '';
    }

    const diffMs = Date.now() - date.getTime();
    if (diffMs <= 0) {
      return 'gerade eben';
    }

    const diffSeconds = Math.floor(diffMs / 1000);
    if (diffSeconds < 60) {
      return 'vor wenigen Sekunden';
    }

    const diffMinutes = Math.floor(diffSeconds / 60);
    if (diffMinutes < 60) {
      return `vor ${diffMinutes} ${diffMinutes === 1 ? 'Minute' : 'Minuten'}`;
    }

    const diffHours = Math.floor(diffMinutes / 60);
    if (diffHours < 24) {
      return `vor ${diffHours} ${diffHours === 1 ? 'Stunde' : 'Stunden'}`;
    }

    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) {
      return `vor ${diffDays} ${diffDays === 1 ? 'Tag' : 'Tagen'}`;
    }

    const diffWeeks = Math.floor(diffDays / 7);
    if (diffWeeks < 5) {
      return `vor ${diffWeeks} ${diffWeeks === 1 ? 'Woche' : 'Wochen'}`;
    }

    const diffMonths = Math.floor(diffDays / 30);
    if (diffMonths < 12) {
      return `vor ${diffMonths} ${diffMonths === 1 ? 'Monat' : 'Monaten'}`;
    }

    const diffYears = Math.floor(diffDays / 365);
    return `vor ${diffYears} ${diffYears === 1 ? 'Jahr' : 'Jahren'}`;
  };

  const describeMovement = (movement: Movement): string => {
    const deviceLabel =
      movement.product_name ??
      movement.serial_number ??
      movement.device_id;

    switch (movement.action) {
      case 'intake':
        return movement.to_zone_name
          ? `${deviceLabel} in ${movement.to_zone_name} eingecheckt`
          : `${deviceLabel} ins Lager eingecheckt`;
      case 'outtake':
        return movement.to_job_description
          ? `${deviceLabel} für ${movement.to_job_description} ausgebucht`
          : `${deviceLabel} aus dem Lager ausgebucht`;
      case 'transfer':
        if (movement.from_zone_name && movement.to_zone_name) {
          return `${deviceLabel} von ${movement.from_zone_name} nach ${movement.to_zone_name} verschoben`;
        }
        if (movement.to_zone_name) {
          return `${deviceLabel} nach ${movement.to_zone_name} verschoben`;
        }
        if (movement.from_zone_name) {
          return `${deviceLabel} aus ${movement.from_zone_name} entnommen`;
        }
        return `${deviceLabel} verschoben`;
      case 'return':
        return `${deviceLabel} zurückgebucht`;
      case 'move':
        return `${deviceLabel} bewegt`;
      default:
        return `${deviceLabel} (${movement.action})`;
    }
  };

  const activityItems = recentActivity.slice(0, 5);

  const statCards = [
    {
      title: 'Im Lager',
      value: stats.in_storage,
      icon: Warehouse,
      color: 'from-gray-600 to-gray-800',
      textColor: 'text-gray-300',
    },
    {
      title: 'Auf Job',
      value: stats.on_job,
      icon: Package,
      color: 'from-accent-red to-red-700',
      textColor: 'text-accent-red',
    },
    {
      title: 'Defekt',
      value: stats.defective,
      icon: AlertTriangle,
      color: 'from-yellow-600 to-yellow-800',
      textColor: 'text-yellow-500',
    },
    {
      title: 'Gesamt',
      value: stats.total,
      icon: TrendingUp,
      color: 'from-blue-600 to-blue-800',
      textColor: 'text-blue-400',
    },
  ];

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-accent-red"></div>
      </div>
    );
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <div>
        <h2 className="text-2xl sm:text-3xl font-bold text-white mb-1 sm:mb-2">Dashboard</h2>
        <p className="text-sm sm:text-base text-gray-400">Lagerübersicht und Statistiken</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-6">
        {statCards.map((card) => {
          const Icon = card.icon;
          return (
            <div
              key={card.title}
              className="glass rounded-xl sm:rounded-2xl p-4 sm:p-6 hover:bg-white/20 transition-all duration-300 group"
            >
              <div className="flex items-center justify-between mb-3 sm:mb-4">
                <div className={`p-2 sm:p-3 rounded-lg sm:rounded-xl bg-gradient-to-br ${card.color} bg-opacity-20`}>
                  <Icon className={`w-5 h-5 sm:w-6 sm:h-6 ${card.textColor}`} />
                </div>
                <div className="text-xs sm:text-sm text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity">
                  Live
                </div>
              </div>
              <div className="space-y-1">
                <p className="text-gray-400 text-xs sm:text-sm font-medium">{card.title}</p>
                <p className={`text-3xl sm:text-4xl font-bold ${card.textColor}`}>{card.value}</p>
              </div>
            </div>
          );
        })}
      </div>

      {/* Recent Activity */}
      <div className="glass-dark rounded-xl sm:rounded-2xl p-4 sm:p-6">
        <h3 className="text-lg sm:text-xl font-bold text-white mb-3 sm:mb-4">Letzte Aktivität</h3>
        {activityItems.length === 0 ? (
          <div className="text-sm sm:text-base text-gray-400">Noch keine Aktivitäten erfasst.</div>
        ) : (
          <div className="space-y-2 sm:space-y-3">
            {activityItems.map((activity) => (
              <div
                key={activity.movement_id}
                className="flex items-center gap-3 sm:gap-4 p-3 sm:p-4 glass rounded-lg sm:rounded-xl hover:bg-white/10 transition-colors"
              >
                <div className="w-2 h-2 rounded-full bg-accent-red animate-pulse flex-shrink-0"></div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm sm:text-base text-white font-medium truncate">
                    {describeMovement(activity)}
                  </p>
                  <div className="flex flex-wrap items-center gap-2 text-xs sm:text-sm text-gray-400">
                    <span>{formatRelativeTime(activity.timestamp) || 'gerade eben'}</span>
                    {activity.performed_by && (
                      <>
                        <span className="hidden sm:inline text-gray-600">•</span>
                        <span className="truncate">{activity.performed_by}</span>
                      </>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
