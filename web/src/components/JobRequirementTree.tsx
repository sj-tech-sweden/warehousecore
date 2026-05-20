import { useMemo } from 'react';
import { ChevronRight, ChevronsDown, ChevronsUp, Package, Search, X } from 'lucide-react';
import type { DeviceTreeCategory, DeviceTreeDevice } from '../lib/api';

export interface JobRequirementSelection {
  product_id: number;
  product_name: string;
  quantity: number;
  assigned: number;
}

interface RequirementTreeProduct {
  id: number;
  name: string;
  available: number;
  total: number;
  isConsumable: boolean;
  isAccessory: boolean;
  stockQuantity?: number;
  unit?: string;
}

interface RequirementTreeSubbiercategory {
  id: string;
  name: string;
  products: RequirementTreeProduct[];
}

interface RequirementTreeSubcategory {
  id: string;
  name: string;
  subbiercategories: RequirementTreeSubbiercategory[];
  products: RequirementTreeProduct[];
}

interface RequirementTreeCategory {
  id: string;
  name: string;
  subcategories: RequirementTreeSubcategory[];
  products: RequirementTreeProduct[];
}

interface JobRequirementTreeProps {
  treeData: DeviceTreeCategory[];
  loading: boolean;
  search: string;
  onSearchChange: (value: string) => void;
  expandedNodes: Set<string>;
  onToggleNode: (nodeId: string) => void;
  onExpandAll: (nodeIds: string[]) => void;
  onCollapseAll: () => void;
  selections: JobRequirementSelection[];
  onSetSelection: (selection: { product_id: number; product_name: string; quantity: number }) => void;
}

