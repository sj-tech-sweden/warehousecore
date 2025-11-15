import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Eye,
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
  const [products, setProducts] = useState<Product[]>([]);
  const [loadingProducts, setLoadingProducts] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewProduct, setViewProduct] = useState<Product | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [editingProduct, setEditingProduct] = useState<number | null>(null);
  const [formData, setFormData] = useState<ProductFormData>(initialFormData);
  const [categories, setCategories] = useState<Category[]>([]);
  const [subcategories, setSubcategories] = useState<Subcategory[]>([]);
  const [subbiercategories, setSubbiercategories] = useState<Subbiercategory[]>([]);
  const [brands, setBrands] = useState<Brand[]>([]);
  const [manufacturers, setManufacturers] = useState<Manufacturer[]>([]);
  const [metadataLoaded, setMetadataLoaded] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'cards'>('table');
  const [searchTerm, setSearchTerm] = useState('');
  const [categoryFilter, setCategoryFilter] = useState<number | ''>('');
  const [refreshing, setRefreshing] = useState(false);
  const scrollPosition = useRef(0);

  const debouncedSearch = useDebouncedValue(searchTerm, 300);

  useEffect(() => {
    const html = document.documentElement;
    const body = document.body;
    const anyModalOpen = modalOpen || !!viewProduct;
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
      const [catRes, subRes, subbierRes, brandRes, manufacturerRes] = await Promise.all([
        api.get<Category[]>('/admin/categories'),
        api.get<Subcategory[]>('/admin/subcategories'),
        api.get<Subbiercategory[]>('/admin/subbiercategories'),
        api.get<Brand[]>('/admin/brands'),
        api.get<Manufacturer[]>('/admin/manufacturers'),
      ]);

      setCategories(catRes.data || []);
      setSubcategories(subRes.data || []);
      setSubbiercategories(subbierRes.data || []);
      setBrands(brandRes.data || []);
      setManufacturers(manufacturerRes.data || []);
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

  const closeModal = useCallback(() => {
    setModalOpen(false);
    setEditingProduct(null);
    setFormData(initialFormData);
  }, []);

  const closeDetailModal = () => {
    setViewProduct(null);
    setLoadingDetail(false);
  };

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
    } catch (error) {
      console.error('Failed to load product details:', error);
      window.alert('Produkt konnte nicht geladen werden.');
    }
  };

  const handleDelete = async (productId: number, name: string) => {
    if (!window.confirm(`Produkt "${name}" wirklich löschen?`)) {
      return;
    }

    try {
      await api.delete(`/admin/products/${productId}`);
      await fetchProducts(searchTerm, categoryFilter);
    } catch (error) {
      console.error('Failed to delete product:', error);
      window.alert('Fehler beim Löschen des Produkts.');
    }
  };

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (!formData.name.trim()) {
      window.alert('Der Produktname ist erforderlich.');
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
    };

    try {
      let productId = editingProduct;

      if (editingProduct) {
        await api.put(`/admin/products/${editingProduct}`, payload);
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
            window.alert('Produkt erstellt, aber Geräte konnten nicht angelegt werden.');
          }
        }
      }

      await fetchProducts(searchTerm, categoryFilter);
      closeModal();
    } catch (error) {
      console.error('Failed to save product:', error);
      window.alert('Fehler beim Speichern des Produkts.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleViewProduct = async (productId: number) => {
    try {
      setLoadingDetail(true);
      const { data } = await api.get<Product>(`/admin/products/${productId}`);
      setViewProduct(data);
    } catch (error) {
      console.error('Failed to load product details:', error);
      window.alert('Produkt konnte nicht geladen werden.');
    } finally {
      setLoadingDetail(false);
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
      .join(' · ') || 'Keiner Kategorie zugeordnet';

  const formatMeasurement = (value?: number | null, unit?: string) =>
    value != null ? `${value.toFixed(2)}${unit ? ` ${unit}` : ''}` : '—';

  const formatCurrency = (value?: number | null) =>
    value != null ? `${value.toFixed(2)} €` : '—';

  const averageDailyPrice =
    sortedProducts.length === 0
      ? null
      : sortedProducts.reduce((sum, product) => sum + (product.item_cost_per_day ?? 0), 0) /
        sortedProducts.length;

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        <div>
          <h2 className="text-xl font-bold text-white">Produkte verwalten</h2>
          <p className="text-sm text-gray-400">
            {sortedProducts.length} Produkte geladen
            {hasActiveFilters ? ' • Filter aktiv' : ''}
          </p>
        </div>

        <div className="flex flex-col gap-3">
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <input
                value={searchTerm}
                onChange={event => setSearchTerm(event.target.value)}
                placeholder="Suchen (Name, Beschreibung …)"
                className="w-full rounded-lg bg-white/10 py-2 pl-9 pr-3 text-sm text-white placeholder-gray-500 outline-none transition focus:bg-white/15 focus:ring-1 focus:ring-accent-red"
              />
            </div>

            <select
              value={categoryFilter}
              onChange={event => handleCategoryFilterChange(event.target.value)}
              className="rounded-lg bg-white/10 px-3 py-2 text-sm text-white outline-none transition focus:bg-white/15 focus:ring-1 focus:ring-accent-red"
            >
              <option value="">Alle Kategorien</option>
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
                Filter zurücksetzen
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
              <span className="hidden sm:inline">Tabelle</span>
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
              <span className="hidden sm:inline">Karten</span>
            </button>
            <button
              type="button"
              onClick={handleRefresh}
              disabled={refreshing}
              className="flex items-center gap-2 rounded-lg bg-white/10 px-3 py-2 text-sm font-semibold text-white transition hover:bg-white/20 disabled:opacity-50"
            >
              <RefreshCcw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
              <span className="hidden sm:inline">Aktualisieren</span>
            </button>
            <button
              onClick={handleOpenCreateModal}
              className="flex items-center gap-2 rounded-xl bg-accent-red px-4 py-2 font-semibold text-white hover:shadow-lg"
            >
              <Plus className="h-4 w-4" />
              <span className="hidden sm:inline">Neues Produkt</span>
              <span className="sm:hidden">Neu</span>
            </button>
          </div>
        </div>
      </div>

      <div className="rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-gray-300 backdrop-blur">
        <div className="flex flex-wrap items-center gap-6">
          <span>
            <strong className="text-white">{sortedProducts.length}</strong> Produkte
          </span>
          <span>
            Durchschnittlicher Tagespreis:{' '}
            <strong className="text-white">
              {averageDailyPrice != null ? `${averageDailyPrice.toFixed(2)} €` : '—'}
            </strong>
          </span>
          {categoryFilter !== '' && (
            <span>
              Gefiltert nach Kategorie:{' '}
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
              Suchbegriff: <strong className="text-white">"{debouncedSearch}"</strong>
            </span>
          )}
        </div>
      </div>

      {loadingProducts ? (
        <div className="glass rounded-xl p-8 text-center text-gray-400">
          Lade Produkte …
        </div>
      ) : sortedProducts.length === 0 ? (
        <div className="glass rounded-xl p-8 text-center text-gray-400">
          <Package className="mx-auto mb-4 h-12 w-12 text-gray-600" />
          Keine Produkte gefunden
          {hasActiveFilters ? ' – bitte Filter anpassen.' : '.'}
        </div>
      ) : viewMode === 'table' ? (
        <div className="overflow-hidden rounded-xl border border-white/10 bg-white/5">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-white/10 text-sm text-gray-200">
              <thead className="bg-white/5 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">Produkt</th>
                  <th className="px-4 py-3 text-left font-semibold">Kategorie</th>
                  <th className="px-4 py-3 text-left font-semibold">Brand / Hersteller</th>
                  <th className="px-4 py-3 text-left font-semibold">Preis pro Tag</th>
                  <th className="px-4 py-3 text-right font-semibold">Aktionen</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {sortedProducts.map(product => (
                  <tr key={product.product_id} className="hover:bg-white/5">
                    <td className="px-4 py-3 align-top">
                      <div className="flex flex-col">
                        <span className="text-white font-medium">{product.name}</span>
                        <span className="text-xs text-gray-500">
                          ID: {product.product_id}
                          {product.pos_in_category != null
                            ? ` • Position ${product.pos_in_category}`
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
                        <span>{product.brand_name || '—'}</span>
                        <span className="text-xs text-gray-500">
                          {product.manufacturer_name || 'Kein Hersteller hinterlegt'}
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
                          title="Details anzeigen"
                        >
                          <Eye className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleEditProduct(product.product_id)}
                          className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                          title="Bearbeiten"
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(product.product_id, product.name)}
                          className="rounded-lg bg-red-600/80 p-2 text-white transition hover:bg-red-600"
                          title="Löschen"
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
      ) : (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {sortedProducts.map(product => (
            <div key={product.product_id} className="glass rounded-xl p-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold text-white">{product.name}</h3>
                    <span className="rounded-md bg-white/10 px-2 py-0.5 text-xs text-gray-300">
                      #{product.product_id}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-gray-400">{categoryPath(product)}</p>
                  {(product.brand_name || product.manufacturer_name) && (
                    <p className="mt-1 text-xs text-gray-500">
                      {product.brand_name || 'Unbekannte Marke'}
                      {product.manufacturer_name ? ` • ${product.manufacturer_name}` : ''}
                    </p>
                  )}
                  {product.description && (
                    <p className="mt-2 text-xs text-gray-400">{product.description}</p>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => handleViewProduct(product.product_id)}
                    className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                    title="Details anzeigen"
                  >
                    <Eye className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleEditProduct(product.product_id)}
                    className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                    title="Bearbeiten"
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button
                    onClick={() => handleDelete(product.product_id, product.name)}
                    className="rounded-lg bg-red-600/80 p-2 text-white transition hover:bg-red-600"
                    title="Löschen"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {modalOpen && (
        <div
          className="fixed inset-0 z-[5000] flex items-center justify-center bg-black/70 backdrop-blur-sm px-4 py-10"
          onClick={closeModal}
        >
          <div
            className="glass-dark relative flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl shadow-2xl"
            onClick={event => event.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-white/10 px-6 py-5">
              <h2 className="text-2xl font-bold text-white">
                {editingProduct ? 'Produkt bearbeiten' : 'Neues Produkt'}
              </h2>
              <button
                onClick={closeModal}
                className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-white/10 hover:text-white"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto px-6 py-6 space-y-6">
              <div>
                <label className="mb-2 block text-sm font-semibold text-white">
                  Name <span className="text-accent-red">*</span>
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={event => setFormData({ ...formData, name: event.target.value })}
                  className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                  required
                />
              </div>

              <div>
                <label className="mb-2 block text-sm font-semibold text-white">Beschreibung</label>
                <textarea
                  value={formData.description}
                  onChange={event => setFormData({ ...formData, description: event.target.value })}
                  rows={3}
                  className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                />
              </div>

              <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">Kategorie</label>
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
                  >
                    <option value="">Keine</option>
                    {categories.map(category => (
                      <option key={category.category_id} value={category.category_id}>
                        {category.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">Unterkategorie</label>
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
                  >
                    <option value="">Keine</option>
                    {filteredSubcategories.map(sub => (
                      <option key={sub.subcategory_id} value={sub.subcategory_id}>
                        {sub.name}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">
                    Sub-Unterkategorie
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
                  >
                    <option value="">Keine</option>
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
                  <label className="mb-2 block text-sm font-semibold text-white">Brand</label>
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
                  >
                    <option value="">Keine</option>
                    {brandsByName.map(brand => (
                      <option key={brand.brand_id} value={brand.brand_id}>
                        {brand.name}
                        {brand.manufacturer_name ? ` • ${brand.manufacturer_name}` : ''}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">Manufacturer</label>
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
                  >
                    <option value="">Keine</option>
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
                    Preis pro Tag (€)
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
                  />
                </div>
                <div>
                  <label className="mb-2 block text-sm font-semibold text-white">
                    Position in Kategorie
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
                  />
                </div>
              </div>

              <div className="space-y-4 rounded-xl border border-white/10 p-4">
                <h3 className="text-sm font-semibold text-white">Physische Eigenschaften</h3>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-4">
                  {[
                    { label: 'Gewicht (kg)', key: 'weight' as const },
                    { label: 'Höhe (cm)', key: 'height' as const },
                    { label: 'Breite (cm)', key: 'width' as const },
                    { label: 'Tiefe (cm)', key: 'depth' as const },
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
                      />
                    </div>
                  ))}
                </div>
              </div>

              <div className="space-y-4 rounded-xl border border-white/10 p-4">
                <h3 className="text-sm font-semibold text-white">Technische Angaben</h3>
                <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      Wartungsintervall (Tage)
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
                    />
                  </div>
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      Leistungsaufnahme (W)
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
                    />
                  </div>
                </div>
              </div>

              {!editingProduct && (
                <div className="space-y-4 rounded-xl border border-white/10 p-4">
                  <h3 className="text-sm font-semibold text-white">Geräte erstellen (optional)</h3>
                  <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    <div>
                      <label className="mb-2 block text-sm font-semibold text-white">
                        Anzahl Geräte
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
                        placeholder="z. B. 10"
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                      />
                    </div>
                    <div>
                      <label className="mb-2 block text-sm font-semibold text-white">
                        Geräte-Präfix
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
                        placeholder="z. B. LED"
                        className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                      />
                    </div>
                  </div>
                  <p className="text-xs text-gray-400">
                    Geräte werden automatisch mit aufsteigender Nummerierung erstellt (z. B. {formData.device_prefix || 'XXX'}0001).
                  </p>
                </div>
              )}

              <div className="flex flex-col gap-3 pt-4 sm:flex-row">
                <button
                  type="submit"
                  disabled={submitting}
                  className="flex-1 rounded-lg bg-accent-red px-4 py-3 font-semibold text-white transition hover:bg-accent-red/90 disabled:opacity-50"
                >
                  {submitting ? 'Speichern …' : editingProduct ? 'Aktualisieren' : 'Erstellen'}
                </button>
                <button
                  type="button"
                  onClick={closeModal}
                  className="rounded-lg bg-white/10 px-4 py-3 font-semibold text-white transition hover:bg-white/20"
                >
                  Abbrechen
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {viewProduct && (
        <div
          className="fixed inset-0 z-[5200] flex items-center justify-center bg-black/80 backdrop-blur-sm px-4 py-10"
          onClick={closeDetailModal}
        >
          <div
            className="glass-dark relative flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl shadow-2xl"
            onClick={event => event.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-white/10 px-6 py-5">
              <div>
                <h2 className="text-2xl font-bold text-white">{viewProduct.name}</h2>
                <p className="text-sm text-gray-400">Produkt-ID #{viewProduct.product_id}</p>
              </div>
              <button
                onClick={closeDetailModal}
                className="rounded-lg p-2 text-gray-400 transition hover:bg-white/10 hover:text-white"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto px-6 py-6">
              {loadingDetail ? (
                <div className="py-6 text-center text-gray-400">Lade Details …</div>
              ) : (
                <div className="space-y-6 text-sm text-gray-200">
                  <div>
                    <h3 className="text-xs uppercase tracking-wide text-gray-400">Beschreibung</h3>
                    <p className="mt-1 text-base text-white">
                      {viewProduct.description || 'Keine Beschreibung hinterlegt.'}
                    </p>
                  </div>

                  <div className="grid gap-4 rounded-xl bg-white/5 p-4 md:grid-cols-2">
                    <div>
                      <span className="text-xs uppercase text-gray-400">Kategorie</span>
                      <p className="text-white">{categoryPath(viewProduct)}</p>
                    </div>
                    <div>
                      <span className="text-xs uppercase text-gray-400">Brand / Hersteller</span>
                      <p className="text-white">
                        {viewProduct.brand_name || '—'}
                        {viewProduct.manufacturer_name ? ` • ${viewProduct.manufacturer_name}` : ''}
                      </p>
                    </div>
                    <div>
                      <span className="text-xs uppercase text-gray-400">Preis pro Tag</span>
                      <p className="text-white">{formatCurrency(viewProduct.item_cost_per_day)}</p>
                    </div>
                    <div>
                      <span className="text-xs uppercase text-gray-400">Wartungsintervall</span>
                      <p className="text-white">
                        {viewProduct.maintenance_interval != null
                          ? `${viewProduct.maintenance_interval} Tage`
                          : '—'}
                      </p>
                    </div>
                  </div>

                  <div className="grid gap-4 rounded-xl bg-white/5 p-4 md:grid-cols-4">
                    <div>
                      <p className="text-xs uppercase text-gray-400">Gewicht</p>
                      <p className="text-white">{formatMeasurement(viewProduct.weight, 'kg')}</p>
                    </div>
                    <div>
                      <p className="text-xs uppercase text-gray-400">Höhe</p>
                      <p className="text-white">{formatMeasurement(viewProduct.height, 'cm')}</p>
                    </div>
                    <div>
                      <p className="text-xs uppercase text-gray-400">Breite</p>
                      <p className="text-white">{formatMeasurement(viewProduct.width, 'cm')}</p>
                    </div>
                    <div>
                      <p className="text-xs uppercase text-gray-400">Tiefe</p>
                      <p className="text-white">{formatMeasurement(viewProduct.depth, 'cm')}</p>
                    </div>
                  </div>

                  <div className="grid gap-4 rounded-xl bg-white/5 p-4 md:grid-cols-2">
                    <div>
                      <p className="text-xs uppercase text-gray-400">Leistungsaufnahme</p>
                      <p className="text-white">
                        {formatMeasurement(viewProduct.power_consumption, 'W')}
                      </p>
                    </div>
                    <div>
                      <p className="text-xs uppercase text-gray-400">Position in Kategorie</p>
                      <p className="text-white">
                        {viewProduct.pos_in_category != null
                          ? viewProduct.pos_in_category
                          : '—'}
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
