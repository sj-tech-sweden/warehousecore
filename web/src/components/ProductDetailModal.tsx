import { useEffect, useState } from 'react';
import { X, Package, Ruler, Weight, Zap, Tag, Box, DollarSign, Wrench, Barcode, Info, Image as ImageIcon, UploadCloud, Loader2, Eye } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { ModalPortal } from './ModalPortal';
import { useBlockBodyScroll } from '../hooks/useBlockBodyScroll';
import { productPicturesApi, productWebsiteApi } from '../lib/api';
import type { ChangeEvent } from 'react';
import type { ProductPicture } from '../lib/api';
import { useCurrencySymbol } from '../hooks/useCurrencySymbol';

export interface ProductDetail {
  product_id: number;
  name: string;
  description?: string;
  category_name?: string;
  subcategory_name?: string;
  subbiercategory_name?: string;
  brand_name?: string;
  manufacturer_name?: string;
  item_cost_per_day?: number;
  maintenance_interval?: number;
  weight?: number;
  height?: number;
  width?: number;
  depth?: number;
  power_consumption?: number;
  pos_in_category?: number;
  is_accessory?: boolean;
  is_consumable?: boolean;
  stock_quantity?: number;
  min_stock_level?: number;
  generic_barcode?: string;
  price_per_unit?: number;
  count_type_abbreviation?: string;
  device_count?: number;
  website_visible?: boolean;
  website_thumbnail?: string | null;
  website_images?: string[];
}

interface ProductDetailModalProps {
  product: ProductDetail | null;
  isOpen: boolean;
  onClose: () => void;
}

