import { useState, useEffect } from 'react';
import {
  Users,
  ShieldCheck,
  Shield,
  Eye,
  AlertTriangle,
  Trash2,
  ToggleLeft,
  ToggleRight,
  RefreshCw,
} from 'lucide-react';

import { getUsers, deleteUser, updateUser } from '../api/api';
import { useAuth } from '../contexts/AuthContext';
import { fmtDate } from '../utils/date';

import type { SafeUser, UserRole } from '../types';

import PageHeader from '../components/PageHeader';
import StatCard from '../components/StatCard';
import JsonModal from '../components/JsonModal';

const ROLE_LABELS: Record<UserRole, string> = {
  admin: 'Администратор',
  analyst: 'Аналитик',
  viewer: 'Наблюдатель',
};

const ROLE_COLORS: Record<
  UserRole,
  { bg: string; color: string; border: string }
> = {
  admin: {
    bg: 'rgba(255,95,109,0.1)',
    color: '#ff5f6d',
    border: 'rgba(255,95,109,0.3)',
  },
  analyst: {
    bg: 'rgba(255,169,77,0.1)',
    color: '#ffa94d',
    border: 'rgba(255,169,77,0.3)',
  },
  viewer: {
    bg: 'rgba(195,253,184,0.1)',
    color: '#C3FDB8',
    border: 'rgba(195,253,184,0.3)',
  },
};

function RoleBadge({ role }: { role: UserRole }) {
  const c = ROLE_COLORS[role] ?? ROLE_COLORS.viewer;

  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '5px',
        padding: '3px 10px',
        borderRadius: '20px',
        fontSize: '11px',
        fontFamily: 'var(--font-display)',
        fontWeight: 700,
        letterSpacing: '0.06em',
        textTransform: 'uppercase',
        background: c.bg,
        color: c.color,
        border: `1px solid ${c.border}`,
      }}
    >
      {role === 'admin' && <ShieldCheck size={10} />}
      {role === 'analyst' && <Shield size={10} />}
      {role === 'viewer' && <Eye size={10} />}
      {ROLE_LABELS[role]}
    </span>
  );
}

function ActiveBadge({ active }: { active: boolean }) {
  return (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: '5px',
        padding: '3px 10px',
        borderRadius: '20px',
        fontSize: '11px',
        fontFamily: 'var(--font-mono)',
        background: active
          ? 'rgba(195,253,184,0.1)'
          : 'rgba(100,100,140,0.1)',
        color: active ? '#C3FDB8' : '#888bac',
      }}
    >
      <span
        style={{
          width: '6px',
          height: '6px',
          borderRadius: '50%',
          background: active ? '#C3FDB8' : '#888bac',
          display: 'inline-block',
          ...(active
            ? {
                boxShadow: '0 0 0 2px rgba(195,253,184,0.25)',
                animation: 'pulse-glow 2s infinite',
              }
            : {}),
        }}
      />

      {active ? 'Активен' : 'Отключён'}
    </span>
  );
}

