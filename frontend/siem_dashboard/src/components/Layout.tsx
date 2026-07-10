import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import {
  LayoutDashboard, Bell, Zap, BookOpen,
  LogOut, Shield, ChevronRight, Activity, User, Users,
  type LucideIcon,
} from 'lucide-react';
import type { UserRole } from '../types';

const NAV: Array<{
  to: string;
  label: string;
  icon: LucideIcon;
  end?: boolean;
  onlyRole?: UserRole;
}> = [
  { to: '/agents',  label: 'Агенты',       icon: LayoutDashboard, end: true },
  { to: '/alerts',  label: 'Оповещения',       icon: Bell },
  { to: '/events',  label: 'События',      icon: Activity },
  { to: '/actions', label: 'Действия',     icon: Zap },
  { to: '/rules',   label: 'Правила',      icon: BookOpen },
  { to: '/users',   label: 'Пользователи', icon: Users, onlyRole: 'admin' },
];

const ROLE_LABELS: Record<UserRole, string> = {
  admin:   'Администратор',
  analyst: 'Аналитик',
  viewer:  'Наблюдатель',
};

const ROLE_COLORS: Record<UserRole, string> = {
  admin:   '#ff5f6d',
  analyst: '#ffa94d',
  viewer:  'var(--text-secondary)',
};

export default function Layout() {
  const { logout, user } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const role = user?.role as UserRole | undefined;

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <aside style={{
        width: '220px', minWidth: '220px',
        background: 'var(--navy-light)', borderRight: '1px solid var(--navy-border)',
        display: 'flex', flexDirection: 'column',
        position: 'sticky', top: 0, height: '100vh', zIndex: 100,
      }}>
        {/* Logo */}
        <div
          onClick={() => navigate('/agents')}
          style={{
            padding: '22px 20px 18px', borderBottom: '1px solid var(--navy-border)',
            display: 'flex', alignItems: 'center', gap: '10px',
            cursor: 'pointer', userSelect: 'none',
          }}
        >
          <div style={{
            width: '34px', height: '34px', background: 'var(--mint)',
            borderRadius: '8px', display: 'flex', alignItems: 'center',
            justifyContent: 'center', flexShrink: 0,
          }}>
            <Shield size={18} color="var(--navy)" strokeWidth={2.5} />
          </div>
          <div>
            <div style={{
              fontFamily: 'var(--font-display)', fontWeight: 800,
              fontSize: '15px', letterSpacing: '0.02em',
            }}>
              КОНСОЛЬ
            </div>
            <div style={{ fontSize: '10px', color: 'var(--text-secondary)', letterSpacing: '0.12em' }}>
              Аналитика
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, padding: '14px 10px' }}>
          {NAV.filter(item => !item.onlyRole || role === item.onlyRole).map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to} to={to} end={end}
              style={({ isActive }) => ({
                display: 'flex', alignItems: 'center', gap: '10px',
                padding: '10px 12px', borderRadius: '8px', marginBottom: '2px',
                textDecoration: 'none',
                color: isActive ? 'var(--navy)' : 'var(--mint)',
                background: isActive ? 'var(--mint)' : 'transparent',
                fontFamily: 'var(--font-display)', fontWeight: isActive ? 700 : 500,
                fontSize: '13px', letterSpacing: '0.02em',
                transition: 'background var(--transition), color var(--transition)',
              })}
              className="nav-link"
            >
              {({ isActive }) => (
                <>
                  <Icon size={15} strokeWidth={isActive ? 2.5 : 2} />
                  <span style={{ flex: 1 }}>{label}</span>
                  {isActive && <ChevronRight size={12} strokeWidth={2.5} />}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        {/* User info */}
        {user && (
          <div style={{
            padding: '12px 14px', margin: '0 10px 8px',
            background: 'var(--navy)', borderRadius: '8px',
            border: '1px solid var(--navy-border)',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <div style={{
                width: '28px', height: '28px', borderRadius: '50%',
                background: 'var(--mint-glow)', border: '1px solid var(--navy-border)',
                display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0,
              }}>
                <User size={13} color="var(--mint)" />
              </div>
              <div style={{ minWidth: 0 }}>
                <div style={{
                  fontFamily: 'var(--font-display)', fontWeight: 700,
                  fontSize: '12px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                }}>
                  {user.username}
                </div>
                <div style={{
                  fontSize: '10px', fontWeight: 700,
                  color: role ? ROLE_COLORS[role] : 'var(--text-secondary)',
                }}>
                  {role ? ROLE_LABELS[role] : ''}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Logout */}
        <div style={{ padding: '0 10px 16px', borderTop: '1px solid var(--navy-border)', paddingTop: '10px' }}>
          <button
            onClick={handleLogout}
            style={{
              width: '100%', display: 'flex', alignItems: 'center', gap: '10px',
              padding: '10px 12px', borderRadius: '8px', background: 'transparent',
              border: 'none', color: 'var(--text-secondary)', cursor: 'pointer',
              fontFamily: 'var(--font-display)', fontWeight: 500, fontSize: '13px',
              transition: 'color var(--transition), background var(--transition)',
            }}
            onMouseEnter={e => {
              (e.currentTarget).style.color = 'var(--red)';
              (e.currentTarget).style.background = 'rgba(255,95,109,0.08)';
            }}
            onMouseLeave={e => {
              (e.currentTarget).style.color = 'var(--text-secondary)';
              (e.currentTarget).style.background = 'transparent';
            }}
          >
            <LogOut size={15} strokeWidth={2} />
            <span>Выйти</span>
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main style={{ flex: 1, overflow: 'auto', background: 'var(--navy)' }}>
        <Outlet />
      </main>

      <style>{`
        .nav-link:hover:not([aria-current="page"]) {
          background: var(--mint-glow) !important;
          color: var(--mint) !important;
        }
      `}</style>
    </div>
  );
}
