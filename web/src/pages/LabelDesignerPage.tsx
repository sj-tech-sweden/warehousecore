import { useState, useEffect, useRef } from 'react';
import { Trash2, Download, Printer, QrCode, Barcode, Type, Save, Image as ImageIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { labelsApi, devicesApi, casesApi, zonesApi } from '../lib/api';
import type { LabelTemplate, LabelElement, Device, CaseSummary, Zone } from '../lib/api';
import JSZip from 'jszip';
import './LabelDesignerPage.css';

interface DesignElement extends LabelElement {
  id: string;
  image_data?: string; // Base64 encoded image data
}

export default function LabelDesignerPage() {
  const { t } = useTranslation();
  const presetSizes = [
    { name: t('labels.presets.standard'), width: 62, height: 29 },
    { name: t('labels.presets.large'), width: 100, height: 50 },
    { name: t('labels.presets.small'), width: 50, height: 25 },
    { name: t('labels.presets.custom'), width: 0, height: 0 },
  ];
  const [labelWidth, setLabelWidth] = useState(62);
  const [labelHeight, setLabelHeight] = useState(29);
  const [elements, setElements] = useState<DesignElement[]>([]);
  const [selectedElement, setSelectedElement] = useState<string | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [cases, setCases] = useState<CaseSummary[]>([]);
  const [zones, setZones] = useState<Zone[]>([]);
  const [previewDevice, setPreviewDevice] = useState<Device | null>(null);
  const [exporting, setExporting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [templateName, setTemplateName] = useState('');
  const [templates, setTemplates] = useState<LabelTemplate[]>([]);
  const [currentTemplateId, setCurrentTemplateId] = useState<number | null>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const canvasContainerRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const dragCleanupRef = useRef<(() => void) | null>(null);

  // Cancel any in-flight drag/resize when the component unmounts or the
  // window loses focus (e.g. mid-drag Alt+Tab).
  useEffect(() => {
    const cancelDrag = () => dragCleanupRef.current?.();
    window.addEventListener('blur', cancelDrag);
    return () => {
      window.removeEventListener('blur', cancelDrag);
      dragCleanupRef.current?.();
    };
  }, []);

  useEffect(() => {
    loadDevices();
    loadCases();
    loadZones();
    loadTemplates();
  }, []);

  useEffect(() => {
    setTemplateName((current) => current || t('labels.newTemplate'));
  }, [t]);

  const loadDevices = async () => {
    try {
      // Set high limit to load all devices (default backend limit is only 100)
      const { data } = await devicesApi.getAll({ limit: 50000 });
      setDevices(data);
      if (data.length > 0) {
        setPreviewDevice(data[0]);
      }
    } catch (error) {
      console.error('Failed to load devices:', error);
    }
  };

  const loadCases = async () => {
    try {
      const { data } = await casesApi.list({});
      setCases(data.cases || []);
    } catch (error) {
      console.error('Failed to load cases:', error);
    }
  };

  const loadZones = async () => {
    try {
      const { data } = await zonesApi.getAll();
      setZones(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to load zones:', error);
    }
  };

  const loadTemplates = async () => {
    try {
      const { data } = await labelsApi.getTemplates();
      const safeTemplates = Array.isArray(data) ? data : [];
      setTemplates(safeTemplates);

      // Load default template if exists
      const defaultTemplate = safeTemplates.find((t) => t.is_default);
      if (defaultTemplate) {
        loadTemplate(defaultTemplate);
      }
    } catch (error) {
      console.error('Failed to load templates:', error);
    }
  };

  const loadTemplate = (template: LabelTemplate) => {
    setCurrentTemplateId(template.id || null);
    setLabelWidth(template.width);
    setLabelHeight(template.height);
    setTemplateName(template.name);
    try {
      const raw = template.template_json?.trim();
      const parsed = raw ? JSON.parse(raw) : [];
      const safeElements = Array.isArray(parsed) ? parsed : [];
      setElements(safeElements.map((e: LabelElement, i: number) => ({ ...e, id: `elem-${i}` })));
    } catch (error) {
      console.error('Invalid template_json, loading empty design:', error);
      setElements([]);
    }
  };

  const createNewTemplate = () => {
    setCurrentTemplateId(null);
    setTemplateName(t('labels.newTemplate'));
    setLabelWidth(62);
    setLabelHeight(29);
    setElements([]);
    setSelectedElement(null);
  };

  const deleteTemplate = async (id: number) => {
    if (!confirm(t('labels.confirmDelete'))) return;

    try {
      await labelsApi.deleteTemplate(id);
      await loadTemplates();
      if (currentTemplateId === id) {
        createNewTemplate();
      }
      alert(t('labels.templateDeleted'));
    } catch (error) {
      console.error('Failed to delete template:', error);
      alert(t('labels.deleteError'));
    }
  };

  const addElement = (type: 'qrcode' | 'barcode' | 'text' | 'image') => {
    const newElement: DesignElement = {
      id: `elem-${Date.now()}`,
      type,
      x: 5,
      y: 5,
      width: type === 'qrcode' ? 25 : type === 'barcode' ? 50 : type === 'image' ? 20 : 30,
      height: type === 'qrcode' ? 25 : type === 'barcode' ? 15 : type === 'image' ? 20 : 6,
      rotation: 0,
      content: type === 'text' ? 'device_id' : type === 'image' ? '' : 'device_id',
      style: {
        font_size: 10,
        font_weight: 'normal',
        font_family: 'Arial',
        color: '#000000',
        alignment: 'left',
        format: type === 'barcode' ? 'code128' : type === 'qrcode' ? 'qr' : undefined,
      },
    };
    setElements([...elements, newElement]);
    setSelectedElement(newElement.id);
  };

  const handleImageUpload = (elementId: string, file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const base64 = e.target?.result as string;
      updateElement(elementId, { image_data: base64, content: file.name });
    };
    reader.readAsDataURL(file);
  };

  const deleteElement = (id: string) => {
    setElements(elements.filter((e) => e.id !== id));
    if (selectedElement === id) {
      setSelectedElement(null);
    }
  };

  const updateElement = (id: string, updates: Partial<DesignElement>) => {
    setElements(elements.map((e) => (e.id === id ? { ...e, ...updates } : e)));
  };

  const saveTemplate = async () => {
    if (!templateName.trim()) {
      alert(t('labels.templateNameRequired'));
      return;
    }

    setSaving(true);
    try {
      const templateJSON = JSON.stringify(
        elements.map(({ id, ...rest }) => rest)
      );

      const template: LabelTemplate = {
        name: templateName,
        description: '',
        width: labelWidth,
        height: labelHeight,
        template_json: templateJSON,
        is_default: false,
      };

      if (currentTemplateId) {
        // Update existing template
        await labelsApi.updateTemplate(currentTemplateId, template);
        alert(t('labels.templateUpdated'));
      } else {
        // Create new template
        const { data } = await labelsApi.createTemplate(template);
        setCurrentTemplateId(data.id || null);
        alert(t('labels.templateSaved'));
      }

      await loadTemplates();
    } catch (error) {
      console.error('Failed to save template:', error);
      alert(t('labels.saveError'));
    } finally {
      setSaving(false);
    }
  };

  const setAsDefault = async () => {
    if (!currentTemplateId) {
      alert(t('labels.saveFirst'));
      return;
    }

    try {
      await labelsApi.updateTemplate(currentTemplateId, { is_default: true });
      await loadTemplates();
      alert(t('labels.setAsDefaultSuccess'));
    } catch (error) {
      console.error('Failed to set default:', error);
      alert(t('labels.setDefaultError'));
    }
  };

  const renderPreview = async () => {
    if (!previewDevice || !canvasRef.current) return;

    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpi = 300;
    const mmToPx = dpi / 25.4;
    const width = labelWidth * mmToPx;
    const height = labelHeight * mmToPx;

    canvas.width = width;
    canvas.height = height;

    // White background
    ctx.fillStyle = '#ffffff';
    ctx.fillRect(0, 0, width, height);

    // Border
    ctx.strokeStyle = '#e0e0e0';
    ctx.lineWidth = 2;
    ctx.strokeRect(0, 0, width, height);

    // Determine if we're previewing a zone or case to enrich the field map
    const isZone = previewDevice.device_id.startsWith('ZONE-');
    const isCase = previewDevice.device_id.startsWith('CASE-');
    const previewZone = isZone
      ? zones.find(z => z.zone_id === previewDevice.zone_id)
      : undefined;
    const previewCase = isCase
      ? cases.find(c => `CASE-${c.case_id}` === previewDevice.device_id)
      : undefined;

    // Resolve parent zone name/code for zone preview
    const parentZone = previewZone?.parent_zone_id
      ? zones.find(z => z.zone_id === previewZone.parent_zone_id)
      : undefined;

    // Build field map from preview device/case/zone (mirrors backend field resolution)
    // Compute product dimensions string if individual dimensions available
    const productDims = (() => {
      const pw = previewDevice.product_width;
      const ph = previewDevice.product_height;
      const pd = previewDevice.product_depth;
      if (pw != null && ph != null && pd != null) {
        return `${pw.toFixed(1)}x${ph.toFixed(1)}x${pd.toFixed(1)} cm`;
      }
      return '';
    })();

    const fieldMap: Record<string, string> = {
      // Device fields
      device_id: previewDevice.device_id,
      device_name: previewDevice.device_id,
      name: previewDevice.product_name || '',
      product_name: previewDevice.product_name || '',
      product_description: previewDevice.product_description || '',
      serial_number: previewDevice.serial_number || '',
      barcode: previewDevice.barcode || '',
      rfid: previewDevice.rfid || '',
      qr_code: previewDevice.qr_code || '',
      status: previewDevice.status || '',
      zone_name: isZone ? (previewZone?.name || previewDevice.zone_name || '') : (previewDevice.zone_name || ''),
      zone_code: isZone ? (previewZone?.code || previewDevice.zone_code || '') : (previewDevice.zone_code || ''),
      case_name: previewDevice.case_name || '',
      notes: previewDevice.notes || '',
      purchase_date: previewDevice.purchase_date || '',
      // Product metadata fields (populated from enriched device detail)
      subcategory: previewDevice.subcategory || '',
      product: previewDevice.product_name || '',
      category: previewDevice.product_category || '',
      manufacturer: previewDevice.manufacturer_name || '',
      manufacturer_name: previewDevice.manufacturer_name || '',
      brand: previewDevice.brand_name || '',
      brand_name: previewDevice.brand_name || '',
      condition_rating: previewDevice.condition_rating != null ? previewDevice.condition_rating.toFixed(1) : '',
      usage_hours: previewDevice.usage_hours != null ? `${previewDevice.usage_hours} h` : '',
      product_weight: previewDevice.product_weight != null ? String(previewDevice.product_weight) : '',
      product_dimensions: productDims,
      maintenance_interval: previewDevice.maintenance_interval != null ? String(previewDevice.maintenance_interval) : '',
      power_consumption: previewDevice.power_consumption != null ? String(previewDevice.power_consumption) : '',
      // Case fields — only populated when the preview entity is a case
      case_id: isCase ? previewDevice.device_id : '',
      description: isCase ? (previewCase?.description || '') : '',
      dimensions: isCase ? (previewCase?.width != null && previewCase?.height != null && previewCase?.depth != null
        ? `${previewCase.width.toFixed(1)}x${previewCase.height.toFixed(1)}x${previewCase.depth.toFixed(1)} cm`
        : '') : '',
      weight: isCase ? (previewCase?.weight != null ? `${previewCase.weight.toFixed(1)} kg` : '') : '',
      rfid_tag: isCase ? (previewCase?.rfid_tag || '') : '',
      // Zone fields — only populated when the preview entity is a zone
      code: isZone ? (previewZone?.code || previewDevice.zone_code || '') : '',
      zone_id: isZone && previewDevice.zone_id != null ? String(previewDevice.zone_id) : '',
      type: isZone ? (previewZone?.type || '') : '',
      zone_type: isZone ? (previewZone?.type || '') : '',
      location: isZone ? (previewZone?.location || '') : '',
      capacity: isZone && previewZone?.capacity != null ? String(previewZone.capacity) : '',
      parent_name: parentZone?.name || '',
      parent_code: parentZone?.code || '',
    };

    // Render elements
    for (const elem of elements) {
      const x = elem.x * mmToPx;
      const y = elem.y * mmToPx;
      const w = elem.width * mmToPx;
      const h = elem.height * mmToPx;

      // Resolve content from field map
      const content = fieldMap[elem.content] !== undefined ? fieldMap[elem.content] : elem.content;

      if (elem.type === 'qrcode') {
        try {
          const { data } = await labelsApi.generateQRCode(content, Math.floor(w));
          const img = new Image();
          await new Promise<void>((resolve) => {
            img.onload = () => {
              ctx.drawImage(img, x, y, w, h);
              resolve();
            };
            img.src = data.image_data;
          });
        } catch (err) {
          console.error('QR code generation failed:', err);
        }
      } else if (elem.type === 'barcode') {
        try {
          const { data } = await labelsApi.generateBarcode(content, Math.floor(w), Math.floor(h));
          const img = new Image();
          await new Promise<void>((resolve) => {
            img.onload = () => {
              ctx.drawImage(img, x, y, w, h);
              resolve();
            };
            img.src = data.image_data;
          });
        } catch (err) {
          console.error('Barcode generation failed:', err);
        }
      } else if (elem.type === 'image') {
        if (elem.image_data) {
          try {
            const img = new Image();
            await new Promise<void>((resolve) => {
              img.onload = () => {
                ctx.drawImage(img, x, y, w, h);
                resolve();
              };
              img.src = elem.image_data!;
            });
          } catch (err) {
            console.error('Image rendering failed:', err);
          }
        }
      } else if (elem.type === 'text') {
        ctx.fillStyle = elem.style.color || '#000000';
        ctx.font = `${elem.style.font_weight || 'normal'} ${(elem.style.font_size || 10) * (dpi / 96)}px ${elem.style.font_family || 'Arial'}`;
        ctx.textAlign = (elem.style.alignment as CanvasTextAlign) || 'left';
        ctx.fillText(content, x, y + (elem.style.font_size || 10) * (dpi / 96));
      }
    }
  };

  useEffect(() => {
    if (previewDevice && !isDragging) {
      renderPreview();
    }
  }, [elements, labelWidth, labelHeight, previewDevice, isDragging]);

  const generateAllLabels = async () => {
    const totalItems = devices.length + cases.length + zones.length;
    if (totalItems === 0) {
      alert(t('labels.noDevicesOrCases'));
      return;
    }

    // Find default template
    const defaultTemplate = templates.find((t) => t.is_default);
    if (!defaultTemplate) {
      alert(t('labels.setDefaultFirst'));
      return;
    }

    // Save current state
    const originalTemplateId = currentTemplateId;

    setExporting(true);
    let successCount = 0;
    let failCount = 0;
    try {
      // Load default template temporarily
      loadTemplate(defaultTemplate);
      await new Promise((r) => setTimeout(r, 1000)); // Wait for template to fully load

      // Generate labels for all devices
      for (const device of devices) {
        setPreviewDevice(device);
        await new Promise((r) => setTimeout(r, 500)); // Longer wait for canvas render

        const canvas = canvasRef.current;
        if (canvas) {
          const imageData = canvas.toDataURL('image/png');
          try {
            await labelsApi.saveLabel(device.device_id, imageData);
            successCount++;
          } catch (error) {
            console.error(`Failed to save label for ${device.device_id}:`, error);
            failCount++;
          }
        }
      }

      // Generate labels for all cases
      for (const caseItem of cases) {
        // Convert case to device-like object for rendering
        const caseAsDevice: Device = {
          device_id: `CASE-${caseItem.case_id}`,
          product_name: caseItem.name,
          status: caseItem.status,
          zone_code: caseItem.zone_code,
          zone_name: caseItem.zone_name,
          zone_id: caseItem.zone_id,
        };

        setPreviewDevice(caseAsDevice);
        await new Promise((r) => setTimeout(r, 500)); // Same wait time as devices for consistent rendering

        const canvas = canvasRef.current;
        if (canvas) {
          const imageData = canvas.toDataURL('image/png');
          try {
            await labelsApi.saveCaseLabel(caseItem.case_id, imageData);
            successCount++;
          } catch (error) {
            console.error(`Failed to save label for CASE-${caseItem.case_id}:`, error);
            failCount++;
          }
        }
      }

      // Generate labels for all zones
      for (const zone of zones) {
        const zoneAsDevice: Device = {
          device_id: `ZONE-${zone.zone_id}`,
          product_name: zone.name,
          status: zone.type,
          zone_code: zone.code,
          zone_name: zone.name,
          zone_id: zone.zone_id,
        };

        setPreviewDevice(zoneAsDevice);
        await new Promise((r) => setTimeout(r, 500));

        const canvas = canvasRef.current;
        if (canvas) {
          const imageData = canvas.toDataURL('image/png');
          try {
            await labelsApi.saveZoneLabel(zone.zone_id, imageData);
            successCount++;
          } catch (error) {
            console.error(`Failed to save label for ZONE-${zone.zone_id}:`, error);
            failCount++;
          }
        }
      }

      alert(t('labels.labelsGenerated', {
        success: successCount,
        total: totalItems,
        devices: devices.length,
        cases: cases.length,
        errors: failCount > 0 ? `\n${t('labels.errorsCount', { count: failCount })}` : '',
      }));
    } catch (error) {
      console.error('Generation failed:', error);
      alert(t('labels.generateError'));
    } finally {
      setExporting(false);
      if (devices.length > 0) {
        setPreviewDevice(devices[0]);
      }
      // Restore original template
      if (originalTemplateId) {
        const originalTemplate = templates.find((t) => t.id === originalTemplateId);
        if (originalTemplate) {
          loadTemplate(originalTemplate);
        }
      }
    }
  };

  const generateMissingLabels = async () => {
    // Filter devices and cases without labels
    const devicesWithoutLabels = devices.filter(d => !d.label_path);
    const casesWithoutLabels = cases.filter(c => !c.label_path);
    // Only filter zones if the API returns label_path; otherwise treat all zones as needing labels
    const zonesIncludeLabelPath = zones.some(z => 'label_path' in z);
    const zonesWithoutLabels = zonesIncludeLabelPath ? zones.filter(z => !z.label_path) : zones;
    const totalMissing = devicesWithoutLabels.length + casesWithoutLabels.length + zonesWithoutLabels.length;

    if (totalMissing === 0) {
      alert(t('labels.allHaveLabels'));
      return;
    }

    if (!confirm(t('labels.missingLabelsFound', { count: totalMissing, devices: devicesWithoutLabels.length, cases: casesWithoutLabels.length }))) {
      return;
    }

    // Find default template
    const defaultTemplate = templates.find((t) => t.is_default);
    if (!defaultTemplate) {
      alert(t('labels.setDefaultFirst'));
      return;
    }

    // Save current state
    const originalTemplateId = currentTemplateId;

    setExporting(true);
    let successCount = 0;
    let failCount = 0;
    try {
      // Load default template temporarily
      loadTemplate(defaultTemplate);
      await new Promise((r) => setTimeout(r, 1000)); // Wait for template to fully load

      // Generate labels for devices without labels
      for (const device of devicesWithoutLabels) {
        setPreviewDevice(device);
        await new Promise((r) => setTimeout(r, 500)); // Longer wait for canvas render

        const canvas = canvasRef.current;
        if (canvas) {
          const imageData = canvas.toDataURL('image/png');
          try {
            await labelsApi.saveLabel(device.device_id, imageData);
            successCount++;
          } catch (error) {
            console.error(`Failed to save label for ${device.device_id}:`, error);
            failCount++;
          }
        }
      }

      // Generate labels for cases without labels
      for (const caseItem of casesWithoutLabels) {
        // Convert case to device-like object for rendering
        const caseAsDevice: Device = {
          device_id: `CASE-${caseItem.case_id}`,
          product_name: caseItem.name,
          status: caseItem.status,
          zone_code: caseItem.zone_code,
          zone_name: caseItem.zone_name,
          zone_id: caseItem.zone_id,
        };

        setPreviewDevice(caseAsDevice);
        await new Promise((r) => setTimeout(r, 500)); // Same wait time as devices for consistent rendering

        const canvas = canvasRef.current;
        if (canvas) {
          const imageData = canvas.toDataURL('image/png');
          try {
            await labelsApi.saveCaseLabel(caseItem.case_id, imageData);
            successCount++;
          } catch (error) {
            console.error(`Failed to save label for CASE-${caseItem.case_id}:`, error);
            failCount++;
          }
        }
      }

      // Generate labels for zones without labels
      for (const zone of zonesWithoutLabels) {
        const zoneAsDevice: Device = {
          device_id: `ZONE-${zone.zone_id}`,
          product_name: zone.name,
          status: zone.type,
          zone_code: zone.code,
          zone_name: zone.name,
          zone_id: zone.zone_id,
        };

        setPreviewDevice(zoneAsDevice);
        await new Promise((r) => setTimeout(r, 500));

        const canvas = canvasRef.current;
        if (canvas) {
          const imageData = canvas.toDataURL('image/png');
          try {
            await labelsApi.saveZoneLabel(zone.zone_id, imageData);
            successCount++;
          } catch (error) {
            console.error(`Failed to save label for ZONE-${zone.zone_id}:`, error);
            failCount++;
          }
        }
      }

      alert(t('labels.missingLabelsGenerated', {
        success: successCount,
        total: totalMissing,
        devices: devicesWithoutLabels.length,
        cases: casesWithoutLabels.length,
        errors: failCount > 0 ? `\n${t('labels.errorsCount', { count: failCount })}` : '',
      }));

      // Reload devices, cases and zones to refresh label_path info
      await loadDevices();
      await loadCases();
      await loadZones();
    } catch (error) {
      console.error('Generation failed:', error);
      alert(t('labels.generateError'));
    } finally {
      setExporting(false);
      if (devices.length > 0) {
        setPreviewDevice(devices[0]);
      }
      // Restore original template
      if (originalTemplateId) {
        const originalTemplate = templates.find((t) => t.id === originalTemplateId);
        if (originalTemplate) {
          loadTemplate(originalTemplate);
        }
      }
    }
  };

  const exportAllLabels = async () => {
    setExporting(true);
    try {
      const zip = new JSZip();
      let exportedCount = 0;
      let skippedCount = 0;

      // Export device labels (only those that already have labels)
      for (const device of devices) {
        if (device.label_path) {
          try {
            const response = await fetch(device.label_path);
            if (response.ok) {
              const blob = await response.blob();
              zip.file(`devices/${device.device_id}_label.png`, blob);
              exportedCount++;
            } else {
              console.warn(`Failed to fetch label for ${device.device_id}`);
              skippedCount++;
            }
          } catch (error) {
            console.error(`Error fetching label for ${device.device_id}:`, error);
            skippedCount++;
          }
        } else {
          skippedCount++;
        }
      }

      // Export case labels (only those that already have labels)
      for (const caseItem of cases) {
        if (caseItem.label_path) {
          try {
            const response = await fetch(caseItem.label_path);
            if (response.ok) {
              const blob = await response.blob();
              zip.file(`cases/CASE-${caseItem.case_id}_label.png`, blob);
              exportedCount++;
            } else {
              console.warn(`Failed to fetch label for CASE-${caseItem.case_id}`);
              skippedCount++;
            }
          } catch (error) {
            console.error(`Error fetching label for CASE-${caseItem.case_id}:`, error);
            skippedCount++;
          }
        } else {
          skippedCount++;
        }
      }

      // Export zone labels (only those that already have labels)
      for (const zone of zones) {
        if (zone.label_path) {
          try {
            const response = await fetch(zone.label_path);
            if (response.ok) {
              const blob = await response.blob();
              zip.file(`zones/ZONE-${zone.zone_id}_label.png`, blob);
              exportedCount++;
            } else {
              console.warn(`Failed to fetch label for ZONE-${zone.zone_id}`);
              skippedCount++;
            }
          } catch (error) {
            console.error(`Error fetching label for ZONE-${zone.zone_id}:`, error);
            skippedCount++;
          }
        } else {
          skippedCount++;
        }
      }

      if (exportedCount === 0) {
        alert(t('labels.noGeneratedLabels'));
        return;
      }

      // Generate ZIP file and trigger download
      const blob = await zip.generateAsync({ type: 'blob' });
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.download = `labels_export_${new Date().toISOString().split('T')[0]}.zip`;
      link.click();
      URL.revokeObjectURL(link.href);

      alert(t('labels.labelsExported', {
        count: exportedCount,
        skipped: skippedCount > 0 ? `\n${t('labels.skippedCount', { count: skippedCount })}` : '',
      }));
    } catch (error) {
      console.error('Export failed:', error);
      alert(t('labels.exportError'));
    } finally {
      setExporting(false);
    }
  };

  const handlePrint = () => {
    if (!canvasRef.current) return;
    const dataUrl = canvasRef.current.toDataURL('image/png');
    const printWindow = window.open('', '', 'width=800,height=600');
    if (printWindow) {
      printWindow.document.write(`
        <html>
          <head><title>${t('labels.printPreview')}</title></head>
          <body style="margin:0;display:flex;justify-content:center;align-items:center;">
            <img src="${dataUrl}" onload="window.print();window.close();" />
          </body>
        </html>
      `);
      printWindow.document.close();
    }
  };

  const handleElementMouseDown = (e: React.MouseEvent, id: string) => {
    e.preventDefault();
    e.stopPropagation();
    setSelectedElement(id);

    const elem = elements.find((el) => el.id === id);
    if (!elem || !canvasRef.current) return;

    const canvas = canvasRef.current;
    const startX = e.clientX;
    const startY = e.clientY;
    const origX = elem.x;
    const origY = elem.y;
    const elemWidth = elem.width;
    const elemHeight = elem.height;

    setIsDragging(true);

    const handleMouseMove = (me: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      const deltaXMm = ((me.clientX - startX) / rect.width) * labelWidth;
      const deltaYMm = ((me.clientY - startY) / rect.height) * labelHeight;
      const newX = Math.max(0, Math.min(labelWidth - elemWidth, origX + deltaXMm));
      const newY = Math.max(0, Math.min(labelHeight - elemHeight, origY + deltaYMm));
      setElements((prev) =>
        prev.map((el) =>
          el.id === id
            ? { ...el, x: Math.round(newX * 10) / 10, y: Math.round(newY * 10) / 10 }
            : el
        )
      );
    };

    const cleanup = () => {
      setIsDragging(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', cleanup);
      document.removeEventListener('pointercancel', cleanup);
      dragCleanupRef.current = null;
    };

    dragCleanupRef.current = cleanup;
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', cleanup);
    document.addEventListener('pointercancel', cleanup);
  };

  const handleResizeMouseDown = (e: React.MouseEvent, id: string, direction: string) => {
    e.preventDefault();
    e.stopPropagation();

    const elem = elements.find((el) => el.id === id);
    if (!elem || !canvasRef.current) return;

    const canvas = canvasRef.current;
    const startX = e.clientX;
    const startY = e.clientY;
    const origX = elem.x;
    const origY = elem.y;
    const origW = elem.width;
    const origH = elem.height;

    setIsDragging(true);

    const handleMouseMove = (me: MouseEvent) => {
      const rect = canvas.getBoundingClientRect();
      const deltaXMm = ((me.clientX - startX) / rect.width) * labelWidth;
      const deltaYMm = ((me.clientY - startY) / rect.height) * labelHeight;

      let newX = origX, newY = origY, newW = origW, newH = origH;

      if (direction.includes('e')) newW = Math.max(3, origW + deltaXMm);
      if (direction.includes('s')) newH = Math.max(2, origH + deltaYMm);
      if (direction.includes('w')) {
        newW = Math.max(3, origW - deltaXMm);
        newX = origX + origW - newW;
      }
      if (direction.includes('n')) {
        newH = Math.max(2, origH - deltaYMm);
        newY = origY + origH - newH;
      }

      newX = Math.max(0, Math.min(labelWidth - newW, newX));
      newY = Math.max(0, Math.min(labelHeight - newH, newY));
      // Clamp width/height to remaining label space from the (possibly adjusted) origin
      newW = Math.min(newW, labelWidth - newX);
      newH = Math.min(newH, labelHeight - newY);

      setElements((prev) =>
        prev.map((el) =>
          el.id === id
            ? {
                ...el,
                x: Math.round(newX * 10) / 10,
                y: Math.round(newY * 10) / 10,
                width: Math.round(newW * 10) / 10,
                height: Math.round(newH * 10) / 10,
              }
            : el
        )
      );
    };

    const cleanup = () => {
      setIsDragging(false);
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', cleanup);
      document.removeEventListener('pointercancel', cleanup);
      dragCleanupRef.current = null;
    };

    dragCleanupRef.current = cleanup;
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', cleanup);
    document.addEventListener('pointercancel', cleanup);
  };

  const selectedElem = elements.find((e) => e.id === selectedElement);

  return (
    <div className="label-designer">
      <div className="label-designer-header">
        <h1>🏷️ {t('labels.title')}</h1>
        <p>{t('labels.subtitle')}</p>
      </div>

      <div className="designer-grid">
        {/* Left Panel - Toolbar & Properties */}
        <div className="designer-panel glass-dark">
          {/* Template Management */}
          <div className="panel-section">
            <h3>{t('labels.template')}</h3>
            <input
              type="text"
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
              className="input-field"
              placeholder={t('labels.templateName')}
            />
            <select
              className="input-select w-full"
              value={currentTemplateId || ''}
              onChange={(e) => {
                const id = e.target.value;
                if (id === 'new') {
                  createNewTemplate();
                } else if (id) {
                  const template = templates.find((t) => t.id === parseInt(id));
                  if (template) loadTemplate(template);
                }
              }}
            >
              <option value="new">+ {t('labels.newTemplate')}</option>
              {templates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} {t.is_default ? '★' : ''}
                </option>
              ))}
            </select>
            <div className="button-group">
              <button onClick={saveTemplate} disabled={saving} className="btn-save">
                <Save size={18} /> {saving ? t('labels.savingTemplate') : currentTemplateId ? t('common.update') : t('common.save')}
              </button>
              {currentTemplateId && !templates.find((t) => t.id === currentTemplateId)?.is_default && (
                <button onClick={setAsDefault} className="btn-add">
                  {t('labels.setAsDefault')} ★
                </button>
              )}
              {currentTemplateId && (
                <button onClick={() => deleteTemplate(currentTemplateId)} className="btn-delete-small">
                  <Trash2 size={18} /> {t('common.delete')}
                </button>
              )}
            </div>
          </div>

          <div className="panel-section">
            <h3>{t('labels.labelSize')}</h3>
            <select
              className="input-select"
              onChange={(e) => {
                const preset = presetSizes[parseInt(e.target.value)];
                if (preset.width > 0) {
                  setLabelWidth(preset.width);
                  setLabelHeight(preset.height);
                }
              }}
            >
              {presetSizes.map((size, i) => (
                <option key={i} value={i}>
                  {size.name}
                </option>
              ))}
            </select>
            <div className="input-group">
              <input
                type="number"
                value={labelWidth}
                onChange={(e) => setLabelWidth(Number(e.target.value))}
                className="input-field-small"
                placeholder={t('labels.width')}
              />
              <span>×</span>
              <input
                type="number"
                value={labelHeight}
                onChange={(e) => setLabelHeight(Number(e.target.value))}
                className="input-field-small"
                placeholder={t('labels.height')}
              />
              <span style={{ fontSize: '0.85rem', color: 'rgba(255,255,255,0.5)' }}>mm</span>
            </div>
          </div>

          <div className="panel-section">
            <h3>{t('labels.addElement')}</h3>
            <div className="button-group">
              <button onClick={() => addElement('qrcode')} className="btn-add">
                <QrCode size={18} /> {t('labels.qrCode')}
              </button>
              <button onClick={() => addElement('barcode')} className="btn-add">
                <Barcode size={18} /> {t('labels.barcode')}
              </button>
              <button onClick={() => addElement('text')} className="btn-add">
                <Type size={18} /> {t('labels.text')}
              </button>
              <button onClick={() => addElement('image')} className="btn-add">
                <ImageIcon size={18} /> {t('labels.image')}
              </button>
            </div>
          </div>

          {selectedElem && (
            <div className="panel-section">
              <div className="section-header">
                <h3>{t('labels.properties')}</h3>
                <button onClick={() => deleteElement(selectedElem.id)} className="btn-delete-small">
                  <Trash2 size={14} />
                </button>
              </div>

              <div className="property-grid">
                <label>{t('labels.fields.xPosition')}</label>
                <input
                  type="number"
                  value={selectedElem.x}
                  onChange={(e) => updateElement(selectedElem.id, { x: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>{t('labels.fields.yPosition')}</label>
                <input
                  type="number"
                  value={selectedElem.y}
                  onChange={(e) => updateElement(selectedElem.id, { y: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>{t('labels.fields.width')}</label>
                <input
                  type="number"
                  value={selectedElem.width}
                  onChange={(e) => updateElement(selectedElem.id, { width: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>{t('labels.fields.height')}</label>
                <input
                  type="number"
                  value={selectedElem.height}
                  onChange={(e) => updateElement(selectedElem.id, { height: Number(e.target.value) })}
                  className="input-field-small"
                />

                {selectedElem.type !== 'image' && (
                  <>
                    <label>{t('labels.content')}</label>
                    <select
                      value={selectedElem.content}
                      onChange={(e) => updateElement(selectedElem.id, { content: e.target.value })}
                      className="input-field-small"
                    >
                      <optgroup label={t('labels.contentGroups.device')}>
                        <option value="device_id">{t('labels.contentOptions.deviceId')}</option>
                        <option value="serial_number">{t('labels.contentOptions.serialNumber')}</option>
                        <option value="barcode">{t('labels.contentOptions.barcode')}</option>
                        <option value="rfid">{t('labels.contentOptions.rfid')}</option>
                        <option value="qr_code">{t('labels.contentOptions.qrCode')}</option>
                        <option value="status">{t('labels.contentOptions.status')}</option>
                        <option value="condition_rating">{t('labels.contentOptions.conditionRating')}</option>
                        <option value="usage_hours">{t('labels.contentOptions.usageHours')}</option>
                        <option value="purchase_date">{t('labels.contentOptions.purchaseDate')}</option>
                        <option value="notes">{t('labels.contentOptions.notes')}</option>
                      </optgroup>
                      <optgroup label={t('labels.contentGroups.product')}>
                        <option value="product_name">{t('labels.contentOptions.productName')}</option>
                        <option value="product_description">{t('labels.contentOptions.productDescription')}</option>
                        <option value="subcategory">{t('labels.contentOptions.subcategory')}</option>
                        <option value="category">{t('labels.contentOptions.category')}</option>
                        <option value="manufacturer">{t('labels.contentOptions.manufacturer')}</option>
                        <option value="brand">{t('labels.contentOptions.brand')}</option>
                        <option value="product_weight">{t('labels.contentOptions.productWeight')}</option>
                        <option value="product_dimensions">{t('labels.contentOptions.productDimensions')}</option>
                        <option value="maintenance_interval">{t('labels.contentOptions.maintenanceInterval')}</option>
                        <option value="power_consumption">{t('labels.contentOptions.powerConsumption')}</option>
                      </optgroup>
                      <optgroup label={t('labels.contentGroups.location')}>
                        <option value="zone_name">{t('labels.contentOptions.zoneName')}</option>
                        <option value="zone_code">{t('labels.contentOptions.zoneCode')}</option>
                        <option value="case_name">{t('labels.contentOptions.caseName')}</option>
                      </optgroup>
                      <optgroup label={t('labels.contentGroups.case')}>
                        <option value="case_id">{t('labels.contentOptions.caseId')}</option>
                        <option value="name">{t('labels.contentOptions.caseName')}</option>
                        <option value="description">{t('labels.contentOptions.description')}</option>
                        <option value="rfid_tag">{t('labels.contentOptions.rfidTag')}</option>
                        <option value="dimensions">{t('labels.contentOptions.dimensions')}</option>
                        <option value="weight">{t('labels.contentOptions.weight')}</option>
                      </optgroup>
                      <optgroup label={t('labels.contentGroups.zone')}>
                        <option value="code">{t('labels.contentOptions.zoneCode')}</option>
                        <option value="zone_name">{t('labels.contentOptions.zoneName')}</option>
                        <option value="zone_type">{t('labels.contentOptions.zoneType')}</option>
                        <option value="location">{t('labels.contentOptions.location')}</option>
                        <option value="capacity">{t('labels.contentOptions.capacity')}</option>
                        <option value="parent_name">{t('labels.contentOptions.parentName')}</option>
                        <option value="parent_code">{t('labels.contentOptions.parentCode')}</option>
                      </optgroup>
                    </select>
                  </>
                )}

                {selectedElem.type === 'image' && (
                  <>
                    <label>{t('labels.fields.uploadImage')}</label>
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/jpg,image/svg+xml"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (file) handleImageUpload(selectedElem.id, file);
                      }}
                      className="input-field-small"
                      style={{ padding: '0.25rem' }}
                    />
                    {selectedElem.image_data && (
                      <div style={{ gridColumn: '1 / -1', fontSize: '0.75rem', color: 'rgba(255,255,255,0.5)' }}>
                        ✓ {selectedElem.content}
                      </div>
                    )}
                  </>
                )}

                {selectedElem.type === 'text' && (
                  <>
                    <label>{t('labels.fields.fontSize')}</label>
                    <input
                      type="number"
                      value={selectedElem.style.font_size}
                      onChange={(e) =>
                        updateElement(selectedElem.id, {
                          style: { ...selectedElem.style, font_size: Number(e.target.value) },
                        })
                      }
                      className="input-field-small"
                    />

                    <label>{t('labels.fields.fontFamily')}</label>
                    <select
                      value={selectedElem.style.font_family}
                      onChange={(e) =>
                        updateElement(selectedElem.id, {
                          style: { ...selectedElem.style, font_family: e.target.value },
                        })
                      }
                      className="input-field-small"
                    >
                      <option value="Arial">Arial</option>
                      <option value="Ubuntu">Ubuntu</option>
                      <option value="Aptos">Aptos</option>
                      <option value="Times New Roman">Times New Roman</option>
                      <option value="Courier New">Courier New</option>
                      <option value="Verdana">Verdana</option>
                      <option value="Georgia">Georgia</option>
                    </select>

                    <label>{t('labels.fields.fontWeight')}</label>
                    <select
                      value={selectedElem.style.font_weight}
                      onChange={(e) =>
                        updateElement(selectedElem.id, {
                          style: { ...selectedElem.style, font_weight: e.target.value },
                        })
                      }
                      className="input-field-small"
                    >
                      <option value="normal">{t('labels.fontWeight.normal')}</option>
                      <option value="bold">{t('labels.fontWeight.bold')}</option>
                    </select>
                  </>
                )}
              </div>
            </div>
          )}

          <div className="panel-section">
            <input
              type="text"
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
              className="input-field"
              placeholder={t('labels.templateName')}
            />
            <button onClick={saveTemplate} disabled={saving} className="btn-save">
              <Save size={18} /> {saving ? t('common.saving') : t('labels.saveTemplate')}
            </button>
          </div>
        </div>

        {/* Center - Canvas Preview */}
        <div className="designer-canvas-area glass-dark">
          <div className="canvas-header">
            <h3>{t('labels.preview')}</h3>
            <select
              value={previewDevice?.device_id || ''}
              onChange={(e) => {
                const value = e.target.value;
                if (value.startsWith('CASE-')) {
                  // It's a case — use data already in the cases list (includes rfid_tag, dimensions, etc.)
                  const caseId = parseInt(value.replace('CASE-', ''));
                  const caseItem = cases.find((c) => c.case_id === caseId);
                  if (caseItem) {
                    const caseAsDevice: Device = {
                      device_id: `CASE-${caseItem.case_id}`,
                      product_name: caseItem.name,
                      status: caseItem.status,
                      zone_code: caseItem.zone_code,
                      zone_name: caseItem.zone_name,
                      zone_id: caseItem.zone_id,
                    };
                    setPreviewDevice(caseAsDevice);
                  }
                } else if (value.startsWith('ZONE-')) {
                  // It's a zone — data already in zones list
                  const zoneId = parseInt(value.replace('ZONE-', ''));
                  const zone = zones.find((z) => z.zone_id === zoneId);
                  if (zone) {
                    const zoneAsDevice: Device = {
                      device_id: `ZONE-${zone.zone_id}`,
                      product_name: zone.name,
                      status: zone.type,
                      zone_code: zone.code,
                      zone_name: zone.name,
                      zone_id: zone.zone_id,
                    };
                    setPreviewDevice(zoneAsDevice);
                  }
                } else {
                  // It's a device — fetch full details (product metadata, rfid, notes, etc.)
                  devicesApi.getById(value).then(({ data }) => {
                    setPreviewDevice(data);
                  }).catch((err) => {
                    // Fall back to summary data on error
                    console.error('Failed to fetch device details for preview:', err);
                    const device = devices.find((d) => d.device_id === value);
                    if (device) setPreviewDevice(device);
                  });
                }
              }}
              className="input-select-small"
            >
              <optgroup label={t('labels.previewGroups.devices')}>
                {devices.map((d) => (
                  <option key={d.device_id} value={d.device_id}>
                    {d.device_id} - {d.product_name}
                  </option>
                ))}
              </optgroup>
              <optgroup label={t('labels.previewGroups.cases')}>
                {cases.map((c) => (
                  <option key={`CASE-${c.case_id}`} value={`CASE-${c.case_id}`}>
                    CASE-{c.case_id} - {c.name}
                  </option>
                ))}
              </optgroup>
              <optgroup label={t('labels.previewGroups.zones')}>
                {zones.map((z) => (
                  <option key={`ZONE-${z.zone_id}`} value={`ZONE-${z.zone_id}`}>
                    {z.code} - {z.name}
                  </option>
                ))}
              </optgroup>
            </select>
          </div>
          <div className="canvas-drag-hint">{t('labels.dragHint')}</div>
          <div className="canvas-wrapper">
            <div className="canvas-interactive-container" ref={canvasContainerRef}>
              <canvas ref={canvasRef} className="label-canvas" style={{ display: 'block' }} />
              <div
                className="canvas-overlay"
                onClick={() => setSelectedElement(null)}
              >
                {labelWidth > 0 && labelHeight > 0 && elements.map((elem) => (
                  <div
                    key={elem.id}
                    className={`element-overlay${selectedElement === elem.id ? ' selected' : ''}`}
                    style={{
                      left: `${(elem.x / labelWidth) * 100}%`,
                      top: `${(elem.y / labelHeight) * 100}%`,
                      width: `${(elem.width / labelWidth) * 100}%`,
                      height: `${(elem.height / labelHeight) * 100}%`,
                    }}
                    title={`${elem.type}: ${elem.content || ''} (x: ${elem.x}mm, y: ${elem.y}mm)`}
                    onMouseDown={(e) => handleElementMouseDown(e, elem.id)}
                    onClick={(e) => {
                      e.stopPropagation();
                      setSelectedElement(elem.id);
                    }}
                  >
                    <div className="element-overlay-label">
                      {elem.type === 'qrcode' && <QrCode size={10} />}
                      {elem.type === 'barcode' && <Barcode size={10} />}
                      {elem.type === 'text' && <Type size={10} />}
                      {elem.type === 'image' && <ImageIcon size={10} />}
                    </div>
                    {selectedElement === elem.id && (
                      <>
                        <div className="resize-handle resize-nw" onMouseDown={(e) => handleResizeMouseDown(e, elem.id, 'nw')} />
                        <div className="resize-handle resize-ne" onMouseDown={(e) => handleResizeMouseDown(e, elem.id, 'ne')} />
                        <div className="resize-handle resize-sw" onMouseDown={(e) => handleResizeMouseDown(e, elem.id, 'sw')} />
                        <div className="resize-handle resize-se" onMouseDown={(e) => handleResizeMouseDown(e, elem.id, 'se')} />
                      </>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* Right Panel - Elements List & Actions */}
        <div className="designer-panel glass-dark">
          <div className="panel-section">
            <h3>{t('labels.elements')} ({elements.length})</h3>
            <div className="elements-list">
              {elements.map((elem) => (
                <div
                  key={elem.id}
                  className={`element-item ${selectedElement === elem.id ? 'active' : ''}`}
                  onClick={() => setSelectedElement(elem.id)}
                >
                  <div className="element-icon">
                    {elem.type === 'qrcode' && <QrCode size={16} />}
                    {elem.type === 'barcode' && <Barcode size={16} />}
                    {elem.type === 'text' && <Type size={16} />}
                    {elem.type === 'image' && <ImageIcon size={16} />}
                  </div>
                  <div className="element-info">
                    <div className="element-type">{elem.type === 'qrcode' ? t('labels.qrCode') : elem.type === 'barcode' ? t('labels.barcode') : elem.type === 'image' ? t('labels.image') : t('labels.text')}</div>
                    <div className="element-content">{elem.content || t('labels.noImage')}</div>
                  </div>
                  <button onClick={(e) => { e.stopPropagation(); deleteElement(elem.id); }} className="btn-delete-mini">
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
              {elements.length === 0 && (
                <div className="empty-state">{t('labels.noElements')}</div>
              )}
            </div>
          </div>

          <div className="panel-section">
            <h3>{t('labels.actions')}</h3>
            <div className="action-buttons">
              <button onClick={handlePrint} disabled={!previewDevice} className="btn-action">
                <Printer size={18} /> {t('labels.printPreview')}
              </button>
              <button onClick={generateMissingLabels} disabled={exporting || (devices.length === 0 && cases.length === 0 && zones.length === 0)} className="btn-action btn-primary" style={{ backgroundColor: '#10b981' }}>
                <Save size={18} /> <span className="hidden sm:inline">{exporting ? t('labels.generating') : t('labels.generateMissing')}</span><span className="sm:hidden">{exporting ? t('labels.generating') : t('labels.missingShort')}</span>
              </button>
              <button onClick={generateAllLabels} disabled={exporting || (devices.length === 0 && cases.length === 0 && zones.length === 0)} className="btn-action btn-primary">
                <Save size={18} /> <span className="hidden sm:inline">{exporting ? t('labels.generatingCount', { count: devices.length + cases.length + zones.length }) : t('labels.generateAllWithCount', { devices: devices.length, cases: cases.length, zones: zones.length })}</span><span className="sm:hidden">{exporting ? t('labels.generating') : t('labels.allShort')}</span>
              </button>
              <button onClick={exportAllLabels} disabled={exporting || (devices.length === 0 && cases.length === 0 && zones.length === 0)} className="btn-action">
                <Download size={18} /> <span className="hidden sm:inline">{exporting ? t('labels.exporting') : t('labels.exportAll')}</span><span className="sm:hidden">{exporting ? t('labels.exportShortLoading') : t('labels.exportShort')}</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
