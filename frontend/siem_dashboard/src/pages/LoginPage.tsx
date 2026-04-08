import { useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { Shield, Eye, EyeOff, AlertCircle, UserPlus, LogIn } from 'lucide-react';

type Tab = 'login' | 'register';

export default function LoginPage() {
  const { login, register } = useAuth();
  const [tab, setTab]           = useState<Tab>('login');
  const [username, setUsername] = useState('');
  const [email, setEmail]       = useState('');
  const [password, setPassword] = useState('');
  const [showPw, setShowPw]     = useState(false);
  const [error, setError]       = useState('');
  const [loading, setLoading]   = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!username.trim() || !password.trim()) {
      setError('Заполните все обязательные поля');
      return;
    }
    if (tab === 'register' && !email.trim()) {
      setError('Email обязателен для регистрации');
      return;
    }

    setLoading(true);
    try {
      if (tab === 'login') {
        await login(username, password);
      } else {
        await register(username, email, password);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Произошла ошибка');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', background: 'var(--navy)',
      position: 'relative', overflow: 'hidden',
    }}>
      {/* Background grid */}
      <div style={{
        position: 'absolute', inset: 0,
        backgroundImage: `
          linear-gradient(rgba(195,253,184,0.04) 1px, transparent 1px),
          linear-gradient(90deg, rgba(195,253,184,0.04) 1px, transparent 1px)`,
        backgroundSize: '40px 40px', pointerEvents: 'none',
      }} />
      <div style={{
        position: 'absolute', top: '-200px', right: '-200px',
        width: '600px', height: '600px', borderRadius: '50%',
        background: 'radial-gradient(circle, rgba(195,253,184,0.06) 0%, transparent 70%)',
        pointerEvents: 'none',
      }} />

      {/* Card */}
      <div style={{
        margin: 'auto', width: '100%', maxWidth: '420px',
        padding: '20px', position: 'relative', zIndex: 1,
        animation: 'fadeIn 0.4s ease both',
      }}>
        {/* Logo */}
        <div style={{ textAlign: 'center', marginBottom: '32px' }}>
          <div style={{
            width: '56px', height: '56px', background: 'var(--mint)',
            borderRadius: '14px', display: 'flex', alignItems: 'center',
            justifyContent: 'center', margin: '0 auto 14px',
            boxShadow: '0 0 28px rgba(195,253,184,0.28)',
          }}>
            <Shield size={28} color="var(--navy)" strokeWidth={2.5} />
          </div>
          <h1 style={{
            fontFamily: 'var(--font-display)', fontWeight: 800,
            fontSize: '26px', letterSpacing: '-0.01em', marginBottom: '4px',
          }}>
            SIEM Console
          </h1>
          <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>
            Система управления безопасностью
          </p>
        </div>

        {/* Form card */}
        <div style={{
          background: 'var(--navy-light)', border: '1px solid var(--navy-border)',
          borderRadius: '16px', overflow: 'hidden',
        }}>
          {/* Tabs */}
          <div style={{ display: 'flex', borderBottom: '1px solid var(--navy-border)' }}>
            {(['login', 'register'] as Tab[]).map(t => (
              <button
                key={t}
                onClick={() => { setTab(t); setError(''); }}
                style={{
                  flex: 1, padding: '14px', border: 'none', cursor: 'pointer',
                  fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '13px',
                  background: tab === t ? 'var(--navy)' : 'transparent',
                  color: tab === t ? 'var(--mint)' : 'var(--text-secondary)',
                  borderBottom: tab === t ? '2px solid var(--mint)' : '2px solid transparent',
                  display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '7px',
                  transition: 'all var(--transition)',
                }}
              >
                {t === 'login' ? <LogIn size={14} /> : <UserPlus size={14} />}
                {t === 'login' ? 'Вход' : 'Регистрация'}
              </button>
            ))}
          </div>

          <form onSubmit={handleSubmit} style={{ padding: '28px' }}>
            {/* Username */}
            <div style={{ marginBottom: '16px' }}>
              <label style={labelStyle}>Логин</label>
              <input
                type="text" value={username} autoComplete="username"
                onChange={e => setUsername(e.target.value)} required
                placeholder="имя пользователя" style={inputStyle}
              />
            </div>

            {/* Email (only register) */}
            {tab === 'register' && (
              <div style={{ marginBottom: '16px' }}>
                <label style={labelStyle}>Email</label>
                <input
                  type="email" value={email} autoComplete="email"
                  onChange={e => setEmail(e.target.value)} required
                  placeholder="user@example.com" style={inputStyle}
                />
              </div>
            )}

            {/* Password */}
            <div style={{ marginBottom: '24px' }}>
              <label style={labelStyle}>Пароль</label>
              <div style={{ position: 'relative' }}>
                <input
                  type={showPw ? 'text' : 'password'} value={password}
                  autoComplete={tab === 'login' ? 'current-password' : 'new-password'}
                  onChange={e => setPassword(e.target.value)} required
                  placeholder={tab === 'register' ? 'минимум 8 символов' : '••••••••'}
                  style={{ ...inputStyle, paddingRight: '44px' }}
                />
                <button
                  type="button" onClick={() => setShowPw(v => !v)}
                  style={{
                    position: 'absolute', right: '12px', top: '50%',
                    transform: 'translateY(-50%)', background: 'none', border: 'none',
                    color: 'var(--text-secondary)', cursor: 'pointer', padding: '2px', display: 'flex',
                  }}
                >
                  {showPw ? <EyeOff size={15} /> : <Eye size={15} />}
                </button>
              </div>
            </div>

            {error && (
              <div style={{
                display: 'flex', alignItems: 'center', gap: '8px',
                padding: '10px 14px', borderRadius: '8px', marginBottom: '18px',
                background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)',
                color: '#ff5f6d', fontSize: '13px',
              }}>
                <AlertCircle size={14} /> {error}
              </div>
            )}

            {tab === 'register' && (
              <div style={{
                padding: '10px 14px', borderRadius: '8px', marginBottom: '18px',
                background: 'rgba(195,253,184,0.07)', border: '1px solid rgba(195,253,184,0.15)',
                color: 'var(--text-secondary)', fontSize: '12px', lineHeight: '1.5',
              }}>
                Первый зарегистрированный пользователь автоматически получает роль{' '}
                <span style={{ color: 'var(--mint)', fontWeight: 700 }}>администратора</span>.
              </div>
            )}

            <button
              type="submit" disabled={loading}
              style={{
                width: '100%', padding: '12px', borderRadius: '10px', border: 'none',
                background: loading ? 'var(--mint-dim)' : 'var(--mint)',
                color: 'var(--navy)',
                fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '14px',
                letterSpacing: '0.04em', cursor: loading ? 'not-allowed' : 'pointer',
                transition: 'background var(--transition)',
              }}
            >
              {loading
                ? (tab === 'login' ? 'Входим...' : 'Регистрируемся...')
                : (tab === 'login' ? 'Войти' : 'Зарегистрироваться')}
            </button>
          </form>
        </div>

        <div style={{ textAlign: 'center', marginTop: '20px', fontSize: '11px', color: 'var(--navy-border)' }}>
          v0.1.0-dev · Несанкционированный доступ запрещён
        </div>
      </div>
    </div>
  );
}

const labelStyle: React.CSSProperties = {
  display: 'block', marginBottom: '7px',
  fontSize: '11px', fontFamily: 'var(--font-display)', fontWeight: 700,
  letterSpacing: '0.1em', color: 'var(--text-secondary)', textTransform: 'uppercase',
};

const inputStyle: React.CSSProperties = {
  width: '100%', padding: '11px 14px',
  background: 'var(--navy)', border: '1px solid var(--navy-border)',
  borderRadius: '8px', color: 'var(--mint)',
  fontFamily: 'var(--font-mono)', fontSize: '13px', outline: 'none',
  transition: 'border-color var(--transition)', boxSizing: 'border-box',
};
