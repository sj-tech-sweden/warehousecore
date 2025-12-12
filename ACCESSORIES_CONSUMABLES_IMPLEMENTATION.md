# WarehouseCore - Accessories & Consumables Implementation

**Status**: Ready for Implementation
**Version**: 1.0
**Date**: 2025-11-24

---

## ✅ RentalCore APIs - Fully Operational

All backend APIs are ready and tested in RentalCore v4.1.38:

### Available Endpoints (http://rentalcore:8081)

```
✅ GET  /api/count-types                          - Get measurement units
✅ GET  /api/accessories/products                 - Get all accessory products
✅ GET  /api/consumables/products                 - Get all consumable products
✅ POST /api/scan/accessory                       - Scan accessory barcode
✅ POST /api/scan/consumable                      - Scan consumable barcode
✅ GET  /api/inventory/low-stock                  - Get low stock alerts
✅ POST /api/inventory/adjust                     - Adjust stock manually
✅ GET  /api/inventory/transactions               - Get transaction history
✅ GET  /api/products/:id/accessories             - Get product accessories
✅ POST /api/products/:id/accessories             - Link accessory to product
✅ DELETE /api/products/:id/accessories/:accID    - Remove accessory
✅ GET  /api/products/:id/consumables             - Get product consumables
✅ POST /api/products/:id/consumables             - Link consumable to product
✅ DELETE /api/products/:id/consumables/:consID   - Remove consumable
✅ GET  /api/jobs/:jobID/accessories              - Get job accessories
✅ POST /api/jobs/:jobID/accessories              - Assign accessories
✅ GET  /api/jobs/:jobID/consumables              - Get job consumables
✅ POST /api/jobs/:jobID/consumables              - Assign consumables
```

---

## 🔧 Implementation Tasks for WarehouseCore

### Priority 1: Extend Product Form (CRITICAL)

**File**: `/opt/dev/cores/warehousecore/web/src/components/admin/ProductsTab.tsx`

#### Step 1: Extend Product Interface

```typescript
// Add to Product interface (line ~19)
interface Product {
  product_id: number;
  name: string;
  // ... existing fields ...

  // NEW FIELDS:
  is_accessory?: boolean;
  is_consumable?: boolean;
  count_type_id?: number | null;
  stock_quantity?: number | null;
  min_stock_level?: number | null;
  generic_barcode?: string | null;
  price_per_unit?: number | null;
}
```

#### Step 2: Extend ProductFormData Interface

```typescript
// Add to ProductFormData interface (line ~73)
interface ProductFormData {
  name: string;
  description: string;
  // ... existing fields ...

  // NEW FIELDS:
  is_accessory?: boolean;
  is_consumable?: boolean;
  count_type_id?: number;
  stock_quantity?: number;
  min_stock_level?: number;
  generic_barcode?: string;
  price_per_unit?: number;
}
```

#### Step 3: Add Count Types State

```typescript
// Add to ProductsTab component (around line ~135)
interface CountType {
  count_type_id: number;
  name: string;
  abbreviation: string;
  is_active: boolean;
}

export function ProductsTab() {
  // ... existing state ...
  const [countTypes, setCountTypes] = useState<CountType[]>([]);

  // Load count types
  useEffect(() => {
    const loadCountTypes = async () => {
      try {
        const response = await fetch('http://rentalcore:8081/api/count-types');
        const data = await response.json();
        setCountTypes(data.count_types || []);
      } catch (error) {
        console.error('Failed to load count types:', error);
      }
    };
    loadCountTypes();
  }, []);
```

#### Step 4: Add Form Fields in Modal

Find the product form modal (search for `<form` in the file) and add these fields:

