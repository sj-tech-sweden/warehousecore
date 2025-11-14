import { useState } from 'react';
import { Package, Box } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ProductsTab } from '../components/admin/ProductsTab';
import { ProductPackagesTab } from '../components/admin/ProductPackagesTab';

type TabType = 'products' | 'packages';

export function ProductsPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<TabType>('products');

  const tabs = [
    { id: 'products' as TabType, label: 'Produkte', icon: Package },
    { id: 'packages' as TabType, label: 'Produktpakete', icon: Box },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Package className="w-8 h-8 text-accent-red" />
        <div>
          <h1 className="text-3xl font-bold text-white">{t('productManagement.productsTitle')}</h1>
          <p className="text-gray-400">{t('productManagement.productsSubtitle')}</p>
        </div>
      </div>

      {/* Tabs */}
      <div className="glass-dark rounded-2xl p-2">
        <div className="flex gap-2">
          {tabs.map(tab => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex items-center justify-center gap-2 px-6 py-3 rounded-xl font-semibold transition-all ${
                  activeTab === tab.id
                    ? 'bg-accent-red text-white shadow-lg'
                    : 'text-gray-400 hover:bg-white/5 hover:text-white'
                }`}
              >
                <Icon className="w-5 h-5" />
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Tab Content */}
      <div className="glass-dark rounded-2xl p-6">
        {activeTab === 'products' && <ProductsTab />}
        {activeTab === 'packages' && <ProductPackagesTab />}
      </div>
    </div>
  );
}
