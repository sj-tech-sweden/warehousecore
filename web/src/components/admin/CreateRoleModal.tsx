import { useState, useEffect, useMemo } from 'react';
import { ModalPortal } from '../../components/ModalPortal';
import { useTranslation } from 'react-i18next';
import { api } from '../../lib/api';

type PermissionDefs = Record<string, string[]>;

const fallbackPermissionDefs: PermissionDefs = {
  admin: ['admin.*', 'super_admin.*', 'manage.*'],
  warehouse: ['warehouse.*', 'warehouse.manage', 'warehouse.reports', 'warehouse.scan', 'warehouse.view'],
  roles: ['role.read', 'role.create', 'role.update', 'role.delete'],
  users: ['user.read', 'user.create', 'user.update', 'user.delete', 'user.assign_role', 'user.revoke_role'],
  devices: ['device.read', 'device.create', 'device.update', 'device.delete', 'device.maintenance', 'device.location'],
  jobs: ['job.read', 'job.create', 'job.update', 'job.delete', 'job.assign_device', 'job.manage_templates'],
  system: ['audit.read', 'permission.read', 'system.admin'],
};

function mergeSelectedPermissions(defs: PermissionDefs, selected: string[]): PermissionDefs {
  const merged: PermissionDefs = Object.fromEntries(
    Object.entries(defs).map(([group, permissions]) => [group, [...permissions]])
  );

  const existing = new Set(Object.values(merged).flat());
  const extraSelected = selected.filter(permission => !existing.has(permission));

  if (extraSelected.length > 0) {
    merged.assigned = Array.from(new Set([...(merged.assigned || []), ...extraSelected]));
  }

  return merged;
}

function asStringList(input: unknown): string[] {
  if (Array.isArray(input)) {
    return input.map(item => (typeof item === 'string' ? item : String(item ?? '').trim())).filter(Boolean);
  }
  if (typeof input === 'string') {
    return [input].filter(Boolean);
  }
  return [];
}

function normalizePermissionDefs(input: unknown): PermissionDefs {
  const out: PermissionDefs = {};

  const addGroup = (groupName: string, permissions: unknown) => {
    const cleaned = asStringList(permissions);
    if (cleaned.length === 0) {
      return;
    }
    const key = (groupName || 'general').trim() || 'general';
    const existing = out[key] || [];
    out[key] = Array.from(new Set([...existing, ...cleaned]));
  };

  if (Array.isArray(input)) {
    if (input.every(item => typeof item === 'string')) {
      addGroup('general', input);
    } else {
      input.forEach(item => {
        if (typeof item === 'string') {
          addGroup('general', [item]);
          return;
        }
        if (item && typeof item === 'object') {
          const obj = item as Record<string, unknown>;
          const group = String(obj.group ?? obj.name ?? obj.category ?? 'general');
          const list = obj.permissions ?? obj.items ?? obj.values;
          addGroup(group, list);
        }
      });
    }
  } else if (input && typeof input === 'object') {
    const obj = input as Record<string, unknown>;
    Object.entries(obj).forEach(([group, value]) => {
      if (Array.isArray(value) || typeof value === 'string') {
        addGroup(group, value);
        return;
      }
      if (value && typeof value === 'object') {
        const nested = value as Record<string, unknown>;
        addGroup(group, nested.permissions ?? nested.items ?? nested.values);
      }
    });
  }

  return Object.keys(out).length > 0 ? out : fallbackPermissionDefs;
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onSaved?: () => void;
  role?: {
    id: number;
    name: string;
    description?: string;
    permissions?: unknown;
  } | null;
}

function parseSelectedPermissions(input: unknown): string[] {
  const out = new Set<string>();

  const collect = (value: unknown) => {
    if (!value) {
      return;
    }

    if (Array.isArray(value)) {
      value.forEach(item => collect(item));
      return;
    }

    if (typeof value === 'string') {
      const trimmed = value.trim();
      if (!trimmed) {
        return;
      }

      // Strings may be raw permission names or JSON-encoded permission payloads.
      if ((trimmed.startsWith('[') && trimmed.endsWith(']')) || (trimmed.startsWith('{') && trimmed.endsWith('}'))) {
        try {
          const parsed = JSON.parse(trimmed);
          collect(parsed);
          return;
        } catch {
          // Treat non-JSON strings as a single permission.
        }
      }

      out.add(trimmed);
      return;
    }

    if (typeof value === 'object') {
      const obj = value as Record<string, unknown>;

      if (obj.permissions !== undefined) {
        collect(obj.permissions);
      }

      Object.values(obj).forEach(entry => collect(entry));
    }
  };

  collect(input);

  if (out.size > 0) {
    return Array.from(out);
  }

  if (!input) {
    return [];
  }
  if (Array.isArray(input)) {
    return input.filter((v): v is string => typeof v === 'string');
  }
  if (typeof input === 'string') {
    try {
      const parsed = JSON.parse(input);
      if (Array.isArray(parsed)) {
        return parsed.filter((v): v is string => typeof v === 'string');
      }
    } catch {
      return [];
    }
  }
  return [];
}

