interface LegendItem {
  name: string;
  value: number;
  color: string;
}

interface PieLegendProps {
  items: LegendItem[];
}

export default function PieLegend({ items }: PieLegendProps) {
  const total = items.reduce((s, i) => s + i.value, 0);
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '7px', marginTop: '12px' }}>
      {items.map(item => (
        <div key={item.name} style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '12px' }}>
          <div style={{ width: '10px', height: '10px', borderRadius: '3px', background: item.color, flexShrink: 0 }} />
          <span style={{ flex: 1, color: 'var(--text-secondary)', textTransform: 'capitalize' }}>{item.name}</span>
          <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--mint)', fontWeight: 700 }}>{item.value}</span>
          {total > 0 && (
            <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--navy-border)', fontSize: '11px', minWidth: '36px', textAlign: 'right' }}>
              {Math.round((item.value / total) * 100)}%
            </span>
          )}
        </div>
      ))}
    </div>
  );
}
