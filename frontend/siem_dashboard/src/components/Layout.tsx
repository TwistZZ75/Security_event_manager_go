import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import {
  LayoutDashboard,
  Bell,
  Zap,
  BookOpen,
  LogOut,
  Shield,
  ChevronRight,
  Activity,
} from 'lucide-react';

const NAV = [
  { to: '/',        label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/alerts',  label: 'Alerts',    icon: Bell },
  { to: '/events',  label: 'Events',    icon: Activity },
  { to: '/actions', label: 'Actions',   icon: Zap },
  { to: '/rules',   label: 'Rules',     icon: BookOpen },
];

export default function Layout() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <aside style={{
        width: '220px',
        minWidth: '220px',
        background: 'var(--navy-light)',
        borderRight: '1px solid var(--navy-border)',
        display: 'flex',
        flexDirection: 'column',
        position: 'sticky',
        top: 0,
        height: '100vh',
        zIndex: 100,
      }}>
        {/* Logo — clickable, goes to home */}
        <div
          onClick={() => navigate('/')}
          style={{
            padding: '24px 20px 20px',
            borderBottom: '1px solid var(--navy-border)',
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            cursor: 'pointer',
            userSelect: 'none',
          }}
        >
          <div style={{
            width: '34px',
            height: '34px',
            background: 'var(--mint)',
            borderRadius: '8px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}>
            <Shield size={18} color="var(--navy)" strokeWidth={2.5} />
          </div>
          <div>
            <div style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '15px', letterSpacing: '0.02em' }}>
              SIEM
            </div>
            <div style={{ fontSize: '10px', color: 'var(--text-secondary)', letterSpacing: '0.12em' }}>
              CONSOLE
            </div>
          </div>
        </div>

        {/* Nav */}
        <nav style={{ flex: 1, padding: '16px 10px' }}>
          {NAV.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              style={({ isActive }) => ({
                display: 'flex',
                alignItems: 'center',
                gap: '10px',
                padding: '10px 12px',
                borderRadius: '8px',
                marginBottom: '4px',
                textDecoration: 'none',
                color: isActive ? 'var(--navy)' : 'var(--mint)',
                background: isActive ? 'var(--mint)' : 'transparent',
                fontFamily: 'var(--font-display)',
                fontWeight: isActive ? 700 : 500,
                fontSize: '13px',
                letterSpacing: '0.02em',
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

        {/* Footer */}
        <div style={{ padding: '16px 10px', borderTop: '1px solid var(--navy-border)' }}>
          <button
            onClick={handleLogout}
            style={{
              width: '100%',
              display: 'flex',
              alignItems: 'center',
              gap: '10px',
              padding: '10px 12px',
              borderRadius: '8px',
              background: 'transparent',
              border: 'none',
              color: 'var(--text-secondary)',
              cursor: 'pointer',
              fontFamily: 'var(--font-display)',
              fontWeight: 500,
              fontSize: '13px',
              transition: 'color var(--transition), background var(--transition)',
            }}
            onMouseEnter={e => {
              (e.currentTarget as HTMLButtonElement).style.color = 'var(--red)';
              (e.currentTarget as HTMLButtonElement).style.background = 'rgba(255,95,109,0.08)';
            }}
            onMouseLeave={e => {
              (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-secondary)';
              (e.currentTarget as HTMLButtonElement).style.background = 'transparent';
            }}
          >
            <LogOut size={15} strokeWidth={2} />
            <span>Logout</span>
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