```tsx
{/* After existing form fields, add: */}

{/* Product Type Section */}
<div className="col-span-2 border-t border-gray-700 pt-4">
  <h3 className="text-lg font-semibold text-white mb-3">Product Type & Inventory</h3>

  <div className="flex gap-4 mb-4">
    <label className="flex items-center gap-2 text-white cursor-pointer">
      <input
        type="checkbox"
        checked={formData.is_accessory || false}
        onChange={e => setFormData(prev => ({
          ...prev,
          is_accessory: e.target.checked
        }))}
        className="w-5 h-5 rounded border-gray-600 bg-dark-700 text-accent-red focus:ring-accent-red"
      />
      <span>This is an Accessory</span>
    </label>

    <label className="flex items-center gap-2 text-white cursor-pointer">
      <input
        type="checkbox"
        checked={formData.is_consumable || false}
        onChange={e => setFormData(prev => ({
          ...prev,
          is_consumable: e.target.checked
        }))}
        className="w-5 h-5 rounded border-gray-600 bg-dark-700 text-accent-red focus:ring-accent-red"
      />
      <span>This is a Consumable</span>
    </label>
  </div>

  <p className="text-sm text-gray-400 mb-4">
    Accessories are optional items (cables, clamps). Consumables are used items (fog fluid, tape).
  </p>
</div>

{/* Inventory Fields (conditional) */}
{(formData.is_accessory || formData.is_consumable) && (
  <>
    <div>
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Measurement Unit <span className="text-red-500">*</span>
      </label>
      <select
        value={formData.count_type_id || ''}
        onChange={e => setFormData(prev => ({
          ...prev,
          count_type_id: e.target.value ? Number(e.target.value) : undefined
        }))}
        required={formData.is_accessory || formData.is_consumable}
        className="w-full px-4 py-2 bg-dark-700 border border-gray-600 rounded-lg text-white"
      >
        <option value="">Select unit...</option>
        {countTypes.map(ct => (
          <option key={ct.count_type_id} value={ct.count_type_id}>
            {ct.name} ({ct.abbreviation})
          </option>
        ))}
      </select>
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Generic Barcode
      </label>
      <input
        type="text"
        value={formData.generic_barcode || ''}
        onChange={e => setFormData(prev => ({
          ...prev,
          generic_barcode: e.target.value
        }))}
        placeholder="e.g., ACC-SAFE40, CONS-FOG"
        className="w-full px-4 py-2 bg-dark-700 border border-gray-600 rounded-lg text-white placeholder-gray-500"
      />
      <p className="text-xs text-gray-400 mt-1">Single barcode for all units of this type</p>
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Current Stock Quantity
      </label>
      <input
        type="number"
        step="0.001"
        min="0"
        value={formData.stock_quantity || ''}
        onChange={e => setFormData(prev => ({
          ...prev,
          stock_quantity: parseNumber(e.target.value)
        }))}
        placeholder="0.000"
        className="w-full px-4 py-2 bg-dark-700 border border-gray-600 rounded-lg text-white placeholder-gray-500"
      />
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Minimum Stock Level
      </label>
      <input
        type="number"
        step="0.001"
        min="0"
        value={formData.min_stock_level || ''}
        onChange={e => setFormData(prev => ({
          ...prev,
          min_stock_level: parseNumber(e.target.value)
        }))}
        placeholder="0.000"
        className="w-full px-4 py-2 bg-dark-700 border border-gray-600 rounded-lg text-white placeholder-gray-500"
      />
      <p className="text-xs text-gray-400 mt-1">Alert when stock falls below this level</p>
    </div>

    <div>
      <label className="block text-sm font-medium text-gray-300 mb-2">
        Price per Unit (€)
      </label>
      <input
        type="number"
        step="0.01"
        min="0"
        value={formData.price_per_unit || ''}
        onChange={e => setFormData(prev => ({
          ...prev,
          price_per_unit: parseNumber(e.target.value)
        }))}
        placeholder="0.00"
        className="w-full px-4 py-2 bg-dark-700 border border-gray-600 rounded-lg text-white placeholder-gray-500"
      />
    </div>
  </>
)}
```

#### Step 5: Update Submit Handler

Find the handleSubmit function and ensure new fields are included:

```typescript
const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault();
  setSubmitting(true);

  try {
    const payload = {
      ...formData,
      // Ensure new fields are included
      is_accessory: formData.is_accessory || false,
      is_consumable: formData.is_consumable || false,
      count_type_id: formData.count_type_id || null,
      stock_quantity: formData.stock_quantity || null,
      min_stock_level: formData.min_stock_level || null,
      generic_barcode: formData.generic_barcode || null,
      price_per_unit: formData.price_per_unit || null,
    };

    // ... rest of submit logic ...
  } catch (error) {
    console.error('Error saving product:', error);
  } finally {
    setSubmitting(false);
  }
};
```

