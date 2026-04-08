import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Building2,
  Eye,
  LayoutGrid,
  List,
  Pencil,
  Plus,
  RefreshCcw,
  Search,
  Trash2,
  X,
} from 'lucide-react';
import { api } from '../../lib/api';
import { ModalPortal } from '../ModalPortal';
import { useTranslation } from 'react-i18next';
import { useCurrencySymbol } from '../../hooks/useCurrencySymbol';
import { formatDateTimeISO } from '../../lib/utils';

interface RentalEquipment {
  equipment_id: number;
  product_name: string;
  supplier_name: string;
  rental_price: number;
  customer_price: number;
  category?: string | null;
  description?: string | null;
  notes?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface RentalEquipmentFormData {
  product_name: string;
  supplier_name: string;
  rental_price: number;
  customer_price: number;
  category: string;
  description: string;
  notes: string;
  is_active: boolean;
}

const initialFormData: RentalEquipmentFormData = {
  product_name: '',
  supplier_name: '',
  rental_price: 0,
  customer_price: 0,
  category: '',
  description: '',
  notes: '',
  is_active: true,
};

const parseNumber = (value: string): number => {
  const trimmed = value.trim();
  if (trimmed === '') {
    return 0;
  }
  const parsed = Number(trimmed);
  return Number.isNaN(parsed) ? 0 : parsed;
};

function useDebouncedValue<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(handle);
  }, [value, delay]);

  return debounced;
}