export function ProductDetailModal({ product, isOpen, onClose }: ProductDetailModalProps) {
  const { t, i18n } = useTranslation();
  const currencySymbol = useCurrencySymbol();
  useBlockBodyScroll(isOpen);

  const [pictures, setPictures] = useState<ProductPicture[]>([]);
  const [loadingPictures, setLoadingPictures] = useState(false);
  const [uploadingPictures, setUploadingPictures] = useState(false);
  const [pictureError, setPictureError] = useState<string | null>(null);
  const [picturesUnavailable, setPicturesUnavailable] = useState(false);
  const [previewIndex, setPreviewIndex] = useState<number | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [lightboxLoading, setLightboxLoading] = useState(false);
  const [websiteVisible, setWebsiteVisible] = useState(false);
  const [selectedImages, setSelectedImages] = useState<Set<string>>(new Set());
  const [websiteThumbnail, setWebsiteThumbnail] = useState<string | null>(null);
  const [savingWebsite, setSavingWebsite] = useState(false);
  const [websiteMessage, setWebsiteMessage] = useState<string | null>(null);

  const formatCurrency = (value?: number) => {
    if (value == null) return '—';
    const locale = i18n.language?.startsWith('de') ? 'de-DE' : 'en-US';
    const formatted = new Intl.NumberFormat(locale, {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value);
    return `${formatted} ${currencySymbol}`;
  };

  const formatMeasurement = (value?: number, unit?: string) => {
    if (value == null) return '—';
    return `${value.toFixed(2)} ${unit || ''}`;
  };

  const categoryPath = () => {
    if (!product) return '—';
    const parts = [product.category_name, product.subcategory_name, product.subbiercategory_name].filter(Boolean);
    return parts.length > 0 ? parts.join(' › ') : '—';
  };

  const loadPictures = async () => {
    if (!product) return;
    setLoadingPictures(true);
    setPictureError(null);
    setPicturesUnavailable(false);
    try {
      const response = await productPicturesApi.list(product.product_id);
      setPictures(response.data.pictures || []);
    } catch (error) {
      console.error('Failed to load product pictures', error);
      const status = (error as { response?: { status?: number } })?.response?.status;
      if (status === 503 || status === 500) {
        setPicturesUnavailable(true);
        setPictureError(t('modals.productDetail.errors.storageUnavailable'));
      } else {
        setPictureError(t('modals.productDetail.errors.loadPictures'));
      }
      setPictures([]);
    } finally {
      setLoadingPictures(false);
    }
  };

  useEffect(() => {
    if (isOpen && product) {
      loadPictures();
      setWebsiteVisible(Boolean(product.website_visible));
      setSelectedImages(new Set(product.website_images || []));
      setWebsiteThumbnail(product.website_thumbnail || null);
      setWebsiteMessage(null);
    } else {
      setPictures([]);
      setPictureError(null);
      setSelectedImages(new Set());
      setWebsiteThumbnail(null);
      setWebsiteMessage(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, product?.product_id]);

  const handleUploadPictures = async (event: ChangeEvent<HTMLInputElement>) => {
    if (!product) return;
    const files = event.target.files;
    if (!files || files.length === 0) return;

    const now = new Date().toISOString();
    const tempPreviews: ProductPicture[] = Array.from(files as ArrayLike<File>).map(file => {
      const url = URL.createObjectURL(file);
      return {
        file_name: file.name,
        size: file.size,
        content_type: file.type || 'image/*',
        modified_at: now,
        download_url: url,
        thumbnail_url: url,
        preview_url: url,
        temporary: true,
      };
    });
    if (tempPreviews.length > 0) {
      setPictures(prev => [...tempPreviews, ...prev]);
    }

    setUploadingPictures(true);
    setPictureError(null);
    try {
      await productPicturesApi.upload(product.product_id, files);
      await loadPictures();
    } catch (error) {
      console.error('Failed to upload product pictures', error);
      setPictureError(t('modals.productDetail.errors.upload'));
      setPictures(prev => prev.filter(pic => !pic.temporary));
    } finally {
      tempPreviews.forEach(pic => {
        if (pic.download_url.startsWith('blob:')) {
          URL.revokeObjectURL(pic.download_url);
        }
      });
      setUploadingPictures(false);
      event.target.value = '';
    }
  };

  const handleDeletePicture = async (fileName: string) => {
    if (!product) return;
    setDeleting(fileName);
    setPictureError(null);
    try {
      await productPicturesApi.delete(product.product_id, fileName);
      await loadPictures();
      if (previewIndex !== null && pictures[previewIndex]?.file_name === fileName) {
        setPreviewIndex(null);
      }
    } catch (error) {
      console.error('Failed to delete product picture', error);
      setPictureError(t('modals.productDetail.errors.deletePicture'));
    } finally {
      setDeleting(null);
    }
  };

  const toggleWebsiteImage = (fileName: string) => {
    setSelectedImages(prev => {
      const next = new Set(prev);
      if (next.has(fileName)) {
        next.delete(fileName);
        if (websiteThumbnail === fileName) {
          setWebsiteThumbnail(null);
        }
      } else {
        next.add(fileName);
      }
      return next;
    });
    setWebsiteMessage(null);
  };

  const persistWebsiteSettings = async (overrideVisible?: boolean) => {
    if (!product) return;
    setSavingWebsite(true);
    setWebsiteMessage(null);
    setPictureError(null);
    const images = Array.from(selectedImages);
    try {
      await productWebsiteApi.update(product.product_id, {
        website_visible: overrideVisible ?? websiteVisible,
        website_images: images,
        website_thumbnail: websiteThumbnail ?? undefined,
      });
      setWebsiteMessage(t('modals.productDetail.website.saved'));
    } catch (error) {
      console.error('Failed to save website settings', error);
      setPictureError(t('modals.productDetail.errors.saveWebsite'));
    } finally {
      setSavingWebsite(false);
    }
  };

  const handleSaveWebsite = async () => {
    await persistWebsiteSettings();
  };

  useEffect(() => {
    if (previewIndex === null || pictures.length === 0) return;
    const current = pictures[previewIndex];
    setLightboxLoading(false);
    // Preload next/prev lightly without blocking
    const preload = (url: string) => {
      if (!url) return;
      const img = new Image();
      img.src = url;
    };
    const next = pictures[(previewIndex + 1) % pictures.length];
    const prev = pictures[(previewIndex - 1 + pictures.length) % pictures.length];
    preload(current.download_url);
    preload(next.download_url);
    preload(prev.download_url);
  }, [previewIndex, pictures]);

  const formatDate = (iso: string) => {
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return '—';
    const locale = i18n.language?.startsWith('de') ? 'de-DE' : 'en-US';
    return date.toLocaleString(locale);
  };

  if (!isOpen || !product) return null;

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
        <div className="glass-dark w-full max-w-4xl rounded-2xl shadow-2xl border border-white/10 flex flex-col max-h-[90vh]">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-white/10">
            <div className="flex items-center gap-3">
              <Package className="w-8 h-8 text-accent-red" />
              <div>
                <h2 className="text-2xl font-bold text-white">{product.name}</h2>
                <p className="text-sm text-gray-400">
                  {t('modals.productDetail.productId', { id: product.product_id })}
                  {product.device_count !== undefined && ` • ${t('modals.productDetail.deviceCount', { count: product.device_count })}`}
                </p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="px-4 py-2 rounded-lg text-sm font-semibold bg-white/10 text-white hover:bg-white/20 transition-colors"
              aria-label={t('common.close')}
              title={t('common.close')}
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Content */}
          <div className="overflow-y-auto p-6 space-y-6">
            <div className="glass rounded-xl p-4">
              <div className="flex items-center justify-between mb-3 gap-3">
                <div className="flex items-center gap-2">
                  <ImageIcon className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.images.title')}</h3>
                </div>
                <label className="inline-flex items-center gap-2 rounded-lg bg-white/10 px-3 py-2 text-sm font-semibold text-white cursor-pointer hover:bg-white/20 transition disabled:opacity-60 disabled:cursor-not-allowed">
                  <UploadCloud className="w-4 h-4" />
                  <span>{uploadingPictures ? t('common.loading') : t('modals.productDetail.images.upload')}</span>
                  <input
                    type="file"
                    accept="image/*"
                    multiple
                    onChange={handleUploadPictures}
                    disabled={uploadingPictures || picturesUnavailable}
                    className="hidden"
                  />
                </label>
              </div>
              {pictureError && <p className="text-sm text-red-400 mb-2">{pictureError}</p>}
              {picturesUnavailable ? (
                <p className="text-gray-400">{t('modals.productDetail.images.notConfigured')}</p>
              ) : loadingPictures ? (
                <div className="flex items-center gap-2 text-gray-300">
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>{t('modals.productDetail.images.loading')}</span>
                </div>
              ) : pictures.length === 0 ? (
                <p className="text-gray-400">{t('modals.productDetail.images.empty')}</p>
              ) : (
                <div className="grid grid-cols-2 gap-3 md:grid-cols-3">
                  {pictures.map((picture, index) => (
                    <div
                      key={picture.file_name}
                      className="group relative overflow-hidden rounded-lg border border-white/10 bg-white/5 cursor-zoom-in"
                      onClick={() => setPreviewIndex(index)}
                    >
                      <img
                        src={picture.thumbnail_url || picture.preview_url || picture.download_url}
                        alt={t('modals.productDetail.images.alt', { name: product.name })}
                        className="h-36 w-full object-cover transition duration-300 group-hover:scale-105"
                        loading="lazy"
                      />
                      <button
                        type="button"
                        onClick={event => {
                          event.stopPropagation();
                          handleDeletePicture(picture.file_name);
                        }}
                        className="absolute right-2 top-2 rounded-full bg-black/70 px-2 py-1 text-xs text-white opacity-90 transition group-hover:opacity-100 disabled:opacity-50"
                        disabled={deleting === picture.file_name}
                        title={t('modals.productDetail.images.deleteTitle')}
                      >
                        {deleting === picture.file_name ? '...' : t('common.delete')}
                      </button>
                      <div className="absolute inset-0 bg-black/50 opacity-0 transition-opacity duration-200 group-hover:opacity-100" />
                      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-2 text-xs text-white">
                        <p className="font-semibold break-all">{picture.file_name}</p>
                        <p className="text-gray-300">{formatDate(picture.modified_at)}</p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Website Settings */}
            <div className="glass rounded-xl p-4 space-y-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Eye className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.website.title')}</h3>
                </div>
                <label className="flex items-center gap-2 text-sm text-gray-200 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    className="h-4 w-4 rounded border-gray-600 text-accent-red focus:ring-accent-red"
                    checked={websiteVisible}
                    onChange={e => {
                      const next = e.target.checked;
                      setWebsiteVisible(next);
                      void persistWebsiteSettings(next);
                    }}
                  />
                  {t('modals.productDetail.website.showOnWebsite')}
                </label>
              </div>
              {pictures.length > 0 ? (
                <div className="space-y-2">
                  <p className="text-sm text-gray-400">{t('modals.productDetail.website.selectImages')}</p>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {pictures.map(pic => (
                      <div key={pic.file_name} className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 p-2 min-w-0">
                        <img
                          src={pic.thumbnail_url || pic.preview_url || pic.download_url}
                          alt=""
                          className="h-16 w-16 rounded object-cover flex-shrink-0"
                          loading="lazy"
                        />
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-white font-semibold break-all">{pic.file_name}</p>
                          <div className="flex items-center gap-3 mt-1 flex-wrap">
                            <label className="flex items-center gap-1 text-xs text-gray-200">
                              <input
                                type="checkbox"
                                checked={selectedImages.has(pic.file_name)}
                                onChange={() => toggleWebsiteImage(pic.file_name)}
                              />
                              {t('modals.productDetail.website.onWebsite')}
                            </label>
                            <label className="flex items-center gap-1 text-xs text-gray-200">
                              <input
                                type="radio"
                                name="website-thumb"
                                disabled={!selectedImages.has(pic.file_name)}
                                checked={websiteThumbnail === pic.file_name}
                                onChange={() => {
                                  if (!selectedImages.has(pic.file_name)) toggleWebsiteImage(pic.file_name);
                                  setWebsiteThumbnail(pic.file_name);
                                  setWebsiteMessage(null);
                                }}
                              />
                              {t('modals.productDetail.website.thumbnail')}
                            </label>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                  <div className="flex items-center justify-end gap-3">
                    {websiteMessage && <span className="text-xs text-green-400">{websiteMessage}</span>}
                    <button
                      onClick={handleSaveWebsite}
                      disabled={savingWebsite}
                      className="px-4 py-2 rounded-lg bg-accent-red text-white text-sm font-semibold hover:bg-accent-red/90 transition disabled:opacity-60"
                    >
                      {savingWebsite ? t('common.saving') : t('modals.productDetail.website.save')}
                    </button>
                  </div>
                </div>
              ) : (
                <p className="text-sm text-gray-400">{t('modals.productDetail.website.noImages')}</p>
              )}
            </div>

            {/* Type Badges */}
            <div className="flex gap-2">
              {product.is_consumable && (
                <span className="px-3 py-1 rounded-full bg-blue-500/20 text-blue-400 text-sm font-semibold">
                  {t('zoneDetail.consumable')}
                </span>
              )}
              {product.is_accessory && (
                <span className="px-3 py-1 rounded-full bg-purple-500/20 text-purple-400 text-sm font-semibold">
                  {t('zoneDetail.accessory')}
                </span>
              )}
              {!product.is_consumable && !product.is_accessory && (
                <span className="px-3 py-1 rounded-full bg-green-500/20 text-green-400 text-sm font-semibold">
                  {t('modals.productDetail.standardProduct')}
                </span>
              )}
            </div>

            {/* Description */}
            {product.description && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-2 mb-2">
                  <Info className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('cases.description')}</h3>
                </div>
                <p className="text-gray-300">{product.description}</p>
              </div>
            )}

            {/* Category & Classification */}
            <div className="glass rounded-xl p-4">
              <div className="flex items-center gap-2 mb-3">
                <Tag className="w-5 h-5 text-accent-red" />
                <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.classification')}</h3>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <div>
                  <p className="text-sm text-gray-400">{t('modals.productDetail.categoryPath')}</p>
                  <p className="text-white font-medium">{categoryPath()}</p>
                </div>
                {product.brand_name && (
                  <div>
                    <p className="text-sm text-gray-400">{t('admin.tabs.brands')}</p>
                    <p className="text-white font-medium">{product.brand_name}</p>
                  </div>
                )}
                {product.manufacturer_name && (
                  <div>
                    <p className="text-sm text-gray-400">{t('modals.productDetail.manufacturer')}</p>
                    <p className="text-white font-medium">{product.manufacturer_name}</p>
                  </div>
                )}
                {product.pos_in_category != null && (
                  <div>
                    <p className="text-sm text-gray-400">{t('modals.productDetail.positionInCategory')}</p>
                    <p className="text-white font-medium">{product.pos_in_category}</p>
                  </div>
                )}
              </div>
            </div>

            {/* Pricing */}
            {(product.item_cost_per_day != null || product.price_per_unit != null) && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-2 mb-3">
                  <DollarSign className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.prices')}</h3>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {product.item_cost_per_day != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('modals.productDetail.pricePerDay')}</p>
                      <p className="text-2xl font-bold text-white">{formatCurrency(product.item_cost_per_day)}</p>
                    </div>
                  )}
                  {product.price_per_unit != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('modals.productDetail.pricePerUnit')}</p>
                      <p className="text-2xl font-bold text-white">{formatCurrency(product.price_per_unit)}</p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Stock Information (for consumables/accessories) */}
            {(product.is_consumable || product.is_accessory) && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Box className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.stock')}</h3>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                  <div className="bg-white/5 rounded-lg p-3">
                    <p className="text-sm text-gray-400">{t('modals.productDetail.currentStock')}</p>
                    <p className="text-xl font-bold text-white">
                      {product.stock_quantity != null ? product.stock_quantity : '—'} {product.count_type_abbreviation || ''}
                    </p>
                  </div>
                  <div className="bg-white/5 rounded-lg p-3">
                    <p className="text-sm text-gray-400">{t('modals.productDetail.minStock')}</p>
                    <p className="text-xl font-bold text-white">
                      {product.min_stock_level != null ? product.min_stock_level : '—'} {product.count_type_abbreviation || ''}
                    </p>
                  </div>
                  {product.generic_barcode && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('devices.barcode')}</p>
                      <p className="text-sm font-mono text-white">{product.generic_barcode}</p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Physical Properties */}
            {(product.weight != null || product.height != null || product.width != null || product.depth != null) && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Ruler className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.physicalProperties')}</h3>
                </div>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  {product.weight != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <div className="flex items-center gap-2 mb-1">
                        <Weight className="w-4 h-4 text-gray-400" />
                        <p className="text-sm text-gray-400">{t('cases.weight')}</p>
                      </div>
                      <p className="text-lg font-semibold text-white">{formatMeasurement(product.weight, 'kg')}</p>
                    </div>
                  )}
                  {product.height != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('casesPage.height')}</p>
                      <p className="text-lg font-semibold text-white">{formatMeasurement(product.height, 'cm')}</p>
                    </div>
                  )}
                  {product.width != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('casesPage.width')}</p>
                      <p className="text-lg font-semibold text-white">{formatMeasurement(product.width, 'cm')}</p>
                    </div>
                  )}
                  {product.depth != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('casesPage.depth')}</p>
                      <p className="text-lg font-semibold text-white">{formatMeasurement(product.depth, 'cm')}</p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Technical Details */}
            {(product.power_consumption != null || product.maintenance_interval != null) && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Zap className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.technicalDetails')}</h3>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {product.power_consumption != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <p className="text-sm text-gray-400">{t('modals.productDetail.powerConsumption')}</p>
                      <p className="text-lg font-semibold text-white">{formatMeasurement(product.power_consumption, 'W')}</p>
                    </div>
                  )}
                  {product.maintenance_interval != null && (
                    <div className="bg-white/5 rounded-lg p-3">
                      <div className="flex items-center gap-2 mb-1">
                        <Wrench className="w-4 h-4 text-gray-400" />
                        <p className="text-sm text-gray-400">{t('modals.productDetail.maintenanceInterval')}</p>
                      </div>
                      <p className="text-lg font-semibold text-white">{t('modals.productDetail.days', { count: product.maintenance_interval })}</p>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Barcode */}
            {product.generic_barcode && !product.is_consumable && !product.is_accessory && (
              <div className="glass rounded-xl p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Barcode className="w-5 h-5 text-accent-red" />
                  <h3 className="text-lg font-semibold text-white">{t('modals.productDetail.identification')}</h3>
                </div>
                <div className="bg-white/5 rounded-lg p-3">
                  <p className="text-sm text-gray-400">{t('modals.productDetail.genericBarcode')}</p>
                  <p className="text-lg font-mono text-white">{product.generic_barcode}</p>
                </div>
              </div>
            )}
          </div>

          {previewIndex !== null && pictures[previewIndex] && (
            <div className="fixed inset-0 z-[130] flex items-center justify-center bg-black/90 p-4">
              <div className="relative w-full max-w-5xl">
                <button
                  onClick={() => setPreviewIndex(null)}
                  className="absolute right-4 top-4 rounded-full bg-white/10 px-3 py-2 text-sm font-semibold text-white hover:bg-white/20"
                >
                  {t('common.close')}
                </button>
                <div className="absolute left-2 top-1/2 -translate-y-1/2">
                  <button
                    onClick={() =>
                      setPreviewIndex(prev => {
                        if (prev === null) return prev;
                        return (prev - 1 + pictures.length) % pictures.length;
                      })
                    }
                    className="rounded-full bg-white/10 px-3 py-2 text-white hover:bg-white/20"
                  >
                    ‹
                  </button>
                </div>
                <div className="absolute right-2 top-1/2 -translate-y-1/2">
                  <button
                    onClick={() =>
                      setPreviewIndex(prev => {
                        if (prev === null) return prev;
                        return (prev + 1) % pictures.length;
                      })
                    }
                    className="rounded-full bg-white/10 px-3 py-2 text-white hover:bg-white/20"
                  >
                    ›
                  </button>
                </div>
                <div className="overflow-hidden rounded-xl border border-white/10 bg-white/5">
                  <img
                    src={pictures[previewIndex].download_url}
                    alt={pictures[previewIndex].file_name}
                    className="max-h-[80vh] w-full object-contain bg-black"
                    onLoad={() => setLightboxLoading(false)}
                    loading="eager"
                  />
                  {lightboxLoading && (
                    <div className="absolute inset-0 flex items-center justify-center bg-black/60 text-white">
                      {t('common.loading')}
                    </div>
                  )}
                </div>
                <div className="mt-3 flex items-center justify-between text-sm text-gray-200">
                  <div>
                    <p className="font-semibold text-white">{pictures[previewIndex].file_name}</p>
                    <p>{formatDate(pictures[previewIndex].modified_at)}</p>
                  </div>
                  <button
                    onClick={() => handleDeletePicture(pictures[previewIndex].file_name)}
                    className="rounded-lg bg-red-600 px-4 py-2 font-semibold text-white hover:bg-red-700 disabled:opacity-60"
                    disabled={deleting === pictures[previewIndex].file_name}
                  >
                    {deleting === pictures[previewIndex].file_name ? t('modals.productDetail.images.deleting') : t('modals.productDetail.images.delete')}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Footer */}
          <div className="flex justify-end gap-3 p-6 border-t border-white/10">
            <button
              onClick={onClose}
              className="px-6 py-2 rounded-lg text-sm font-semibold bg-white/10 text-white hover:bg-white/20 transition-colors"
            >
              {t('common.close')}
            </button>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
