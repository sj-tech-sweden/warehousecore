import { useState, useEffect } from 'react';
import { ModalPortal } from '../../components/ModalPortal';
import { useTranslation } from 'react-i18next';
import { api } from '../../lib/api';

interface Role { id: number; name: string }
interface Props {
  isOpen: boolean;
  onClose: () => void;
  onCreated?: () => void;
  roles: Role[];
}

export function CreateUserModal({ isOpen, onClose, onCreated, roles }: Props) {
  const { t } = useTranslation();
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [password, setPassword] = useState('');
  const [selectedRoles, setSelectedRoles] = useState<number[]>([]);
  const [forceChange, setForceChange] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(()=>{
    if (!isOpen) {
      setUsername(''); setEmail(''); setFirstName(''); setLastName(''); setPassword(''); setSelectedRoles([]); setForceChange(false);
    }
  },[isOpen]);

  if (!isOpen) return null;

  const toggleRole = (id:number) => setSelectedRoles(prev => prev.includes(id) ? prev.filter(x=>x!==id) : [...prev, id]);

  const handleSave = async () => {
    if (!username.trim() || !email.includes('@') || password.length < 6) {
      alert(t('admin.rolesTab.validationError'));
      return;
    }
    setSaving(true);
    try {
      await api.post('/admin/users', { username, email, password, first_name: firstName, last_name: lastName, role_ids: selectedRoles, force_password_change: forceChange });
      onCreated && onCreated();
      onClose();
    } catch (err:any) {
      alert(`${t('common.error')}: ` + (err.response?.data?.error || err.message));
    } finally { setSaving(false); }
  };

  return (
    <ModalPortal>
      <div className="fixed inset-0 z-50 flex items-center justify-center">
        <div className="absolute inset-0 bg-black/60" onClick={onClose} />
        <div className="relative z-10 w-full max-w-2xl bg-dark rounded-xl p-6 border border-white/10">
          <h3 className="text-2xl font-bold text-white mb-2">{t('admin.rolesTab.createUser')}</h3>
          <p className="text-gray-400 text-sm mb-4">{t('admin.rolesTab.createUserSubtitle')}</p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <input placeholder={t('admin.rolesTab.username')} autoComplete="username" value={username} onChange={e=>setUsername(e.target.value)} className="px-3 py-2 rounded-lg glass text-white" />
            <input placeholder={t('admin.rolesTab.email')} autoComplete="email" value={email} onChange={e=>setEmail(e.target.value)} className="px-3 py-2 rounded-lg glass text-white" />
            <input placeholder={t('admin.rolesTab.firstName')} value={firstName} onChange={e=>setFirstName(e.target.value)} className="px-3 py-2 rounded-lg glass text-white" />
            <input placeholder={t('admin.rolesTab.lastName')} value={lastName} onChange={e=>setLastName(e.target.value)} className="px-3 py-2 rounded-lg glass text-white" />
            <input type="password" placeholder={t('admin.rolesTab.password')} autoComplete="new-password" value={password} onChange={e=>setPassword(e.target.value)} className="px-3 py-2 rounded-lg glass text-white" />
          </div>

          <div className="mt-4">
            <label className="block text-sm font-semibold text-gray-400 mb-2">{t('admin.rolesTab.assignRoles')}</label>
            <div className="flex flex-wrap gap-2">
              {roles.map(r => (
                <label key={r.id} className="inline-flex items-center gap-2 px-3 py-1 rounded-lg glass cursor-pointer">
                  <input type="checkbox" checked={selectedRoles.includes(r.id)} onChange={()=>toggleRole(r.id)} className="w-4 h-4" />
                  <span className="text-sm text-white">{r.name}</span>
                </label>
              ))}
            </div>
          </div>

          <div className="mt-3">
            <label className="inline-flex items-center gap-2 text-sm text-gray-300">
              <input type="checkbox" checked={forceChange} onChange={e=>setForceChange(e.target.checked)} className="w-4 h-4" />
              <span>{t('admin.rolesTab.forcePasswordChange')}</span>
            </label>
          </div>

          <div className="flex justify-end gap-2 mt-4">
            <button onClick={onClose} className="px-4 py-2 rounded-lg bg-white/10 text-white">{t('common.close')}</button>
            <button onClick={handleSave} disabled={saving} className="px-4 py-2 rounded-lg bg-green-600 text-white">{saving ? t('common.saving') : t('admin.rolesTab.createUser')}</button>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
