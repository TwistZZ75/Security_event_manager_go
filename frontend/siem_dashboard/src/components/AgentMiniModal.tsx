import { useState, useEffect } from 'react';
import { Monitor, Wifi, WifiOff, X, Clock, Tag } from 'lucide-react';
import { getAgent } from '../api/api';
import type { Agent } from '../types';

interface AgentMiniModalProps {
  hostname: string;
  onClose: () => void;
}

export default function AgentMiniModal({ hostname, onClose }: AgentMiniModalProps) {
  const [agent, setAgent] = useState<Agent | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getAgent(hostname)
      .then(setAgent)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));

    // Escape to close
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', handler);
    // Block body scroll
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', handler);
      document.body.style.overflow = prev;
    };
  }, [hostname, onClose]);

  const isOnline = agent ? (Date.now() - new Date(agent.last_seen).getTime()) < 2 * 60 * 1000 : false;

  return (
    <div
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
      style={{
        position: 'fixed', inset: 0,
        background: 'rgba(10,14,30,0.85)',
        zIndex: 1100,
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'center',
        padding: '40px 20px',
        overflowY: 'auto',
        outline: 'none',
      }}
    >
      <div
        style={{
          background: 'var(--navy-light)',
          border: '1px solid var(--navy-border)',
          borderRadius: '16px',
          width: '100%',
          maxWidth: '520px',
          overflow: 'hidden',
        }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{
          padding: '16px 20px',
          borderBottom: '1px solid var(--navy-border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Monitor size={16} style={{ color: 'var(--mint)' }} />
            <span style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '14px' }}>{hostname}</span>
          </div>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', padding: '4px' }}>
            <X size={16} />
          </button>
        </div>

        {/* Body */}
        <div style={{ padding: '20px' }}>
          {loading && <div style={{ color: 'var(--text-secondary)', fontSize: '13px', textAlign: 'center', padding: '20px 0' }}>Loading agent info…</div>}
          {error && <div style={{ color: '#ff5f6d', fontSize: '13px', textAlign: 'center', padding: '20px 0' }}>Agent not found: {error}</div>}
          {agent && (
            <>
              {/* Status badge */}
              <div style={{ marginBottom: '18px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                {isOnline
                  ? <><Wifi size={14} style={{ color: 'var(--mint)' }} /><span style={{ color: 'var(--mint)', fontWeight: 700, fontSize: '13px' }}>Online</span></>
                  : <><WifiOff size={14} style={{ color: '#888bac' }} /><span style={{ color: '#888bac', fontWeight: 700, fontSize: '13px' }}>Offline</span></>
                }
              </div>

              {/* Fields */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
                {[
                  ['OS', agent.os],
                  ['OS Version', agent.os_version],
                  ['Agent Version', agent.agent_version],
                  ['IP Address', agent.ip_address],
                  ['Agent ID', agent.agent_id],
                  ['Registered', fmt(agent.registered_at)],
                  ['Last Seen', fmt(agent.last_seen)],
                ].map(([label, value]) => (
                  <div key={label} style={{ background: 'var(--navy)', borderRadius: '8px', padding: '10px 12px' }}>
                    <div style={{ fontSize: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-display)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.07em', marginBottom: '4px' }}>{label}</div>
                    <div style={{ fontSize: '12px', fontFamily: 'var(--font-mono)', color: 'var(--mint)', wordBreak: 'break-all' }}>{value || '—'}</div>
                  </div>
                ))}
              </div>

              {/* Metadata */}
              {agent.metadata && Object.keys(agent.metadata).length > 0 && (
                <div style={{ marginTop: '14px' }}>
                  <div style={{ fontSize: '10px', color: 'var(--text-secondary)', fontFamily: 'var(--font-display)', fontWeight: 700, textTransform: 'uppercase', letterSpacing: '0.07em', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '5px' }}>
                    <Tag size={10} /> Metadata
                  </div>
                  <div style={{ background: 'var(--navy)', borderRadius: '8px', padding: '10px 12px', display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {Object.entries(agent.metadata).map(([k, v]) => (
                      <span key={k} style={{ fontSize: '11px', fontFamily: 'var(--font-mono)', background: 'var(--mint-glow)', color: 'var(--mint)', padding: '2px 7px', borderRadius: '4px' }}>
                        {k}: {v}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div style={{ padding: '12px 20px', borderTop: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-secondary)', fontSize: '11px' }}>
          <Clock size={11} /> Last seen: {agent ? fmt(agent.last_seen) : '…'}
        </div>
      </div>
    </div>
  );
}

function fmt(s?: string) {
  if (!s) return '—';
  const d = new Date(s);
  return isNaN(d.getTime()) ? '—' : d.toLocaleString();
}
