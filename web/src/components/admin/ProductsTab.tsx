import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Eye,
  GitBranch,
  LayoutGrid,
  List,
  Package,
  Pencil,
  Plus,
  RefreshCcw,
  Search,
  Trash2,
  X,
} from 'lucide-react';
import { api } from '../../lib/api';
import { ModalPortal } from '../ModalPortal';
import { DeviceTreeTab } from './DeviceTreeTab';
import { ProductDependenciesModal } from '../ProductDependenciesModal';
import { ProductDetailModal } from '../ProductDetailModal';
import { ProductDevicesModal } from '../ProductDevicesModal';

interface Product {
  product_id: number;
  name: string;
  category_id?: number | null;
  subcategory_id?: string | number | null;
  subbiercategory_id?: string | number | null;
  manufacturer_id?: number | null;
  brand_id?: number | null;
  description?: string | null;
  maintenance_interval?: number | null;
  item_cost_per_day?: number | null;
  weight?: number | null;
  height?: number | null;
  width?: number | null;
  depth?: number | null;
  power_consumption?: number | null;
  pos_in_category?: number | null;
  category_name?: string | null;
  subcategory_name?: string | null;
  subbiercategory_name?: string | null;
  brand_name?: string | null;
  manufacturer_name?: string | null;
  // Accessories & Consumables fields
  is_accessory?: boolean;
  is_consumable?: boolean;
  count_type_id?: number | null;
  stock_quantity?: number | null;
  min_stock_level?: number | null;
  generic_barcode?: string | null;
  price_per_unit?: number | null;
  count_type_name?: string | null;
  count_type_abbr?: string | null;
  website_visible?: boolean;
  website_thumbnail?: string | null;
  website_images?: string[];
}

interface Category {
  category_id: number;
  name: string;
}

interface Subcategory {
  subcategory_id: string | number;
  name: string;
  category_id: number;
}

interface Subbiercategory {
  subbiercategory_id: string | number;
  name: string;
  subcategory_id: string | number;
}

interface Brand {
  brand_id: number;
  name: string;
  manufacturer_id?: number | null;
  manufacturer_name?: string | null;
}

interface Manufacturer {
  manufacturer_id: number;
  name: string;
  website?: string | null;
}

interface ProductFormData {
  name: string;
  description: string;
  category_id?: number;
  subcategory_id?: string | number;
  subbiercategory_id?: string | number;
  brand_id?: number;
  manufacturer_id?: number;
  item_cost_per_day?: number;
  weight?: number;
  height?: number;
  width?: number;
  depth?: number;
  maintenance_interval?: number;
  power_consumption?: number;
  pos_in_category?: number;
  device_quantity?: number;
  device_prefix?: string;
  // Accessories & Consumables fields
  is_accessory?: boolean;
  is_consumable?: boolean;
  count_type_id?: number;
  stock_quantity?: number;
  min_stock_level?: number;
  generic_barcode?: string;
  price_per_unit?: number;
}

interface Device {
  device_id: string;
  product_id?: number;
  status: string;
  serial_number?: string;
  barcode?: string;
}

interface CountType {
  count_type_id: number;
  name: string;
  abbreviation: string;
  is_active: boolean;
}

const initialFormData: ProductFormData = {
  name: '',
  description: '',
};

const parseNumber = (value: string): number | undefined => {
  const trimmed = value.trim();
  if (trimmed === '') {
    return undefined;
  }
  const parsed = Number(trimmed);
  return Number.isNaN(parsed) ? undefined : parsed;
};

const parseInteger = (value: string): number | undefined => {
  const trimmed = value.trim();
  if (trimmed === '') {
    return undefined;
  }
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isNaN(parsed) ? undefined : parsed;
};

function useDebouncedValue<T>(value: T, delay: number) {
  const [debounced, setDebounced] = useState(value);

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delay);
    return () => window.clearTimeout(handle);
  }, [value, delay]);

  return debounced;
}

