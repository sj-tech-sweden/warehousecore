import { useState } from 'react';
import { Settings, Users, Layers, Lightbulb, Cpu, FolderTree, Database, Ruler, KeyRound } from 'lucide-react';
import { ZoneTypesTab } from '../components/admin/ZoneTypesTab';
import { LEDSettingsTab } from '../components/admin/LEDSettingsTab';
import { RolesTab } from '../components/admin/RolesTab';
import { LEDControllersTab } from '../components/admin/LEDControllersTab';
import { CategoriesTab } from '../components/admin/CategoriesTab';
import { APISettingsTab } from '../components/admin/APISettingsTab';
import { CountTypesTab } from '../components/admin/CountTypesTab';
import { APIKeysTab } from '../components/admin/APIKeysTab';

type TabType = 'zonetypes' | 'led' | 'controllers' | 'categories' | 'counttypes' | 'roles' | 'apisettings' | 'apikeys';

export function AdminPage() {
  const [activeTab, setActiveTab] = useState<TabType>('zonetypes');

  const tabs = [
    { id: 'zonetypes' as TabType, label: 'Lagertypen', icon: Layers },
    { id: 'led' as TabType, label: 'LED-Verhalten', icon: Lightbulb },
    { id: 'controllers' as TabType, label: 'ESP-Controller', icon: Cpu },
    { id: 'categories' as TabType, label: 'Kategorien', icon: FolderTree },
    { id: 'counttypes' as TabType, label: 'Mess-Einheiten', icon: Ruler },
    { id: 'roles' as TabType, label: 'Rollen & Benutzer', icon: Users },
    { id: 'apisettings' as TabType, label: 'API-Einstellungen', icon: Database },
    { id: 'apikeys' as TabType, label: 'API-Keys', icon: KeyRound },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Settings className="w-8 h-8 text-accent-red" />
        <div>
          <h1 className="text-3xl font-bold text-white">Admin-Dashboard</h1>
          <p className="text-gray-400">Systemeinstellungen verwalten</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="glass-dark rounded-2xl p-2">
        <div className="flex gap-2 overflow-x-auto scrollbar-thin">
          {tabs.map(tab => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center justify-center gap-2 px-4 sm:px-6 py-3 rounded-xl font-semibold transition-all whitespace-nowrap flex-shrink-0 ${
                  activeTab === tab.id
                    ? 'bg-accent-red text-white shadow-lg'
                    : 'text-gray-400 hover:bg-white/5 hover:text-white'
                }`}
              >
                <Icon className="w-5 h-5" />
                <span className="hidden sm:inline">{tab.label}</span>
                <span className="sm:hidden text-xs">{tab.label.split(' ')[0]}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab Content */}
      <div className="glass-dark rounded-2xl p-6">
        {activeTab === 'zonetypes' && <ZoneTypesTab />}
        {activeTab === 'led' && <LEDSettingsTab />}
        {activeTab === 'controllers' && <LEDControllersTab />}
        {activeTab === 'categories' && <CategoriesTab />}
        {activeTab === 'counttypes' && <CountTypesTab />}
        {activeTab === 'roles' && <RolesTab />}
        {activeTab === 'apisettings' && <APISettingsTab />}
        {activeTab === 'apikeys' && <APIKeysTab />}
      </div>
    </div>
  );
}
