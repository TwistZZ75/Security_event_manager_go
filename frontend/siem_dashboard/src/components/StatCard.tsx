interface StatCardProps {
  label: string;
  value: number | string;
  sub?: string;
  accent?: string;
  icon?: React.ReactNode;
}

export default function StatCard({ label, value, sub, accent, icon }: StatCardProps) {
  return (
    <div style={{
      background: 'var(--navy-light)',
      border: `1px solid ${accent ? accent + '40' : 'var(--navy-border)'}`,
      borderRadius: '12px',
      padding: '20px 22px',
      display: 'flex',
      flexDirection: 'column',
      gap: '6px',
      position: 'relative',
      overflow: 'hidden',
    }}>
      {accent && (
        <div style={{
          position: 'absolute',
          top: 0, left: 0, right: 0,
          height: '3px',
          background: accent,
          borderRadius: '12px 12px 0 0',
        }} />
      )}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontSize: '11px', fontFamily: 'var(--font-display)', fontWeight: 600, letterSpacing: '0.1em', color: 'var(--text-secondary)', textTransform: 'uppercase' }}>
          {label}
        </span>
        {icon && (
          <span style={{ color: accent || 'var(--mint)', opacity: 0.7 }}>{icon}</span>
        )}
      </div>
      <div style={{
        fontSize: '32px',
        fontFamily: 'var(--font-display)',
        fontWeight: 800,
        color: accent || 'var(--mint)',
        lineHeight: 1,
      }}>
        {value}
      </div>
      {sub && (
        <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
          {sub}
        </div>
      )}
    </div>
  );
}
