import { useState, useEffect } from 'react';
import {
  PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts';
import { Monitor, Wifi, WifiOff, AlertTriangle } from 'lucide-react';
import { getAgents } from '../api/api';
import type { Agent } from '../types';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import StatusBadge from '../components/StatusBadge';
import PageHeader from '../components/PageHeader';
import PieLegend from '../components/PieLegend';

const MINT = '#C3FDB8';
const MINT_DIM = '#a0e095';
const RED = '#ff5f6d';
const GRAY = '#888bac';
const ORANGE = '#ffa94d';


export default function DashboardPage() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getAgents()
      .then(setAgents)
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  // Stats
  const online = agents.filter(a => a.Status === 'online').length;
  const offline = agents.filter(a => a.Status === 'offline').length;
  const errored = agents.filter(a => a.Status === 'error').length;

  // OS distribution
  const osMap: Record<string, number> = {};
  agents.forEach(a => {
    const os = a.OSVersion || 'Unknown';
    osMap[os] = (osMap[os] || 0) + 1;
  });
  const osData = Object.entries(osMap).map(([name, value]) => ({ name, value }));

  // Pie data for status
  const pieData = [
    { name: 'Online',  value: online,  color: MINT },
    { name: 'Offline', value: offline, color: GRAY },
    { name: 'Error',   value: errored, color: RED },
  ].filter(d => d.value > 0);

  // Bar chart - agents by OS
  const OS_COLORS = [MINT, MINT_DIM, ORANGE, '#64b4ff', '#b47cff'];

  // Sort: online first, then alphabetical
  const sortedAgents = [...agents].sort((a, b) => {
    if (a.Status === 'online' && b.Status !== 'online') return -1;
    if (a.Status !== 'online' && b.Status === 'online') return 1;
return (a.Hostname || '').localeCompare(b.Hostname || '');
  });

  const columns = [
    {
      key: 'hostname',
      header: 'Hostname',
      render: (a: Agent) => (
        <span style={{ fontFamily: 'var(--font-mono)', fontWeight: 700 }}>{a.Hostname}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      width: '120px',
      render: (a: Agent) => <StatusBadge status={a.Status} />,
    },
    {
      key: 'os',
      header: 'OS',
      render: (a: Agent) => (
        <span style={{ color: 'var(--text-secondary)' }}>
          {a.OSVersion}
        </span>
      ),
    },
    {
      key: 'ip',
      header: 'IPAddress',
      render: (a: Agent) => (
        <span style={{ fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)', fontSize: '12px' }}>
          {a.IPAddress}
        </span>
      ),
    },
    {
      key: 'version',
      header: 'AgentVersion',
      render: (a: Agent) => (
        <span style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>
          {a.AgentVersion}
        </span>
      ),
    },
    {
      key: 'last_seen',
      header: 'LastSeen',
      render: (a: Agent) => (
        <span style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>
          {formatDate(a.LastSeen)}
        </span>
      ),
    },
  ];

  const customTooltip = ({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; payload: { color: string } }> }) => {
    if (!active || !payload?.length) return null;
    return (
      <div style={{
        background: 'var(--navy-light)',
        border: '1px solid var(--navy-border)',
        borderRadius: '8px',
        padding: '10px 14px',
        fontFamily: 'var(--font-mono)',
        fontSize: '12px',
      }}>
        <div style={{ color: payload[0].payload.color, fontWeight: 700 }}>{payload[0].name}</div>
        <div>{payload[0].value} agents</div>
      </div>
    );
  };

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader
        title="Dashboard"
        subtitle="Agent overview and system health"
      />

      <div style={{ padding: '0 32px' }}>
        {error && (
          <div style={{
            padding: '12px 16px', borderRadius: '8px', marginBottom: '20px',
            background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)',
            color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center',
          }}>
            <AlertTriangle size={14} /> {error}
          </div>
        )}

        {loading ? (
          <LoadingSpinner />
        ) : (
          <>
            {/* Stat cards */}
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Total Agents" value={agents.length} icon={<Monitor size={18} />} accent={MINT} />
              <StatCard label="Online"  value={online}  icon={<Wifi size={18} />}    accent={MINT}   sub="connected agents" />
              <StatCard label="Offline" value={offline} icon={<WifiOff size={18} />} accent={GRAY}   sub="disconnected" />
              <StatCard label="Errors"  value={errored} icon={<AlertTriangle size={18} />} accent={RED} sub="need attention" />
            </div>

            {/* Charts */}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', marginBottom: '28px' }}>
              {/* Status pie */}
                            <div style={chartCardStyle}>
                <h3 style={chartTitleStyle}>Agent Status</h3>
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

              {/* OS distribution bar */}
              <div style={chartCardStyle}>
                <h3 style={chartTitleStyle}>OS Distribution</h3>
                {osData.length === 0 ? (
                  <EmptyChart />
                ) : (
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={osData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                      <XAxis
                        dataKey="name"
                        tick={{ fill: 'var(--text-secondary)', fontSize: 11, fontFamily: 'var(--font-mono)' }}
                        axisLine={{ stroke: 'var(--navy-border)' }}
                        tickLine={false}
                      />
                      <YAxis
                        tick={{ fill: 'var(--text-secondary)', fontSize: 11, fontFamily: 'var(--font-mono)' }}
                        axisLine={false}
                        tickLine={false}
                        allowDecimals={false}
                      />
                      <Tooltip content={customTooltip as React.FC} />
                      <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                        {osData.map((_, i) => (
                          <Cell key={i} fill={OS_COLORS[i % OS_COLORS.length]} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </div>
            </div>

            {/* Table */}
            <div>
              <h2 style={{
                fontFamily: 'var(--font-display)',
                fontWeight: 700,
                fontSize: '16px',
                marginBottom: '16px',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}>
                <Monitor size={16} />
                Agents
              </h2>
              <PaginatedTable
                data={sortedAgents}
                columns={columns}
                keyFn={a => a.ID}
              />
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
  if (isNaN(d.getTime())) return '—';
  return d.toLocaleString();
}

const chartCardStyle: React.CSSProperties = {
  background: 'var(--navy-light)',
  border: '1px solid var(--navy-border)',
  borderRadius: '12px',
  padding: '20px',
};

const chartTitleStyle: React.CSSProperties = {
  fontFamily: 'var(--font-display)',
  fontWeight: 700,
  fontSize: '14px',
  marginBottom: '14px',
  color: 'var(--text-secondary)',
  textTransform: 'uppercase',
  letterSpacing: '0.08em',
};

function EmptyChart() {
  return (
    <div style={{ height: '220px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>
      No data available
    </div>
  );
}

function LoadingSpinner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{
        width: '36px', height: '36px', borderRadius: '50%',
        border: '3px solid var(--navy-border)',
        borderTopColor: 'var(--mint)',
        animation: 'spin 0.8s linear infinite',
      }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Loading agents...</span>
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}
