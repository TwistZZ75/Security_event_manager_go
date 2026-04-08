import { useState, useEffect } from 'react';
import { PieChart, Pie, Cell, AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { Bell, AlertTriangle, CheckCircle, Clock, XCircle, RotateCcw } from 'lucide-react';
import { getAlerts, updateAlertStatus } from '../api/api';
import { useAuth } from '../contexts/AuthContext';
import { fmtDate, parseDate } from '../utils/date';
import type { Alert } from '../types';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import SeverityBadge from '../components/SeverityBadge';
import StatusBadge from '../components/StatusBadge';
import PageHeader from '../components/PageHeader';
import PieLegend from '../components/PieLegend';
import JsonModal from '../components/JsonModal';

const SEV_ORDER = ['critical', 'high', 'medium', 'low'];
const SEV_COLORS: Record<string, string> = {
  critical: '#ff3055', high: '#ff5f6d', medium: '#ffa94d', low: '#C3FDB8',
};
const SEV_LABELS: Record<string, string> = {
  critical: 'Критический', high: 'Высокий', medium: 'Средний', low: 'Низкий',
};

const isResolved = (s: string) => s === 'resolved' || s === 'false_positive';

export default function AlertsPage() {
  const { username } = useAuth();
  const [alerts, setAlerts]       = useState<Alert[]>([]);
  const [loading, setLoading]     = useState(true);
  const [error, setError]         = useState('');
  const [selected, setSelected]   = useState<Alert | null>(null);
  const [statusChoice, setStatus] = useState('');
  const [notes, setNotes]         = useState('');
  const [sending, setSending]     = useState(false);
  const [sendError, setSendError] = useState('');

  useEffect(() => {
    setLoading(true);
    getAlerts()
      .then(data => setAlerts(data ?? []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const openModal  = (a: Alert) => { setSelected(a); setStatus(''); setNotes(''); setSendError(''); };
  const closeModal = () => setSelected(null);

  const handleSend = async () => {
    if (!selected || !statusChoice) return;
    setSending(true);
    setSendError('');
    try {
      await updateAlertStatus(selected.id, statusChoice, notes);
      setAlerts(prev => prev.map(a => a.id === selected.id ? { ...a, status: statusChoice as Alert['status'] } : a));
      closeModal();
    } catch (e) {
      setSendError(e instanceof Error ? e.message : 'Ошибка обновления');
    } finally {
      setSending(false);
    }
  };

  const critical = alerts.filter(a => a.severity === 'critical').length;
  const open     = alerts.filter(a => a.status === 'open').length;
  const resolved = alerts.filter(a => isResolved(a.status)).length;

  const pieData = SEV_ORDER
    .map(s => ({ name: SEV_LABELS[s], value: alerts.filter(a => a.severity === s).length, color: SEV_COLORS[s] }))
    .filter(d => d.value > 0);

  const now = Date.now();
  const buckets: Record<number, number> = {};
  for (let i = 0; i < 24; i++) buckets[i] = 0;
  alerts.forEach(a => {
    const diff = Math.floor((now - new Date(a.created_at).getTime()) / 3_600_000);
    if (diff >= 0 && diff < 24) buckets[23 - diff]++;
  });
  const areaData = Array.from({ length: 24 }, (_, i) => ({ h: `${i}ч`, count: buckets[i] }));

  const sortedAlerts = [...alerts].sort((a, b) => {
    const si = (s: string) => SEV_ORDER.indexOf(s);
    if (si(a.severity) !== si(b.severity)) return si(a.severity) - si(b.severity);
    return parseDate(a.created_at).getTime() - parseDate(b.created_at).getTime();
  });

  const STATUS_ACTIVE = [
    { value: 'acknowledged',   label: 'Принять в работу' },
    { value: 'resolved',       label: 'Закрыть как решённый' },
    { value: 'false_positive', label: 'Ложная тревога' },
  ];
  const STATUS_CLOSED = [
    { value: 'open',           label: 'Переоткрыть', icon: <RotateCcw size={12} /> },
    { value: 'resolved',       label: 'Закрыть как решённый' },
    { value: 'false_positive', label: 'Ложная тревога' },
  ];
  const statusOptions = selected && selected.status !== 'open' ? STATUS_CLOSED : STATUS_ACTIVE;

  const columns = [
    { key: 'id', header: '№', width: '60px',
      render: (a: Alert) => <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>#{a.id}</span> },
    { key: 'severity', header: 'Важность', width: '120px',
      sortValue: (a: Alert) => SEV_ORDER.indexOf(a.severity),
      render: (a: Alert) => <SeverityBadge severity={a.severity} /> },
    { key: 'title', header: 'Заголовок',
      render: (a: Alert) => (
        <div>
          <div style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '13px' }}>{a.title}</div>
          <div style={{ color: 'var(--text-secondary)', fontSize: '11px', marginTop: '2px' }}>{a.rule_name}</div>
        </div>
      ) },
    { key: 'status', header: 'Статус', width: '160px',
      sortValue: (a: Alert) => ['open','acknowledged','resolved','false_positive'].indexOf(a.status),
      render: (a: Alert) => <StatusBadge status={a.status} /> },
    { key: 'created_at', header: 'Создан',
      sortValue: (a: Alert) => parseDate(a.created_at),
      render: (a: Alert) => <span style={{ color: 'var(--text-secondary)', fontSize: '11px' }}>{fmtDate(a.created_at)}</span> },
  ];

  const alertModalFooter = selected && (
    <div style={{ padding: '20px 22px' }}>
      <div style={{ marginBottom: '14px' }}>
        <div style={sectionLabel}>Изменить статус</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
          {statusOptions.map(opt => (
            <label key={opt.value} style={{ display: 'flex', alignItems: 'center', gap: '9px', cursor: 'pointer', fontSize: '13px', fontFamily: 'var(--font-display)' }}>
              <input type="radio" name="alert_status" value={opt.value}
                checked={statusChoice === opt.value} onChange={() => setStatus(opt.value)}
                style={{ accentColor: 'var(--mint)', width: '15px', height: '15px', cursor: 'pointer' }} />
              <span style={{ display: 'flex', alignItems: 'center', gap: '5px', color: statusChoice === opt.value ? 'var(--mint)' : 'var(--text-secondary)' }}>
                {('icon' in opt && opt.icon) as React.ReactNode}
                {opt.label}
              </span>
            </label>
          ))}
        </div>
      </div>
      <div style={{ marginBottom: '14px' }}>
        <div style={sectionLabel}>Примечание</div>
        <textarea value={notes} onChange={e => setNotes(e.target.value)}
          placeholder="Необязательное примечание..." rows={3}
          style={{ width: '100%', boxSizing: 'border-box', background: 'var(--navy)', border: '1px solid var(--navy-border)', borderRadius: '8px', color: 'var(--mint)', fontFamily: 'var(--font-mono)', fontSize: '12px', padding: '10px 12px', resize: 'vertical', outline: 'none', lineHeight: '1.5' }} />
      </div>
      {username && (
        <div style={{ fontSize: '11px', color: 'var(--text-secondary)', marginBottom: '10px', fontFamily: 'var(--font-mono)' }}>
          Будет записано как: <span style={{ color: 'var(--mint)' }}>{username}</span>
        </div>
      )}
      {sendError && (
        <div style={{ color: '#ff5f6d', fontSize: '12px', marginBottom: '10px', display: 'flex', gap: '6px', alignItems: 'center' }}>
          <AlertTriangle size={12} /> {sendError}
        </div>
      )}
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button onClick={handleSend} disabled={!statusChoice || sending}
          style={{ padding: '9px 22px', borderRadius: '9px', border: 'none', background: (!statusChoice || sending) ? 'var(--navy-border)' : 'var(--mint)', color: (!statusChoice || sending) ? 'var(--text-secondary)' : 'var(--navy)', fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '13px', cursor: (!statusChoice || sending) ? 'not-allowed' : 'pointer' }}>
          {sending ? 'Отправка...' : 'Применить'}
        </button>
      </div>
    </div>
  );

  const tt = ({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; payload: { color?: string } }> }) => {
    if (!active || !payload?.length) return null;
    return <div style={ttStyle}><div style={{ color: payload[0].payload.color || 'var(--mint)', fontWeight: 700 }}>{payload[0].name}</div><div>{payload[0].value}</div></div>;
  };

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader title="Алерты" subtitle="Инциденты безопасности и тревоги" />
      <div style={{ padding: '0 32px' }}>
        {error && <div style={errorBox}><XCircle size={14} /> {error}</div>}
        {loading ? <Spinner label="алертов" /> : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Всего"       value={alerts.length} icon={<Bell size={18} />}          accent="#C3FDB8" />
              <StatCard label="Критических" value={critical}      icon={<AlertTriangle size={18} />}  accent="#ff3055" />
              <StatCard label="Открытых"    value={open}          icon={<Clock size={18} />}           accent="#ffa94d" />
              <StatCard label="Закрытых"    value={resolved}      icon={<CheckCircle size={18} />}    accent="#C3FDB8" sub="вкл. ложные тревоги" />
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.6fr', gap: '20px', marginBottom: '28px' }}>
              <div style={card}>
                <h3 style={cardTitle}>По важности</h3>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <div style={{ flex: '0 0 160px' }}>
                    {pieData.length === 0 ? <EmptyChart /> : (
                      <ResponsiveContainer width="100%" height={160}>
                        <PieChart>
                          <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={42} outerRadius={68} paddingAngle={3}>
                            {pieData.map((e, i) => <Cell key={i} fill={e.color} stroke="transparent" />)}
                          </Pie>
                          <Tooltip content={tt as React.FC} />
                        </PieChart>
                      </ResponsiveContainer>
                    )}
                  </div>
                  <div style={{ flex: 1 }}><PieLegend items={pieData} /></div>
                </div>
              </div>
              <div style={card}>
                <h3 style={cardTitle}>Последние 24 часа</h3>
                <ResponsiveContainer width="100%" height={200}>
                  <AreaChart data={areaData} margin={{ top: 5, right: 10, left: -20, bottom: 0 }}>
                    <defs><linearGradient id="ag" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#C3FDB8" stopOpacity={0.3} /><stop offset="95%" stopColor="#C3FDB8" stopOpacity={0} /></linearGradient></defs>
                    <XAxis dataKey="h" tick={{ fill: 'var(--text-secondary)', fontSize: 9, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} interval={5} />
                    <YAxis tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} allowDecimals={false} />
                    <Tooltip content={tt as React.FC} />
                    <Area type="monotone" dataKey="count" stroke="#C3FDB8" strokeWidth={2} fill="url(#ag)" />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>
            <div>
              <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}><Bell size={16} /> Список алертов</h2>
              <PaginatedTable data={sortedAlerts} columns={columns} keyFn={a => a.id} onRowClick={openModal} />
            </div>
          </>
        )}
      </div>
      {selected && (
        <JsonModal title={selected.title} subtitle={<><SeverityBadge severity={selected.severity} /><StatusBadge status={selected.status} /></>} data={selected} onClose={closeModal} footer={alertModalFooter} />
      )}
    </div>
  );
}

function Spinner({ label }: { label: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Загрузка {label}...</span>
      <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
    </div>
  );
}
function EmptyChart() { return <div style={{ height: '160px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>Нет данных</div>; }

const errorBox: React.CSSProperties = { padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' };
const card: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '12px', padding: '20px' };
const cardTitle: React.CSSProperties = { fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '14px', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em' };
const sectionLabel: React.CSSProperties = { fontSize: '11px', fontFamily: 'var(--font-display)', fontWeight: 700, color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: '10px' };
const ttStyle: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '8px', padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: '12px' };
