import { useState, useEffect } from 'react';
import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell, PieChart, Pie } from 'recharts';
import { Activity, AlertTriangle, Download } from 'lucide-react';
import { getEvents } from '../api/api';
import { fmtDateShort, parseDate } from '../utils/date';
import type { NormalizedEvent } from '../types';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import PageHeader from '../components/PageHeader';
import PieLegend from '../components/PieLegend';
import JsonModal from '../components/JsonModal';
import AgentMiniModal from '../components/AgentMiniModal';

// Все возможные значения severity от парсеров (XML: "Info","Warning","Error","Critical"; syslog/auth: аналогично)
const SEV_COLORS: Record<string, string> = {
  // Capitalized (XML parser output)
  Critical: '#ff3055', Error: '#ff5f6d', Warning: '#ffa94d',
  Info: '#C3FDB8',     Verbose: '#888bac', Undefined: '#555577',
  // Lowercase
  critical: '#ff3055', error: '#ff5f6d', warning: '#ffa94d',
  info: '#C3FDB8',     verbose: '#888bac',
  // ALL CAPS
  CRITICAL: '#ff3055', ERROR: '#ff5f6d', WARNING: '#ffa94d', INFO: '#C3FDB8',
  // Suricata
  high: '#ff5f6d', medium: '#ffa94d', low: '#C3FDB8',
  High: '#ff5f6d', Medium: '#ffa94d', Low: '#C3FDB8',
  // Auth parser
  danger: '#ff3055', DANGER: '#ff3055',
};
const CAT_COLORS = ['#C3FDB8', '#a0e095', '#64b4ff', '#ffa94d', '#b47cff', '#ff5f6d'];
const LIMIT_OPTIONS = [100, 500, 1000, 5000, 10000];

