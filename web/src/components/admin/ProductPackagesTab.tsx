import { useEffect, useRef, useState } from 'react';
import {
  Plus,
  Pencil,
  Trash2,
  X,
  Search,
  RefreshCcw,
  Eye,
} from 'lucide-react';
import { api } from '../../lib/api';
import { ModalPortal } from '../ModalPortal';

interface ProductPackage {
  package_id: number;
  product_id: number;
  package_code: string;
  name: string;
  description?: string | null;
  price?: number | string | null;
  total_items: number;
  created_at: string;
  updated_at: string;
  aliases?: string[];
  category_id?: number | null;
  category_name?: string | null;
  subcategory_id?: string | null;
}

interface PackageItemDetail {
  package_item_id: number;
  product_id: number;
  product_name: string;
  quantity: number;
  category_name?: string | null;
  brand_name?: string | null;
}

interface ProductPackageWithItems extends ProductPackage {
  items?: PackageItemDetail[];
}

interface Product {
  product_id: number;
  name: string;
  category_name?: string | null;
  brand_name?: string | null;
}

interface PackageFormData {
  name: string;
  description: string;
  price: string;
  items: Array<{
    product_id: number;
    quantity: number;
  }>;
  aliases: string[];
  category_id: number | '';
  subcategory_id: string;
  subbiercategory_id: string;
  device_quantity?: number;
  device_prefix?: string;
}

interface Device {
  device_id: string;
  product_id?: number;
  status: string;
  serial_number?: string;
  barcode?: string;
}

const initialFormData: PackageFormData = {
  name: '',
  description: '',
  price: '',
  items: [],
  aliases: [],
  category_id: '',
  subcategory_id: '',
  subbiercategory_id: '',
  device_quantity: undefined,
  device_prefix: '',
};

function formatPrice(value?: number | string | null, fallback = '-') {
  if (value === undefined || value === null || value === '') {
    return fallback;
  }

  const numericValue = typeof value === 'number' ? value : parseFloat(value);
  if (Number.isNaN(numericValue)) {
    return fallback;
  }

  return `${numericValue.toFixed(2)} €`;
}

