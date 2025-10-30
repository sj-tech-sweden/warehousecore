import { useState, useEffect } from 'react';
import { Plus, Package, X, Trash2 } from 'lucide-react';
import { api } from '../../lib/api';

interface Product {
  product_id: number;
  name: string;
  category_name?: string;
  subcategory_name?: string;
  subbiercategory_name?: string;
  item_cost_per_day?: number;
  description?: string;
}

interface Category {
  category_id: number;
  name: string;
}

interface Subcategory {
  subcategory_id: number | string;
  name: string;
  category_id: number;
}

interface Subbiercategory {
  subbiercategory_id: number | string;
  name: string;
  subcategory_id: number | string;
}

interface ProductFormData {
  name: string;
  description?: string;
  category_id?: number;
  subcategory_id?: number | string;
  subbiercategory_id?: number | string;
  item_cost_per_day?: number;
  device_quantity?: number;
  device_prefix?: string;
}

export function ProductsTab() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingProduct, setEditingProduct] = useState<number | null>(null);
  const [formData, setFormData] = useState<ProductFormData>({
    name: '',
    description: '',
  });
  const [categories, setCategories] = useState<Category[]>([]);
  const [subcategories, setSubcategories] = useState<Subcategory[]>([]);
  const [subbiercategories, setSubbiercategories] = useState<Subbiercategory[]>([]);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    loadProducts();
    loadCategories();
  }, []);

  const loadProducts = async () => {
    try {
      setLoading(true);
      const { data } = await api.get('/admin/products');
      setProducts(data || []);
    } catch (error) {
      console.error('Failed to load products:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadCategories = async () => {
    try {
      const [catRes, subRes, subbierRes] = await Promise.all([
        api.get('/admin/categories'),
        api.get('/admin/subcategories'),
        api.get('/admin/subbiercategories'),
      ]);
      setCategories(catRes.data || []);
      setSubcategories(subRes.data || []);
      setSubbiercategories(subbierRes.data || []);
    } catch (error) {
      console.error('Failed to load categories:', error);
    }
  };

  const handleOpenCreateModal = () => {
    setEditingProduct(null);
    setFormData({
      name: '',
      description: '',
    });
    setModalOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name.trim()) return;

    setSubmitting(true);
    try {
      let productId: number;

      if (editingProduct) {
        await api.put(`/admin/products/${editingProduct}`, formData);
        productId = editingProduct;
      } else {
        const { data } = await api.post('/admin/products', formData);
        productId = data.product_id;
      }

      // Create devices if quantity is specified
      if (formData.device_quantity && formData.device_quantity > 0 && !editingProduct) {
        try {
          await api.post('/products/create-devices', {
            product_id: productId,
            quantity: formData.device_quantity,
            prefix: formData.device_prefix || '',
          });
        } catch (deviceError) {
          console.error('Failed to create devices:', deviceError);
          alert('Produkt erstellt, aber Fehler beim Erstellen der Geräte');
        }
      }

      setModalOpen(false);
      await loadProducts();
    } catch (error) {
      console.error('Failed to save product:', error);
      alert('Fehler beim Speichern des Produkts');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (productId: number) => {
    if (!confirm('Produkt wirklich löschen?')) return;

    try {
      await api.delete(`/admin/products/${productId}`);
      await loadProducts();
    } catch (error) {
      console.error('Failed to delete product:', error);
      alert('Fehler beim Löschen des Produkts');
    }
  };

  const filteredSubcategories = formData.category_id
    ? subcategories.filter(s => s.category_id === formData.category_id)
    : [];

  const filteredSubbiercategories = formData.subcategory_id
    ? subbiercategories.filter(s => s.subcategory_id === formData.subcategory_id)
    : [];

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h2 className="text-xl font-bold text-white">Produkte verwalten</h2>
        <button
          onClick={handleOpenCreateModal}
          className="px-4 py-2 bg-accent-red text-white rounded-xl font-semibold hover:shadow-lg flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          Neues Produkt
        </button>
      </div>

      {loading ? (
        <p className="text-gray-400">Lade Produkte...</p>
      ) : products.length === 0 ? (
        <div className="glass rounded-xl p-8 text-center">
          <Package className="w-16 h-16 text-gray-600 mx-auto mb-4" />
          <p className="text-gray-400">Keine Produkte vorhanden</p>
        </div>
      ) : (
        <div className="space-y-2">
          {products.map(product => (
            <div key={product.product_id} className="glass rounded-xl p-4">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <h3 className="text-white font-semibold">{product.name}</h3>
                  <p className="text-gray-400 text-sm">
                    {[product.category_name, product.subcategory_name, product.subbiercategory_name]
                      .filter(Boolean)
                      .join(' > ')}
                  </p>
                  {product.description && (
                    <p className="text-gray-500 text-xs mt-1">{product.description}</p>
                  )}
                </div>
                <div className="flex items-center gap-3">
                  {product.item_cost_per_day && (
                    <div className="text-right">
                      <p className="text-white font-semibold">{product.item_cost_per_day.toFixed(2)} €</p>
                      <p className="text-gray-400 text-xs">pro Tag</p>
                    </div>
                  )}
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleDelete(product.product_id)}
                      className="p-2 bg-red-600/80 hover:bg-red-600 rounded-lg transition-colors"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Product Form Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-[70] bg-black/60 backdrop-blur-sm overflow-y-auto">
          <div className="flex justify-center pt-8 pb-8 px-4">
            <div className="glass-dark rounded-2xl w-full max-w-2xl shadow-2xl">
            <div className="flex items-center justify-between p-6 border-b border-white/10">
              <h2 className="text-2xl font-bold text-white">
                {editingProduct ? 'Produkt bearbeiten' : 'Neues Produkt'}
              </h2>
              <button
                onClick={() => setModalOpen(false)}
                className="p-2 rounded-lg hover:bg-white/10 transition-colors text-gray-400 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-semibold text-white mb-2">
                  Name <span className="text-accent-red">*</span>
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
                  required
                />
              </div>

              <div>
                <label className="block text-sm font-semibold text-white mb-2">Beschreibung</label>
                <textarea
                  value={formData.description || ''}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={3}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
                />
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label className="block text-sm font-semibold text-white mb-2">Kategorie</label>
                  <select
                    value={formData.category_id || ''}
                    onChange={(e) => {
                      const value = e.target.value;
                      setFormData({
                        ...formData,
                        category_id: value ? Number(value) : undefined,
                        subcategory_id: undefined,
                        subbiercategory_id: undefined,
                      });
                    }}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:border-accent-red transition-colors"
                  >
                    <option value="">Keine</option>
                    {categories.map(cat => (
                      <option key={cat.category_id} value={cat.category_id}>{cat.name}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-semibold text-white mb-2">Unterkategorie</label>
                  <select
                    value={formData.subcategory_id || ''}
                    onChange={(e) => {
                      const value = e.target.value;
                      setFormData({
                        ...formData,
                        subcategory_id: value || undefined,
                        subbiercategory_id: undefined,
                      });
                    }}
                    disabled={!formData.category_id}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:border-accent-red transition-colors disabled:opacity-50"
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
                  <label className="block text-sm font-semibold text-white mb-2">Sub-Unterkategorie</label>
                  <select
                    value={formData.subbiercategory_id || ''}
                    onChange={(e) => {
                      const value = e.target.value;
                      setFormData({
                        ...formData,
                        subbiercategory_id: value || undefined,
                      });
                    }}
                    disabled={!formData.subcategory_id}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white focus:outline-none focus:border-accent-red transition-colors disabled:opacity-50"
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

              <div>
                <label className="block text-sm font-semibold text-white mb-2">Preis pro Tag (€)</label>
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  value={formData.item_cost_per_day || ''}
                  onChange={(e) => setFormData({
                    ...formData,
                    item_cost_per_day: e.target.value ? parseFloat(e.target.value) : undefined,
                  })}
                  className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
                />
              </div>

              {!editingProduct && (
                <>
                  <div className="border-t border-white/10 pt-4">
                    <h3 className="text-lg font-semibold text-white mb-3">Geräte erstellen (optional)</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-semibold text-white mb-2">Anzahl Geräte</label>
                        <input
                          type="number"
                          min="0"
                          value={formData.device_quantity || ''}
                          onChange={(e) => setFormData({
                            ...formData,
                            device_quantity: e.target.value ? parseInt(e.target.value) : undefined,
                          })}
                          className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
                          placeholder="z.B. 10"
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-semibold text-white mb-2">Geräte-Präfix</label>
                        <input
                          type="text"
                          value={formData.device_prefix || ''}
                          onChange={(e) => setFormData({
                            ...formData,
                            device_prefix: e.target.value,
                          })}
                          className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-accent-red transition-colors"
                          placeholder="z.B. LED"
                        />
                      </div>
                    </div>
                    <p className="text-xs text-gray-400 mt-2">
                      Geräte werden automatisch mit IDs wie {formData.device_prefix || 'XXX'}0001, {formData.device_prefix || 'XXX'}0002, etc. erstellt
                    </p>
                  </div>
                </>
              )}

              <div className="flex gap-3 pt-4">
                <button
                  type="submit"
                  disabled={submitting}
                  className="flex-1 px-4 py-3 bg-accent-red/80 hover:bg-accent-red rounded-lg font-semibold text-white transition-colors disabled:opacity-50"
                >
                  {submitting ? 'Speichern...' : editingProduct ? 'Aktualisieren' : 'Erstellen'}
                </button>
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
                  className="px-4 py-3 bg-white/10 hover:bg-white/20 rounded-lg font-semibold text-white transition-colors"
                >
                  Abbrechen
                </button>
              </div>
            </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