export default function EventsPage() {
  const [events, setEvents] = useState<NormalizedEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<NormalizedEvent | null>(null);
  const [limit, setLimit] = useState<number>(10000);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [agentHostname, setAgentHostname] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    getEvents(limit)
      .then(data => {
        setEvents(data ?? []);
        setTotalCount(data?.length ?? 0);
      })
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [limit]);

  const categories: Record<string, number> = {};
  const severities: Record<string, number> = {};
  events.forEach(e => {
    categories[e.event_category] = (categories[e.event_category] || 0) + 1;
    severities[e.severity] = (severities[e.severity] || 0) + 1;
  });

  const catData = Object.entries(categories)
    .sort((a, b) => b[1] - a[1]).slice(0, 8)
    .map(([name, value]) => ({ name: name.length > 20 ? name.slice(0, 18) + '…' : name, value }));

  const sevData = Object.entries(severities).map(([name, value]) => ({
    name, value, color: SEV_COLORS[name] || '#888bac',
  }));

  const columns = [
    { key: 'timestamp', header: 'Time', width: '110px',
      render: (e: NormalizedEvent) => <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '11px' }}>{fmtDateShort(e.timestamp)}</span>,
      sortValue: (e: NormalizedEvent) => parseDate(e.timestamp) },
    { key: 'severity', header: 'Sev', width: '80px',
      sortValue: (row: NormalizedEvent) => ['Error','Warning','Info'].indexOf(row.severity),
      render: (e: NormalizedEvent) => (
        <span style={{ fontSize: '10px', fontFamily: 'var(--font-display)', fontWeight: 700, padding: '2px 8px', borderRadius: '10px', textTransform: 'uppercase', color: SEV_COLORS[e.severity] || 'var(--text-secondary)', background: (SEV_COLORS[e.severity] || '#888') + '18' }}>
          {e.severity}
        </span>
      ) },
    { key: 'pc_name', header: 'Host', width: '120px', sortValue: (e: NormalizedEvent) => e.pc_name,
      render: (e: NormalizedEvent) => (
        <span
          onClick={ev => { ev.stopPropagation(); setAgentHostname(e.pc_name); }}
          title="Click to view agent info"
          style={{
            fontFamily: 'var(--font-mono)', fontSize: '12px', fontWeight: 700,
            color: 'var(--mint)', cursor: 'pointer',
            textDecoration: 'underline dotted', textUnderlineOffset: '3px',
          }}
        >{e.pc_name}</span>
      ) },
    { key: 'username', header: 'User', width: '90px',
      render: (e: NormalizedEvent) => <span style={{ fontFamily: 'var(--font-mono)', fontSize: '12px', color: 'var(--text-secondary)' }}>{e.username || '—'}</span> },
    { key: 'category', header: 'Category', width: '150px', sortValue: (e: NormalizedEvent) => e.event_category,
      render: (e: NormalizedEvent) => <span style={{ fontSize: '11px', color: 'var(--mint)', background: 'var(--mint-glow)', padding: '2px 7px', borderRadius: '4px' }}>{e.event_category}</span> },
    { key: 'event_description', header: 'Description', width: 'auto',
      render: (e: NormalizedEvent) => <span style={{ display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', color: 'var(--text-secondary)', fontSize: '12px',
      }}>{e.event_description.length > 80 ? e.event_description.slice(0, 78) + '…' : e.event_description}</span> },
    { key: 'process_name', header: 'Process', width: '150px',
      render: (e: NormalizedEvent) => <span style={{ fontFamily: 'var(--font-mono)', fontSize: '11px', color: 'var(--text-secondary)', display: 'block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',}}>{e.process_name || '—'}</span> },
    { key: 'source', header: 'Source', width: '120px',sortValue: (e: NormalizedEvent) => e.source,
      render: (e: NormalizedEvent) => <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>{e.source}</span> },
  ];

  const customTooltip = ({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; payload: { color?: string } }> }) => {
    if (!active || !payload?.length) return null;
    return (
      <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '8px', padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
        <div style={{ color: payload[0].payload.color || 'var(--mint)', fontWeight: 700 }}>{payload[0].name}</div>
        <div>{payload[0].value} events</div>
      </div>
    );
  };

  const sortedEvents = [...events].sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader title="Events" subtitle="Normalized security events from all agents" />
      <div style={{ padding: '0 32px' }}>
        {error && (
          <div style={{ padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' }}>
            <AlertTriangle size={14} /> {error}
          </div>
        )}

        {loading ? <Spinner /> : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Total Events"  value={events.length}                                  icon={<Activity size={18} />} accent="#C3FDB8" />
              <StatCard label="Categories"    value={Object.keys(categories).length}                 icon={<Activity size={18} />} accent="#64b4ff" />
              <StatCard label="Sources"       value={Object.keys({ ...Object.fromEntries(events.map(e => [e.source, 1])) }).length} icon={<Activity size={18} />} accent="#ffa94d" />
              <StatCard label="Unique Hosts"  value={new Set(events.map(e => e.pc_name)).size}       icon={<Activity size={18} />} accent="#b47cff" sub="hosts" />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.6fr', gap: '20px', marginBottom: '28px' }}>
              <div style={card}>
                <h3 style={cardTitle}>By Severity</h3>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <div style={{ flex: '0 0 150px' }}>
                    {sevData.length === 0 ? <EmptyChart /> : (
                      <ResponsiveContainer width="100%" height={150}>
                        <PieChart>
                          <Pie data={sevData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={40} outerRadius={64} paddingAngle={3}>
                            {sevData.map((e, i) => <Cell key={i} fill={e.color} stroke="transparent" />)}
                          </Pie>
                          <Tooltip content={({ active, payload }) => {
                            if (!active || !payload?.length) return null;
                            const p = payload[0].payload as { name: string; value: number; color: string };
                            return <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '8px', padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: '12px' }}><div style={{ color: p.color, fontWeight: 700 }}>{p.name}</div><div>{p.value}</div></div>;
                          }} />
                        </PieChart>
                      </ResponsiveContainer>
                    )}
                  </div>
                  <div style={{ flex: 1, minWidth: '90px' }}>
                    <PieLegend items={sevData} />
                  </div>
                </div>
              </div>

              <div style={card}>
                <h3 style={cardTitle}>Top Categories</h3>
                {catData.length === 0 ? <EmptyChart /> : (
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={catData} layout="vertical" margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                      <XAxis type="number" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} allowDecimals={false} />
                      <YAxis type="category" dataKey="name" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} width={120} />
                      <Tooltip content={customTooltip as React.FC} />
                      <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                        {catData.map((_, i) => <Cell key={i} fill={CAT_COLORS[i % CAT_COLORS.length]} />)}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </div>
            </div>

            <div>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '16px', flexWrap: 'wrap', gap: '8px' }}>
                <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '16px', display: 'flex', alignItems: 'center', gap: '8px', margin: 0 }}>
                  <Activity size={16} /> Events
                  <span style={{ fontSize: '11px', fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)', fontWeight: 400 }}>
                    ({events.length} uploaded)
                  </span>
                </h2>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Upload:</span>
                  <select
                    value={limit}
                    onChange={e => setLimit(Number(e.target.value))}
                    style={{
                      background: 'var(--navy-light)',
                      border: '1px solid var(--navy-border)',
                      color: 'var(--mint)',
                      borderRadius: '6px',
                      padding: '5px 10px',
                      fontSize: '12px',
                      fontFamily: 'var(--font-mono)',
                      cursor: 'pointer',
                      outline: 'none',
                    }}
                  >
                    {[100, 500, 1000, 5000, 10000].map(n => (
                      <option key={n} value={n}>{n.toLocaleString()} events</option>
                    ))}
                  </select>
                </div>
              </div>
              <PaginatedTable data={sortedEvents} columns={columns} keyFn={e => e.id} onRowClick={setSelected} />
            </div>
          </>
        )}
      </div>

      {selected && (
        <JsonModal
          title={selected.event_category || 'Event'}
          subtitle={
            <span style={{ fontSize: '10px', fontFamily: 'var(--font-display)', fontWeight: 700, padding: '2px 8px', borderRadius: '10px', textTransform: 'uppercase', color: SEV_COLORS[selected.severity] || 'var(--text-secondary)', background: (SEV_COLORS[selected.severity] || '#888') + '18' }}>
              {selected.severity}
            </span>
          }
          data={selected}
          onClose={() => setSelected(null)}
        />
      )}

      {agentHostname && (
        <AgentMiniModal
          hostname={agentHostname}
          onClose={() => setAgentHostname(null)}
        />
      )}
    </div>
  );
}

// Legacy fmt alias for modal subtitle usage
function fmt(s?: string) { return fmtDateShort(s); }
function EmptyChart() { return <div style={{ height: '150px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>No data</div>; }
function Spinner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Loading events...</span>
      <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
    </div>
  );
}
const card: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '12px', padding: '20px' };
const cardTitle: React.CSSProperties = { fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '14px', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em' };
