type Severity = 'low' | 'medium' | 'high' | 'critical';

const COLORS: Record<Severity, { bg: string; color: string; border: string }> = {
  low:      { bg: 'rgba(195,253,184,0.1)',  color: '#C3FDB8', border: 'rgba(195,253,184,0.3)' },
  medium:   { bg: 'rgba(255,169,77,0.12)',  color: '#ffa94d', border: 'rgba(255,169,77,0.3)' },
  high:     { bg: 'rgba(255,95,109,0.12)',  color: '#ff5f6d', border: 'rgba(255,95,109,0.3)' },
  critical: { bg: 'rgba(255,0,60,0.15)',    color: '#ff3055', border: 'rgba(255,0,60,0.4)' },
};

export default function SeverityBadge({ severity }: { severity: string }) {
  const s = (severity as Severity) in COLORS ? (severity as Severity) : 'low';
  const c = COLORS[s];
  return (
    <span style={{
      display: 'inline-flex',
      alignItems: 'center',
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
    }}>
      {severity}
    </span>
  );
}