export function CreateRoleModal({ isOpen, onClose, onSaved, role }: Props) {
  const { t } = useTranslation();
  const isEditing = Boolean(role);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [permissionDefs, setPermissionDefs] = useState<PermissionDefs>({});
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  const visiblePermissionDefs = useMemo(
    () => mergeSelectedPermissions(permissionDefs, selectedPermissions),
    [permissionDefs, selectedPermissions]
  );

  useEffect(()=>{
    if (!isOpen) return;
    const loadPermissions = async ()=>{
      try {
        // try a couple common endpoints
        let res:any;
        try { res = await api.get('/security/api/permissions/definitions'); } catch(e) { }
        if (!res) {
          try { res = await api.get('/admin/permissions'); } catch(e) { }
        }
        if (res && res.data && (res.data.permissions || res.data)) {
          const perms = res.data.permissions || res.data;
          setPermissionDefs(normalizePermissionDefs(perms));
          return;
        }
      } catch (err) {
        // ignore
      }

      setPermissionDefs(fallbackPermissionDefs);
    };
    loadPermissions();
  },[isOpen]);

  useEffect(() => {
    if (!isOpen) {
      setName('');
      setDescription('');
      setSelectedPermissions([]);
      return;
    }

    if (role) {
      setName(role.name || '');
      setDescription(role.description || '');
      setSelectedPermissions(parseSelectedPermissions(role.permissions));
    }
  }, [isOpen, role]);

  if (!isOpen) return null;

  const handleSave = async () => {
    if (!name.trim()) {
      alert(t('admin.rolesTab.roleNameRequired'));
      return;
    }
    setSaving(true);
    try {
      const permissions = selectedPermissions;
      if (isEditing && role?.id) {
        await api.put(`/admin/roles/${role.id}`, { name: name.trim(), description: description.trim(), permissions });
      } else {
        await api.post('/admin/roles', { name: name.trim(), description: description.trim(), permissions });
      }
      onSaved && onSaved();
      onClose();
    } catch (err:any) {
      alert(`${t('common.error')}: ` + (err.response?.data?.error || err.message));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-60 flex items-center justify-center">
        <div className="absolute inset-0 bg-black/60" onClick={onClose} />
        <div className="relative w-full max-w-2xl bg-dark rounded-xl p-6 border border-white/10 z-70">
          <h3 className="text-2xl font-bold text-white mb-2">{isEditing ? t('admin.rolesTab.editRole') : t('admin.rolesTab.createRole')}</h3>
          <p className="text-gray-400 text-sm mb-4">{isEditing ? t('admin.rolesTab.editRoleSubtitle') : t('admin.rolesTab.createRoleSubtitle')}</p>

          <div className="space-y-3">
            <div>
              <label htmlFor="create-role-name" className="block text-sm text-gray-300">{t('admin.rolesTab.roleName')}</label>
              <input id="create-role-name" value={name} onChange={e=>setName(e.target.value)} className="w-full px-3 py-2 rounded-lg glass text-white" />
            </div>
            <div>
              <label htmlFor="create-role-description" className="block text-sm text-gray-300">{t('admin.rolesTab.roleDesc')}</label>
              <input id="create-role-description" value={description} onChange={e=>setDescription(e.target.value)} className="w-full px-3 py-2 rounded-lg glass text-white" />
            </div>
            <div>
              <label className="block text-sm text-gray-300 mb-2">{t('admin.rolesTab.rolePermissions')}</label>
              {Object.keys(visiblePermissionDefs).length === 0 ? (
                <p className="text-sm text-gray-400">{t('roleGuard.loading') || 'Loading permissions...'}</p>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 max-h-60 overflow-auto">
                  {Object.entries(visiblePermissionDefs).map(([group, perms])=> (
                    <div key={group} className="p-2 rounded-md border border-white/5">
                      <div className="text-xs text-gray-300 font-semibold mb-1">{group}</div>
                      <div className="flex flex-col gap-1">
                        {(Array.isArray(perms) ? perms : []).map(p=> (
                          <label key={p} className="inline-flex items-center gap-2 text-sm text-white">
                            <input type="checkbox" checked={selectedPermissions.includes(p)} onChange={()=>{
                              setSelectedPermissions(prev => prev.includes(p) ? prev.filter(x=>x!==p) : [...prev,p]);
                            }} className="w-4 h-4" />
                            <span className="truncate">{p}</span>
                          </label>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          <div className="flex justify-end gap-2 mt-4">
            <button onClick={onClose} className="px-4 py-2 rounded-lg bg-white/10 text-white">{t('common.close')}</button>
            <button onClick={handleSave} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white">{saving ? t('common.saving') : (isEditing ? t('admin.rolesTab.saveRole') : t('admin.rolesTab.createRole'))}</button>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
