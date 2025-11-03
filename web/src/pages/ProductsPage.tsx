import { Package } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ProductsTab } from '../components/admin/ProductsTab';

export function ProductsPage() {
  const { t } = useTranslation();

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Package className="w-8 h-8 text-accent-red" />
        <div>
          <h1 className="text-3xl font-bold text-white">{t('productManagement.productsTitle')}</h1>
          <p className="text-gray-400">{t('productManagement.productsSubtitle')}</p>
        </div>
      </div>

      <div className="glass-dark rounded-2xl p-6">
        <ProductsTab />
      </div>
    </div>
  );
}
