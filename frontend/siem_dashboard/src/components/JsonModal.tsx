import { useEffect, useRef } from 'react';
import { X } from 'lucide-react';

interface JsonModalProps {
  title: string;
  subtitle?: React.ReactNode;
  data: unknown;
  onClose: () => void;
  footer?: React.ReactNode;
}

export default function JsonModal({ title, subtitle, data, onClose, footer }: JsonModalProps) {
  // Сохраняем позицию скролла до открытия
  const savedScrollY = useRef(0);
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Запоминаем позицию и сразу скроллим оверлей в начало
    savedScrollY.current = window.scrollY;
    // Фокус на оверлей — чтобы не нужно было мотать страницу вверх
    overlayRef.current?.focus();

    // Блокируем скролл страницы позади модала
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    return () => {
      document.body.style.overflow = prev;
      // Восстанавливаем позицию после закрытия
      window.scrollTo({ top: savedScrollY.current, behavior: 'instant' });
    };
  }, []);

  // Закрытие по Escape
  useEffect(() => {
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [onClose]);

  return (
    <div
      ref={overlayRef}
      tabIndex={-1}
      onClick={e => { if (e.target === e.currentTarget) onClose(); }}
      style={{
        position: 'fixed', inset: 0,
        background: 'rgba(10,14,30,0.85)',
        zIndex: 1000,
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
          maxWidth: '720px',
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
        }}
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div style={{
          padding: '18px 22px',
          borderBottom: '1px solid var(--navy-border)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: '12px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
            <span style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '15px' }}>{title}</span>
            {subtitle}
          </div>
          <button onClick={onClose} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', display: 'flex', flexShrink: 0, padding: '4px' }}>
            <X size={18} />
          </button>
        </div>

        {/* JSON body */}
        <div style={{ maxHeight: '55vh', overflowY: 'auto' }}>
          <pre style={{
            padding: '20px', margin: 0,
            fontFamily: 'var(--font-mono)', fontSize: '12px',
            lineHeight: '1.7', color: 'var(--mint)',
            background: 'var(--navy)', overflowX: 'auto',
          }}>
            {JSON.stringify(data, null, 2)}
          </pre>
        </div>

        {/* Optional footer */}
        {footer && (
          <div style={{ borderTop: '1px solid var(--navy-border)' }}>
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