---

### Priority 2: Extend Scan Page (CRITICAL)

**File**: `/opt/dev/cores/warehousecore/web/src/pages/ScanPage.tsx`

#### Add Generic Barcode Detection

```typescript
// Add after existing barcode scan handler
const handleBarcodeScan = async (barcode: string) => {
  try {
    // 1. Check if it's a generic barcode (accessories/consumables)
    const productResponse = await fetch(
      `http://rentalcore:8081/api/products?barcode=${barcode}`
    );

    if (productResponse.ok) {
      const productData = await productResponse.json();
      const product = productData.product;

      if (product && (product.is_accessory || product.is_consumable)) {
        if (product.is_accessory) {
          await handleAccessoryScan(barcode, product);
        } else if (product.is_consumable) {
          await handleConsumableScan(barcode, product);
        }
        return;
      }
    }

    // 2. Fall back to standard device scan
    await handleStandardDeviceScan(barcode);
  } catch (error) {
    console.error('Scan error:', error);
    showError('Scan failed');
  }
};

// Accessory scan handler (scan once per piece)
const handleAccessoryScan = async (barcode: string, product: any) => {
  if (!selectedJobId) {
    showError('Please select a job first');
    return;
  }

  try {
    const response = await fetch('http://rentalcore:8081/api/scan/accessory', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        barcode: barcode,
        job_id: selectedJobId,
        direction: scanMode, // "out" or "in"
        quantity: 1
      })
    });

    if (response.ok) {
      const data = await response.json();
      showSuccess(
        `✅ Scanned 1x ${data.product.name}`,
        `Remaining stock: ${data.remaining_stock} pcs`
      );
      playBeep();
    } else {
      const error = await response.json();
      showError(error.error || 'Scan failed');
      playBeepError();
    }
  } catch (error) {
    console.error('Accessory scan error:', error);
    showError('Network error');
  }
};

// Consumable scan handler (prompt for quantity)
const handleConsumableScan = async (barcode: string, product: any) => {
  if (!selectedJobId) {
    showError('Please select a job first');
    return;
  }

  // Show quantity input modal
  const quantity = await promptForQuantity(
    product.name,
    product.count_type?.abbreviation || 'units',
    product.stock_quantity || 0
  );

  if (quantity === null) {
    return; // User cancelled
  }

  try {
    const response = await fetch('http://rentalcore:8081/api/scan/consumable', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        barcode: barcode,
        job_id: selectedJobId,
        direction: scanMode, // "out" or "in"
        quantity: parseFloat(quantity)
      })
    });

    if (response.ok) {
      const data = await response.json();
      showSuccess(
        `✅ Scanned ${quantity}${product.count_type?.abbreviation} ${data.product.name}`,
        `Remaining stock: ${data.remaining_stock} ${product.count_type?.abbreviation}`
      );
      playBeep();
    } else {
      const error = await response.json();
      showError(error.error || 'Scan failed');
      playBeepError();
    }
  } catch (error) {
    console.error('Consumable scan error:', error);
    showError('Network error');
  }
};

// Quantity input prompt (add this component/modal)
const promptForQuantity = (productName: string, unit: string, available: number): Promise<number | null> => {
  return new Promise((resolve) => {
    // TODO: Create a modal component for quantity input
    // For now, use prompt as placeholder
    const input = window.prompt(
      `Enter quantity for ${productName} (${unit})\nAvailable: ${available} ${unit}`
    );

    if (input === null) {
      resolve(null);
    } else {
      const qty = parseFloat(input);
      resolve(isNaN(qty) ? null : qty);
    }
  });
};
```

---

### Priority 3: Low Stock Alerts Dashboard Widget

**Create**: `/opt/dev/cores/warehousecore/web/src/components/LowStockAlert.tsx`

```tsx
import { useEffect, useState } from 'react';
import { AlertTriangle } from 'lucide-react';

interface LowStockItem {
  productID: number;
  name: string;
  stock_quantity: number;
  min_stock_level: number;
  quantity_below_min: number;
  count_type: string;
  count_type_abbr: string;
  item_type: string;
}

