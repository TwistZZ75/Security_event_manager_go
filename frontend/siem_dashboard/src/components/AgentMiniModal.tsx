import { useState, useEffect, useRef } from 'react';
import { Monitor, Wifi, WifiOff, X, Clock, Tag, AlertTriangle } from 'lucide-react';
import { getAgent } from '../api/api';
import type { Agent } from '../types';
import { fmtDate } from '../utils/date';

interface AgentMiniModalProps {
  hostname: string;
  onClose: () => void;
}

export default function AgentMiniModal({ hostname, onClose }: AgentMiniModalProps) {
  const [agent, setAgent]     = useState<Agent | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError]     = useState('');
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setLoading(true);
    setError('');
    setAgent(null);

    const tryLoad = async () => {
      try {
        const data = await getAgent(hostname);
        setAgent(data);
      } catch {
        const short = hostname.split('.')[0];
        if (short && short !== hostname) {
          try { const data = await getAgent(short); setAgent(data); return; } catch {}
        }
        setError(`Агент "${hostname}" не найден в системе`);
      } finally {
        setLoading(false);
      }
    };
    tryLoad();

    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', handler);
    overlayRef.current?.focus();
    return () => document.removeEventListener('keydown', handler);
  }, [hostname, onClose]);

  const isOnline = agent
    ? (Date.now() - new Date(agent.LastSeen.replace(/([Zz]|[+-]\d{2}:?\d{2})$/, '')).getTime()) < 2 * 60 * 1000
    : false;

  const fields: [string, string][] = agent ? [
    ['Имя хоста',      agent.Hostname       || '—'],
    ['ID агента',      agent.AgentID        || '—'],
    ['ОС',             agent.OS             || '—'],
    ['Версия ОС',      agent.OSVersion      || '—'],
    ['Версия агента',  agent.AgentVersion   || '—'],
    ['IP-адрес',       agent.IPAddress      || '—'],
    ['Зарегистрирован', fmtDate(agent.RegisteredAt)],
    ['Последняя связь', fmtDate(agent.LastSeen)],
  ] : [];

  return (
    <div ref={overlayRef} tabIndex={-1}
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
      style={{ position: 'fixed', inset: 0, background: 'rgba(10,14,30,0.82)', zIndex: 1100, display: 'flex', alignItems: 'flex-start', justifyContent: 'center', padding: '60px 20px 40px', overflowY: 'auto', outline: 'none' }}>
      <div onClick={e => e.stopPropagation()}
        style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '16px', width: '100%', maxWidth: '520px', overflow: 'hidden' }}>
        {/* Шапка */}
        <div style={{ padding: '16px 20px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Monitor size={16} style={{ color: 'var(--mint)' }} />
            <span style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '14px' }}>{hostname}</span>
            {agent && (
              <span style={{ display: 'flex', alignItems: 'center', gap: '4px', fontSize: '11px', fontWeight: 700, color: isOnline ? 'var(--mint)' : '#888bac' }}>
                {isOnline ? <Wifi size={11} /> : <WifiOff size={11} />}
                {isOnline ? 'В сети' : 'Оффлайн'}
              </span>
            )}
          </div>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', padding: '4px' }}><X size={16} /></button>
        </div>

        {/* Тело */}
        <div style={{ padding: '20px' }}>
          {loading && (
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '120px', flexDirection: 'column', gap: '10px' }}>
              <div style={{ width: '28px', height: '28px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
              <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Загрузка данных агента...</span>
              <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
            </div>
          )}

          {error && !loading && (
            <div style={{ display: 'flex', alignItems: 'flex-start', gap: '10px', padding: '14px', borderRadius: '10px', background: 'rgba(255,95,109,0.08)', border: '1px solid rgba(255,95,109,0.2)', color: '#ff5f6d', fontSize: '13px' }}>
              <AlertTriangle size={16} style={{ flexShrink: 0, marginTop: '1px' }} />
              <div>
                <div style={{ fontWeight: 700, marginBottom: '4px' }}>{error}</div>
                <div style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>Убедитесь, что агент зарегистрирован и hostname совпадает с pc_name в событии.</div>
              </div>
            </div>
          )}

          {agent && !loading && (
            <>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '10px', marginBottom: '14px' }}>
                {fields.map(([label, value]) => (
                  <div key={label} style={{ background: 'var(--navy)', borderRadius: '8px', padding: '10px 12px' }}>
                    <div style={{ fontSize: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-display)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.07em', marginBottom: '4px' }}>{label}</div>
                    <div style={{ fontSize: '12px', fontFamily: 'var(--font-mono)', color: 'var(--mint)', wordBreak: 'break-all' }}>{value}</div>
                  </div>
                ))}
              </div>

              {agent.Metadata && Object.keys(agent.Metadata).length > 0 && (
                <div>
                  <div style={{ fontSize: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-display)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.07em', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '5px' }}>
                    <Tag size={10} /> Метаданные
                  </div>
                  <div style={{ background: 'var(--navy)', borderRadius: '8px', padding: '10px 12px', display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {Object.entries(agent.Metadata).map(([k, v]) => (
                      <span key={k} style={{ fontSize: '11px', fontFamily: 'var(--font-mono)', background: 'var(--mint-glow)', color: 'var(--mint)', padding: '2px 7px', borderRadius: '4px' }}>{k}: {v}</span>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Подвал */}
        <div style={{ padding: '10px 20px', borderTop: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-secondary)', fontSize: '11px', fontFamily: 'var(--font-mono)' }}>
          <Clock size={11} />
          Последняя связь: {agent ? fmtDate(agent.LastSeen) : '—'}
        </div>
      </div>
    </div>
  );
}