export function RentedProductsTab() {
  const { t } = useTranslation();
  const currencySymbol = useCurrencySymbol();
  const [equipment, setEquipment] = useState<RentalEquipment[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewEquipment, setViewEquipment] = useState<RentalEquipment | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [formData, setFormData] = useState<RentalEquipmentFormData>(initialFormData);
  const [suppliers, setSuppliers] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'cards'>(() => {
    return typeof window !== 'undefined' && window.innerWidth < 768 ? 'cards' : 'table';
  });
  const [searchTerm, setSearchTerm] = useState('');
  const [supplierFilter, setSupplierFilter] = useState<string>('');
  const [refreshing, setRefreshing] = useState(false);
  const scrollPosition = useRef(0);

  const debouncedSearch = useDebouncedValue(searchTerm, 300);

  useEffect(() => {
    const handleResize = () => {
      const isMobile = window.innerWidth < 768;
      setViewMode(isMobile ? 'cards' : 'table');
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  useEffect(() => {
    const html = document.documentElement;
    const body = document.body;
    const anyModalOpen = modalOpen || !!viewEquipment;
    if (anyModalOpen) {
      scrollPosition.current = window.scrollY || window.pageYOffset || 0;
      html.classList.add('modal-open');
      body.classList.add('modal-open');
      body.style.position = 'fixed';
      body.style.top = `-${scrollPosition.current}px`;
      body.style.left = '0';
      body.style.right = '0';
      body.style.width = '100%';
      return () => {
        html.classList.remove('modal-open');
        body.classList.remove('modal-open');
        body.style.position = '';
        body.style.top = '';
        body.style.left = '';
        body.style.right = '';
        body.style.width = '';
        window.scrollTo(0, scrollPosition.current);
      };
    }

    return undefined;
  }, [modalOpen, viewEquipment]);

  const fetchEquipment = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (debouncedSearch.trim()) {
        params.search = debouncedSearch.trim();
      }
      if (supplierFilter) {
        params.supplier = supplierFilter;
      }
      const { data } = await api.get<RentalEquipment[]>('/admin/rental-equipment', { params });
      setEquipment(data || []);
    } catch (error) {
      console.error('Failed to load rental equipment:', error);
      setEquipment([]);
    } finally {
      setLoading(false);
    }
  }, [debouncedSearch, supplierFilter]);

  const fetchSuppliers = useCallback(async () => {
    try {
      const { data } = await api.get<string[]>('/admin/rental-equipment/suppliers');
      setSuppliers(data || []);
    } catch (error) {
      console.error('Failed to load suppliers:', error);
    }
  }, []);

  useEffect(() => {
    fetchEquipment();
  }, [fetchEquipment]);

  useEffect(() => {
    fetchSuppliers();
  }, [fetchSuppliers]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    await fetchEquipment();
    await fetchSuppliers();
    setRefreshing(false);
  }, [fetchEquipment, fetchSuppliers]);

  const clearFilters = () => {
    setSearchTerm('');
    setSupplierFilter('');
  };

  const closeModal = useCallback(() => {
    setModalOpen(false);
    setEditingId(null);
    setFormData(initialFormData);
  }, []);

  const closeDetailModal = () => {
    setViewEquipment(null);
  };

  const handleOpenCreateModal = () => {
    setFormData(initialFormData);
    setEditingId(null);
    setModalOpen(true);
  };

  const handleEditEquipment = (item: RentalEquipment) => {
    setFormData({
      product_name: item.product_name,
      supplier_name: item.supplier_name,
      rental_price: item.rental_price,
      customer_price: item.customer_price,
      category: item.category || '',
      description: item.description || '',
      notes: item.notes || '',
      is_active: item.is_active,
    });
    setEditingId(item.equipment_id);
    setModalOpen(true);
  };

  const handleDelete = async (id: number, name: string) => {
    if (!window.confirm(t('admin.rentedProducts.confirmDelete', { name }))) {
      return;
    }

    try {
      await api.delete(`/admin/rental-equipment/${id}`);
      await fetchEquipment();
      await fetchSuppliers();
    } catch (error) {
      console.error('Failed to delete rental equipment:', error);
      window.alert(t('admin.rentedProducts.errors.delete'));
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!formData.product_name.trim()) {
      window.alert(t('admin.rentedProducts.errors.productNameRequired'));
      return;
    }
    if (!formData.supplier_name.trim()) {
      window.alert(t('admin.rentedProducts.errors.supplierRequired'));
      return;
    }

    setSubmitting(true);

    const payload = {
      product_name: formData.product_name.trim(),
      supplier_name: formData.supplier_name.trim(),
      rental_price: formData.rental_price,
      customer_price: formData.customer_price,
      category: formData.category.trim() || null,
      description: formData.description.trim() || null,
      notes: formData.notes.trim() || null,
      is_active: formData.is_active,
    };

    try {
      if (editingId) {
        await api.put(`/admin/rental-equipment/${editingId}`, payload);
      } else {
        await api.post('/admin/rental-equipment', payload);
      }

      await fetchEquipment();
      await fetchSuppliers();
      closeModal();
    } catch (error) {
      console.error('Failed to save rental equipment:', error);
      window.alert(t('admin.rentedProducts.errors.save'));
    } finally {
      setSubmitting(false);
    }
  };

  const sortedEquipment = useMemo(() =>
    [...equipment].sort((a, b) => {
      const supplierCompare = a.supplier_name.localeCompare(b.supplier_name);
      if (supplierCompare !== 0) return supplierCompare;
      return a.product_name.localeCompare(b.product_name);
    }),
    [equipment]
  );

  const hasActiveFilters = debouncedSearch.trim() !== '' || supplierFilter !== '';

  const formatCurrency = (value: number) => `${value.toFixed(2)} ${currencySymbol}`;

  const profitMargin = (rental: number, customer: number) => {
    if (rental === 0) return 0;
    return ((customer - rental) / rental * 100);
  };

  const totalRentalCost = sortedEquipment.reduce((sum, e) => sum + e.rental_price, 0);
  const totalCustomerRevenue = sortedEquipment.reduce((sum, e) => sum + e.customer_price, 0);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        <div>
          <h2 className="text-xl font-bold text-white">{t('admin.rentedProducts.title')}</h2>
          <p className="text-sm text-gray-400">
            {t('admin.rentedProducts.loadedCount', { count: sortedEquipment.length })}
            {hasActiveFilters ? ` ${t('admin.rentedProducts.filtersActive')}` : ''}
          </p>
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={searchTerm}
                onChange={event => setSearchTerm(event.target.value)}
                placeholder={t('admin.rentedProducts.searchPlaceholder')}
                className="w-full rounded-lg bg-white/10 py-2 pl-9 pr-3 text-sm text-white placeholder-gray-500 outline-none transition focus:bg-white/15 focus:ring-1 focus:ring-accent-red"
                title={t('admin.rentedProducts.searchPlaceholder')}
              />
            </div>

            <select
              value={supplierFilter}
              onChange={event => setSupplierFilter(event.target.value)}
              className="rounded-lg bg-white/10 px-3 py-2 text-sm text-white outline-none transition focus:bg-white/15 focus:ring-1 focus:ring-accent-red"
              title={t('admin.rentedProducts.allSuppliers')}
            >
              <option value="">{t('admin.rentedProducts.allSuppliers')}</option>
              {suppliers.map(supplier => (
                <option key={supplier} value={supplier}>
                  {supplier}
                </option>
              ))}
            </select>

            {hasActiveFilters && (
              <button
                type="button"
                onClick={clearFilters}
                className="rounded-lg bg-white/10 px-3 py-2 text-sm text-white hover:bg-white/20 whitespace-nowrap"
              >
                {t('admin.rentedProducts.resetFilters')}
              </button>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setViewMode('table')}
              className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                viewMode === 'table'
                  ? 'bg-white/15 text-white'
                  : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white'
              }`}
            >
              <List className="h-4 w-4" />
              <span className="hidden sm:inline">{t('admin.devices.tableView')}</span>
            </button>
            <button
              type="button"
              onClick={() => setViewMode('cards')}
              className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                viewMode === 'cards'
                  ? 'bg-white/15 text-white'
                  : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white'
              }`}
            >
              <LayoutGrid className="h-4 w-4" />
              <span className="hidden sm:inline">{t('admin.devices.cardView')}</span>
            </button>
            <button
              type="button"
              onClick={handleRefresh}
              disabled={refreshing}
              className="flex items-center gap-2 rounded-lg bg-white/10 px-3 py-2 text-sm font-semibold text-white transition hover:bg-white/20 disabled:opacity-50"
            >
              <RefreshCcw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              <span className="hidden sm:inline">{t('common.refresh')}</span>
            </button>
            <button
              onClick={handleOpenCreateModal}
              className="flex items-center gap-2 rounded-xl bg-accent-red px-4 py-2 font-semibold text-white hover:shadow-lg"
            >
              <Plus className="h-4 w-4" />
              <span className="hidden sm:inline">{t('admin.rentedProducts.newItem')}</span>
              <span className="sm:hidden">{t('common.new')}</span>
            </button>
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-gray-300 backdrop-blur">
        <div className="flex flex-wrap items-center gap-6">
          <span>
            <strong className="text-white">{sortedEquipment.length}</strong> {t('admin.rentedProducts.items')}
          </span>
          <span>
            {t('admin.rentedProducts.totalRentalCost')}: <strong className="text-white">{formatCurrency(totalRentalCost)}</strong>
          </span>
          <span>
            {t('admin.rentedProducts.totalCustomerPrice')}: <strong className="text-green-400">{formatCurrency(totalCustomerRevenue)}</strong>
          </span>
          {supplierFilter && (
            <span>
              {t('admin.rentedProducts.filteredBy')}: <strong className="text-white">{supplierFilter}</strong>
            </span>
          )}
        </div>
      </div>

      {loading ? (
        <div className="glass rounded-xl p-8 text-center text-gray-400">
          {t('admin.rentedProducts.loading')}
        </div>
      ) : sortedEquipment.length === 0 ? (
        <div className="glass rounded-xl p-8 text-center text-gray-400">
          <Building2 className="mx-auto mb-4 h-12 w-12 text-gray-600" />
          {t('admin.rentedProducts.empty')}
          {hasActiveFilters ? ` ${t('admin.rentedProducts.adjustFilters')}` : '.'}
        </div>
      ) : viewMode === 'table' ? (
        <div className="overflow-hidden rounded-xl border border-white/10 bg-white/5">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-white/10 text-sm text-gray-200">
              <thead className="bg-white/5 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.rentedProducts.columns.product')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.rentedProducts.columns.supplier')}</th>
                  <th className="px-4 py-3 text-right font-semibold">{t('admin.rentedProducts.columns.rentalPrice')}</th>
                  <th className="px-4 py-3 text-right font-semibold">{t('admin.rentedProducts.columns.customerPrice')}</th>
                  <th className="px-4 py-3 text-right font-semibold">{t('admin.rentedProducts.columns.margin')}</th>
                  <th className="px-4 py-3 text-center font-semibold">{t('admin.rentedProducts.columns.status')}</th>
                  <th className="px-4 py-3 text-right font-semibold">{t('admin.rentedProducts.columns.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {sortedEquipment.map(item => {
                  const margin = profitMargin(item.rental_price, item.customer_price);
                  return (
                    <tr key={item.equipment_id} className="hover:bg-white/5">
                      <td className="px-4 py-3 align-top">
                        <div className="flex flex-col">
                          <span className="text-white font-medium">{item.product_name}</span>
                          {item.category && (
                            <span className="text-xs text-gray-500">{item.category}</span>
                          )}
                          {item.description && (
                            <span className="mt-1 text-xs text-gray-400 line-clamp-2">
                              {item.description}
                            </span>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 align-top">
                        <div className="flex items-center gap-2">
                          <Building2 className="h-4 w-4 text-blue-400" />
                          <span className="text-gray-300">{item.supplier_name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 align-top text-right text-gray-300">
                        {formatCurrency(item.rental_price)}
                      </td>
                      <td className="px-4 py-3 align-top text-right font-medium text-green-400">
                        {formatCurrency(item.customer_price)}
                      </td>
                      <td className="px-4 py-3 align-top text-right">
                        <span className={margin > 0 ? 'text-green-400' : margin < 0 ? 'text-red-400' : 'text-gray-400'}>
                          {margin > 0 ? '+' : ''}{margin.toFixed(1)}%
                        </span>
                      </td>
                      <td className="px-4 py-3 align-top text-center">
                        <span className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium ${
                          item.is_active
                            ? 'bg-green-500/20 text-green-400'
                            : 'bg-gray-500/20 text-gray-400'
                        }`}>
                          {item.is_active ? t('admin.rentedProducts.active') : t('admin.rentedProducts.inactive')}
                        </span>
                      </td>
                      <td className="px-4 py-3 align-top">
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={() => setViewEquipment(item)}
                            className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                            title={t('admin.rentedProducts.viewDetails')}
                            aria-label={t('admin.rentedProducts.viewDetails')}
                          >
                            <Eye className="h-4 w-4" />
                          </button>
                          <button
                            onClick={() => handleEditEquipment(item)}
                            className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                            title={t('common.edit')}
                            aria-label={t('common.edit')}
                          >
                            <Pencil className="h-4 w-4" />
                          </button>
                          <button
                            onClick={() => handleDelete(item.equipment_id, item.product_name)}
                            className="rounded-lg bg-red-600/80 p-2 text-white transition hover:bg-red-600"
                            title={t('common.delete')}
                            aria-label={t('common.delete')}
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {sortedEquipment.map(item => {
            const margin = profitMargin(item.rental_price, item.customer_price);
            return (
              <div key={item.equipment_id} className="glass rounded-xl p-4">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h3 className="text-lg font-semibold text-white">{item.product_name}</h3>
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                        item.is_active
                          ? 'bg-green-500/20 text-green-400'
                          : 'bg-gray-500/20 text-gray-400'
                      }`}>
                        {item.is_active ? t('admin.rentedProducts.active') : t('admin.rentedProducts.inactive')}
                      </span>
                    </div>
                    <div className="mt-1 flex items-center gap-2 text-sm text-gray-400">
                      <Building2 className="h-4 w-4 text-blue-400" />
                      {item.supplier_name}
                    </div>
                    {item.category && (
                      <p className="mt-1 text-xs text-gray-500">{item.category}</p>
                    )}
                    <div className="mt-3 grid grid-cols-3 gap-2 text-center">
                      <div>
                        <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.rentalPrice')}</p>
                        <p className="text-sm font-medium text-gray-300">{formatCurrency(item.rental_price)}</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.customerPrice')}</p>
                        <p className="text-sm font-medium text-green-400">{formatCurrency(item.customer_price)}</p>
                      </div>
                      <div>
                        <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.margin')}</p>
                        <p className={`text-sm font-medium ${margin > 0 ? 'text-green-400' : margin < 0 ? 'text-red-400' : 'text-gray-400'}`}>
                          {margin > 0 ? '+' : ''}{margin.toFixed(1)}%
                        </p>
                      </div>
                    </div>
                  </div>
                  <div className="flex flex-col gap-2">
                    <button
                      onClick={() => setViewEquipment(item)}
                      className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                      title={t('admin.rentedProducts.viewDetails')}
                      aria-label={t('admin.rentedProducts.viewDetails')}
                    >
                      <Eye className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleEditEquipment(item)}
                      className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                      title={t('common.edit')}
                      aria-label={t('common.edit')}
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(item.equipment_id, item.product_name)}
                      className="rounded-lg bg-red-600/80 p-2 text-white transition hover:bg-red-600"
                      title={t('common.delete')}
                      aria-label={t('common.delete')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Create/Edit Modal */}
      {modalOpen && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-2xl font-bold text-white">
                  {editingId ? t('admin.rentedProducts.editItem') : t('admin.rentedProducts.newItem')}
                </h3>
                <button
                  onClick={closeModal}
                  className="text-gray-400 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors"
                  title={t('common.close')}
                  aria-label={t('common.close')}
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

              <form onSubmit={handleSubmit} className="space-y-6">
                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">
                    {t('admin.rentedProducts.fields.productName')} <span className="text-accent-red">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.product_name}
                    onChange={event => setFormData({ ...formData, product_name: event.target.value })}
                    placeholder={t('admin.rentedProducts.placeholders.productName')}
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    title={t('admin.rentedProducts.fields.productName')}
                    required
                  />
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">
                    {t('admin.rentedProducts.fields.supplier')} <span className="text-accent-red">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.supplier_name}
                    onChange={event => setFormData({ ...formData, supplier_name: event.target.value })}
                    placeholder={t('admin.rentedProducts.placeholders.supplier')}
                    list="supplier-suggestions"
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    title={t('admin.rentedProducts.fields.supplier')}
                    required
                  />
                  <datalist id="supplier-suggestions">
                    {suppliers.map(supplier => (
                      <option key={supplier} value={supplier} />
                    ))}
                  </datalist>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      {t('admin.rentedProducts.fields.rentalPrice')} <span className="text-accent-red">*</span>
                    </label>
                    <div className="relative">
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        value={formData.rental_price || ''}
                        onChange={event =>
                          setFormData({
                            ...formData,
                            rental_price: parseNumber(event.target.value),
                          })
                        }
                        placeholder="15.00"
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 pr-8 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        title={t('admin.rentedProducts.fields.rentalPrice')}
                      />
                      <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">{currencySymbol}</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500">{t('admin.rentedProducts.help.rentalPrice')}</p>
                  </div>

                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      {t('admin.rentedProducts.fields.customerPrice')} <span className="text-accent-red">*</span>
                    </label>
                    <div className="relative">
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        value={formData.customer_price || ''}
                        onChange={event =>
                          setFormData({
                            ...formData,
                            customer_price: parseNumber(event.target.value),
                          })
                        }
                        placeholder="25.00"
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 pr-8 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        title={t('admin.rentedProducts.fields.customerPrice')}
                      />
                      <span className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400">{currencySymbol}</span>
                    </div>
                    <p className="mt-1 text-xs text-gray-500">{t('admin.rentedProducts.help.customerPrice')}</p>
                  </div>
                </div>

                {formData.rental_price > 0 && formData.customer_price > 0 && (
                  <div className="rounded-lg bg-white/5 p-3 text-center">
                    <span className="text-sm text-gray-400">{t('admin.rentedProducts.marginLabel')} </span>
                    <span className={`text-lg font-bold ${
                      profitMargin(formData.rental_price, formData.customer_price) > 0
                        ? 'text-green-400'
                        : 'text-red-400'
                    }`}>
                      {profitMargin(formData.rental_price, formData.customer_price) > 0 ? '+' : ''}
                      {profitMargin(formData.rental_price, formData.customer_price).toFixed(1)}%
                    </span>
                    <span className="ml-2 text-sm text-gray-400">
                      ({formatCurrency(formData.customer_price - formData.rental_price)} {t('admin.rentedProducts.profit')})
                    </span>
                  </div>
                )}

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">{t('products.category')}</label>
                  <input
                    type="text"
                    value={formData.category}
                    onChange={event => setFormData({ ...formData, category: event.target.value })}
                    placeholder={t('admin.rentedProducts.placeholders.category')}
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    title={t('products.category')}
                  />
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">{t('common.description')}</label>
                  <textarea
                    value={formData.description}
                    onChange={event => setFormData({ ...formData, description: event.target.value })}
                    rows={3}
                    placeholder={t('admin.rentedProducts.placeholders.description')}
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    title={t('common.description')}
                  />
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">{t('common.notes')}</label>
                  <textarea
                    value={formData.notes}
                    onChange={event => setFormData({ ...formData, notes: event.target.value })}
                    rows={2}
                    placeholder={t('admin.rentedProducts.placeholders.notes')}
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    title={t('common.notes')}
                  />
                </div>

                <div className="flex items-center gap-3">
                  <input
                    type="checkbox"
                    id="is_active"
                    checked={formData.is_active}
                    onChange={event => setFormData({ ...formData, is_active: event.target.checked })}
                    className="h-4 w-4 rounded border-white/20 bg-white/10 text-accent-red focus:ring-accent-red"
                  />
                  <label htmlFor="is_active" className="text-sm font-medium text-white">
                    {t('admin.rentedProducts.fields.activeHelp')}
                  </label>
                </div>

                <div className="flex gap-3 pt-4">
                  <button
                    type="button"
                    onClick={closeModal}
                    className="flex-1 btn-secondary"
                    disabled={submitting}
                  >
                    {t('common.cancel')}
                  </button>
                  <button type="submit" className="flex-1 btn-primary" disabled={submitting}>
                    {submitting ? t('common.saving') : editingId ? t('common.save') : t('common.create')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </ModalPortal>
      )}

      {/* Detail Modal */}
      {viewEquipment && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-2xl font-bold text-white">{viewEquipment.product_name}</h3>
                <button
                  onClick={closeDetailModal}
                  className="text-gray-400 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors"
                  title={t('common.close')}
                  aria-label={t('common.close')}
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

              <div className="space-y-4">
                <div className="flex items-center gap-3 rounded-lg bg-white/5 p-4">
                  <Building2 className="h-6 w-6 text-blue-400" />
                  <div>
                    <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.supplier')}</p>
                    <p className="text-lg font-medium text-white">{viewEquipment.supplier_name}</p>
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-4">
                  <div className="rounded-lg bg-white/5 p-4 text-center">
                    <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.rentalPrice')}</p>
                    <p className="text-xl font-bold text-gray-300">{formatCurrency(viewEquipment.rental_price)}</p>
                  </div>
                  <div className="rounded-lg bg-white/5 p-4 text-center">
                    <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.customerPrice')}</p>
                    <p className="text-xl font-bold text-green-400">{formatCurrency(viewEquipment.customer_price)}</p>
                  </div>
                  <div className="rounded-lg bg-white/5 p-4 text-center">
                    <p className="text-xs text-gray-500">{t('admin.rentedProducts.columns.margin')}</p>
                    <p className={`text-xl font-bold ${
                      profitMargin(viewEquipment.rental_price, viewEquipment.customer_price) > 0
                        ? 'text-green-400'
                        : 'text-red-400'
                    }`}>
                      {profitMargin(viewEquipment.rental_price, viewEquipment.customer_price) > 0 ? '+' : ''}
                      {profitMargin(viewEquipment.rental_price, viewEquipment.customer_price).toFixed(1)}%
                    </p>
                  </div>
                </div>

                {viewEquipment.category && (
                  <div>
                    <p className="text-xs text-gray-500">{t('products.category')}</p>
                    <p className="text-white">{viewEquipment.category}</p>
                  </div>
                )}

                {viewEquipment.description && (
                  <div>
                    <p className="text-xs text-gray-500">{t('common.description')}</p>
                    <p className="text-white">{viewEquipment.description}</p>
                  </div>
                )}

                {viewEquipment.notes && (
                  <div>
                    <p className="text-xs text-gray-500">{t('common.notes')}</p>
                    <p className="text-gray-300">{viewEquipment.notes}</p>
                  </div>
                )}

                <div className="flex items-center justify-between rounded-lg bg-white/5 p-4">
                  <span className="text-sm text-gray-400">{t('admin.rentedProducts.columns.status')}</span>
                  <span className={`inline-flex items-center rounded-full px-3 py-1 text-sm font-medium ${
                    viewEquipment.is_active
                      ? 'bg-green-500/20 text-green-400'
                      : 'bg-gray-500/20 text-gray-400'
                  }`}>
                    {viewEquipment.is_active ? t('admin.rentedProducts.active') : t('admin.rentedProducts.inactive')}
                  </span>
                </div>

                <div className="text-xs text-gray-500">
                  <p>{t('admin.rentedProducts.createdAt')}: {formatDateTimeISO(viewEquipment.created_at)}</p>
                  <p>{t('admin.rentedProducts.updatedAt')}: {formatDateTimeISO(viewEquipment.updated_at)}</p>
                </div>
              </div>

              <button
                onClick={closeDetailModal}
                className="w-full mt-6 btn-secondary"
              >
                {t('common.close')}
              </button>
            </div>
          </div>
        </ModalPortal>
      )}
    </div>
  );
}