export function ProductPackagesTab() {
  const [packages, setPackages] = useState<ProductPackage[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [viewPackage, setViewPackage] = useState<ProductPackageWithItems | null>(null);
  const [editingPackage, setEditingPackage] = useState<number | null>(null);
  const [formData, setFormData] = useState<PackageFormData>(initialFormData);
  const [searchTerm, setSearchTerm] = useState('');
  const [products, setProducts] = useState<Product[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<number | ''>('');
  const [selectedQuantity, setSelectedQuantity] = useState(1);
  const [aliasInput, setAliasInput] = useState('');
  const [formError, setFormError] = useState<string | null>(null);
  const scrollPosition = useRef(0);
  const viewPackagePriceDisplay = viewPackage ? formatPrice(viewPackage.price, '') : '';
  const [categories, setCategories] = useState<Array<{ categoryID: number; name: string }>>([]);
  const [packageDevices, setPackageDevices] = useState<Device[]>([]);
  const [devicesToDelete, setDevicesToDelete] = useState<Set<string>>(new Set());
  const [loadingDevices, setLoadingDevices] = useState(false);

  const fetchPackages = async () => {
    try {
      setLoading(true);
      const params = searchTerm ? { search: searchTerm } : {};
      const response = await api.get('/admin/product-packages', { params });
      setPackages(response.data || []);
    } catch (error) {
      console.error('Failed to fetch product packages:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchProducts = async () => {
    try {
      const response = await api.get('/admin/products');
      setProducts(response.data || []);
    } catch (error) {
      console.error('Failed to fetch products:', error);
    }
  };

  const fetchCategories = async () => {
    try {
      const response = await api.get('/categories');
      setCategories(response.data || []);
    } catch (error) {
      console.error('Failed to fetch categories:', error);
    }
  };

  useEffect(() => {
    fetchPackages();
  }, [searchTerm]);

  useEffect(() => {
    if ((modalOpen || viewPackage) && products.length === 0) {
      fetchProducts();
    }
    if ((modalOpen || viewPackage) && categories.length === 0) {
      fetchCategories();
    }
  }, [modalOpen, viewPackage, products.length, categories.length]);

  useEffect(() => {
    const html = document.documentElement;
    const body = document.body;
    if (modalOpen || viewPackage) {
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
  }, [modalOpen, viewPackage]);

  const loadPackageDevices = async (productId: number) => {
    setLoadingDevices(true);
    try {
      const { data } = await api.get<Device[]>(`/admin/products/${productId}/devices`);
      setPackageDevices(data || []);
      setDevicesToDelete(new Set());
    } catch (error) {
      console.error('Failed to load package devices:', error);
      setPackageDevices([]);
    } finally {
      setLoadingDevices(false);
    }
  };

  const handleAddDevices = async (productId: number) => {
    const quantity = formData.device_quantity;
    if (!quantity || quantity <= 0) {
      window.alert('Bitte eine gültige Anzahl eingeben.');
      return;
    }

    try {
      await api.post(`/admin/products/${productId}/devices`, {
        product_id: productId,
        quantity: quantity,
        prefix: formData.device_prefix || '',
      });

      // Reload devices
      await loadPackageDevices(productId);

      // Reset device creation fields
      setFormData({ ...formData, device_quantity: undefined, device_prefix: '' });
    } catch (error) {
      console.error('Failed to add devices:', error);
      window.alert('Fehler beim Hinzufügen der Geräte.');
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

  const handleOpenModal = async (pkg?: ProductPackage) => {
    if (pkg) {
      setEditingPackage(pkg.package_id);
      // Fetch full package details with items
      try {
        const res = await api.get<ProductPackageWithItems>(`/admin/product-packages/${pkg.package_id}`);
        const data = res.data;
        setFormData({
          name: data.name,
          description: data.description || '',
          price: data.price?.toString() || '',
          items: data.items?.map(item => ({
            product_id: item.product_id,
            quantity: item.quantity,
          })) || [],
          aliases: data.aliases || [],
          category_id: data.category_id || '',
          subcategory_id: data.subcategory_id || '',
          subbiercategory_id: '',
          device_quantity: undefined,
          device_prefix: '',
        });
        // Load devices for the package's product
        if (data.product_id) {
          await loadPackageDevices(data.product_id);
        }
      } catch (err) {
        console.error('Failed to fetch package details:', err);
      }
    } else {
      setEditingPackage(null);
      setFormData(initialFormData);
      setPackageDevices([]);
      setDevicesToDelete(new Set());
    }
    setAliasInput('');
    setFormError(null);
    setModalOpen(true);
  };

  const handleCloseModal = () => {
    setModalOpen(false);
    setEditingPackage(null);
    setFormData(initialFormData);
    setSelectedProduct('');
    setSelectedQuantity(1);
    setAliasInput('');
    setFormError(null);
    setPackageDevices([]);
    setDevicesToDelete(new Set());
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);

    if (!formData.name.trim()) {
      setFormError('Bitte gib einen Namen für das Paket an.');
      setSubmitting(false);
      return;
    }

    if (formData.items.length === 0) {
      setFormError('Ein Paket muss mindestens ein Produkt enthalten.');
      setSubmitting(false);
      return;
    }

    try {
      const payload = {
        name: formData.name,
        description: formData.description || null,
        price: formData.price ? parseFloat(formData.price) : null,
        items: formData.items,
        aliases: formData.aliases,
        category_id: formData.category_id || null,
        subcategory_id: formData.subcategory_id || null,
        subbiercategory_id: formData.subbiercategory_id || null,
      };

      let productId: number | undefined;

      if (editingPackage) {
        await api.put(`/admin/product-packages/${editingPackage}`, payload);

        // Get the product_id for this package
        const pkgResponse = await api.get<ProductPackageWithItems>(`/admin/product-packages/${editingPackage}`);
        productId = pkgResponse.data.product_id;

        // Delete devices that were marked for deletion
        if (devicesToDelete.size > 0 && productId) {
          const deletePromises = Array.from(devicesToDelete).map(deviceId =>
            api.delete(`/admin/devices/${deviceId}`)
          );
          try {
            await Promise.all(deletePromises);
          } catch (deviceError) {
            console.error('Failed to delete some devices:', deviceError);
            window.alert('Paket gespeichert, aber einige Geräte konnten nicht gelöscht werden.');
          }
        }
      } else {
        const { data } = await api.post<ProductPackageWithItems>('/admin/product-packages', payload);
        productId = data.product_id;

        // Create devices if quantity specified
        if (formData.device_quantity && formData.device_quantity > 0 && productId) {
          try {
            await api.post(`/admin/products/${productId}/devices`, {
              product_id: productId,
              quantity: formData.device_quantity,
              prefix: formData.device_prefix || '',
            });
          } catch (deviceError) {
            console.error('Failed to create devices:', deviceError);
            window.alert('Paket erstellt, aber Geräte konnten nicht angelegt werden.');
          }
        }
      }

      handleCloseModal();
      fetchPackages();
    } catch (error) {
      console.error('Failed to save product package:', error);
      const message =
        (error as any)?.response?.data?.error || 'Fehler beim Speichern des Produktpakets';
      setFormError(message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Möchten Sie dieses Produktpaket wirklich löschen?')) return;

    try {
      await api.delete(`/admin/product-packages/${id}`);
      fetchPackages();
    } catch (error) {
      console.error('Failed to delete product package:', error);
      alert('Fehler beim Löschen des Produktpakets');
    }
  };

  const handleViewPackage = async (pkg: ProductPackage) => {
    try {
      const response = await api.get<ProductPackageWithItems>(`/admin/product-packages/${pkg.package_id}`);
      setViewPackage(response.data);
    } catch (error) {
      console.error('Failed to fetch package details:', error);
    }
  };

  const handleAddItem = () => {
    if (!selectedProduct || selectedQuantity <= 0) return;

    const product = products.find(p => p.product_id === selectedProduct);
    if (!product) return;

    // Check if product already exists in items
    const existingIndex = formData.items.findIndex(item => item.product_id === selectedProduct);

    if (existingIndex >= 0) {
      // Update quantity
      const newItems = [...formData.items];
      newItems[existingIndex].quantity += selectedQuantity;
      setFormData({ ...formData, items: newItems });
    } else {
      // Add new item
      setFormData({
        ...formData,
        items: [...formData.items, { product_id: Number(selectedProduct), quantity: selectedQuantity }],
      });
    }

    setSelectedProduct('');
    setSelectedQuantity(1);
  };

  const handleRemoveItem = (index: number) => {
    const newItems = formData.items.filter((_, i) => i !== index);
    setFormData({ ...formData, items: newItems });
  };

  const handleAddAlias = () => {
    const value = aliasInput.trim();
    if (!value) return;
    const exists = formData.aliases.some(alias => alias.toLowerCase() === value.toLowerCase());
    if (exists) {
      setAliasInput('');
      return;
    }
    setFormData({ ...formData, aliases: [...formData.aliases, value] });
    setAliasInput('');
  };

  const handleRemoveAlias = (aliasToRemove: string) => {
    setFormData({
      ...formData,
      aliases: formData.aliases.filter(alias => alias !== aliasToRemove),
    });
  };

  const getProductName = (productId: number) => {
    const product = products.find(p => p.product_id === productId);
    return product?.name || `Produkt #${productId}`;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div>
          <h2 className="text-2xl font-bold text-white">Produktpakete</h2>
          <p className="text-gray-400">Verwalten Sie Produktpakete für Jobs</p>
        </div>
        <button
          onClick={() => handleOpenModal()}
          className="flex items-center gap-2 rounded-xl bg-accent-red/90 px-5 py-3 text-sm font-semibold text-white shadow-lg shadow-red-900/40 transition-all hover:bg-accent-red focus:outline-none focus:ring-2 focus:ring-red-400"
        >
          <Plus className="w-5 h-5" />
          Neues Paket
        </button>
      </div>

      {/* Search & Filters */}
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Pakete durchsuchen..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-gray-800/50 border border-gray-700 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-accent-red"
          />
        </div>
        <button
          onClick={fetchPackages}
          className="btn-secondary flex items-center gap-2"
        >
          <RefreshCcw className="w-5 h-5" />
          Aktualisieren
        </button>
      </div>

      {/* Packages Table */}
      {loading ? (
        <div className="text-center py-12 text-gray-400">Lade Produktpakete...</div>
      ) : packages.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          Keine Produktpakete gefunden. Erstellen Sie ein neues Paket.
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-700">
                <th className="text-left py-3 px-4 text-gray-400 font-semibold">Name</th>
                <th className="text-left py-3 px-4 text-gray-400 font-semibold">Beschreibung</th>
                <th className="text-right py-3 px-4 text-gray-400 font-semibold">Preis</th>
                <th className="text-right py-3 px-4 text-gray-400 font-semibold">Artikel</th>
                <th className="text-right py-3 px-4 text-gray-400 font-semibold">Aktionen</th>
              </tr>
            </thead>
            <tbody>
              {packages.map((pkg) => (
                <tr key={pkg.package_id} className="border-b border-gray-800 hover:bg-gray-800/30">
                  <td className="py-3 px-4 text-white font-medium">{pkg.name}</td>
                  <td className="py-3 px-4 text-gray-400">{pkg.description || '-'}</td>
                  <td className="py-3 px-4 text-right text-white">
                    {formatPrice(pkg.price)}
                  </td>
                  <td className="py-3 px-4 text-right text-white">{pkg.total_items}</td>
                  <td className="py-3 px-4 text-right">
                    <div className="flex justify-end gap-2">
                      <button
                        onClick={() => handleViewPackage(pkg)}
                        className="p-2 text-blue-400 hover:bg-blue-400/10 rounded-lg transition-colors"
                        title="Anzeigen"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleOpenModal(pkg)}
                        className="p-2 text-yellow-400 hover:bg-yellow-400/10 rounded-lg transition-colors"
                        title="Bearbeiten"
                      >
                        <Pencil className="w-4 h-4" />
                      </button>
                      <button
                        onClick={() => handleDelete(pkg.package_id)}
                        className="p-2 text-red-400 hover:bg-red-400/10 rounded-lg transition-colors"
                        title="Löschen"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create/Edit Modal */}
      {modalOpen && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-2xl font-bold text-white">
                  {editingPackage ? 'Paket bearbeiten' : 'Neues Paket'}
                </h3>
                <button
                  onClick={handleCloseModal}
                  className="text-gray-400 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors"
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

            {formError && (
              <div className="mb-4 rounded-xl border border-red-500/40 bg-red-500/10 px-4 py-2 text-sm text-red-300">
                {formError}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Name *
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Beschreibung
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                  rows={3}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Preis (€)
                </label>
                <input
                  type="number"
                  step="0.01"
                  value={formData.price}
                  onChange={(e) => setFormData({ ...formData, price: e.target.value })}
                  className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Kategorie
                </label>
                <select
                  value={formData.category_id}
                  onChange={(e) => setFormData({ ...formData, category_id: e.target.value ? Number(e.target.value) : '' })}
                  className="w-full px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                >
                  <option value="">Keine Kategorie</option>
                  {categories.map((cat) => (
                    <option key={cat.categoryID} value={cat.categoryID}>
                      {cat.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="border-t border-gray-700 pt-4">
                <h4 className="text-lg font-semibold text-white mb-3">Produkte</h4>

                {/* Add Item Section */}
                <div className="flex gap-2 mb-4">
                  <select
                    value={selectedProduct}
                    onChange={(e) => setSelectedProduct(Number(e.target.value))}
                    className="flex-1 px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                  >
                    <option value="">Produkt auswählen...</option>
                    {products.map((product) => (
                      <option key={product.product_id} value={product.product_id}>
                        {product.name} {product.category_name ? `(${product.category_name})` : ''}
                      </option>
                    ))}
                  </select>
                  <input
                    type="number"
                    min="1"
                    value={selectedQuantity}
                    onChange={(e) => setSelectedQuantity(parseInt(e.target.value) || 1)}
                    className="w-24 px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                    placeholder="Anzahl"
                  />
                  <button
                    type="button"
                    onClick={handleAddItem}
                    className="btn-primary flex items-center gap-2 min-w-[120px]"
                  >
                    <Plus className="w-4 h-4" />
                    Hinzufügen
                  </button>
                </div>

                {/* Items List */}
                {formData.items.length === 0 ? (
                  <p className="text-gray-400 text-center py-4">Keine Produkte hinzugefügt</p>
                ) : (
                  <div className="space-y-2">
                    {formData.items.map((item, index) => (
                      <div key={index} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                        <div>
                          <span className="text-white font-medium">{getProductName(item.product_id)}</span>
                          <span className="text-gray-400 ml-2">x {item.quantity}</span>
                        </div>
                        <button
                          type="button"
                          onClick={() => handleRemoveItem(index)}
                          className="text-red-400 hover:bg-red-400/10 p-2 rounded-lg transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="border-t border-gray-700 pt-4">
                <h4 className="text-lg font-semibold text-white mb-2">OCR-Zuordnungen</h4>
                <p className="text-sm text-gray-400 mb-3">
                  Definiere Schlagwörter oder Kürzel, die bei OCR-Erkennung automatisch diesem Paket zugeordnet werden.
                </p>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={aliasInput}
                    onChange={(e) => setAliasInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault();
                        handleAddAlias();
                      }
                    }}
                    className="flex-1 px-4 py-2 bg-gray-800 border border-gray-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-accent-red"
                    placeholder="z.B. Basic Audio Set"
                  />
                  <button
                    type="button"
                    onClick={handleAddAlias}
                    className="btn-secondary px-4"
                  >
                    Hinzufügen
                  </button>
                </div>
                {formData.aliases.length === 0 ? (
                  <p className="text-gray-500 text-sm mt-2">Noch keine Schlüsselwörter definiert.</p>
                ) : (
                  <div className="flex flex-wrap gap-2 mt-3">
                    {formData.aliases.map((alias) => (
                      <span
                        key={alias}
                        className="px-3 py-1 rounded-full bg-white/10 text-white text-sm flex items-center gap-2"
                      >
                        {alias}
                        <button
                          type="button"
                          onClick={() => handleRemoveAlias(alias)}
                          className="text-gray-400 hover:text-white"
                          aria-label="Alias entfernen"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>

              {/* Device Management Section */}
              <div className="border-t border-gray-700 pt-4">
                <h4 className="text-lg font-semibold text-white mb-3">
                  {editingPackage ? 'Geräte verwalten' : 'Geräte erstellen (optional)'}
                </h4>

                {editingPackage && (
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-gray-300">
                        {packageDevices.length} Gerät(e) zugeordnet
                      </span>
                      {loadingDevices && (
                        <span className="text-xs text-gray-400">Lade...</span>
                      )}
                    </div>

                    {packageDevices.length > 0 && (
                      <div className="max-h-48 overflow-y-auto space-y-2 rounded-lg bg-white/5 p-3">
                        {packageDevices.map(device => (
                          <div
                            key={device.device_id}
                            className={`flex items-center justify-between rounded-lg px-3 py-2 transition ${
                              devicesToDelete.has(device.device_id)
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
                              className={`p-2 rounded-lg transition ${
                                devicesToDelete.has(device.device_id)
                                  ? 'bg-red-500/20 text-red-300 hover:bg-red-500/30'
                                  : 'text-gray-400 hover:bg-white/10 hover:text-red-400'
                              }`}
                              title={devicesToDelete.has(device.device_id) ? 'Löschen rückgängig' : 'Zum Löschen markieren'}
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </div>
                        ))}
                      </div>
                    )}

                    {devicesToDelete.size > 0 && (
                      <p className="text-xs text-red-300">
                        {devicesToDelete.size} Gerät(e) zum Löschen markiert. Änderungen werden beim Speichern angewendet.
                      </p>
                    )}
                  </div>
                )}

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 mt-4">
                  <div>
                    <label className="mb-2 block text-sm font-semibold text-white">
                      Anzahl Geräte {editingPackage && 'hinzufügen'}
                    </label>
                    <input
                      type="number"
                      min="0"
                      value={formData.device_quantity ?? ''}
                      onChange={event =>
                        setFormData({
                          ...formData,
                          device_quantity: event.target.value ? parseInt(event.target.value) : undefined,
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
                      value={formData.device_prefix || ''}
                      onChange={event =>
                        setFormData({
                          ...formData,
                          device_prefix: event.target.value,
                        })
                      }
                      placeholder="z. B. PKG"
                      className="w-full rounded-lg border border-white/20 bg-white/10 px-3 py-2 text-white placeholder-gray-500 outline-none transition focus:border-accent-red"
                    />
                  </div>
                </div>

                {editingPackage && (
                  <button
                    type="button"
                    onClick={async () => {
                      const pkgResponse = await api.get<ProductPackageWithItems>(`/admin/product-packages/${editingPackage}`);
                      if (pkgResponse.data.product_id) {
                        await handleAddDevices(pkgResponse.data.product_id);
                      }
                    }}
                    disabled={!formData.device_quantity || formData.device_quantity <= 0}
                    className="w-full mt-3 rounded-lg bg-green-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Geräte jetzt hinzufügen
                  </button>
                )}

                <p className="text-xs text-gray-400 mt-3">
                  Geräte werden automatisch mit aufsteigender Nummerierung erstellt (z. B. {formData.device_prefix || 'PKG'}0001).
                </p>
              </div>

              <div className="flex gap-3 pt-4">
                <button
                  type="button"
                  onClick={handleCloseModal}
                  className="flex-1 btn-secondary"
                  disabled={submitting}
                >
                  Abbrechen
                </button>
                <button type="submit" className="flex-1 btn-primary" disabled={submitting}>
                  {submitting ? 'Speichert...' : editingPackage ? 'Speichern' : 'Erstellen'}
                </button>
              </div>
            </form>
          </div>
          </div>
        </ModalPortal>
      )}

      {/* View Modal */}
      {viewPackage && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-2xl font-bold text-white">{viewPackage.name}</h3>
                <button
                  onClick={() => setViewPackage(null)}
                  className="text-gray-400 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors"
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

            <div className="space-y-4">
              {viewPackage.description && (
                <div>
                  <h4 className="text-sm font-medium text-gray-400 mb-1">Beschreibung</h4>
                  <p className="text-white">{viewPackage.description}</p>
                </div>
              )}

              {viewPackagePriceDisplay && (
                <div>
                  <h4 className="text-sm font-medium text-gray-400 mb-1">Preis</h4>
                  <p className="text-white text-xl font-bold">{viewPackagePriceDisplay}</p>
                </div>
              )}

              <div>
                <h4 className="text-sm font-medium text-gray-400 mb-1">OCR-Schlüsselwörter</h4>
                {viewPackage.aliases && viewPackage.aliases.length > 0 ? (
                  <div className="flex flex-wrap gap-2">
                    {viewPackage.aliases.map((alias) => (
                      <span key={alias} className="px-3 py-1 rounded-full bg-white/10 text-white text-sm">
                        {alias}
                      </span>
                    ))}
                  </div>
                ) : (
                  <p className="text-gray-500 text-sm">Keine Schlüsselwörter definiert.</p>
                )}
              </div>

              <div>
                <h4 className="text-sm font-medium text-gray-400 mb-3">Produkte ({viewPackage.total_items} Artikel)</h4>
                {viewPackage.items && viewPackage.items.length > 0 ? (
                  <div className="space-y-2">
                    {viewPackage.items.map((item) => (
                      <div key={item.package_item_id} className="bg-gray-800 rounded-lg p-3">
                        <div className="flex justify-between items-center">
                          <div>
                            <span className="text-white font-medium">{item.product_name}</span>
                            {item.category_name && (
                              <span className="text-gray-400 text-sm ml-2">({item.category_name})</span>
                            )}
                          </div>
                          <span className="text-white font-semibold">x {item.quantity}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <p className="text-gray-400">Keine Produkte</p>
                )}
              </div>
            </div>

            <button
              onClick={() => setViewPackage(null)}
              className="w-full mt-6 btn-secondary"
            >
              Schließen
            </button>
          </div>
          </div>
        </ModalPortal>
      )}
    </div>
  );
}
