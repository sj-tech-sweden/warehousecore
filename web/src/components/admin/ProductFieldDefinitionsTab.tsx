import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Pencil, Trash2, X } from 'lucide-react';
import { productFieldDefinitionsApi, type ProductFieldDefinition } from '../../lib/api';
import { ModalPortal } from '../ModalPortal';

type FieldType = ProductFieldDefinition['field_type'];

interface FormState {
  name: string;
  label: string;
  field_type: FieldType;
  unit: string;
  sort_order: number;
  is_required: boolean;
  optionsText: string; // one option per line; converted to/from JSON on save/load
}

const emptyForm: FormState = {
  name: '',
  label: '',
  field_type: 'text',
  unit: '',
  sort_order: 0,
  is_required: false,
  optionsText: '',
};

export function ProductFieldDefinitionsTab() {
  const { t } = useTranslation();
  const [definitions, setDefinitions] = useState<ProductFieldDefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await productFieldDefinitionsApi.list();
      setDefinitions(data || []);
    } catch (e) {
      console.error('Failed to load field definitions:', e);
      window.alert(t('admin.fieldDefinitions.errors.load'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    load();
  }, [load]);

  const openCreate = () => {
    setForm(emptyForm);
    setEditingId(null);
    setModalOpen(true);
  };

  const openEdit = (def: ProductFieldDefinition) => {
    let optionsText = '';
    if (def.field_type === 'select' && def.options) {
      try {
        const parsed = JSON.parse(def.options) as string[];
        optionsText = parsed.join('\n');
      } catch {
        optionsText = def.options;
      }
    }
    setForm({
      name: def.name,
      label: def.label,
      field_type: def.field_type,
      unit: def.unit ?? '',
      sort_order: def.sort_order,
      is_required: def.is_required,
      optionsText,
    });
    setEditingId(def.id);
    setModalOpen(true);
  };

  const closeModal = () => {
    setModalOpen(false);
    setEditingId(null);
    setForm(emptyForm);
  };

  const handleSave = async () => {
    if (!form.name.trim() || !form.label.trim()) return;

    let options: string | null = null;
    if (form.field_type === 'select') {
      const lines = form.optionsText
        .split('\n')
        .map(l => l.trim())
        .filter(l => l.length > 0);
      options = JSON.stringify(lines);
    }

    const payload: Omit<ProductFieldDefinition, 'id'> = {
      name: form.name.trim(),
      label: form.label.trim(),
      field_type: form.field_type,
      unit: form.unit.trim() || null,
      sort_order: form.sort_order,
      is_required: form.is_required,
      options,
    };

    setSaving(true);
    try {
      if (editingId !== null) {
        await productFieldDefinitionsApi.update(editingId, payload);
      } else {
        await productFieldDefinitionsApi.create(payload);
      }
      closeModal();
      await load();
    } catch (e) {
      console.error('Failed to save field definition:', e);
      window.alert(t('admin.fieldDefinitions.errors.save'));
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (def: ProductFieldDefinition) => {
    if (!window.confirm(t('admin.fieldDefinitions.confirmDelete'))) return;
    try {
      await productFieldDefinitionsApi.delete(def.id);
      await load();
    } catch (e) {
      console.error('Failed to delete field definition:', e);
      window.alert(t('admin.fieldDefinitions.errors.delete'));
    }
  };

  const fieldTypeLabel = (type: string) => t(`admin.fieldDefinitions.types.${type}`, { defaultValue: type });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-bold text-white">{t('admin.fieldDefinitions.title')}</h2>
          <p className="text-sm text-gray-400">{t('admin.fieldDefinitions.subtitle')}</p>
        </div>
        <button
          onClick={openCreate}
          className="flex items-center gap-2 rounded-xl bg-accent-red px-4 py-2 font-semibold text-white hover:shadow-lg"
        >
          <Plus className="h-4 w-4" />
          {t('admin.fieldDefinitions.newField')}
        </button>
      </div>

      {loading ? (
        <div className="rounded-xl border border-white/10 bg-white/5 p-8 text-center text-gray-400">
          {t('common.loading')}
        </div>
      ) : definitions.length === 0 ? (
        <div className="rounded-xl border border-white/10 bg-white/5 p-8 text-center text-gray-400">
          {t('admin.fieldDefinitions.noFields')}
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-white/10 bg-white/5">
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-white/10 text-sm text-gray-200">
              <thead className="bg-white/5 text-xs uppercase tracking-wide text-gray-400">
                <tr>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.name')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.label')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.type')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.unit')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.options')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.required')}</th>
                  <th className="px-4 py-3 text-left font-semibold">{t('admin.fieldDefinitions.columns.sortOrder')}</th>
                  <th className="px-4 py-3 text-right font-semibold">{t('admin.fieldDefinitions.columns.actions')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {definitions.map(def => (
                  <tr key={def.id} className="hover:bg-white/5">
                    <td className="px-4 py-3 font-mono text-xs text-white">{def.name}</td>
                    <td className="px-4 py-3 text-white">{def.label}</td>
                    <td className="px-4 py-3 text-gray-300">{fieldTypeLabel(def.field_type)}</td>
                    <td className="px-4 py-3 text-gray-400">{def.unit || '—'}</td>
                    <td className="px-4 py-3 text-gray-400 max-w-xs truncate">
                      {def.field_type === 'select' && def.options
                        ? (() => {
                            try {
                              return (JSON.parse(def.options) as string[]).join(', ');
                            } catch {
                              return def.options;
                            }
                          })()
                        : '—'}
                    </td>
                    <td className="px-4 py-3">
                      {def.is_required ? (
                        <span className="rounded-full bg-accent-red/20 px-2 py-0.5 text-xs text-accent-red">
                          {t('common.yes')}
                        </span>
                      ) : (
                        <span className="text-gray-500">{t('common.no')}</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-400">{def.sort_order}</td>
                    <td className="px-4 py-3">
                      <div className="flex justify-end gap-2">
                        <button
                          onClick={() => openEdit(def)}
                          className="rounded-lg bg-white/10 p-2 text-gray-200 transition hover:bg-white/20 hover:text-white"
                          title={t('common.edit')}
                        >
                          <Pencil className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(def)}
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
      )}

      {modalOpen && (
        <ModalPortal>
          <div className="fixed inset-0 z-[120] flex min-h-screen items-center justify-center bg-black/80 p-4">
            <div className="glass-dark rounded-2xl border border-white/10 shadow-2xl p-6 max-w-lg w-full max-h-[90vh] overflow-y-auto">
              <div className="flex justify-between items-center mb-6">
                <h3 className="text-xl font-bold text-white">
                  {editingId !== null
                    ? t('admin.fieldDefinitions.editField')
                    : t('admin.fieldDefinitions.createField')}
                </h3>
                <button
                  onClick={closeModal}
                  disabled={saving}
                  className="text-gray-400 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors disabled:opacity-50"
                  title={t('common.close')}
                  aria-label={t('common.close')}
                >
                  <X className="w-6 h-6" />
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">
                    {t('admin.fieldDefinitions.fields.name')} <span className="text-accent-red">*</span>
                  </label>
                  <input
                    type="text"
                    value={form.name}
                    onChange={e => setForm({ ...form, name: e.target.value })}
                    placeholder={t('admin.fieldDefinitions.fields.namePlaceholder')}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 outline-none focus:border-accent-red"
                    disabled={editingId !== null}
                  />
                  {editingId !== null && (
                    <p className="mt-1 text-xs text-gray-500">Field name cannot be changed after creation.</p>
                  )}
                </div>

                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">
                    {t('admin.fieldDefinitions.fields.label')} <span className="text-accent-red">*</span>
                  </label>
                  <input
                    type="text"
                    value={form.label}
                    onChange={e => setForm({ ...form, label: e.target.value })}
                    placeholder={t('admin.fieldDefinitions.fields.labelPlaceholder')}
                    className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 outline-none focus:border-accent-red"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-300">
                      {t('admin.fieldDefinitions.fields.type')}
                    </label>
                    <select
                      value={form.field_type}
                      onChange={e => setForm({ ...form, field_type: e.target.value as FieldType })}
                      className="w-full px-3 py-2 bg-dark-700 border border-gray-600 rounded-lg text-white focus:ring-1 focus:ring-accent-red focus:border-accent-red"
                    >
                      <option value="text">{t('admin.fieldDefinitions.types.text')}</option>
                      <option value="number">{t('admin.fieldDefinitions.types.number')}</option>
                      <option value="integer">{t('admin.fieldDefinitions.types.integer')}</option>
                      <option value="select">{t('admin.fieldDefinitions.types.select')}</option>
                      <option value="boolean">{t('admin.fieldDefinitions.types.boolean')}</option>
                    </select>
                  </div>

                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-300">
                      {t('admin.fieldDefinitions.fields.unit')}
                    </label>
                    <input
                      type="text"
                      value={form.unit}
                      onChange={e => setForm({ ...form, unit: e.target.value })}
                      placeholder={t('admin.fieldDefinitions.fields.unitPlaceholder')}
                      className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 outline-none focus:border-accent-red"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 items-center">
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-300">
                      {t('admin.fieldDefinitions.fields.sortOrder')}
                    </label>
                    <input
                      type="number"
                      value={form.sort_order}
                      onChange={e => setForm({ ...form, sort_order: parseInt(e.target.value, 10) || 0 })}
                      className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white outline-none focus:border-accent-red"
                    />
                  </div>
                  <div className="flex items-center gap-2 pt-5">
                    <input
                      type="checkbox"
                      id="is_required"
                      checked={form.is_required}
                      onChange={e => setForm({ ...form, is_required: e.target.checked })}
                      className="w-5 h-5 rounded"
                    />
                    <label htmlFor="is_required" className="text-sm text-gray-300 cursor-pointer">
                      {t('admin.fieldDefinitions.fields.required')}
                    </label>
                  </div>
                </div>

                {form.field_type === 'select' && (
                  <div>
                    <label className="mb-1 block text-sm font-medium text-gray-300">
                      {t('admin.fieldDefinitions.fields.options')}
                    </label>
                    <textarea
                      value={form.optionsText}
                      onChange={e => setForm({ ...form, optionsText: e.target.value })}
                      placeholder={t('admin.fieldDefinitions.fields.optionsPlaceholder')}
                      rows={4}
                      className="w-full px-3 py-2 bg-white/10 border border-white/20 rounded-lg text-white placeholder-gray-500 outline-none focus:border-accent-red"
                    />
                    <p className="mt-1 text-xs text-gray-500">{t('admin.fieldDefinitions.fields.optionsHelp')}</p>
                  </div>
                )}
              </div>

              <div className="flex gap-3 pt-6">
                <button
                  type="button"
                  onClick={closeModal}
                  disabled={saving}
                  className="flex-1 rounded-lg border border-white/20 px-4 py-2 text-white hover:bg-white/10 transition disabled:opacity-50"
                >
                  {t('common.cancel')}
                </button>
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={saving || !form.name.trim() || !form.label.trim()}
                  className="flex-1 rounded-lg bg-accent-red px-4 py-2 font-semibold text-white transition hover:bg-accent-red/80 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {saving ? t('common.saving') : editingId !== null ? t('common.save') : t('common.create')}
                </button>
              </div>
            </div>
          </div>
        </ModalPortal>
      )}
    </div>
  );
}
