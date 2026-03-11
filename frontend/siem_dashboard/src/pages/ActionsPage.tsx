import { useState, useEffect } from 'react';
import {
  PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis,
  Tooltip, ResponsiveContainer,
} from 'recharts';
import { Zap, CheckCircle, XCircle, Clock, AlertTriangle } from 'lucide-react';
import { getActions } from '../api/api';
import type { Action } from '../types';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import StatusBadge from '../components/StatusBadge';
import PageHeader from '../components/PageHeader';
import PieLegend from '../components/PieLegend';

export default function ActionsPage() {
  const [actions, setActions] = useState<Action[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getActions()
      .then(setActions)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  const success  = actions.filter(a => a.status === 'success').length;
  const failed   = actions.filter(a => a.status === 'failed').length;
  const pending  = actions.filter(a => a.status === 'pending').length;

  const typeMap: Record<string, number> = {};
  actions.forEach(a => { typeMap[a.action_type] = (typeMap[a.action_type] || 0) + 1; });
  const typeData = Object.entries(typeMap).map(([name, value]) => ({ name: name.replace(/_/g, ' '), value }));

  const pieData = [
    { name: 'Success', value: success, color: '#C3FDB8' },
    { name: 'Failed',  value: failed,  color: '#ff5f6d' },
    { name: 'Pending', value: pending, color: '#ffe066' },
  ].filter(d => d.value > 0);

  const TYPE_COLORS = ['#C3FDB8', '#a0e095', '#64b4ff', '#ffa94d', '#b47cff', '#ff5f6d', '#ffe066', '#ff8800', '#00e5ff'];

  const sortedActions = [...actions].sort((a, b) =>
    new Date(b.executed_at).getTime() - new Date(a.executed_at).getTime()
  );

  const columns = [
    {
      key: 'id',
      header: 'ID',
      width: '60px',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>#{a.id}</span>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      render: (a: Action) => (
        <span style={{
          fontFamily: 'var(--font-mono)', fontSize: '11px',
          background: 'var(--mint-glow)', color: 'var(--mint)',
          padding: '3px 8px', borderRadius: '5px', whiteSpace: 'nowrap',
        }}>
          {a.action_type}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      width: '120px',
      render: (a: Action) => <StatusBadge status={a.status} />,
    },
    {
      key: 'target',
      header: 'Target',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
          {a.target || '—'}
        </span>
      ),
    },
    {
      key: 'alert_id',
      header: 'Alert',
      width: '80px',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
          #{a.alert_id}
        </span>
      ),
    },
    {
      key: 'result',
      header: 'Result',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>
          {a.error
            ? <span style={{ color: '#ff5f6d' }}>{a.error}</span>
            : (a.result || '—')}
        </span>
      ),
    },
    {
      key: 'executed_at',
      header: 'Executed At',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>
          {formatDate(a.executed_at)}
        </span>
      ),
    },
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

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader title="Actions" subtitle="Automated response actions history" />
      <div style={{ padding: '0 32px' }}>
        {error && <ErrorBox msg={error} />}
        {loading ? <LoadingSpinner label="actions" /> : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Total"   value={actions.length} icon={<Zap size={18} />}         accent="#C3FDB8" />
              <StatCard label="Success" value={success}        icon={<CheckCircle size={18} />}  accent="#C3FDB8" />
              <StatCard label="Failed"  value={failed}         icon={<XCircle size={18} />}      accent="#ff5f6d" />
              <StatCard label="Pending" value={pending}        icon={<Clock size={18} />}        accent="#ffe066" />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.6fr', gap: '20px', marginBottom: '28px' }}>
              <div style={chartCardStyle}>
                <h3 style={chartTitleStyle}>Status Distribution</h3>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <div style={{ flex: '0 0 160px' }}>
                    {pieData.length === 0 ? <EmptyChart /> : (
                      <ResponsiveContainer width="100%" height={160}>
                        <PieChart>
                          <Pie data={pieData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={42} outerRadius={68} paddingAngle={3}>
                            {pieData.map((entry, i) => <Cell key={i} fill={entry.color} stroke="transparent" />)}
                          </Pie>
                          <Tooltip content={customTooltip as React.FC} />
                        </PieChart>
                      </ResponsiveContainer>
                    )}
                  </div>
                  <div style={{ flex: 1, minWidth: '90px' }}>
                    <PieLegend items={pieData} />
                  </div>
                </div>
              </div>
              <div style={chartCardStyle}>
                <h3 style={chartTitleStyle}>Actions by Type</h3>
                {typeData.length === 0 ? <EmptyChart /> : (
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={typeData} layout="vertical" margin={{ top: 5, right: 10, left: 20, bottom: 0 }}>
                      <XAxis type="number" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} allowDecimals={false} />
                      <YAxis type="category" dataKey="name" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} width={100} />
                      <Tooltip content={customTooltip as React.FC} />
                      <Bar dataKey="value" radius={[0, 4, 4, 0]}>
                        {typeData.map((_, i) => <Cell key={i} fill={TYPE_COLORS[i % TYPE_COLORS.length]} />)}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </div>
            </div>

            <div>
              <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <Zap size={16} /> Actions
              </h2>
              <PaginatedTable data={sortedActions} columns={columns} keyFn={a => a.id} />
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function formatDate(str?: string) {
  if (!str) return '—';
  const d = new Date(str);
  return isNaN(d.getTime()) ? '—' : d.toLocaleString();
}

function ErrorBox({ msg }: { msg: string }) {
  return (
    <div style={{ padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' }}>
      <AlertTriangle size={14} /> {msg}
    </div>
  );
}

const chartCardStyle: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '12px', padding: '20px' };
const chartTitleStyle: React.CSSProperties = { fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '14px', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em' };
function EmptyChart() { return <div style={{ height: '220px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>No data available</div>; }
function LoadingSpinner({ label }: { label: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Loading {label}...</span>
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}