export function ProductsTab() {
  const { t } = useTranslation();
  const [products, setProducts] = useState<Product[]>([]);
  const [loadingProducts, setLoadingProducts] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewProduct, setViewProduct] = useState<Product | null>(null);
  const [editingProduct, setEditingProduct] = useState<number | null>(null);
  const [dependenciesModal, setDependenciesModal] = useState<{ productId: number; productName: string } | null>(null);
  const [formData, setFormData] = useState<ProductFormData>(initialFormData);
  const [categories, setCategories] = useState<Category[]>([]);
  const [subcategories, setSubcategories] = useState<Subcategory[]>([]);
  const [subbiercategories, setSubbiercategories] = useState<Subbiercategory[]>([]);
  const [brands, setBrands] = useState<Brand[]>([]);
  const [manufacturers, setManufacturers] = useState<Manufacturer[]>([]);
  const [countTypes, setCountTypes] = useState<CountType[]>([]);
  const [metadataLoaded, setMetadataLoaded] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'cards' | 'tree'>(() => {
    // Set default based on screen width: mobile (<768px) = cards, desktop = table
    return typeof window !== 'undefined' && window.innerWidth < 768 ? 'cards' : 'table';
  });
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<number | ''>('');
  const [refreshing, setRefreshing] = useState(false);
  // const scrollPosition = useRef(0);
  const [productDevices, setProductDevices] = useState<Device[]>([]);
  const [devicesToDelete, setDevicesToDelete] = useState<Set<string>>(new Set());
  const [loadingDevices, setLoadingDevices] = useState(false);
  const [devicesModal, setDevicesModal] = useState<{ productId: number; productName: string } | null>(null);
  const [devicesModalDevices, setDevicesModalDevices] = useState<Device[]>([]);
  const [loadingDevicesModal, setLoadingDevicesModal] = useState(false);

  const debouncedSearch = useDebouncedValue(searchTerm, 300);

  // Update view mode based on screen size
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
    const anyModalOpen = modalOpen || !!viewProduct;
    if (anyModalOpen) {
      // scrollPosition.current = window.scrollY || window.pageYOffset || 0; // Not needed if we don't fix body
      html.classList.add('modal-open');
      body.classList.add('modal-open');
      // Fix: Don't set position:fixed on body as it breaks modal positioning relative to viewport
      // body.style.position = 'fixed';
      // body.style.top = `-${scrollPosition.current}px`;
      // body.style.left = '0';
      // body.style.right = '0';
      // body.style.width = '100%';
      return () => {
        html.classList.remove('modal-open');
        body.classList.remove('modal-open');
        // body.style.position = '';
        // body.style.top = '';
        // body.style.left = '';
        // body.style.right = '';
        // body.style.width = '';
        // window.scrollTo(0, scrollPosition.current);
      };
    }

    return undefined;
  }, [modalOpen, viewProduct]);

  const fetchProducts = useCallback(
    async (searchValue?: string, categoryId?: number | '') => {
      setLoadingProducts(true);
      try {
        const params: Record<string, string> = {};
        if (searchValue && searchValue.trim().length > 0) {
          params.search = searchValue.trim();
        }
        if (categoryId !== '' && typeof categoryId === 'number') {
          params.category_id = String(categoryId);
        }
        const { data } = await api.get<Product[]>('/admin/products', { params });
        setProducts(data || []);
      } catch (error) {
        console.error('Failed to load products:', error);
        setProducts([]);
      } finally {
        setLoadingProducts(false);
      }
    },
    []
  );

  const loadMetadata = useCallback(async () => {
    try {
      const [catRes, subRes, subbierRes, brandRes, manufacturerRes, countTypeRes] = await Promise.all([
        api.get<Category[]>('/admin/categories'),
        api.get<Subcategory[]>('/admin/subcategories'),
        api.get<Subbiercategory[]>('/admin/subbiercategories'),
        api.get<Brand[]>('/admin/brands'),
        api.get<Manufacturer[]>('/admin/manufacturers'),
        api.get<CountType[]>('/admin/count-types'),
      ]);

      setCategories(catRes.data || []);
      setSubcategories(subRes.data || []);
      setSubbiercategories(subbierRes.data || []);
      setBrands(brandRes.data || []);
      setManufacturers(manufacturerRes.data || []);
      setCountTypes(countTypeRes.data || []);

      setMetadataLoaded(true);
    } catch (error) {
      console.error('Failed to load metadata:', error);
    }
  }, []);

  const ensureMetadataLoaded = useCallback(async () => {
    if (!metadataLoaded) {
      await loadMetadata();
    }
  }, [metadataLoaded, loadMetadata]);

  useEffect(() => {
    fetchProducts(debouncedSearch, categoryFilter);
  }, [fetchProducts, debouncedSearch, categoryFilter]);

  useEffect(() => {
    loadMetadata();
  }, [loadMetadata]);

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    await fetchProducts(searchTerm, categoryFilter);
    setRefreshing(false);
  }, [fetchProducts, searchTerm, categoryFilter]);

  const clearFilters = () => {
    setSearchTerm('');
    setCategoryFilter('');
  };

  const handleCategoryFilterChange = (value: string) => {
    if (!value) {
      setCategoryFilter('');
      return;
    }
    setCategoryFilter(Number(value));
  };

  const mapProductToFormData = useCallback(
    (product: Product): ProductFormData => ({
      name: product.name ?? '',
      description: product.description ?? '',
      category_id: product.category_id ?? undefined,
      subcategory_id: product.subcategory_id ?? undefined,
      subbiercategory_id: product.subbiercategory_id ?? undefined,
      brand_id: product.brand_id ?? undefined,
      manufacturer_id: product.manufacturer_id ?? undefined,
      item_cost_per_day: product.item_cost_per_day ?? undefined,
      weight: product.weight ?? undefined,
      height: product.height ?? undefined,
      width: product.width ?? undefined,
      depth: product.depth ?? undefined,
      maintenance_interval: product.maintenance_interval ?? undefined,
      power_consumption: product.power_consumption ?? undefined,
      pos_in_category: product.pos_in_category ?? undefined,
      device_quantity: undefined,
      device_prefix: '',
      // Accessories & Consumables fields
      is_accessory: product.is_accessory ?? false,
      is_consumable: product.is_consumable ?? false,
      count_type_id: product.count_type_id ?? undefined,
      stock_quantity: product.stock_quantity ?? undefined,
      min_stock_level: product.min_stock_level ?? undefined,
      generic_barcode: product.generic_barcode ?? '',
      price_per_unit: product.price_per_unit ?? undefined,
    }),
    []
  );

  const filteredSubcategories = useMemo(() => {
    if (!formData.category_id) {
      return subcategories;
    }
    return subcategories.filter(sub => sub.category_id === formData.category_id);
  }, [subcategories, formData.category_id]);

  const filteredSubbiercategories = useMemo(() => {
    if (!formData.subcategory_id) {
      return subbiercategories;
    }
    return subbiercategories.filter(subbier => subbier.subcategory_id === formData.subcategory_id);
  }, [subbiercategories, formData.subcategory_id]);

  const loadProductDevices = useCallback(async (productId: number) => {
    setLoadingDevices(true);
    try {
      const { data } = await api.get<Device[]>(`/admin/products/${productId}/devices`);
      setProductDevices(data || []);
      setDevicesToDelete(new Set());
    } catch (error) {
      console.error('Failed to load product devices:', error);
      setProductDevices([]);
    } finally {
      setLoadingDevices(false);
    }
  }, []);

  const closeModal = useCallback(() => {
    setModalOpen(false);
    setEditingProduct(null);
    setFormData(initialFormData);
    setProductDevices([]);
    setDevicesToDelete(new Set());
  }, []);

  const closeDetailModal = () => {
    setViewProduct(null);
  };

  const handleOpenDevicesModal = useCallback(async (productId: number, productName: string) => {
    setDevicesModal({ productId, productName });
    setLoadingDevicesModal(true);
    try {
      const { data } = await api.get<Device[]>(`/admin/products/${productId}/devices`);
      setDevicesModalDevices(data || []);
    } catch (error) {
      console.error('Failed to load devices for modal:', error);
      setDevicesModalDevices([]);
    } finally {
      setLoadingDevicesModal(false);
    }
  }, []);

  const closeDevicesModal = useCallback(() => {
    setDevicesModal(null);
    setDevicesModalDevices([]);
  }, []);

  const handleOpenCreateModal = async () => {
    await ensureMetadataLoaded();
    setFormData(initialFormData);
    setEditingProduct(null);
    setModalOpen(true);
  };

  const handleEditProduct = async (productId: number) => {
    try {
      await ensureMetadataLoaded();
      const { data } = await api.get<Product>(`/admin/products/${productId}`);
      setFormData(mapProductToFormData(data));
      setEditingProduct(productId);
      setModalOpen(true);
      await loadProductDevices(productId);
    } catch (error) {
      console.error('Failed to load product details:', error);
      window.alert(t('admin.products.errors.loadProduct'));
    }
  };

  const handleAddDevices = async () => {
    if (!editingProduct) return;

    const quantity = formData.device_quantity;
    if (!quantity || quantity <= 0) {
      window.alert(t('admin.products.errors.invalidDeviceCount'));
      return;
    }

    try {
      await api.post(`/admin/products/${editingProduct}/devices`, {
        product_id: editingProduct,
        quantity: quantity,
        prefix: formData.device_prefix || '',
      });

      // Reload devices
      await loadProductDevices(editingProduct);

      // Reset device creation fields
      setFormData({ ...formData, device_quantity: undefined, device_prefix: '' });
    } catch (error) {
      console.error('Failed to add devices:', error);
      window.alert(t('admin.products.errors.addDevices'));
    }
  };

  const handleRemoveDevice = (deviceId: string) => {
    setDevicesToDelete(prev => {
      const newSet = new Set(prev);
      if (newSet.has(deviceId)) {
        newSet.delete(deviceId);
      } else {
        newSet.add(deviceId);
      }
      return newSet;
    });
  };

  const handleDelete = async (productId: number, name: string) => {
    if (!window.confirm(t('admin.products.confirmDelete', { name }))) {
      return;
    }

    try {
      await api.delete(`/admin/products/${productId}`);
      await fetchProducts(searchTerm, categoryFilter);
    } catch (error) {
      console.error('Failed to delete product:', error);
      window.alert(t('admin.products.errors.delete'));
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!formData.name.trim()) {
      window.alert(t('admin.products.errors.nameRequired'));
      return;
    }

    setSubmitting(true);

    const payload = {
      name: formData.name.trim(),
      description: formData.description.trim() || null,
      category_id: formData.category_id ?? null,
      subcategory_id: formData.subcategory_id ?? null,
      subbiercategory_id: formData.subbiercategory_id ?? null,
      brand_id: formData.brand_id ?? null,
      manufacturer_id: formData.manufacturer_id ?? null,
      item_cost_per_day: formData.item_cost_per_day ?? null,
      weight: formData.weight ?? null,
      height: formData.height ?? null,
      width: formData.width ?? null,
      depth: formData.depth ?? null,
      maintenance_interval: formData.maintenance_interval ?? null,
      power_consumption: formData.power_consumption ?? null,
      pos_in_category: formData.pos_in_category ?? null,
      // Accessories & Consumables fields
      is_accessory: formData.is_accessory ?? false,
      is_consumable: formData.is_consumable ?? false,
      count_type_id: formData.count_type_id ?? null,
      stock_quantity: formData.stock_quantity ?? null,
      min_stock_level: formData.min_stock_level ?? null,
      generic_barcode: formData.generic_barcode?.trim() || null,
      price_per_unit: formData.price_per_unit ?? null,
    };

    try {
      let productId = editingProduct;

      if (editingProduct) {
        await api.put(`/admin/products/${editingProduct}`, payload);

        // Handle device deletions if editing
        if (devicesToDelete.size > 0) {
          const deletePromises = Array.from(devicesToDelete).map(deviceId =>
            api.delete(`/admin/devices/${deviceId}`)
          );
          try {
            await Promise.all(deletePromises);
          } catch (deviceError) {
            console.error('Failed to delete some devices:', deviceError);
            window.alert(t('admin.products.errors.partialDeviceDelete'));
          }
        }
      } else {
        const { data } = await api.post<Product>('/admin/products', payload);
        productId = data.product_id;

        if (formData.device_quantity && formData.device_quantity > 0 && productId) {
          try {
            await api.post(`/admin/products/${productId}/devices`, {
              product_id: productId,
              quantity: formData.device_quantity,
              prefix: formData.device_prefix || '',
            });
          } catch (deviceError) {
            console.error('Failed to create devices:', deviceError);
            window.alert(t('admin.products.errors.deviceCreate'));
          }
        }
      }

      await fetchProducts(searchTerm, categoryFilter);
      closeModal();
    } catch (error) {
      console.error('Failed to save product:', error);
      window.alert(t('admin.products.errors.save'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleViewProduct = async (productId: number) => {
    try {
      const { data } = await api.get<Product>(`/admin/products/${productId}`);
      setViewProduct(data);
    } catch (error) {
      console.error('Failed to load product details:', error);
      window.alert(t('admin.products.errors.loadProduct'));
    }
  };

  const brandsByName = useMemo(() => brands, [brands]);
  const manufacturerOptions = useMemo(() => manufacturers, [manufacturers]);
  const sortedProducts = useMemo(() => [...products].sort((a, b) => a.name.localeCompare(b.name)), [
    products,
  ]);

  const hasActiveFilters = debouncedSearch.trim() !== '' || categoryFilter !== '';

  const categoryPath = (product: Product) =>
    [product.category_name, product.subcategory_name, product.subbiercategory_name]
      .filter(Boolean)
      .join(' · ') || t('admin.products.noCategoryAssigned');

  const formatCurrency = (value?: number | null) =>
    value != null ? `${value.toFixed(2)} €` : t('admin.products.dash');

  const averageDailyPrice =
    sortedProducts.length === 0
      ? null
      : sortedProducts.reduce((sum, product) => sum + (product.item_cost_per_day ?? 0), 0) /
      sortedProducts.length;

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        <div>
          <h2 className="text-xl font-bold text-white">{t('admin.products.title')}</h2>
          <p className="text-sm text-gray-400">
            {t('admin.products.loadedCount', { count: sortedProducts.length })}
            {hasActiveFilters ? ` • ${t('admin.products.filtersActive')}` : ''}
          </p>
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={searchTerm}
                onChange={event => setSearchTerm(event.target.value)}
                placeholder={t('admin.products.searchPlaceholder')}
                className="w-full rounded-lg bg-white/10 py-2 pl-9 pr-3 text-sm text-white placeholder-gray-500 outline-none transition focus:bg-white/15 focus:ring-1 focus:ring-accent-red"
                title={t('common.search')}
              />
            </div>

            <select
              value={categoryFilter}
              onChange={event => handleCategoryFilterChange(event.target.value)}
              className="rounded-lg bg-white/10 px-3 py-2 text-sm text-white outline-none transition focus:bg-white/15 focus:ring-1 focus:ring-accent-red"
              title={t('products.category')}
            >
              <option value="">{t('admin.products.allCategories')}</option>
              {categories.map(category => (
                <option key={category.category_id} value={category.category_id}>
                  {category.name}
                </option>
              ))}
            </select>

            {hasActiveFilters && (
              <button
                type="button"
                onClick={clearFilters}
                className="rounded-lg bg-white/10 px-3 py-2 text-sm text-white hover:bg-white/20 whitespace-nowrap"
              >
                {t('admin.products.resetFilters')}
              </button>
            )}
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={() => setViewMode('table')}
              className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${viewMode === 'table'
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
              className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${viewMode === 'cards'
                ? 'bg-white/15 text-white'
                : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white'
                }`}
            >
              <LayoutGrid className="h-4 w-4" />
              <span className="hidden sm:inline">{t('admin.devices.cardView')}</span>
            </button>
            <button
              type="button"
              onClick={() => setViewMode('tree')}
              className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${viewMode === 'tree'
                ? 'bg-white/15 text-white'
                : 'bg-white/5 text-gray-400 hover:bg-white/10 hover:text-white'
                }`}
            >
              <GitBranch className="h-4 w-4" />
              <span className="hidden sm:inline">{t('admin.products.deviceTree')}</span>
            </button>
            <button
              type="button"
              onClick={handleRefresh}
              disabled={refreshing}
              className="flex items-center gap-2 rounded-lg bg-white/10 px-3 py-2 text-sm font-semibold text-white transition hover:bg-white/20 disabled:opacity-50"
            >
              <RefreshCcw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              <span className="hidden sm:inline">{t('common.update')}</span>
            </button>
            <button
              onClick={handleOpenCreateModal}
              className="flex items-center gap-2 rounded-xl bg-accent-red px-4 py-2 font-semibold text-white hover:shadow-lg"
            >
              <Plus className="h-4 w-4" />
              <span className="hidden sm:inline">{t('admin.devices.newDevice')}</span>
              <span className="sm:hidden">{t('common.create')}</span>
            </button>
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-gray-300 backdrop-blur">
        <div className="flex flex-wrap items-center gap-6">
          <span>
            <strong className="text-white">{sortedProducts.length}</strong> {t('products.title')}
          </span>
          <span>
            {t('admin.products.avgDailyPrice')}:{' '}
            <strong className="text-white">
              {averageDailyPrice != null ? `${averageDailyPrice.toFixed(2)} €` : t('admin.products.dash')}
            </strong>
          </span>
          {categoryFilter !== '' && (
            <span>
              {t('admin.products.filteredByCategory')}:{' '}
              <strong className="text-white">
                {
                  categories.find(category => category.category_id === categoryFilter)
                    ?.name
                }
              </strong>
            </span>
          )}
          {debouncedSearch.trim() && (
            <span>
              {t('admin.products.searchTerm')}: <strong className="text-white">"{debouncedSearch}"</strong>
            </span>
          )}
        </div>
      </div>

      {loadingProducts ? (
        <div className="glass rounded-xl p-8 text-center text-gray-400">
          {t('admin.products.loading')}
        </div>
      ) : sortedProducts.length === 0 ? (
        <div className="glass rounded-xl p-8 text-center text-gray-400">
          <Package className="mx-auto mb-4 h-12 w-12 text-gray-600" />
          {t('admin.products.empty')}
          {hasActiveFilters ? ` ${t('admin.products.adjustFilters')}` : '.'}
        </div>
      ) : viewMode === 'table' ? (
        <div className="overflow-hidden rounded-xl border border-white/10 bg-white/5">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-white/10 text-sm text-gray-200">
              <thead className="bg-white/5 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">{t('products.title')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('products.category')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.products.brandManufacturer')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.products.pricePerDay')}</th>
                  <th className="px-4 py-3 text-right font-semibold">{t('labels.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {sortedProducts.map(product => (
                  <tr key={product.product_id} className="hover:bg-white/5">
                    <td className="px-4 py-3 align-top">
                      <div className="flex flex-col">
                        <span className="text-white font-medium">{product.name}</span>
                        <span className="text-xs text-gray-500">
                          {t('admin.ledControllers.id')}: {product.product_id}
                          {product.pos_in_category != null
                            ? ` • ${t('admin.products.position')} ${product.pos_in_category}`
                            : ''}
                        </span>
                        {product.description && (
                          <span className="mt-1 text-xs text-gray-400 line-clamp-2">
                            {product.description}
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 align-top text-sm text-gray-300">
                      {categoryPath(product)}
                    </td>
                    <td className="px-4 py-3 align-top text-sm text-gray-300">
                      <div className="flex flex-col">
                        <span>{product.brand_name || t('admin.products.dash')}</span>
                        <span className="text-xs text-gray-500">
                          {product.manufacturer_name || t('admin.products.noManufacturer')}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-3 align-top text-sm text-gray-200">
                      {formatCurrency(product.item_cost_per_day)}
                    </td>
                    <td className="px-4 py-3 align-top">
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={() => handleViewProduct(product.product_id)}
                          className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                          title={t('casesPage.details')}
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleEditProduct(product.product_id)}
                          className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                          title={t('common.edit')}
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleOpenDevicesModal(product.product_id, product.name)}
                          className="rounded-lg bg-cyan-600/80 p-2 text-white transition hover:bg-cyan-600"
                          title={t('admin.products.manageDevices')}
                        >
                          <Box className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => setDependenciesModal({ productId: product.product_id, productName: product.name })}
                          className="rounded-lg bg-purple-600/80 p-2 text-white transition hover:bg-purple-600"
                          title={t('admin.products.manageDependencies')}
                        >
                          <GitBranch className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(product.product_id, product.name)}
                          className="rounded-lg bg-red-600/80 p-2 text-white transition hover:bg-red-600"
                          title={t('common.delete')}
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : viewMode === 'cards' ? (
        <div className="grid w-full gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {sortedProducts.map(product => (
            <div key={product.product_id} className="glass rounded-xl p-4 min-w-0">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0 space-y-1">
                  <div className="flex items-center gap-2 min-w-0">
                    <h3 className="text-lg font-semibold text-white break-words leading-tight">{product.name}</h3>
                    <span className="rounded-md bg-white/10 px-2 py-0.5 text-xs text-gray-300 whitespace-nowrap">
                      #{product.product_id}
                    </span>
                  </div>
                  <p className="text-sm text-gray-400 break-words">{categoryPath(product)}</p>
                  {(product.brand_name || product.manufacturer_name) && (
                    <p className="text-xs text-gray-500 break-words">
                      {product.brand_name || t('admin.products.unknownBrand')}
                      {product.manufacturer_name ? ` • ${product.manufacturer_name}` : ''}
                    </p>
                  )}
                  {product.description && (
                    <p className="text-xs text-gray-400 break-words line-clamp-3">{product.description}</p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleViewProduct(product.product_id)}
                    className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                    title={t('casesPage.details')}
                  >
                    <Eye className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleEditProduct(product.product_id)}
                    className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                    title={t('common.edit')}
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleOpenDevicesModal(product.product_id, product.name)}
                    className="rounded-lg bg-cyan-600/80 p-2 text-white transition hover:bg-cyan-600"
                    title={t('admin.products.manageDevices')}
                  >
                    <Box className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => setDependenciesModal({ productId: product.product_id, productName: product.name })}
                    className="rounded-lg bg-purple-600/80 p-2 text-white transition hover:bg-purple-600"
                    title={t('admin.products.manageDependencies')}
                  >
                    <GitBranch className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(product.product_id, product.name)}
                    className="rounded-lg bg-red-600/80 p-2 text-white transition hover:bg-red-600"
                    title={t('common.delete')}
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="mt-2">
          <DeviceTreeTab />
        </div>
      )}

      {modalOpen && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-3xl w-full max-h-[90vh] overflow-y-auto">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-2xl font-bold text-white">
                  {editingProduct ? t('admin.products.editProduct') : t('admin.products.newProduct')}
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
                    {t('common.name')} <span className="text-accent-red">*</span>
                  </label>
                  <input
                    type="text"
                    value={formData.name}
                    onChange={event => setFormData({ ...formData, name: event.target.value })}
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    placeholder={t('common.name')}
                    title={t('common.name')}
                    required
                  />
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">{t('common.description')}</label>
                  <textarea
                    value={formData.description}
                    onChange={event => setFormData({ ...formData, description: event.target.value })}
                    rows={3}
                    className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    placeholder={t('common.description')}
                    title={t('common.description')}
                  />
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">{t('products.category')}</label>
                    <select
                      value={formData.category_id ?? ''}
                      onChange={event => {
                        const value = event.target.value ? Number(event.target.value) : undefined;
                        setFormData({
                          ...formData,
                          category_id: value,
                          subcategory_id: undefined,
                          subbiercategory_id: undefined,
                        });
                      }}
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white outline-none transition focus:border-accent-red"
                      title={t('products.category')}
                    >
                      <option value="">{t('admin.products.none')}</option>
                      {categories.map(category => (
                        <option key={category.category_id} value={category.category_id}>
                          {category.name}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">{t('admin.categories.levels.subcategories')}</label>
                    <select
                      value={formData.subcategory_id ?? ''}
                      onChange={event => {
                        const value = event.target.value || undefined;
                        setFormData({
                          ...formData,
                          subcategory_id: value,
                          subbiercategory_id: undefined,
                        });
                      }}
                      disabled={!formData.category_id}
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white outline-none transition focus:border-accent-red disabled:opacity-50"
                      title={t('admin.categories.levels.subcategories')}
                    >
                      <option value="">{t('admin.products.none')}</option>
                      {filteredSubcategories.map(sub => (
                        <option key={sub.subcategory_id} value={sub.subcategory_id}>
                          {sub.name}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      {t('admin.categories.levels.subSubcategories')}
                    </label>
                    <select
                      value={formData.subbiercategory_id ?? ''}
                      onChange={event => {
                        const value = event.target.value || undefined;
                        setFormData({
                          ...formData,
                          subbiercategory_id: value,
                        });
                      }}
                      disabled={!formData.subcategory_id}
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white outline-none transition focus:border-accent-red disabled:opacity-50"
                      title={t('admin.categories.levels.subSubcategories')}
                    >
                      <option value="">{t('admin.products.none')}</option>
                      {filteredSubbiercategories.map(subbier => (
                        <option key={subbier.subbiercategory_id} value={subbier.subbiercategory_id}>
                          {subbier.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">{t('products.brand')}</label>
                    <select
                      value={formData.brand_id ?? ''}
                      onChange={event => {
                        const value = event.target.value ? Number(event.target.value) : undefined;
                        let manufacturerId = formData.manufacturer_id;
                        if (value) {
                          const brand = brandsByName.find(b => b.brand_id === value);
                          if (brand?.manufacturer_id) {
                            manufacturerId = brand.manufacturer_id;
                          }
                        }
                        setFormData({
                          ...formData,
                          brand_id: value,
                          manufacturer_id: manufacturerId ?? formData.manufacturer_id,
                        });
                      }}
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white outline-none transition focus:border-accent-red"
                      title={t('products.brand')}
                    >
                      <option value="">{t('admin.products.none')}</option>
                      {brandsByName.map(brand => (
                        <option key={brand.brand_id} value={brand.brand_id}>
                          {brand.name}
                          {brand.manufacturer_name ? ` • ${brand.manufacturer_name}` : ''}
                        </option>
                      ))}
                    </select>
                  </div>

                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">{t('products.manufacturer')}</label>
                    <select
                      value={formData.manufacturer_id ?? ''}
                      onChange={event => {
                        const value = event.target.value ? Number(event.target.value) : undefined;
                        setFormData({
                          ...formData,
                          manufacturer_id: value,
                        });
                      }}
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white outline-none transition focus:border-accent-red"
                      title={t('products.manufacturer')}
                    >
                      <option value="">{t('admin.products.none')}</option>
                      {manufacturerOptions.map(manufacturer => (
                        <option key={manufacturer.manufacturer_id} value={manufacturer.manufacturer_id}>
                          {manufacturer.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      {t('admin.products.pricePerDay')} (€)
                    </label>
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      value={formData.item_cost_per_day ?? ''}
                      onChange={event =>
                        setFormData({
                          ...formData,
                          item_cost_per_day: parseNumber(event.target.value),
                        })
                      }
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                      title={t('admin.products.pricePerDay')}
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      {t('admin.products.positionInCategory')}
                    </label>
                    <input
                      type="number"
                      value={formData.pos_in_category ?? ''}
                      onChange={event =>
                        setFormData({
                          ...formData,
                          pos_in_category: parseInteger(event.target.value),
                        })
                      }
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                      title={t('admin.products.positionInCategory')}
                    />
                  </div>
                </div>

                <div className="space-y-4 rounded-xl border border-white/10 p-4">
                  <h3 className="text-sm font-semibold text-white">{t('modals.productDetail.physicalProperties')}</h3>
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
                    {[
                      { label: t('cases.weight'), key: 'weight' as const },
                      { label: t('casesPage.height'), key: 'height' as const },
                      { label: t('casesPage.width'), key: 'width' as const },
                      { label: t('casesPage.depth'), key: 'depth' as const },
                    ].map(field => (
                      <div key={field.key}>
                        <label className="mb-2 block text-sm font-semibold text-white">
                          {field.label}
                        </label>
                        <input
                          type="number"
                          step="0.01"
                          min="0"
                          value={formData[field.key] ?? ''}
                          onChange={event =>
                            setFormData({
                              ...formData,
                              [field.key]: parseNumber(event.target.value),
                            })
                          }
                          className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                          title={field.label}
                        />
                      </div>
                    ))}
                  </div>
                </div>

                <div className="space-y-4 rounded-xl border border-white/10 p-4">
                  <h3 className="text-sm font-semibold text-white">{t('modals.productDetail.technicalDetails')}</h3>
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                    <div>
                      <label className="mb-2 block text-sm font-semibold text-white">
                        {t('modals.productDetail.maintenanceInterval')} ({t('modals.productDetail.days', { count: 1 })})
                      </label>
                      <input
                        type="number"
                        min="0"
                        value={formData.maintenance_interval ?? ''}
                        onChange={event =>
                          setFormData({
                            ...formData,
                            maintenance_interval: parseInteger(event.target.value),
                          })
                        }
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        title={t('modals.productDetail.maintenanceInterval')}
                      />
                    </div>
                    <div>
                      <label className="mb-2 block text-sm font-semibold text-white">
                        {t('modals.productDetail.powerConsumption')} (W)
                      </label>
                      <input
                        type="number"
                        step="0.01"
                        min="0"
                        value={formData.power_consumption ?? ''}
                        onChange={event =>
                          setFormData({
                            ...formData,
                            power_consumption: parseNumber(event.target.value),
                          })
                        }
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        title={t('modals.productDetail.powerConsumption')}
                      />
                    </div>
                  </div>
                </div>

                <div className="space-y-4 rounded-xl border border-white/10 p-4">
                  <h3 className="text-sm font-semibold text-white">{t('admin.products.typeInventory')}</h3>

                  <div className="flex gap-4 mb-4">
                    <label className="flex items-center gap-2 text-white cursor-pointer">
                      <input
                        type="checkbox"
                        checked={formData.is_accessory || false}
                        onChange={e => setFormData({ ...formData, is_accessory: e.target.checked })}
                        className="w-5 h-5 rounded border-white/20 bg-white/10 text-accent-red focus:ring-accent-red"
                      />
                      <span>{t('admin.products.isAccessory')}</span>
                    </label>

                    <label className="flex items-center gap-2 text-white cursor-pointer">
                      <input
                        type="checkbox"
                        checked={formData.is_consumable || false}
                        onChange={e => setFormData({ ...formData, is_consumable: e.target.checked })}
                        className="w-5 h-5 rounded border-white/20 bg-white/10 text-accent-red focus:ring-accent-red"
                      />
                      <span>{t('admin.products.isConsumable')}</span>
                    </label>
                  </div>

                  <p className="text-xs text-gray-400 mb-4">{t('admin.products.typeInventoryHelp')}</p>

                  {(formData.is_accessory || formData.is_consumable) && (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                      <div>
                        <label className="mb-2 block text-sm font-semibold text-white">
                          {t('admin.products.measurementUnit')} <span className="text-accent-red">*</span>
                        </label>
                        <select
                          value={formData.count_type_id || ''}
                          onChange={e => setFormData({
                            ...formData,
                            count_type_id: e.target.value ? Number(e.target.value) : undefined
                          })}
                          required={formData.is_accessory || formData.is_consumable}
                          className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white outline-none transition focus:border-accent-red"
                          title={t('admin.products.measurementUnit')}
                        >
                          <option value="">{t('admin.products.selectUnit')}</option>
                          {countTypes.map(ct => (
                            <option key={ct.count_type_id} value={ct.count_type_id}>
                              {ct.name} ({ct.abbreviation})
                            </option>
                          ))}
                        </select>
                      </div>

                      <div>
                        <label className="mb-2 block text-sm font-semibold text-white">
                          {t('admin.products.genericBarcode')}
                        </label>
                        <input
                          type="text"
                          value={formData.generic_barcode || ''}
                          onChange={e => setFormData({ ...formData, generic_barcode: e.target.value })}
                          placeholder={t('admin.products.genericBarcodePlaceholder')}
                          className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        />
                        <p className="text-xs text-gray-400 mt-1">{t('admin.products.singleBarcodeHint')}</p>
                      </div>

                      <div>
                        <label className="mb-2 block text-sm font-semibold text-white">
                          {t('admin.products.currentStockQuantity')}
                        </label>
                        <input
                          type="number"
                          step="0.001"
                          min="0"
                          value={formData.stock_quantity ?? ''}
                          onChange={e => setFormData({
                            ...formData,
                            stock_quantity: parseNumber(e.target.value)
                          })}
                          placeholder="0"
                          className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        />
                      </div>

                      <div>
                        <label className="mb-2 block text-sm font-semibold text-white">
                          {t('admin.products.minimumStockLevel')}
                        </label>
                        <input
                          type="number"
                          step="0.001"
                          min="0"
                          value={formData.min_stock_level ?? ''}
                          onChange={e => setFormData({
                            ...formData,
                            min_stock_level: parseNumber(e.target.value)
                          })}
                          placeholder={t('admin.products.lowStockThreshold')}
                          className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        />
                      </div>

                      <div>
                        <label className="mb-2 block text-sm font-semibold text-white">
                          {t('admin.products.pricePerUnit')} (€)
                        </label>
                        <input
                          type="number"
                          step="0.01"
                          min="0"
                          value={formData.price_per_unit ?? ''}
                          onChange={e => setFormData({
                            ...formData,
                            price_per_unit: parseNumber(e.target.value)
                          })}
                          placeholder="0.00"
                          className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                        />
                      </div>
                    </div>
                  )}
                </div>

                <div className="space-y-4 rounded-xl border border-white/10 p-4">
                  <h3 className="text-sm font-semibold text-white">
                    {editingProduct ? t('admin.products.manageDevices') : t('admin.products.createDevicesOptional')}
                  </h3>

                  {editingProduct && (
                    <div className="space-y-3">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-gray-300">
                          {t('admin.products.assignedDevices', { count: productDevices.length })}
                        </span>
                        {loadingDevices && (
                          <span className="text-xs text-gray-400">{t('common.loading')}</span>
                        )}
                      </div>

                      {productDevices.length > 0 && (
                        <div className="max-h-48 overflow-y-auto space-y-2 rounded-lg bg-white/5 p-3">
                          {productDevices.map(device => (
                            <div
                              key={device.device_id}
                              className={`flex items-center justify-between rounded-lg px-3 py-2 transition ${devicesToDelete.has(device.device_id)
                                ? 'bg-red-500/20 border border-red-500/50'
                                : 'bg-white/5 hover:bg-white/10'
                                }`}
                            >
                              <div className="flex-1">
                                <span className="text-sm font-medium text-white">
                                  {device.device_id}
                                </span>
                                <span className="ml-2 text-xs text-gray-400">
                                  {device.status}
                                </span>
                              </div>
                              <button
                                type="button"
                                onClick={() => handleRemoveDevice(device.device_id)}
                                className={`rounded-lg px-3 py-1 text-xs font-semibold transition ${devicesToDelete.has(device.device_id)
                                  ? 'bg-gray-600 text-white hover:bg-gray-700'
                                  : 'bg-red-600/80 text-white hover:bg-red-600'
                                  }`}
                              >
                                {devicesToDelete.has(device.device_id) ? t('admin.products.keep') : t('common.remove')}
                              </button>
                            </div>
                          ))}
                        </div>
                      )}

                      {devicesToDelete.size > 0 && (
                        <p className="text-xs text-red-400">
                          {t('admin.products.markedDeleteOnSave', { count: devicesToDelete.size })}
                        </p>
                      )}
                    </div>
                  )}

                  <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div>
                      <label className="mb-2 block text-sm font-semibold text-white">
                        {editingProduct ? t('admin.products.deviceCountAdd') : t('admin.products.deviceCount')}
                      </label>
                      <input
                        type="number"
                        min="0"
                        value={formData.device_quantity ?? ''}
                        onChange={event =>
                          setFormData({
                            ...formData,
                            device_quantity: parseInteger(event.target.value),
                          })
                        }
                        placeholder={t('admin.products.deviceCountPlaceholder')}
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                      />
                    </div>
                    <div>
                      <label className="mb-2 block text-sm font-semibold text-white">
                        {t('admin.products.devicePrefix')}
                      </label>
                      <input
                        type="text"
                        value={formData.device_prefix ?? ''}
                        onChange={event =>
                          setFormData({
                            ...formData,
                            device_prefix: event.target.value,
                          })
                        }
                        placeholder={t('admin.products.devicePrefixPlaceholder')}
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                      />
                    </div>
                  </div>

                  {editingProduct && (
                    <button
                      type="button"
                      onClick={handleAddDevices}
                      disabled={!formData.device_quantity || formData.device_quantity <= 0}
                      className="w-full rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {t('admin.products.addDevicesNow')}
                    </button>
                  )}

                  <p className="text-xs text-gray-400">
                    {t('admin.products.autoNumberingHint', { prefix: formData.device_prefix || 'XXX' })}
                  </p>
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
                    {submitting ? t('common.saving') : editingProduct ? t('common.save') : t('common.create')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </ModalPortal>
      )}

      <ProductDetailModal
        product={viewProduct ? {
          product_id: viewProduct.product_id,
          name: viewProduct.name,
          description: viewProduct.description || undefined,
          website_visible: viewProduct.website_visible,
          website_thumbnail: viewProduct.website_thumbnail || undefined,
          website_images: viewProduct.website_images || [],
          category_name: viewProduct.category_name || undefined,
          subcategory_name: viewProduct.subcategory_name || undefined,
          subbiercategory_name: viewProduct.subbiercategory_name || undefined,
          brand_name: viewProduct.brand_name || undefined,
          manufacturer_name: viewProduct.manufacturer_name || undefined,
          item_cost_per_day: viewProduct.item_cost_per_day || undefined,
          maintenance_interval: viewProduct.maintenance_interval || undefined,
          weight: viewProduct.weight || undefined,
          height: viewProduct.height || undefined,
          width: viewProduct.width || undefined,
          depth: viewProduct.depth || undefined,
          power_consumption: viewProduct.power_consumption || undefined,
          pos_in_category: viewProduct.pos_in_category || undefined,
          is_accessory: viewProduct.is_accessory,
          is_consumable: viewProduct.is_consumable,
          stock_quantity: viewProduct.stock_quantity || undefined,
          min_stock_level: viewProduct.min_stock_level || undefined,
          generic_barcode: viewProduct.generic_barcode || undefined,
          price_per_unit: viewProduct.price_per_unit || undefined,
          count_type_abbreviation: viewProduct.count_type_abbr || undefined,
        } : null}
        isOpen={!!viewProduct}
        onClose={closeDetailModal}
      />

      {/* Product Dependencies Modal */}
      {dependenciesModal && (
        <ProductDependenciesModal
          productId={dependenciesModal.productId}
          productName={dependenciesModal.productName}
          onClose={() => setDependenciesModal(null)}
        />
      )}

      {/* Product Devices Modal */}
      {/* onLocate, onOpenZone, and onOpenDevice are intentional no-ops here: */}
      {/* LED/zone navigation is not applicable in the product admin context. */}
      <ProductDevicesModal
        isOpen={!!devicesModal}
        onClose={closeDevicesModal}
        productName={devicesModal?.productName ?? ''}
        devices={devicesModalDevices}
        loading={loadingDevicesModal}
        onLocate={() => {}}
        onOpenZone={() => {}}
        onOpenDevice={() => {}}
      />
    </div>
  );
}
