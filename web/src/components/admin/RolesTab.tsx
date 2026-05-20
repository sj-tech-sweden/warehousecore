import { useState, useEffect, useMemo, useRef } from 'react';
import { Users, Shield, Save } from 'lucide-react';
import { api } from '../../lib/api';
import { useAuth } from '../../contexts/AuthContext';
import { useTranslation } from 'react-i18next';
import { CreateUserModal } from './CreateUserModal';
import { CreateRoleModal } from './CreateRoleModal';

interface Role {
  id: number;
  name: string;
  description: string;
  permissions?: unknown;
}

interface UserWithRoles {
  userID: number;
  username: string;
  email: string;
  first_name: string;
  last_name: string;
  roles: Role[];
}

export function RolesTab() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const [users, setUsers] = useState<UserWithRoles[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [selectedUser, setSelectedUser] = useState<UserWithRoles | null>(null);
  const [selectedRoles, setSelectedRoles] = useState<number[]>([]);
  const [saving, setSaving] = useState(false);
  const [userSaving, setUserSaving] = useState(false);
  const [emailEdit, setEmailEdit] = useState('');
  const [displayNameEdit, setDisplayNameEdit] = useState('');
  const [resetPwMsg, setResetPwMsg] = useState('');
  const [editingRole, setEditingRole] = useState<Role | null>(null);
  const resetPwTimerRef = useRef<number | null>(null);

  const currentRoles = useMemo(
    () => (user?.Roles ?? user?.roles ?? []) as Role[],
    [user]
  );
  const restrictedRoleNames = useMemo(() => ['admin', 'manager', 'super_admin'], []);
  const canManageRestricted = useMemo(
    () =>
      currentRoles.some(role => {
        const name = (role.name ?? '').toLowerCase();
        return name === 'admin' || name === 'super_admin';
      }),
    [currentRoles]
  );

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    return () => {
      if (resetPwTimerRef.current !== null) {
        window.clearTimeout(resetPwTimerRef.current);
      }
    };
  }, []);

  const loadData = async () => {
    try {
      const [usersRes, rolesRes] = await Promise.all([
        api.get('/admin/users'),
        api.get('/admin/roles'),
      ]);
      setUsers(usersRes.data);
      setRoles(rolesRes.data);
    } catch (error) {
      console.error('Failed to load data:', error);
    }
  };

  // creation now handled by modals (state moved into modal components)
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [userModalOpen, setUserModalOpen] = useState(false);

  const openCreateRoleModal = () => {
    setEditingRole(null);
    setRoleModalOpen(true);
  };

  const openEditRoleModal = (role: Role) => {
    setEditingRole(role);
    setRoleModalOpen(true);
  };

  const selectUser = (user: UserWithRoles) => {
    setSelectedUser(user);
    setSelectedRoles(user.roles.map(r => r.id));
    setEmailEdit(user.email || '');
    setDisplayNameEdit(`${user.first_name || ''} ${user.last_name || ''}`);
  };

  const toggleRole = (roleId: number) => {
    const role = roles.find(r => r.id === roleId);
    if (!role) return;

    const isRestricted = restrictedRoleNames.includes((role.name ?? '').toLowerCase());
    if (isRestricted && !canManageRestricted) {
      return;
    }

    setSelectedRoles(prev =>
      prev.includes(roleId)
        ? prev.filter(id => id !== roleId)
        : [...prev, roleId]
    );
  };

  const handleSave = async () => {
    if (!selectedUser) return;

    setSaving(true);
    try {
      await api.put(`/admin/users/${selectedUser.userID}/roles`, {
        role_ids: selectedRoles,
      });
      await loadData();
      setSelectedUser(null);
      alert(t('admin.rolesTab.messages.updated'));
    } catch (error: any) {
      alert(`${t('common.error')}: ` + (error.response?.data?.error || error.message));
    } finally {
      setSaving(false);
    }
  };


  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3 justify-between">
        <div className="flex items-center gap-3">
          <Users className="w-6 h-6 text-blue-400" />
          <div>
            <h2 className="text-xl font-bold text-white">{t('admin.rolesTab.title')}</h2>
            <p className="text-gray-400 text-sm">{t('admin.rolesTab.subtitle')}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={()=>setUserModalOpen(true)} className="px-3 py-2 rounded-lg bg-green-600 text-white">{t('admin.rolesTab.createUser')}</button>
          <button onClick={openCreateRoleModal} className="px-3 py-2 rounded-lg bg-blue-600 text-white">{t('admin.rolesTab.createRole')}</button>
        </div>
      </div>

      <div className="glass rounded-xl p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-white font-semibold">{t('admin.rolesTab.roles')}</h3>
          <span className="text-xs text-gray-400">{roles.length}</span>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-2">
          {roles.map(role => (
            <div key={role.id} className="rounded-lg border border-white/10 p-3 bg-white/5">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-white font-semibold truncate">{role.name}</p>
                  <p className="text-gray-400 text-sm line-clamp-2">{role.description || t('admin.rolesTab.noDescription')}</p>
                </div>
                <button
                  onClick={() => openEditRoleModal(role)}
                  className="px-2 py-1 rounded-md bg-white/10 text-xs text-white hover:bg-white/20"
                >
                  {t('admin.rolesTab.editRole')}
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Users List */}
        <div className="space-y-2">
          {/* Create user moved to modal */}
          <h3 className="text-white font-semibold mb-3">{t('admin.rolesTab.users')}</h3>
          {users.map(user => (
            <div
              key={user.userID}
              onClick={() => selectUser(user)}
              className={`glass rounded-xl p-4 cursor-pointer transition-all ${
                selectedUser?.userID === user.userID
                  ? 'ring-2 ring-accent-red'
                  : 'hover:bg-white/5'
              }`}
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-accent-red/20 flex items-center justify-center flex-shrink-0">
                  <Users className="w-5 h-5 text-accent-red" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-white font-semibold truncate">
                    {user.first_name} {user.last_name}
                  </p>
                  <p className="text-gray-400 text-sm truncate">{user.email}</p>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {user.roles.map(role => (
                      <span
                        key={role.id}
                        className="px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-400 text-xs"
                      >
                        {role.name}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Modals */}
        {userModalOpen && (
          <CreateUserModal isOpen={userModalOpen} onClose={()=>setUserModalOpen(false)} onCreated={loadData} roles={roles} />
        )}
        {roleModalOpen && (
          <CreateRoleModal
            isOpen={roleModalOpen}
            onClose={()=>{ setRoleModalOpen(false); setEditingRole(null); }}
            onSaved={loadData}
            role={editingRole}
          />
        )}

        {/* Roles Editor */}
        <div className="glass rounded-xl p-6">
          {/* Role/Create area now mainly shows editor and list; modals handle create */}
          {/* Create role moved to modal */}
          {selectedUser ? (
            <div className="space-y-4">
              <div className="space-y-3">
                <label htmlFor="admin-edit-user-email" className="block text-sm font-semibold text-gray-400">Email</label>
                <input id="admin-edit-user-email" value={emailEdit} onChange={(e)=>setEmailEdit(e.target.value)} className="w-full px-3 py-2 rounded-lg glass text-white" />
                <label htmlFor="admin-edit-user-display-name" className="block text-sm font-semibold text-gray-400 mt-2">Display name</label>
                <input id="admin-edit-user-display-name" value={displayNameEdit} onChange={(e)=>setDisplayNameEdit(e.target.value)} className="w-full px-3 py-2 rounded-lg glass text-white" />
                <div className="flex gap-2 mt-3">
                  <button onClick={async ()=>{
                    if (!selectedUser) return;
                    setUserSaving(true);
                    try {
                      await api.put(`/admin/users/${selectedUser.userID}`, {
                        email: emailEdit,
                        display_name: displayNameEdit,
                      });
                      await loadData();
                      alert(t('admin.rolesTab.messages.userUpdated'));
                    } catch (err:any) {
                      alert(`${t('common.error')}: ` + (err.response?.data?.error || err.message));
                    } finally { setUserSaving(false); }
                  }} disabled={userSaving} className={`px-4 py-2 rounded-lg font-semibold text-white ${userSaving ? 'bg-gray-600' : 'bg-blue-600'}`}>{t('admin.rolesTab.updateUser')}</button>
                  <button onClick={async ()=>{
                    if (!selectedUser) return;
                    setResetPwMsg('');
                    try {
                      const res = await api.post(`/admin/users/${selectedUser.userID}/reset-password`);
                      if (res.data && res.data.new_password) {
                        setResetPwMsg(t('admin.rolesTab.resetPasswordReturned', { pw: res.data.new_password }));
                      } else {
                        setResetPwMsg(t('admin.rolesTab.resetPasswordSuccess'));
                      }
                    } catch (err:any) {
                      alert(`${t('common.error')}: ` + (err.response?.data?.error || err.message));
                    }
                    if (resetPwTimerRef.current !== null) {
                      window.clearTimeout(resetPwTimerRef.current);
                    }
                    resetPwTimerRef.current = window.setTimeout(() => {
                      setResetPwMsg('');
                      resetPwTimerRef.current = null;
                    }, 10000);
                  }} className="px-4 py-2 rounded-lg font-semibold text-white bg-red-600">{t('admin.rolesTab.resetPassword')}</button>
                </div>
                {resetPwMsg && <p className="text-sm text-green-400 mt-2">{resetPwMsg}</p>}
              </div>
              <div>
                <h3 className="text-white font-semibold mb-2">
                  {t('admin.rolesTab.rolesFor', { name: `${selectedUser.first_name} ${selectedUser.last_name}` })}
                </h3>
                <p className="text-gray-400 text-sm">{selectedUser.email}</p>
              </div>

              <div className="space-y-2">
                {roles.map(role => {
                  const isRestricted = restrictedRoleNames.includes((role.name ?? '').toLowerCase());
                  const disabled = isRestricted && !canManageRestricted;
                  return (
                    <label
                      key={role.id}
                      className={`flex items-center gap-3 p-3 rounded-lg glass transition-colors ${
                        disabled ? 'cursor-not-allowed opacity-70' : 'cursor-pointer hover:bg-white/5'
                      }`}
                      title={disabled ? t('admin.rolesTab.restrictedTitle') : undefined}
                    >
                      <input
                        type="checkbox"
                        checked={selectedRoles.includes(role.id)}
                        onChange={() => toggleRole(role.id)}
                        className="w-5 h-5 rounded accent-accent-red"
                        disabled={disabled}
                      />
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <Shield className="w-4 h-4 text-blue-400" />
                          <span className="text-white font-semibold">{role.name}</span>
                        </div>
                        <p className="text-gray-400 text-sm">{role.description}</p>
                        {disabled && (
                          <p className="text-xs text-gray-500 mt-1">
                            {t('admin.rolesTab.restrictedDescription')}
                          </p>
                        )}
                      </div>
                    </label>
                  );
                })}
              </div>

              <button
                onClick={handleSave}
                disabled={saving}
                className={`w-full py-3 px-6 rounded-xl font-semibold text-white transition-all flex items-center justify-center gap-2 ${
                  saving
                    ? 'bg-gray-600 cursor-not-allowed'
                    : 'bg-gradient-to-r from-accent-red to-red-700 hover:shadow-lg'
                }`}
              >
                <Save className="w-5 h-5" />
                <span>{saving ? t('common.saving') : t('common.save')}</span>
              </button>
              
            </div>
          ) : (
            <div className="text-center text-gray-400 py-12">
              <Users className="w-12 h-12 mx-auto mb-3 opacity-50" />
              <p>{t('admin.rolesTab.selectUserHint')}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
