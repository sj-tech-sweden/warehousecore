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

interface ProductPackage {
  package_id: number;
  package_code: string;
  name: string;
  description?: string | null;
  price?: number | null;
  total_items: number;
  created_at: string;
  updated_at: string;
  aliases?: string[];
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
}

const initialFormData: PackageFormData = {
  name: '',
  description: '',
  price: '',
  items: [],
  aliases: [],
};

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

  useEffect(() => {
    fetchPackages();
  }, [searchTerm]);

  useEffect(() => {
    if ((modalOpen || viewPackage) && products.length === 0) {
      fetchProducts();
    }
  }, [modalOpen, viewPackage, products.length]);

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

  const handleOpenModal = (pkg?: ProductPackage) => {
    if (pkg) {
      setEditingPackage(pkg.package_id);
      // Fetch full package details with items
      api.get<ProductPackageWithItems>(`/admin/product-packages/${pkg.package_id}`)
        .then(res => {
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
          });
        })
        .catch(err => console.error('Failed to fetch package details:', err));
    } else {
      setEditingPackage(null);
      setFormData(initialFormData);
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
      };

      if (editingPackage) {
        await api.put(`/admin/product-packages/${editingPackage}`, payload);
      } else {
        await api.post('/admin/product-packages', payload);
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
                    {pkg.price ? `${pkg.price.toFixed(2)} €` : '-'}
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
        <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
          <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="flex justify-between items-center mb-6">
              <h3 className="text-2xl font-bold text-white">
                {editingPackage ? 'Paket bearbeiten' : 'Neues Paket'}
              </h3>
              <button onClick={handleCloseModal} className="text-gray-400 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors">
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
      )}

      {/* View Modal */}
      {viewPackage && (
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

              {viewPackage.price && (
                <div>
                  <h4 className="text-sm font-medium text-gray-400 mb-1">Preis</h4>
                  <p className="text-white text-xl font-bold">{viewPackage.price.toFixed(2)} €</p>
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
      )}
    </div>
  );
}
