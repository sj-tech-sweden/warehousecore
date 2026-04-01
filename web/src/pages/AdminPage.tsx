import { useState } from 'react';
import { Settings, Users, Layers, Lightbulb, Cpu, FolderTree, Database, Ruler, KeyRound, Tag, Download, Box } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ZoneTypesTab } from '../components/admin/ZoneTypesTab';
import { LEDSettingsTab } from '../components/admin/LEDSettingsTab';
import { RolesTab } from '../components/admin/RolesTab';
import { LEDControllersTab } from '../components/admin/LEDControllersTab';
import { CategoriesTab } from '../components/admin/CategoriesTab';
import { APISettingsTab } from '../components/admin/APISettingsTab';
import { CountTypesTab } from '../components/admin/CountTypesTab';
import { APIKeysTab } from '../components/admin/APIKeysTab';
import { BrandsManufacturersTab } from '../components/admin/BrandsManufacturersTab';
import { ExportTab } from '../components/admin/ExportTab';
import { DevicesTab } from '../components/admin/DevicesTab';

type TabType = 'zonetypes' | 'led' | 'controllers' | 'categories' | 'brands' | 'counttypes' | 'roles' | 'apisettings' | 'apikeys' | 'export' | 'devices';

export function AdminPage() {
  const [activeTab, setActiveTab] = useState<TabType>('zonetypes');
  const { t } = useTranslation();

  const tabs = [
    { id: 'zonetypes' as TabType, label: t('admin.tabs.zoneTypes'), icon: Layers },
    { id: 'led' as TabType, label: t('admin.tabs.led'), icon: Lightbulb },
    { id: 'controllers' as TabType, label: t('admin.tabs.controllers'), icon: Cpu },
    { id: 'categories' as TabType, label: t('admin.tabs.categories'), icon: FolderTree },
    { id: 'brands' as TabType, label: t('admin.tabs.brands'), icon: Tag },
    { id: 'counttypes' as TabType, label: t('admin.tabs.countTypes'), icon: Ruler },
    { id: 'roles' as TabType, label: t('admin.tabs.roles'), icon: Users },
    { id: 'devices' as TabType, label: t('admin.tabs.devices'), icon: Box },
    { id: 'apisettings' as TabType, label: t('admin.tabs.apiSettings'), icon: Database },
    { id: 'apikeys' as TabType, label: t('admin.tabs.apiKeys'), icon: KeyRound },
    { id: 'export' as TabType, label: t('admin.tabs.export'), icon: Download },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Settings className="w-8 h-8 text-accent-red" />
        <div>
          <h1 className="text-3xl font-bold text-white">{t('admin.title')}</h1>
          <p className="text-gray-400">{t('admin.subtitle')}</p>
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
        {activeTab === 'brands' && <BrandsManufacturersTab />}
        {activeTab === 'counttypes' && <CountTypesTab />}
        {activeTab === 'roles' && <RolesTab />}
        {activeTab === 'devices' && <DevicesTab />}
        {activeTab === 'apisettings' && <APISettingsTab />}
        {activeTab === 'apikeys' && <APIKeysTab />}
        {activeTab === 'export' && <ExportTab />}
      </div>
    </div>
  );
}
