import { useState, useEffect } from 'react';
import { PieChart, Pie, Cell, AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { Bell, AlertTriangle, CheckCircle, Clock, XCircle, RotateCcw } from 'lucide-react';
import { getAlerts, updateAlertStatus } from '../api/api';
import { useAuth } from '../contexts/AuthContext';
import type { Alert } from '../types';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import SeverityBadge from '../components/SeverityBadge';
import StatusBadge from '../components/StatusBadge';
import PageHeader from '../components/PageHeader';
import PieLegend from '../components/PieLegend';
import JsonModal from '../components/JsonModal';

const SEV_ORDER = ['critical', 'high', 'medium', 'low'];
const SEV_COLORS: Record<string, string> = { critical: '#ff3055', high: '#ff5f6d', medium: '#ffa94d', low: '#C3FDB8' };

export default function AlertsPage() {
  const { username } = useAuth();   
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<Alert | null>(null);

  // alert status form state
  const [statusChoice, setStatusChoice] = useState<string>('');
  const [notes, setNotes] = useState('');
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState('');

  const loadAlerts = () => {
    setLoading(true);
    getAlerts()
      .then(setAlerts)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => { loadAlerts(); }, []);

  const openModal = (a: Alert) => {
    setSelected(a);
    setStatusChoice('');
    setNotes('');
    setSendError('');
  };

  const closeModal = () => { setSelected(null); };

  const handleSend = async () => {
    if (!selected || !statusChoice) return;
    setSending(true);
    setSendError('');
    try {
      await updateAlertStatus(selected.id, statusChoice, notes);
      // update local state
      setAlerts(prev =>
        prev.map(a => a.id === selected.id ? { ...a, status: statusChoice as Alert['status'] } : a)
      );
      closeModal();
    } catch (e) {
      setSendError(e instanceof Error ? e.message : 'Failed to update');
    } finally {
      setSending(false);
    }
  };

  const critical = alerts.filter(a => a.severity === 'critical').length;
  const open     = alerts.filter(a => a.status === 'open').length;
  const resolved = alerts.filter(a => a.status === 'resolved').length + alerts.filter(a=>a.status === 'false_positive').length;

  const pieData = SEV_ORDER
    .map(s => ({ name: s, value: alerts.filter(a => a.severity === s).length, color: SEV_COLORS[s] }))
    .filter(d => d.value > 0);

  // last 24h hourly buckets
  const now = Date.now();
  const buckets: Record<number, number> = {};
  for (let i = 23; i >= 0; i--) buckets[i] = 0;
  alerts.forEach(a => {
    const diff = Math.floor((now - new Date(a.created_at).getTime()) / 3600000);
    if (diff >= 0 && diff < 24) buckets[23 - diff] = (buckets[23 - diff] || 0) + 1;
  });
  const areaData = Array.from({ length: 24 }, (_, i) => ({ h: `${i}h`, count: buckets[i] || 0 }));

  const sortedAlerts = [...alerts].sort((a, b) => {
    const si = (s: string) => SEV_ORDER.indexOf(s);
    if (si(a.severity) !== si(b.severity)) return si(a.severity) - si(b.severity);
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
  });

  const columns = [
    { key: 'id', header: 'ID', width: '60px',
      render: (a: Alert) => <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>#{a.id}</span> },
    { key: 'severity', header: 'Severity', width: '110px',
      render: (a: Alert) => <SeverityBadge severity={a.severity} /> },
    { key: 'title', header: 'Title',
      render: (a: Alert) => (
        <div>
          <div style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '13px' }}>{a.title}</div>
          <div style={{ color: 'var(--text-secondary)', fontSize: '11px', marginTop: '2px' }}>{a.rule_name}</div>
        </div>
      ) },
    { key: 'status', header: 'Status', width: '130px',
      render: (a: Alert) => <StatusBadge status={a.status} /> },
    { key: 'created_at', header: 'Created',
      render: (a: Alert) => <span style={{ color: 'var(--text-secondary)', fontSize: '11px' }}>{fmt(a.created_at)}</span> },
  ];

  const customTooltip = ({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; payload: { color?: string } }> }) => {
    if (!active || !payload?.length) return null;
    return (
      <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '8px', padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
        <div style={{ color: payload[0].payload.color || 'var(--mint)', fontWeight: 700 }}>{payload[0].name}</div>
        <div>{payload[0].value}</div>
      </div>
    );
  };

  const STATUS_OPTIONS = [
    { value: 'open',          label: 'Close (reopen as open)' },
    { value: 'acknowledged',  label: 'Acknowledge' },
    { value: 'resolved',      label: 'Mark as Resolved' },
    { value: 'false_positive',label: 'Mark as False Positive' },
  ];

  const alertModalFooter = selected && (
    <div style={{ padding: '20px 22px' }}>
      {/* Radio buttons */}
      <div style={{ marginBottom: '14px' }}>
        <div style={{ fontSize: '11px', fontFamily: 'var(--font-display)', fontWeight: 700, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: '10px' }}>
          Change Status
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {STATUS_OPTIONS.map(opt => (
            <label key={opt.value} style={{ display: 'flex', alignItems: 'center', gap: '9px', cursor: 'pointer', fontSize: '13px', fontFamily: 'var(--font-display)' }}>
              <input
                type="radio"
                name="alert_status"
                value={opt.value}
                checked={statusChoice === opt.value}
                onChange={() => setStatusChoice(opt.value)}
                style={{ accentColor: 'var(--mint)', width: '15px', height: '15px', cursor: 'pointer' }}
              />
              <span style={{ color: statusChoice === opt.value ? 'var(--mint)' : 'var(--text-secondary)' }}>
                {opt.label}
              </span>
            </label>
          ))}
        </div>
      </div>

      {/* Notes */}
      <div style={{ marginBottom: '14px' }}>
        <div style={{ fontSize: '11px', fontFamily: 'var(--font-display)', fontWeight: 700, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: '8px' }}>
          Notes
        </div>
        <textarea
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes about this status change..."
          rows={3}
          style={{
            width: '100%', boxSizing: 'border-box',
            background: 'var(--navy)', border: '1px solid var(--navy-border)',
            borderRadius: '8px', color: 'var(--mint)', fontFamily: 'var(--font-mono)',
            fontSize: '12px', padding: '10px 12px', resize: 'vertical',
            outline: 'none', lineHeight: '1.5',
          }}
        />
      </div>

      {sendError && (
        <div style={{ color: '#ff5f6d', fontSize: '12px', marginBottom: '10px', display: 'flex', gap: '6px', alignItems: 'center' }}>
          <AlertTriangle size={12} /> {sendError}
        </div>
      )}

      {/* Send button */}
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button
          onClick={handleSend}
          disabled={!statusChoice || sending}
          style={{
            display: 'flex', alignItems: 'center', gap: '7px',
            padding: '9px 20px', borderRadius: '9px', border: 'none',
            background: (!statusChoice || sending) ? 'var(--navy-border)' : 'var(--mint)',
            color: (!statusChoice || sending) ? 'var(--text-secondary)' : 'var(--navy)',
            fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '13px',
            cursor: (!statusChoice || sending) ? 'not-allowed' : 'pointer',
            transition: 'background var(--transition)',
          }}
        >
          {sending ? 'Sending...' : 'Send'}
        </button>
      </div>
    </div>
  );

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader title="Alerts" subtitle="Security alerts and incidents" />
      <div style={{ padding: '0 32px' }}>
        {error && <ErrorBox msg={error} />}
        {loading ? <Spinner label="alerts" /> : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Total"    value={alerts.length} icon={<Bell size={18} />}        accent="#C3FDB8" />
              <StatCard label="Critical" value={critical}      icon={<AlertTriangle size={18} />} accent="#ff3055" />
              <StatCard label="Open"     value={open}          icon={<Clock size={18} />}        accent="#ffa94d" />
              <StatCard label="Resolved" value={resolved}      icon={<CheckCircle size={18} />} accent="#C3FDB8" />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.6fr', gap: '20px', marginBottom: '28px' }}>
              {/* Pie + legend */}
              <div style={card}>
                <h3 style={cardTitle}>By Severity</h3>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <div style={{ flex: '0 0 160px' }}>
                    {pieData.length === 0 ? <EmptyChart /> : (
                      <ResponsiveContainer width="100%" height={160}>
                        <PieChart>
                          <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={42} outerRadius={68} paddingAngle={3}>
                            {pieData.map((e, i) => <Cell key={i} fill={e.color} stroke="transparent" />)}
                          </Pie>
                          <Tooltip content={customTooltip as React.FC} />
                        </PieChart>
                      </ResponsiveContainer>
                    )}
                  </div>
                  <div style={{ flex: 1, minWidth: '100px' }}>
                    <PieLegend items={pieData} />
                  </div>
                </div>
              </div>

              {/* Area chart */}
              <div style={card}>
                <h3 style={cardTitle}>Last 24h</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={areaData} margin={{ top: 5, right: 10, left: -20, bottom: 0 }}>
                    <defs>
                      <linearGradient id="alertGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#C3FDB8" stopOpacity={0.3} />
                        <stop offset="95%" stopColor="#C3FDB8" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <XAxis dataKey="h" tick={{ fill: 'var(--text-secondary)', fontSize: 9, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} interval={5} />
                    <YAxis tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} allowDecimals={false} />
                    <Tooltip content={customTooltip as React.FC} />
                    <Area type="monotone" dataKey="count" stroke="#C3FDB8" strokeWidth={2} fill="url(#alertGrad)" />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>

            <div>
              <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <Bell size={16} /> Alerts
              </h2>
              <PaginatedTable data={sortedAlerts} columns={columns} keyFn={a => a.id} onRowClick={openModal} />
            </div>
          </>
        )}
      </div>

      {selected && (
        <JsonModal
          title={selected.title}
          subtitle={<><SeverityBadge severity={selected.severity} /><StatusBadge status={selected.status} /></>}
          data={selected}
          onClose={closeModal}
          footer={alertModalFooter}
        />
      )}
    </div>
  );
}

function fmt(s?: string) {
  if (!s) return '—';
  const d = new Date(s);
  return isNaN(d.getTime()) ? '—' : d.toLocaleString();
}

function ErrorBox({ msg }: { msg: string }) {
  return (
    <div style={{ padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' }}>
      <XCircle size={14} /> {msg}
    </div>
  );
}
function Spinner({ label }: { label: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Loading {label}...</span>
      <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
    </div>
  );
}
function EmptyChart() { return <div style={{ height: '160px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>No data</div>; }
const card: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '12px', padding: '20px' };
const cardTitle: React.CSSProperties = { fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '14px', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em' };
