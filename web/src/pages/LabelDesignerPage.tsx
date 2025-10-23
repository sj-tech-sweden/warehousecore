import { useState, useEffect, useRef } from 'react';
import { Trash2, Download, Printer, QrCode, Barcode, Type, Save } from 'lucide-react';
import { labelsApi, devicesApi } from '../lib/api';
import type { LabelTemplate, LabelElement, Device } from '../lib/api';
import './LabelDesignerPage.css';

interface DesignElement extends LabelElement {
  id: string;
}

const PRESET_SIZES = [
  { name: '62x29mm (Standard)', width: 62, height: 29 },
  { name: '100x50mm (Groß)', width: 100, height: 50 },
  { name: '50x25mm (Klein)', width: 50, height: 25 },
  { name: 'Custom', width: 0, height: 0 },
];

export default function LabelDesignerPage() {
  const [labelWidth, setLabelWidth] = useState(62);
  const [labelHeight, setLabelHeight] = useState(29);
  const [elements, setElements] = useState<DesignElement[]>([]);
  const [selectedElement, setSelectedElement] = useState<string | null>(null);
  const [devices, setDevices] = useState<Device[]>([]);
  const [previewDevice, setPreviewDevice] = useState<Device | null>(null);
  const [exporting, setExporting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [templateName, setTemplateName] = useState('Mein Label Template');
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    loadDevices();
    loadDefaultTemplate();
  }, []);

  const loadDevices = async () => {
    try {
      const { data } = await devicesApi.getAll();
      setDevices(data);
      if (data.length > 0) {
        setPreviewDevice(data[0]);
      }
    } catch (error) {
      console.error('Failed to load devices:', error);
    }
  };

  const loadDefaultTemplate = async () => {
    try {
      const { data } = await labelsApi.getTemplates();
      const defaultTemplate = data.find((t) => t.is_default);
      if (defaultTemplate) {
        setLabelWidth(defaultTemplate.width);
        setLabelHeight(defaultTemplate.height);
        setTemplateName(defaultTemplate.name);
        const parsed = JSON.parse(defaultTemplate.template_json);
        setElements(parsed.map((e: LabelElement, i: number) => ({ ...e, id: `elem-${i}` })));
      }
    } catch (error) {
      console.error('Failed to load template:', error);
    }
  };

  const addElement = (type: 'qrcode' | 'barcode' | 'text') => {
    const newElement: DesignElement = {
      id: `elem-${Date.now()}`,
      type,
      x: 5,
      y: 5,
      width: type === 'qrcode' ? 25 : type === 'barcode' ? 50 : 30,
      height: type === 'qrcode' ? 25 : type === 'barcode' ? 15 : 6,
      rotation: 0,
      content: type === 'text' ? 'device_id' : 'device_id',
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
    setSaving(true);
    try {
      const templateJSON = JSON.stringify(
        elements.map(({ id, ...rest }) => rest)
      );

      const template: LabelTemplate = {
        name: templateName,
        description: 'Globales Label Template',
        width: labelWidth,
        height: labelHeight,
        template_json: templateJSON,
        is_default: true,
      };

      await labelsApi.createTemplate(template);
      alert('Template erfolgreich gespeichert!');
    } catch (error) {
      console.error('Failed to save template:', error);
      alert('Fehler beim Speichern des Templates');
    } finally {
      setSaving(false);
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

    // Render elements
    for (const elem of elements) {
      const x = elem.x * mmToPx;
      const y = elem.y * mmToPx;
      const w = elem.width * mmToPx;
      const h = elem.height * mmToPx;

      // Get content
      let content = elem.content;
      if (elem.content === 'device_id') content = previewDevice.device_id;
      else if (elem.content === 'product_name') content = previewDevice.product_name || '';
      else if (elem.content === 'device_name') content = previewDevice.device_id;

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
      } else if (elem.type === 'text') {
        ctx.fillStyle = elem.style.color || '#000000';
        ctx.font = `${elem.style.font_weight || 'normal'} ${(elem.style.font_size || 10) * (dpi / 96)}px ${elem.style.font_family || 'Arial'}`;
        ctx.textAlign = (elem.style.alignment as CanvasTextAlign) || 'left';
        ctx.fillText(content, x, y + (elem.style.font_size || 10) * (dpi / 96));
      }
    }
  };

  useEffect(() => {
    if (previewDevice) {
      renderPreview();
    }
  }, [elements, labelWidth, labelHeight, previewDevice]);

  const exportAllLabels = async () => {
    if (devices.length === 0) {
      alert('Keine Devices gefunden!');
      return;
    }

    setExporting(true);
    try {
      for (const device of devices) {
        setPreviewDevice(device);
        await new Promise((r) => setTimeout(r, 500));

        const canvas = canvasRef.current;
        if (canvas) {
          const link = document.createElement('a');
          link.download = `label-${device.device_id}.png`;
          link.href = canvas.toDataURL('image/png');
          link.click();
        }
      }
      alert(`${devices.length} Labels erfolgreich exportiert!`);
    } catch (error) {
      console.error('Export failed:', error);
      alert('Fehler beim Export');
    } finally {
      setExporting(false);
      if (devices.length > 0) {
        setPreviewDevice(devices[0]);
      }
    }
  };

  const handlePrint = () => {
    if (!canvasRef.current) return;
    const dataUrl = canvasRef.current.toDataURL('image/png');
    const printWindow = window.open('', '', 'width=800,height=600');
    if (printWindow) {
      printWindow.document.write(`
        <html>
          <head><title>Print Label</title></head>
          <body style="margin:0;display:flex;justify-content:center;align-items:center;">
            <img src="${dataUrl}" onload="window.print();window.close();" />
          </body>
        </html>
      `);
      printWindow.document.close();
    }
  };

  const selectedElem = elements.find((e) => e.id === selectedElement);

  return (
    <div className="label-designer">
      <div className="label-designer-header">
        <h1>🏷️ Label Designer</h1>
        <p>Globales Label-Template für alle Devices</p>
      </div>

      <div className="designer-grid">
        {/* Left Panel - Toolbar & Properties */}
        <div className="designer-panel glass-dark">
          <div className="panel-section">
            <h3>Label-Größe</h3>
            <select
              className="input-select"
              onChange={(e) => {
                const preset = PRESET_SIZES[parseInt(e.target.value)];
                if (preset.width > 0) {
                  setLabelWidth(preset.width);
                  setLabelHeight(preset.height);
                }
              }}
            >
              {PRESET_SIZES.map((size, i) => (
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
                className="input-field"
                placeholder="Breite (mm)"
              />
              <span>×</span>
              <input
                type="number"
                value={labelHeight}
                onChange={(e) => setLabelHeight(Number(e.target.value))}
                className="input-field"
                placeholder="Höhe (mm)"
              />
            </div>
          </div>

          <div className="panel-section">
            <h3>Elemente hinzufügen</h3>
            <div className="button-group">
              <button onClick={() => addElement('qrcode')} className="btn-add">
                <QrCode size={18} /> QR-Code
              </button>
              <button onClick={() => addElement('barcode')} className="btn-add">
                <Barcode size={18} /> Barcode
              </button>
              <button onClick={() => addElement('text')} className="btn-add">
                <Type size={18} /> Text
              </button>
            </div>
          </div>

          {selectedElem && (
            <div className="panel-section">
              <div className="section-header">
                <h3>Eigenschaften</h3>
                <button onClick={() => deleteElement(selectedElem.id)} className="btn-delete-small">
                  <Trash2 size={14} />
                </button>
              </div>

              <div className="property-grid">
                <label>X Position (mm)</label>
                <input
                  type="number"
                  value={selectedElem.x}
                  onChange={(e) => updateElement(selectedElem.id, { x: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>Y Position (mm)</label>
                <input
                  type="number"
                  value={selectedElem.y}
                  onChange={(e) => updateElement(selectedElem.id, { y: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>Breite (mm)</label>
                <input
                  type="number"
                  value={selectedElem.width}
                  onChange={(e) => updateElement(selectedElem.id, { width: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>Höhe (mm)</label>
                <input
                  type="number"
                  value={selectedElem.height}
                  onChange={(e) => updateElement(selectedElem.id, { height: Number(e.target.value) })}
                  className="input-field-small"
                />

                <label>Inhalt</label>
                <select
                  value={selectedElem.content}
                  onChange={(e) => updateElement(selectedElem.id, { content: e.target.value })}
                  className="input-field-small"
                >
                  <option value="device_id">Device ID</option>
                  <option value="product_name">Produktname</option>
                  <option value="device_name">Device Name</option>
                </select>

                {selectedElem.type === 'text' && (
                  <>
                    <label>Schriftgröße</label>
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

                    <label>Schriftart</label>
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

                    <label>Schriftstil</label>
                    <select
                      value={selectedElem.style.font_weight}
                      onChange={(e) =>
                        updateElement(selectedElem.id, {
                          style: { ...selectedElem.style, font_weight: e.target.value },
                        })
                      }
                      className="input-field-small"
                    >
                      <option value="normal">Normal</option>
                      <option value="bold">Fett</option>
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
              placeholder="Template Name"
            />
            <button onClick={saveTemplate} disabled={saving} className="btn-save">
              <Save size={18} /> {saving ? 'Speichert...' : 'Template Speichern'}
            </button>
          </div>
        </div>

        {/* Center - Canvas Preview */}
        <div className="designer-canvas-area glass-dark">
          <div className="canvas-header">
            <h3>Vorschau</h3>
            <select
              value={previewDevice?.device_id || ''}
              onChange={(e) => {
                const device = devices.find((d) => d.device_id === e.target.value);
                if (device) setPreviewDevice(device);
              }}
              className="input-select-small"
            >
              {devices.slice(0, 20).map((d) => (
                <option key={d.device_id} value={d.device_id}>
                  {d.device_id} - {d.product_name}
                </option>
              ))}
            </select>
          </div>
          <div className="canvas-wrapper">
            <canvas ref={canvasRef} className="label-canvas" />
          </div>
        </div>

        {/* Right Panel - Elements List & Actions */}
        <div className="designer-panel glass-dark">
          <div className="panel-section">
            <h3>Elemente ({elements.length})</h3>
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
                  </div>
                  <div className="element-info">
                    <div className="element-type">{elem.type === 'qrcode' ? 'QR-Code' : elem.type === 'barcode' ? 'Barcode' : 'Text'}</div>
                    <div className="element-content">{elem.content}</div>
                  </div>
                  <button onClick={(e) => { e.stopPropagation(); deleteElement(elem.id); }} className="btn-delete-mini">
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
              {elements.length === 0 && (
                <div className="empty-state">Keine Elemente</div>
              )}
            </div>
          </div>

          <div className="panel-section">
            <h3>Aktionen</h3>
            <div className="action-buttons">
              <button onClick={handlePrint} disabled={!previewDevice} className="btn-action">
                <Printer size={18} /> Vorschau Drucken
              </button>
              <button onClick={exportAllLabels} disabled={exporting || devices.length === 0} className="btn-action btn-primary">
                <Download size={18} /> {exporting ? `Exportiere ${devices.length}...` : `Alle ${devices.length} Labels Exportieren`}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
