import type { ReactNode } from 'react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Home, Package, MapPin, ScanLine, Wrench, Menu, Briefcase, X, LogOut, User } from 'lucide-react';
import { useState, useEffect } from 'react';
import { useAuth } from '../contexts/AuthContext';

interface LayoutProps {
  children: ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const checkMobile = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (!mobile) {
        setSidebarOpen(true);
      } else {
        setSidebarOpen(false);
      }
    };

    checkMobile();
    window.addEventListener('resize', checkMobile);
    return () => window.removeEventListener('resize', checkMobile);
  }, []);

  const closeSidebar = () => {
    if (isMobile) {
      setSidebarOpen(false);
    }
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  // Get cross-navigation domain from environment variable injected by backend
  // If not set, auto-detect based on current hostname
  const getRentalCoreURL = () => {
    const rentalDomain = (window as any).__APP_CONFIG__?.rentalCoreDomain;
    const protocol = window.location.protocol; // Use same protocol as current page (http/https)

    if (rentalDomain && rentalDomain !== '') {
      // Use configured domain from environment (no port, handled by reverse proxy)
      return `${protocol}//${rentalDomain}`;
    }

    // Auto-detect based on current hostname
    const hostname = window.location.hostname;
    const port = window.location.port;

    // Check if we're on a subdomain setup (e.g., storage.server-nt.de)
    if (hostname.startsWith('storage.')) {
      // Replace 'storage' with 'rent'
      const rentalHost = hostname.replace(/^storage\./, 'rent.');
      return `${protocol}//${rentalHost}`;
    } else if (port === '8082') {
      // Running on port 8082 -> go to 8081 on same host
      return `${protocol}//${hostname}:8081`;
    } else if (port === '') {
      // No port specified (reverse proxy) - keep same host
      return `${protocol}//${hostname}`;
    } else {
      // Default fallback - try :8081 on same host
      return `${protocol}//${hostname}:8081`;
    }
  };

  const rentalCoreURL = getRentalCoreURL();

  // Debug log
  console.log('RentalCore URL:', rentalCoreURL);

  const navItems = [
    { path: '/', icon: Home, label: 'Dashboard' },
    { path: '/scan', icon: ScanLine, label: 'Scan' },
    { path: '/devices', icon: Package, label: 'Geräte' },
    { path: '/zones', icon: MapPin, label: 'Zonen' },
    { path: '/jobs', icon: Briefcase, label: 'Jobs' },
    { path: '/maintenance', icon: Wrench, label: 'Wartung' },
  ];

  return (
    <div className="min-h-screen bg-dark">
      {/* Header */}
      <header className={`fixed top-0 right-0 z-50 glass-dark border-b border-white/10 transition-all duration-300 ${
        !isMobile && sidebarOpen ? 'left-64' : 'left-0'
      }`}>
        <div className="flex items-center justify-between px-3 sm:px-6 py-3 sm:py-4">
          <div className="flex items-center gap-2 sm:gap-4">
            <button
              onClick={() => setSidebarOpen(!sidebarOpen)}
              className="p-2 hover:bg-white/10 rounded-lg transition-colors"
              aria-label="Toggle menu"
            >
              <Menu className="w-5 h-5 sm:w-6 sm:h-6" />
            </button>
            <h1 className="text-lg sm:text-2xl font-bold">
              <span className="text-accent-red">Storage</span>
              <span className="text-white">Core</span>
            </h1>
          </div>
          <div className="text-xs sm:text-sm text-gray-400 hidden sm:block">
            Tsunami Events UG
          </div>
        </div>
      </header>

      {/* Mobile Backdrop */}
      {isMobile && sidebarOpen && (
        <div
          className="fixed inset-0 bg-black/60 z-40 backdrop-blur-sm"
          onClick={closeSidebar}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed left-0 top-0 bottom-0 z-50 glass-dark border-r border-white/10 transition-transform duration-300 ease-in-out ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        } ${isMobile ? 'w-64' : 'w-64 md:translate-x-0'}`}
      >
        {/* Sidebar Header (Mobile only) */}
        <div className="flex items-center justify-between px-4 py-4 border-b border-white/10 md:hidden">
          <h2 className="text-lg font-bold">
            <span className="text-accent-red">Storage</span>
            <span className="text-white">Core</span>
          </h2>
          <button
            onClick={closeSidebar}
            className="p-2 hover:bg-white/10 rounded-lg transition-colors"
            aria-label="Close menu"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav className="p-4 space-y-2 mt-12 md:mt-4">
          {/* Cross-navigation to RentalCore */}
          <a
            href={rentalCoreURL}
            className="flex items-center gap-3 px-4 py-3 rounded-lg transition-all bg-accent-red/10 text-accent-red hover:bg-accent-red hover:text-white shadow-lg shadow-accent-red/10 border border-accent-red/20"
            title="Switch to RentalCore"
          >
            <Briefcase className="w-5 h-5" />
            <span className="font-semibold">RentalCore</span>
          </a>

          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.path;

            return (
              <Link
                key={item.path}
                to={item.path}
                onClick={closeSidebar}
                className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-all ${
                  isActive
                    ? 'bg-accent-red text-white shadow-lg shadow-accent-red/20'
                    : 'text-gray-400 hover:bg-white/10 hover:text-white'
                }`}
              >
                <Icon className="w-5 h-5" />
                <span className="font-medium">{item.label}</span>
              </Link>
            );
          })}

          {/* User Profile & Logout */}
          <div className="absolute bottom-0 left-0 right-0 p-4 border-t border-white/10 bg-dark/50">
            {user && (
              <div className="mb-3 px-4 py-2 rounded-lg bg-white/5">
                <div className="flex items-center gap-2 text-sm">
                  <User className="w-4 h-4 text-accent-red" />
                  <span className="text-gray-300 font-medium">{user.Username}</span>
                </div>
                {user.Email && (
                  <p className="text-xs text-gray-500 mt-1 ml-6">{user.Email}</p>
                )}
              </div>
            )}
            <button
              onClick={handleLogout}
              className="flex items-center gap-3 px-4 py-3 rounded-lg transition-all text-gray-400 hover:bg-red-500/10 hover:text-red-400 w-full"
            >
              <LogOut className="w-5 h-5" />
              <span className="font-medium">Abmelden</span>
            </button>
          </div>
        </nav>
      </aside>

      {/* Main Content */}
      <main
        className={`pt-14 sm:pt-16 transition-all duration-300 ${
          !isMobile && sidebarOpen ? 'md:ml-64' : 'ml-0'
        }`}
      >
        <div className="p-3 sm:p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
