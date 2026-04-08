const STATUS_CONFIG: Record<string, { bg: string; color: string; dot: string; label: string }> = {
  online:         { bg: 'rgba(195,253,184,0.1)', color: '#C3FDB8', dot: '#C3FDB8', label: 'В сети' },
  offline:        { bg: 'rgba(100,100,140,0.1)', color: '#888bac', dot: '#888bac', label: 'Оффлайн' },
  error:          { bg: 'rgba(255,95,109,0.12)', color: '#ff5f6d', dot: '#ff5f6d', label: 'Ошибка' },
  open:           { bg: 'rgba(255,95,109,0.1)',  color: '#ff5f6d', dot: '#ff5f6d', label: 'Открыт' },
  acknowledged:   { bg: 'rgba(255,169,77,0.1)',  color: '#ffa94d', dot: '#ffa94d', label: 'Принят' },
  resolved:       { bg: 'rgba(195,253,184,0.1)', color: '#C3FDB8', dot: '#C3FDB8', label: 'Закрыт' },
  false_positive: { bg: 'rgba(100,100,140,0.1)', color: '#888bac', dot: '#888bac', label: 'Ложная тревога' },
  pending:        { bg: 'rgba(255,224,102,0.1)', color: '#ffe066', dot: '#ffe066', label: 'Ожидание' },
  running:        { bg: 'rgba(100,180,255,0.1)', color: '#64b4ff', dot: '#64b4ff', label: 'Выполняется' },
  success:        { bg: 'rgba(195,253,184,0.1)', color: '#C3FDB8', dot: '#C3FDB8', label: 'Успешно' },
  failed:         { bg: 'rgba(255,95,109,0.1)',  color: '#ff5f6d', dot: '#ff5f6d', label: 'Ошибка' },
  enabled:        { bg: 'rgba(195,253,184,0.1)', color: '#C3FDB8', dot: '#C3FDB8', label: 'Включено' },
  disabled:       { bg: 'rgba(100,100,140,0.1)', color: '#888bac', dot: '#888bac', label: 'Отключено' },
};

const pulse = new Set(['online', 'running']);

export default function StatusBadge({ status }: { status: string }) {
  const c = STATUS_CONFIG[status] ?? STATUS_CONFIG['offline'];
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '6px',
      padding: '3px 10px', borderRadius: '20px',
      fontSize: '11px', fontFamily: 'var(--font-mono)',
      background: c.bg, color: c.color,
    }}>
      <span style={{
        width: '6px', height: '6px', borderRadius: '50%',
        background: c.dot, display: 'inline-block',
        ...(pulse.has(status) ? {
          boxShadow: `0 0 0 2px ${c.dot}30`,
          animation: 'pulse-glow 2s infinite',
        } : {}),
      }} />
      {c.label}
    </span>
  );
}
