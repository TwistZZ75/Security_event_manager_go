import { X } from 'lucide-react';

interface JsonModalProps {
  title: string;
  subtitle?: React.ReactNode;
  data: unknown;
  onClose: () => void;
  footer?: React.ReactNode;
}

export default function JsonModal({ title, subtitle, data, onClose, footer }: JsonModalProps) {
  return (
    <div
      style={{ position: 'fixed', inset: 0, background: 'rgba(10,14,30,0.85)', zIndex: 1000, display: 'flex', alignItems: 'flex-start', justifyContent: 'center', padding: '40px 20px', overflowY: 'auto' }}
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '16px', width: '100%', maxWidth: '720px', overflow: 'hidden', display: 'flex', flexDirection: 'column', animation: 'fadeIn 0.2s ease' }}>
        {/* Header */}
        <div style={{ padding: '18px 22px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
            <span style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '15px' }}>{title}</span>
            {subtitle}
          </div>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', display: 'flex', flexShrink: 0 }}>
            <X size={18} />
          </button>
        </div>

        {/* JSON body */}
        <div style={{ maxHeight: '50vh', overflowY: 'auto' }}>
          <pre style={{ padding: '20px', margin: 0, fontFamily: 'var(--font-mono)', fontSize: '12px', lineHeight: '1.7', color: 'var(--mint)', background: 'var(--navy)', overflowX: 'auto' }}>
            {JSON.stringify(data, null, 2)}
          </pre>
        </div>

        {/* Optional footer (e.g. status form) */}
        {footer && (
          <div style={{ borderTop: '1px solid var(--navy-border)' }}>
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
