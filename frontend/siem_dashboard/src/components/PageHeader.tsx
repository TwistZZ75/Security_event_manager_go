interface PageHeaderProps {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
}

export default function PageHeader({ title, subtitle, action }: PageHeaderProps) {
  return (
    <div style={{
      padding: '28px 32px 0',
      display: 'flex', alignItems: 'flex-end',
      justifyContent: 'space-between', marginBottom: '28px',
    }}>
      <div>
        <h1 style={{
          fontFamily: 'var(--font-display)', fontWeight: 800,
          fontSize: '26px', letterSpacing: '-0.01em',
          lineHeight: 1.1, marginBottom: subtitle ? '5px' : 0,
        }}>
          {title}
        </h1>
        {subtitle && (
          <p style={{ fontSize: '13px', color: 'var(--text-secondary)' }}>{subtitle}</p>
        )}
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}