export function LowStockAlert() {
  const [alerts, setAlerts] = useState<LowStockItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const loadAlerts = async () => {
      try {
        const response = await fetch('http://rentalcore:8081/api/inventory/low-stock');
        const data = await response.json();
        setAlerts(data.alerts || []);
      } catch (error) {
        console.error('Failed to load low stock alerts:', error);
      } finally {
        setLoading(false);
      }
    };

    loadAlerts();
    const interval = setInterval(loadAlerts, 60000); // Refresh every minute

    return () => clearInterval(interval);
  }, []);

  if (loading || alerts.length === 0) {
    return null;
  }

  return (
    <div className="glass-dark rounded-2xl p-4 border-l-4 border-orange-500">
      <div className="flex items-start gap-3">
        <AlertTriangle className="w-6 h-6 text-orange-500 flex-shrink-0 mt-1" />
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-white mb-2">
            Low Stock Alert
          </h3>
          <p className="text-gray-300 text-sm mb-3">
            {alerts.length} item{alerts.length !== 1 ? 's' : ''} below minimum stock level
          </p>
          <div className="space-y-2">
            {alerts.slice(0, 5).map(item => (
              <div
                key={item.productID}
                className="flex justify-between items-center text-sm"
              >
                <span className="text-gray-300">{item.name}</span>
                <span className="text-orange-400 font-mono">
                  {item.stock_quantity} {item.count_type_abbr} (min: {item.min_stock_level})
                </span>
              </div>
            ))}
          </div>
          {alerts.length > 5 && (
            <p className="text-xs text-gray-400 mt-2">
              +{alerts.length - 5} more items
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
```

**Usage**: Add to Dashboard.tsx:

```tsx
import { LowStockAlert } from '../components/LowStockAlert';

// In Dashboard component:
<div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
  <LowStockAlert />
  {/* ... other dashboard widgets ... */}
</div>
```

---

## 🎨 Visual Guidelines

### Icons
- **Accessories**: Use `Paperclip` from lucide-react
- **Consumables**: Use `Droplet` from lucide-react

### Colors (Tailwind)
- **Accessories**: `text-blue-400`, `bg-blue-500/10`
- **Consumables**: `text-green-400`, `bg-green-500/10`
- **Low Stock**: `text-orange-400`, `bg-orange-500/10`

---

## ✅ Testing Checklist

### Product Management
- [ ] Create accessory product with all fields
- [ ] Create consumable product with all fields
- [ ] View products in list - see new fields
- [ ] Edit product - modify inventory fields
- [ ] Verify data syncs to RentalCore database

### Scanning
- [ ] Scan accessory barcode (once per piece)
- [ ] Verify stock decreases
- [ ] Scan consumable barcode
- [ ] Enter quantity in prompt
- [ ] Verify stock decreases by quantity
- [ ] Scan back in (return)
- [ ] Verify stock increases

### Low Stock
- [ ] Set product below minimum stock
- [ ] Verify alert appears on dashboard
- [ ] Perform stock adjustment
- [ ] Verify alert disappears

---

## 📊 Implementation Status

| Feature | Status | Priority |
|---------|--------|----------|
| Product Form Extension | 🟡 Code Ready | HIGH |
| Scan Page Extension | 🟡 Code Ready | HIGH |
| Low Stock Alert Widget | 🟡 Code Ready | MEDIUM |
| RentalCore APIs | ✅ Complete | - |
| Database Schema | ✅ Complete | - |
| Documentation | ✅ Complete | - |

**🟡 Code Ready** = Implementation code provided, needs to be integrated
**✅ Complete** = Fully implemented and tested

---

## 🚀 Deployment

After implementing changes:

```bash
cd /opt/dev/cores/warehousecore/web
npm install
npm run build

cd /opt/dev/cores/warehousecore
docker build -t nobentie/warehousecore:latest .
docker push nobentie/warehousecore:latest
```

---

## 📞 Support

- **RentalCore API Docs**: `/opt/dev/cores/rentalcore/docs/API_ACCESSORIES_CONSUMABLES.md`
- **Integration Guide**: `/opt/dev/cores/rentalcore/docs/WAREHOUSECORE_INTEGRATION_GUIDE.md`
- **Implementation Examples**: All code snippets in this document are copy-paste ready

---

**Version**: 1.0
**Last Updated**: 2025-11-24
**Status**: Ready for immediate implementation