export function JobRequirementTree({
  treeData,
  loading,
  search,
  onSearchChange,
  expandedNodes,
  onToggleNode,
  onExpandAll,
  onCollapseAll,
  selections,
  onSetSelection,
}: JobRequirementTreeProps) {
  const normalizedTree = useMemo(() => normalizeRequirementTree(treeData), [treeData]);
  const selectionMap = useMemo(
    () => new Map(selections.map((selection) => [selection.product_id, selection])),
    [selections],
  );
  const productStatsMap = useMemo(() => buildProductStatsMap(normalizedTree), [normalizedTree]);
  const allExpandableNodeIds = useMemo(() => collectExpandableNodeIds(normalizedTree), [normalizedTree]);
  const searchTerm = search.trim().toLowerCase();
  const { filteredTree, searchExpandedNodeIds } = useMemo(
    () => filterRequirementTree(normalizedTree, searchTerm),
    [normalizedTree, searchTerm],
  );
  const effectiveExpandedNodes = useMemo(() => {
    if (!searchTerm) {
      return expandedNodes;
    }
    return new Set([...expandedNodes, ...searchExpandedNodeIds]);
  }, [expandedNodes, searchExpandedNodeIds, searchTerm]);

  return (
    <div className="space-y-4">
      <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500" />
            <input
              type="text"
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder="Search products"
              className="w-full rounded-xl border border-white/10 bg-black/20 py-2.5 pl-9 pr-10 text-sm text-white placeholder:text-gray-500 focus:border-accent-red focus:outline-none"
            />
            {search && (
              <button
                type="button"
                onClick={() => onSearchChange('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-white"
                aria-label="Clear product search"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => onExpandAll(allExpandableNodeIds)}
              className="inline-flex items-center gap-2 rounded-xl border border-white/10 px-3 py-2 text-sm text-gray-200 hover:bg-white/10"
            >
              <ChevronsDown className="h-4 w-4" />
              Expand All
            </button>
            <button
              type="button"
              onClick={onCollapseAll}
              className="inline-flex items-center gap-2 rounded-xl border border-white/10 px-3 py-2 text-sm text-gray-200 hover:bg-white/10"
            >
              <ChevronsUp className="h-4 w-4" />
              Collapse All
            </button>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(320px,1fr)]">
        <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
          {loading ? (
            <div className="flex min-h-[16rem] items-center justify-center">
              <div className="h-10 w-10 animate-spin rounded-full border-b-2 border-accent-red" />
            </div>
          ) : filteredTree.length === 0 ? (
            <div className="flex min-h-[16rem] items-center justify-center text-sm text-gray-400">
              {searchTerm ? 'No products match your search.' : 'No products available.'}
            </div>
          ) : (
            <div className="max-h-[34rem] space-y-2 overflow-y-auto pr-1">
              {filteredTree.map((category) => (
                <RequirementCategoryNode
                  key={category.id}
                  category={category}
                  expandedNodes={effectiveExpandedNodes}
                  selectionMap={selectionMap}
                  onToggleNode={onToggleNode}
                  onSetSelection={onSetSelection}
                />
              ))}
            </div>
          )}
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
          <div className="mb-3 flex items-center gap-2 text-white">
            <Package className="h-5 w-5 text-accent-red" />
            <h3 className="text-lg font-semibold">Selected Products</h3>
          </div>

          {selections.length === 0 ? (
            <div className="rounded-xl border border-dashed border-white/10 p-4 text-sm text-gray-400">
              No products selected yet.
            </div>
          ) : (
            <div className="overflow-hidden rounded-xl border border-white/10">
              <table className="min-w-full text-sm">
                <thead className="bg-black/20 text-gray-400">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">Product</th>
                    <th className="px-3 py-2 text-left font-medium">Qty</th>
                    <th className="px-3 py-2 text-left font-medium">Available</th>
                  </tr>
                </thead>
                <tbody>
                  {[...selections]
                    .sort((left, right) => left.product_name.localeCompare(right.product_name))
                    .map((selection) => {
                      const stats = productStatsMap.get(selection.product_id) ?? { available: 0, total: 0 };
                      const hasWarning = selection.quantity > stats.available;
                      return (
                        <tr key={selection.product_id} className={`border-t border-white/10 ${hasWarning ? 'bg-amber-500/10' : ''}`}>
                          <td className="px-3 py-2 text-white">{selection.product_name}</td>
                          <td className="px-3 py-2 text-white">{selection.quantity}</td>
                          <td className={`px-3 py-2 ${hasWarning ? 'text-amber-300' : 'text-gray-300'}`}>
                            {stats.available} / {stats.total}
                          </td>
                        </tr>
                      );
                    })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function RequirementCategoryNode({
  category,
  expandedNodes,
  selectionMap,
  onToggleNode,
  onSetSelection,
}: {
  category: RequirementTreeCategory;
  expandedNodes: Set<string>;
  selectionMap: Map<number, JobRequirementSelection>;
  onToggleNode: (nodeId: string) => void;
  onSetSelection: (selection: { product_id: number; product_name: string; quantity: number }) => void;
}) {
  return (
    <RequirementTreeNode
      nodeId={`cat-${category.id}`}
      label={category.name}
      stats={collectRequirementNodeStats(category)}
      expandedNodes={expandedNodes}
      onToggleNode={onToggleNode}
    >
      {category.subcategories.map((subcategory) => (
        <RequirementSubcategoryNode
          key={subcategory.id}
          subcategory={subcategory}
          expandedNodes={expandedNodes}
          selectionMap={selectionMap}
          onToggleNode={onToggleNode}
          onSetSelection={onSetSelection}
        />
      ))}
      {category.products.length > 0 && (
        <RequirementProductGroup
          nodeId={`cat-${category.id}-products`}
          label="Products"
          products={category.products}
          expandedNodes={expandedNodes}
          selectionMap={selectionMap}
          onToggleNode={onToggleNode}
          onSetSelection={onSetSelection}
        />
      )}
    </RequirementTreeNode>
  );
}

function RequirementSubcategoryNode({
  subcategory,
  expandedNodes,
  selectionMap,
  onToggleNode,
  onSetSelection,
}: {
  subcategory: RequirementTreeSubcategory;
  expandedNodes: Set<string>;
  selectionMap: Map<number, JobRequirementSelection>;
  onToggleNode: (nodeId: string) => void;
  onSetSelection: (selection: { product_id: number; product_name: string; quantity: number }) => void;
}) {
  return (
    <RequirementTreeNode
      nodeId={`sub-${subcategory.id}`}
      label={subcategory.name}
      stats={collectRequirementNodeStats(subcategory)}
      expandedNodes={expandedNodes}
      onToggleNode={onToggleNode}
      indent="pl-4"
    >
      {subcategory.subbiercategories.map((group) => (
        <RequirementSubbiercategoryNode
          key={group.id}
          group={group}
          expandedNodes={expandedNodes}
          selectionMap={selectionMap}
          onToggleNode={onToggleNode}
          onSetSelection={onSetSelection}
        />
      ))}
      {subcategory.products.length > 0 && (
        <RequirementProductGroup
          nodeId={`sub-${subcategory.id}-products`}
          label="Products"
          products={subcategory.products}
          expandedNodes={expandedNodes}
          selectionMap={selectionMap}
          onToggleNode={onToggleNode}
          onSetSelection={onSetSelection}
          indent="pl-4"
        />
      )}
    </RequirementTreeNode>
  );
}

function RequirementSubbiercategoryNode({
  group,
  expandedNodes,
  selectionMap,
  onToggleNode,
  onSetSelection,
}: {
  group: RequirementTreeSubbiercategory;
  expandedNodes: Set<string>;
  selectionMap: Map<number, JobRequirementSelection>;
  onToggleNode: (nodeId: string) => void;
  onSetSelection: (selection: { product_id: number; product_name: string; quantity: number }) => void;
}) {
  return (
    <RequirementTreeNode
      nodeId={`grp-${group.id}`}
      label={group.name}
      stats={collectRequirementNodeStats(group)}
      expandedNodes={expandedNodes}
      onToggleNode={onToggleNode}
      indent="pl-8"
    >
      <div className="space-y-2 pl-4">
        {group.products.map((product) => (
          <RequirementProductCard
            key={product.id}
            product={product}
            quantity={selectionMap.get(product.id)?.quantity ?? 0}
            onSetSelection={onSetSelection}
          />
        ))}
      </div>
    </RequirementTreeNode>
  );
}

function RequirementProductGroup({
  nodeId,
  label,
  products,
  expandedNodes,
  selectionMap,
  onToggleNode,
  onSetSelection,
  indent = 'pl-4',
}: {
  nodeId: string;
  label: string;
  products: RequirementTreeProduct[];
  expandedNodes: Set<string>;
  selectionMap: Map<number, JobRequirementSelection>;
  onToggleNode: (nodeId: string) => void;
  onSetSelection: (selection: { product_id: number; product_name: string; quantity: number }) => void;
  indent?: string;
}) {
  return (
    <RequirementTreeNode
      nodeId={nodeId}
      label={label}
      stats={collectProductsStats(products)}
      expandedNodes={expandedNodes}
      onToggleNode={onToggleNode}
      indent={indent}
    >
      <div className="space-y-2 pl-4">
        {products.map((product) => (
          <RequirementProductCard
            key={product.id}
            product={product}
            quantity={selectionMap.get(product.id)?.quantity ?? 0}
            onSetSelection={onSetSelection}
          />
        ))}
      </div>
    </RequirementTreeNode>
  );
}

function RequirementTreeNode({
  nodeId,
  label,
  stats,
  expandedNodes,
  onToggleNode,
  children,
  indent,
}: {
  nodeId: string;
  label: string;
  stats: { products: number; available: number; total: number };
  expandedNodes: Set<string>;
  onToggleNode: (nodeId: string) => void;
  children: React.ReactNode;
  indent?: string;
}) {
  const isExpanded = expandedNodes.has(nodeId);

  return (
    <div className={indent ? indent : ''}>
      <div className="overflow-hidden rounded-xl border border-white/10 bg-black/20">
        <button
          type="button"
          onClick={() => onToggleNode(nodeId)}
          className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left hover:bg-white/5"
        >
          <div className="flex items-center gap-2 text-white">
            <ChevronRight className={`h-4 w-4 transition-transform ${isExpanded ? 'rotate-90' : ''}`} />
            <span className="font-medium">{label}</span>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <span className="rounded-full border border-white/10 px-2 py-1 text-gray-300">{stats.products} products</span>
            <span className={`rounded-full border px-2 py-1 ${stats.available > 0 ? 'border-emerald-400/30 text-emerald-300' : 'border-white/10 text-gray-400'}`}>
              {stats.available}/{stats.total} available
            </span>
          </div>
        </button>
        {isExpanded && <div className="border-t border-white/10 px-2 py-2">{children}</div>}
      </div>
    </div>
  );
}

function RequirementProductCard({
  product,
  quantity,
  onSetSelection,
}: {
  product: RequirementTreeProduct;
  quantity: number;
  onSetSelection: (selection: { product_id: number; product_name: string; quantity: number }) => void;
}) {
  const warning = quantity > product.available;

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-white/10 bg-white/[0.03] p-3 lg:flex-row lg:items-center lg:justify-between">
      <div className="min-w-0">
        <p className="truncate font-medium text-white">{product.name}</p>
        <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-400">
          {product.isConsumable && <span className="rounded-full border border-sky-400/30 px-2 py-0.5 text-sky-300">Consumable</span>}
          {product.isAccessory && <span className="rounded-full border border-fuchsia-400/30 px-2 py-0.5 text-fuchsia-300">Accessory</span>}
          {product.stockQuantity !== undefined && (
            <span className="rounded-full border border-white/10 px-2 py-0.5 text-gray-300">
              Stock {product.stockQuantity}{product.unit ? ` ${product.unit}` : ''}
            </span>
          )}
          <span className={`rounded-full border px-2 py-0.5 ${warning ? 'border-amber-400/30 text-amber-300' : 'border-white/10 text-gray-300'}`}>
            {product.available}/{product.total} available
          </span>
        </div>
      </div>
      <input
        type="number"
        min={0}
        step={1}
        aria-label={`Required quantity for ${product.name}`}
        value={quantity}
        onChange={(event) => onSetSelection({
          product_id: product.id,
          product_name: product.name,
          quantity: Math.max(0, Number(event.target.value) || 0),
        })}
        className="w-28 rounded-xl border border-white/10 bg-black/30 px-3 py-2 text-white focus:border-accent-red focus:outline-none"
      />
    </div>
  );
}

function normalizeRequirementTree(treeData: DeviceTreeCategory[]): RequirementTreeCategory[] {
  return treeData
    .map((category) => ({
      id: String(category.id),
      name: category.name,
      subcategories: (category.subcategories ?? []).map((subcategory) => ({
        id: String(subcategory.id),
        name: subcategory.name,
        subbiercategories: (subcategory.subbiercategories ?? []).map((group) => ({
          id: String(group.id),
          name: group.name,
          products: aggregateProducts(group.devices ?? []),
        })).filter((group) => group.products.length > 0),
        products: aggregateProducts(subcategory.direct_devices ?? []),
      })).filter((subcategory) => subcategory.products.length > 0 || subcategory.subbiercategories.length > 0),
      products: aggregateProducts(category.direct_devices ?? []),
    }))
    .filter((category) => category.products.length > 0 || category.subcategories.length > 0)
    .sort((left, right) => left.name.localeCompare(right.name));
}

function aggregateProducts(devices: DeviceTreeDevice[]): RequirementTreeProduct[] {
  const grouped = new Map<number, RequirementTreeProduct>();

  devices.forEach((device) => {
    const productId = resolveProductId(device);
    if (!productId) {
      return;
    }

    const existing = grouped.get(productId) ?? {
      id: productId,
      name: device.product_name || `Product ${productId}`,
      available: 0,
      total: 0,
      isConsumable: device.is_consumable === true,
      isAccessory: device.is_accessory === true,
      stockQuantity: typeof device.stock_quantity === 'number' ? device.stock_quantity : undefined,
      unit: device.unit,
    };

    existing.name = existing.name || device.product_name || `Product ${productId}`;
    existing.isConsumable = existing.isConsumable || device.is_consumable === true;
    existing.isAccessory = existing.isAccessory || device.is_accessory === true;
    existing.unit = existing.unit || device.unit;

    if (isProductPlaceholder(device)) {
      const stock = Number(device.stock_quantity ?? 0);
      existing.stockQuantity = stock;
      existing.available = Math.max(existing.available, stock);
      existing.total = Math.max(existing.total, stock);
    } else {
      existing.total += 1;
      if (isAvailableStatus(device.status)) {
        existing.available += 1;
      }
    }

    grouped.set(productId, existing);
  });

  return [...grouped.values()].sort((left, right) => left.name.localeCompare(right.name));
}

function resolveProductId(device: DeviceTreeDevice): number | null {
  if (typeof device.product_id === 'number' && Number.isFinite(device.product_id)) {
    return device.product_id;
  }

  if (typeof device.device_id === 'string' && device.device_id.startsWith('PROD-')) {
    const parsed = Number(device.device_id.slice(5));
    return Number.isFinite(parsed) ? parsed : null;
  }

  return null;
}

function isProductPlaceholder(device: DeviceTreeDevice): boolean {
  return typeof device.device_id === 'string' && device.device_id.startsWith('PROD-');
}

function isAvailableStatus(status: string | undefined): boolean {
  const normalized = (status || '').trim().toLowerCase();
  if (!normalized) {
    return false;
  }
  return normalized === 'in_storage' || normalized === 'available';
}

function collectProductsStats(products: RequirementTreeProduct[]) {
  return products.reduce(
    (acc, product) => ({
      products: acc.products + 1,
      available: acc.available + product.available,
      total: acc.total + product.total,
    }),
    { products: 0, available: 0, total: 0 },
  );
}

function collectRequirementNodeStats(node: RequirementTreeCategory | RequirementTreeSubcategory | RequirementTreeSubbiercategory) {
  let stats = collectProductsStats(node.products);

  if ('subcategories' in node) {
    node.subcategories.forEach((subcategory) => {
      const childStats = collectRequirementNodeStats(subcategory);
      stats = {
        products: stats.products + childStats.products,
        available: stats.available + childStats.available,
        total: stats.total + childStats.total,
      };
    });
  }

  if ('subbiercategories' in node) {
    node.subbiercategories.forEach((group) => {
      const childStats = collectRequirementNodeStats(group);
      stats = {
        products: stats.products + childStats.products,
        available: stats.available + childStats.available,
        total: stats.total + childStats.total,
      };
    });
  }

  return stats;
}

function collectExpandableNodeIds(tree: RequirementTreeCategory[]) {
  const nodeIds: string[] = [];
  tree.forEach((category) => {
    nodeIds.push(`cat-${category.id}`);
    if (category.products.length > 0) {
      nodeIds.push(`cat-${category.id}-products`);
    }
    category.subcategories.forEach((subcategory) => {
      nodeIds.push(`sub-${subcategory.id}`);
      if (subcategory.products.length > 0) {
        nodeIds.push(`sub-${subcategory.id}-products`);
      }
      subcategory.subbiercategories.forEach((group) => {
        nodeIds.push(`grp-${group.id}`);
      });
    });
  });
  return nodeIds;
}

function filterRequirementTree(tree: RequirementTreeCategory[], term: string) {
  if (!term) {
    return { filteredTree: tree, searchExpandedNodeIds: new Set<string>() };
  }

  const expanded = new Set<string>();
  const filteredTree = tree
    .map((category) => {
      const categoryProducts = category.products.filter((product) => product.name.toLowerCase().includes(term));
      const subcategories = category.subcategories
        .map((subcategory) => {
          const subProducts = subcategory.products.filter((product) => product.name.toLowerCase().includes(term));
          const groups = subcategory.subbiercategories
            .map((group) => ({
              ...group,
              products: group.products.filter((product) => product.name.toLowerCase().includes(term)),
            }))
            .filter((group) => group.products.length > 0);

          if (subProducts.length > 0 || groups.length > 0) {
            expanded.add(`cat-${category.id}`);
            expanded.add(`sub-${subcategory.id}`);
            if (subProducts.length > 0) {
              expanded.add(`sub-${subcategory.id}-products`);
            }
            groups.forEach((group) => expanded.add(`grp-${group.id}`));
          }

          return {
            ...subcategory,
            products: subProducts,
            subbiercategories: groups,
          };
        })
        .filter((subcategory) => subcategory.products.length > 0 || subcategory.subbiercategories.length > 0);

      if (categoryProducts.length > 0) {
        expanded.add(`cat-${category.id}`);
        expanded.add(`cat-${category.id}-products`);
      }

      return {
        ...category,
        products: categoryProducts,
        subcategories,
      };
    })
    .filter((category) => category.products.length > 0 || category.subcategories.length > 0);

  return { filteredTree, searchExpandedNodeIds: expanded };
}

function buildProductStatsMap(tree: RequirementTreeCategory[]) {
  const stats = new Map<number, { available: number; total: number }>();

  const visitProducts = (products: RequirementTreeProduct[]) => {
    products.forEach((product) => {
      stats.set(product.id, { available: product.available, total: product.total });
    });
  };

  tree.forEach((category) => {
    visitProducts(category.products);
    category.subcategories.forEach((subcategory) => {
      visitProducts(subcategory.products);
      subcategory.subbiercategories.forEach((group) => {
        visitProducts(group.products);
      });
    });
  });

  return stats;
}