export default function UsersPage() {
  const { user: currentUser } = useAuth();

  const [users, setUsers] = useState<SafeUser[]>([]);
  const [loading, setLoading] = useState(true);

  const [error, setError] = useState('');
  const [actionError, setActionError] = useState('');

  const [pendingId, setPendingId] = useState<number | null>(null);

  // modal
  const [selectedUser, setSelectedUser] =
    useState<SafeUser | null>(null);

  const [roleChoice, setRoleChoice] =
    useState<UserRole | ''>('');

  const [savingRole, setSavingRole] = useState(false);
  const [roleError, setRoleError] = useState('');

  const load = () => {
    setLoading(true);
    setError('');

    getUsers()
      .then(setUsers)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, []);

  const totalUsers = users.length;

  const admins = users.filter(
    u => u.role === 'admin'
  ).length;

  const activeUsers = users.filter(
    u => u.is_active
  ).length;

  const openUserModal = (u: SafeUser) => {
    setSelectedUser(u);
    setRoleChoice(u.role);
    setRoleError('');
  };

  const closeUserModal = () => {
    setSelectedUser(null);
  };

  const handleRoleUpdate = async () => {
    if (!selectedUser || !roleChoice) return;

    if (
      selectedUser.id === currentUser?.id &&
      roleChoice !== 'admin'
    ) {
      setRoleError(
        'Нельзя изменить собственную роль администратора.'
      );
      return;
    }

    setSavingRole(true);
    setRoleError('');

    try {
      const updated = await updateUser(selectedUser.id, {
        role: roleChoice,
      });

      setUsers(prev =>
        prev.map(u =>
          u.id === updated.id ? updated : u
        )
      );

      setSelectedUser(updated);

      closeUserModal();
    } catch (e) {
      setRoleError(
        e instanceof Error
          ? e.message
          : 'Ошибка обновления роли'
      );
    } finally {
      setSavingRole(false);
    }
  };

  const handleToggleActive = async (u: SafeUser) => {
    if (u.id === currentUser?.id) {
      setActionError(
        'Нельзя деактивировать собственную учётную запись.'
      );
      return;
    }

    setPendingId(u.id);
    setActionError('');

    try {
      const updated = await updateUser(u.id, {
        is_active: !u.is_active,
      });

      setUsers(prev =>
        prev.map(x =>
          x.id === u.id ? updated : x
        )
      );
    } catch (e) {
      setActionError(
        e instanceof Error
          ? e.message
          : 'Ошибка обновления'
      );
    } finally {
      setPendingId(null);
    }
  };

  const handleDelete = async (u: SafeUser) => {
    if (u.id === currentUser?.id) {
      setActionError(
        'Нельзя удалить собственную учётную запись.'
      );
      return;
    }

    if (
      !confirm(
        `Удалить пользователя «${u.username}»? Это действие необратимо.`
      )
    )
      return;

    setPendingId(u.id);
    setActionError('');

    try {
      await deleteUser(u.id);

      setUsers(prev =>
        prev.filter(x => x.id !== u.id)
      );
    } catch (e) {
      setActionError(
        e instanceof Error
          ? e.message
          : 'Ошибка удаления'
      );
    } finally {
      setPendingId(null);
    }
  };

  const sortedUsers = [...users].sort((a, b) => {
    const roleOrder = {
      admin: 0,
      analyst: 1,
      viewer: 2,
    };

    if (
      roleOrder[a.role] !== roleOrder[b.role]
    ) {
      return (
        roleOrder[a.role] -
        roleOrder[b.role]
      );
    }

    return a.username.localeCompare(
      b.username,
      'ru'
    );
  });

  const userModalFooter = selectedUser && (
    <div style={{ padding: '20px 22px' }}>
      <div style={{ marginBottom: '14px' }}>
        <div style={sectionLabel}>
          Изменить роль
        </div>

        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '8px',
          }}
        >
          {(
            ['admin', 'analyst', 'viewer'] as UserRole[]
          ).map(role => (
            <label
              key={role}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '9px',
                cursor: 'pointer',
                fontSize: '13px',
                fontFamily:
                  'var(--font-display)',
              }}
            >
              <input
                type="radio"
                name="user_role"
                value={role}
                checked={roleChoice === role}
                onChange={() =>
                  setRoleChoice(role)
                }
                style={{
                  accentColor: 'var(--mint)',
                  width: '15px',
                  height: '15px',
                  cursor: 'pointer',
                }}
              />

              <span
                style={{
                  color:
                    roleChoice === role
                      ? 'var(--mint)'
                      : 'var(--text-secondary)',
                }}
              >
                {ROLE_LABELS[role]}
              </span>
            </label>
          ))}
        </div>
      </div>

      {roleError && (
        <div
          style={{
            color: '#ff5f6d',
            fontSize: '12px',
            marginBottom: '10px',
            display: 'flex',
            gap: '6px',
            alignItems: 'center',
          }}
        >
          <AlertTriangle size={12} />
          {roleError}
        </div>
      )}

      <div
        style={{
          display: 'flex',
          justifyContent: 'flex-end',
        }}
      >
        <button
          onClick={handleRoleUpdate}
          disabled={!roleChoice || savingRole}
          style={{
            padding: '9px 22px',
            borderRadius: '9px',
            border: 'none',
            background:
              !roleChoice || savingRole
                ? 'var(--navy-border)'
                : 'var(--mint)',
            color:
              !roleChoice || savingRole
                ? 'var(--text-secondary)'
                : 'var(--navy)',
            fontFamily:
              'var(--font-display)',
            fontWeight: 800,
            fontSize: '13px',
            cursor:
              !roleChoice || savingRole
                ? 'not-allowed'
                : 'pointer',
          }}
        >
          {savingRole
            ? 'Сохранение...'
            : 'Применить'}
        </button>
      </div>
    </div>
  );

  return (
    <div
      style={{ padding: '0 0 40px' }}
      className="animate-in"
    >
      <PageHeader
        title="Пользователи"
        subtitle="Управление учётными записями — только для администраторов"
        action={
          <button
            onClick={load}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: '7px',
              padding: '9px 16px',
              borderRadius: '9px',
              border:
                '1px solid var(--navy-border)',
              background: 'transparent',
              color: 'var(--mint)',
              cursor: 'pointer',
              fontFamily:
                'var(--font-display)',
              fontWeight: 600,
              fontSize: '13px',
            }}
          >
            <RefreshCw size={14} />
            Обновить
          </button>
        }
      />

      <div style={{ padding: '0 32px' }}>
        {error && (
          <div style={errorBox}>
            <AlertTriangle size={14} />
            {error}
          </div>
        )}

        {actionError && (
          <div
            style={{
              ...errorBox,
              marginBottom: '16px',
            }}
          >
            <AlertTriangle size={14} />
            {actionError}

            <button
              onClick={() =>
                setActionError('')
              }
              style={{
                marginLeft: 'auto',
                background: 'none',
                border: 'none',
                color: '#ff5f6d',
                cursor: 'pointer',
                fontSize: '16px',
                lineHeight: 1,
              }}
            >
              ×
            </button>
          </div>
        )}

        {loading ? (
          <Spinner />
        ) : (
          <>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns:
                  'repeat(3, 1fr)',
                gap: '14px',
                marginBottom: '28px',
              }}
            >
              <StatCard
                label="Всего пользователей"
                value={totalUsers}
                icon={<Users size={18} />}
                accent="#C3FDB8"
              />

              <StatCard
                label="Администраторов"
                value={admins}
                icon={<ShieldCheck size={18} />}
                accent="#ff5f6d"
              />

              <StatCard
                label="Активных"
                value={activeUsers}
                icon={<Shield size={18} />}
                accent="#ffa94d"
                sub={`из ${totalUsers}`}
              />
            </div>

            <div
              style={{
                background:
                  'var(--navy-light)',
                border:
                  '1px solid var(--navy-border)',
                borderRadius: '12px',
                overflow: 'hidden',
              }}
            >
              <table
                style={{
                  width: '100%',
                  borderCollapse: 'collapse',
                  tableLayout: 'fixed',
                }}
              >
                <colgroup>
                  <col style={{ width: '50px' }} />
                  <col
                    style={{ width: '150px' }}
                  />
                  <col />
                  <col
                    style={{ width: '140px' }}
                  />
                  <col
                    style={{ width: '130px' }}
                  />
                  <col
                    style={{ width: '160px' }}
                  />
                  <col
                    style={{ width: '160px' }}
                  />
                  <col
                    style={{ width: '100px' }}
                  />
                </colgroup>

                <thead>
                  <tr
                    style={{
                      background:
                        'var(--navy-lighter)',
                      borderBottom:
                        '1px solid var(--navy-border)',
                    }}
                  >
                    {[
                      'ID',
                      'Логин',
                      'Email',
                      'Роль',
                      'Статус',
                      'Создан',
                      'Последний вход',
                      'Действия',
                    ].map(h => (
                      <th
                        key={h}
                        style={{
                          padding: '11px 14px',
                          textAlign: 'left',
                          fontSize: '11px',
                          fontFamily:
                            'var(--font-display)',
                          fontWeight: 700,
                          letterSpacing:
                            '0.08em',
                          color:
                            'var(--text-secondary)',
                          textTransform:
                            'uppercase',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>

                <tbody>
                  {sortedUsers.length === 0 ? (
                    <tr>
                      <td
                        colSpan={8}
                        style={{
                          padding: '32px',
                          textAlign: 'center',
                          color:
                            'var(--text-secondary)',
                          fontSize: '13px',
                        }}
                      >
                        Нет зарегистрированных
                        пользователей
                      </td>
                    </tr>
                  ) : (
                    sortedUsers.map((u, i) => {
                      const isSelf =
                        u.id === currentUser?.id;

                      const busy =
                        pendingId === u.id;

                      return (
                        <tr
                          key={u.id}
                          onClick={() =>
                            openUserModal(u)
                          }
                          style={{
                            cursor: 'pointer',
                            borderBottom:
                              i <
                              sortedUsers.length - 1
                                ? '1px solid var(--navy-border)'
                                : 'none',
                            background: isSelf
                              ? 'rgba(195,253,184,0.04)'
                              : 'transparent',
                            opacity: busy
                              ? 0.6
                              : 1,
                            transition:
                              'opacity 0.2s',
                          }}
                        >
                          <td style={td}>
                            <span
                              style={{
                                fontFamily:
                                  'var(--font-mono)',
                                fontSize: '12px',
                                color:
                                  'var(--text-secondary)',
                              }}
                            >
                              {u.id}
                            </span>
                          </td>

                          <td style={td}>
                            <div
                              style={{
                                display: 'flex',
                                alignItems:
                                  'center',
                                gap: '7px',
                              }}
                            >
                              <span
                                style={{
                                  fontFamily:
                                    'var(--font-display)',
                                  fontWeight: 700,
                                  fontSize:
                                    '13px',
                                }}
                              >
                                {u.username}
                              </span>

                              {isSelf && (
                                <span
                                  style={{
                                    fontSize:
                                      '10px',
                                    background:
                                      'rgba(195,253,184,0.15)',
                                    color:
                                      'var(--mint)',
                                    padding:
                                      '1px 6px',
                                    borderRadius:
                                      '10px',
                                    fontFamily:
                                      'var(--font-display)',
                                    fontWeight: 700,
                                  }}
                                >
                                  вы
                                </span>
                              )}
                            </div>
                          </td>

                          <td style={td}>
                            <span
                              style={{
                                fontFamily:
                                  'var(--font-mono)',
                                fontSize: '12px',
                                color:
                                  'var(--text-secondary)',
                                display: 'block',
                                overflow:
                                  'hidden',
                                textOverflow:
                                  'ellipsis',
                                whiteSpace:
                                  'nowrap',
                              }}
                            >
                              {u.email}
                            </span>
                          </td>

                          <td style={td}>
                            <RoleBadge
                              role={u.role}
                            />
                          </td>

                          <td style={td}>
                            <ActiveBadge
                              active={
                                u.is_active
                              }
                            />
                          </td>

                          <td style={td}>
                            <span
                              style={{
                                fontSize: '12px',
                                color:
                                  'var(--text-secondary)',
                                fontFamily:
                                  'var(--font-mono)',
                              }}
                            >
                              {fmtDate(
                                u.created_at
                              )}
                            </span>
                          </td>

                          <td style={td}>
                            <span
                              style={{
                                fontSize: '12px',
                                color:
                                  'var(--text-secondary)',
                                fontFamily:
                                  'var(--font-mono)',
                              }}
                            >
                              {u.last_login_at
                                ? fmtDate(
                                    u.last_login_at
                                  )
                                : '—'}
                            </span>
                          </td>

                          <td style={td}>
                            <div
                              onClick={e =>
                                e.stopPropagation()
                              }
                              style={{
                                display: 'flex',
                                gap: '6px',
                                alignItems:
                                  'center',
                              }}
                            >
                              <button
                                onClick={() =>
                                  handleToggleActive(
                                    u
                                  )
                                }
                                disabled={
                                  busy || isSelf
                                }
                                title={
                                  u.is_active
                                    ? 'Деактивировать'
                                    : 'Активировать'
                                }
                                style={{
                                  background:
                                    'transparent',
                                  border:
                                    '1px solid var(--navy-border)',
                                  color: isSelf
                                    ? 'var(--navy-border)'
                                    : u.is_active
                                    ? 'var(--mint)'
                                    : '#888bac',
                                  cursor:
                                    busy ||
                                    isSelf
                                      ? 'not-allowed'
                                      : 'pointer',
                                  padding:
                                    '5px 7px',
                                  borderRadius:
                                    '6px',
                                  display:
                                    'flex',
                                  alignItems:
                                    'center',
                                  opacity:
                                    isSelf
                                      ? 0.4
                                      : 1,
                                }}
                              >
                                {u.is_active ? (
                                  <ToggleRight
                                    size={16}
                                    strokeWidth={
                                      2
                                    }
                                  />
                                ) : (
                                  <ToggleLeft
                                    size={16}
                                    strokeWidth={
                                      2
                                    }
                                  />
                                )}
                              </button>

                              <button
                                onClick={() =>
                                  handleDelete(
                                    u
                                  )
                                }
                                disabled={
                                  busy || isSelf
                                }
                                title={
                                  isSelf
                                    ? 'Нельзя удалить себя'
                                    : 'Удалить пользователя'
                                }
                                style={{
                                  background:
                                    'transparent',
                                  border:
                                    '1px solid var(--navy-border)',
                                  color: isSelf
                                    ? 'var(--navy-border)'
                                    : '#ff5f6d',
                                  cursor:
                                    busy ||
                                    isSelf
                                      ? 'not-allowed'
                                      : 'pointer',
                                  padding:
                                    '5px 7px',
                                  borderRadius:
                                    '6px',
                                  display:
                                    'flex',
                                  alignItems:
                                    'center',
                                  opacity:
                                    isSelf
                                      ? 0.4
                                      : 1,
                                }}
                              >
                                <Trash2
                                  size={15}
                                  strokeWidth={
                                    2
                                  }
                                />
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>

            <p
              style={{
                marginTop: '12px',
                fontSize: '12px',
                color: 'var(--text-secondary)',
                fontFamily: 'var(--font-mono)',
              }}
            >
              Собственную учётную запись
              деактивировать и удалить
              невозможно. Первый
              зарегистрированный пользователь
              — всегда администратор.
            </p>
          </>
        )}
      </div>

      {selectedUser && (
        <JsonModal
          title={selectedUser.username}
          subtitle={
            <>
              <RoleBadge
                role={selectedUser.role}
              />
              <ActiveBadge
                active={
                  selectedUser.is_active
                }
              />
            </>
          }
          data={selectedUser}
          onClose={closeUserModal}
          footer={userModalFooter}
        />
      )}
    </div>
  );
}

function Spinner() {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        height: '300px',
        flexDirection: 'column',
        gap: '12px',
      }}
    >
      <div
        style={{
          width: '36px',
          height: '36px',
          borderRadius: '50%',
          border:
            '3px solid var(--navy-border)',
          borderTopColor: 'var(--mint)',
          animation:
            'spin 0.8s linear infinite',
        }}
      />

      <span
        style={{
          color: 'var(--text-secondary)',
          fontSize: '13px',
        }}
      >
        Загрузка пользователей...
      </span>

      <style>
        {`
          @keyframes spin {
            to {
              transform: rotate(360deg)
            }
          }
        `}
      </style>
    </div>
  );
}

const errorBox: React.CSSProperties = {
  padding: '12px 16px',
  borderRadius: '8px',
  marginBottom: '20px',
  background: 'rgba(255,95,109,0.1)',
  border: '1px solid rgba(255,95,109,0.25)',
  color: '#ff5f6d',
  fontSize: '13px',
  display: 'flex',
  gap: '8px',
  alignItems: 'center',
};

const td: React.CSSProperties = {
  padding: '12px 14px',
  fontSize: '13px',
  verticalAlign: 'middle',
};

const sectionLabel: React.CSSProperties = {
  fontSize: '11px',
  fontFamily: 'var(--font-display)',
  fontWeight: 700,
  color: 'var(--text-secondary)',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
  marginBottom: '10px',
};