import { useState, useEffect } from 'react';
import {
  PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis,
  Tooltip, ResponsiveContainer,
} from 'recharts';
import {
  Zap, CheckCircle, XCircle, Clock, AlertTriangle,
  ShieldOff, Wifi, Info,
} from 'lucide-react';
import { getActions, sendAgentCommand } from '../api/api';
import type { Action } from '../types';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import StatusBadge from '../components/StatusBadge';
import PageHeader from '../components/PageHeader';
import { fmtDate, parseDate } from '../utils/date';
import PieLegend from '../components/PieLegend';
import JsonModal from '../components/JsonModal';
import AgentMiniModal from '../components/AgentMiniModal';

// Типы действий с возможностью разблокировки
const UNBLOCK_MAP: Record<string, { cmd: string; label: string; paramKey: string }> = {
  block_account:  { cmd: 'unblock_account',  label: 'Unblock Account',  paramKey: 'username' },
  block_network:  { cmd: 'unblock_network',   label: 'Unblock Network',  paramKey: 'host' },
};

// Для этих типов — только JSON (без кнопок)
const VIEW_ONLY = new Set(['notify', 'kill_process', 'quarantine_file', 'isolate_host']);

type ModalState =
  | { type: 'action'; action: Action }
  | { type: 'agent'; hostname: string }
  | null;

export default function ActionsPage() {
  const [actions, setActions] = useState<Action[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [modal, setModal] = useState<ModalState>(null);
  const [unblocking, setUnblocking] = useState(false);
  const [unblockResult, setUnblockResult] = useState<{ ok: boolean; msg: string } | null>(null);

  const loadActions = () => {
    setLoading(true);
    getActions()
      .then(data => setActions(data ?? []))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  };

  useEffect(() => { loadActions(); }, []);

  // ── Stats ─────────────────────────────────────────────────────────────────
  const success = actions.filter(a => a.status === 'success').length;
  const failed  = actions.filter(a => a.status === 'failed').length;
  const pending = actions.filter(a => a.status === 'pending').length;

  const typeMap: Record<string, number> = {};
  actions.forEach(a => { typeMap[a.action_type] = (typeMap[a.action_type] || 0) + 1; });
  const typeData = Object.entries(typeMap)
    .sort((a, b) => b[1] - a[1])
    .map(([name, value]) => ({ name: name.replace(/_/g, ' '), value }));

  const pieData = [
    { name: 'Success', value: success, color: '#C3FDB8' },
    { name: 'Failed',  value: failed,  color: '#ff5f6d' },
    { name: 'Pending', value: pending, color: '#ffe066' },
  ].filter(d => d.value > 0);

  const TYPE_COLORS = ['#C3FDB8','#a0e095','#64b4ff','#ffa94d','#b47cff','#ff5f6d','#ffe066','#ff8800','#00e5ff'];

  const sortedActions = [...actions].sort((a, b) =>
    parseDate(b.executed_at).getTime() - parseDate(a.executed_at).getTime()
  );

  // ── Row click ─────────────────────────────────────────────────────────────
  const handleRowClick = (a: Action) => {
    setUnblockResult(null);
    setModal({ type: 'action', action: a });
  };

  // ── Target click (open agent mini modal) ──────────────────────────────────
  const handleTargetClick = (e: React.MouseEvent, a: Action) => {
    e.stopPropagation();
    if (!a.target) return;
    // target может быть "username@hostname" или просто "hostname"
    const hostname = a.target.includes('@') ? a.target.split('@')[1] : a.target.split(':')[0];
    if (hostname) setModal({ type: 'agent', hostname });
  };

  // ── Unblock action ────────────────────────────────────────────────────────
  const handleUnblock = async (action: Action) => {
    const unblock = UNBLOCK_MAP[action.action_type];
    if (!unblock) return;

    setUnblocking(true);
    setUnblockResult(null);

    // Вычленяем hostname из target: "username@hostname" или "hostname"
    let hostname = action.target || '';
    let username = '';
    if (hostname.includes('@')) {
      [username, hostname] = hostname.split('@');
    }

    const params: Record<string, string> = {};
    if (unblock.paramKey === 'username' && username) params['username'] = username;
    if (hostname) params['host'] = hostname;

    try {
      await sendAgentCommand(hostname || action.target, unblock.cmd, params);
      setUnblockResult({ ok: true, msg: `${unblock.label} command sent to agent successfully.` });
    } catch (e) {
      setUnblockResult({ ok: false, msg: e instanceof Error ? e.message : 'Failed to send command' });
    } finally {
      setUnblocking(false);
    }
  };

  // ── Columns ───────────────────────────────────────────────────────────────
  const columns = [
    {
      key: 'id', header: 'ID', width: '60px',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>#{a.id}</span>
      ),
    },
    {
      key: 'type', header: 'Type',
      sortValue: (row: Action) => row.action_type,
      render: (a: Action) => {
        const isBlock = !!UNBLOCK_MAP[a.action_type];
        return (
          <span style={{
            fontFamily: 'var(--font-mono)', fontSize: '11px',
            background: isBlock ? 'rgba(255,95,109,0.12)' : 'var(--mint-glow)',
            color: isBlock ? '#ff5f6d' : 'var(--mint)',
            padding: '3px 8px', borderRadius: '5px', whiteSpace: 'nowrap',
          }}>
            {a.action_type}
          </span>
        );
      },
    },
    {
      key: 'status', header: 'Status', width: '120px',
      sortValue: (row: Action) => ['failed', 'pending', 'success', ].indexOf(row.status),
      render: (a: Action) => <StatusBadge status={a.status} />,
    },
    {
      key: 'target', header: 'Target',
      sortValue: (row: Action) => row.target,
      render: (a: Action) => {
        const hasTarget = !!a.target;
        return (
          <span
            onClick={hasTarget ? e => handleTargetClick(e, a) : undefined}
            title={hasTarget ? 'Click to view agent info' : undefined}
            style={{
              color: hasTarget ? 'var(--mint)' : 'var(--text-secondary)',
              fontFamily: 'var(--font-mono)', fontSize: '12px',
              cursor: hasTarget ? 'pointer' : 'default',
              textDecoration: hasTarget ? 'underline dotted' : 'none',
              textUnderlineOffset: '3px',
            }}
          >
            {a.target || '—'}
          </span>
        );
      },
    },
    {
      key: 'alert_id', header: 'Alert', width: '80px',
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>#{a.alert_id}</span>
      ),
    },
    {
      key: 'result', header: 'Result',
      render: (a: Action) => (
        <span
      style={{
        fontSize: '12px',
        display: 'block',
        maxWidth: '300px',        // максимальная ширина содержимого
        overflow: 'hidden',       // обрезаем что не влезает
        whiteSpace: 'nowrap',     // запрещаем перенос строки
      }}
    >
          {a.error
            ? <span style={{ color: '#ff5f6d' }}>{a.error}</span>
            : <span style={{ color: 'var(--text-secondary)' }}>{a.result || '—'}</span>}
        </span>
      ),
    },
    {
      key: 'executed_at', header: 'Executed At',
      sortValue: (a: Action) => parseDate(a.executed_at),
      render: (a: Action) => (
        <span style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>{fmt(a.executed_at)}</span>
      ),
    },
  ];

  // ── Modal footer for block_account / block_network ────────────────────────
  const selectedAction = modal?.type === 'action' ? modal.action : null;
  const unblockInfo = selectedAction ? UNBLOCK_MAP[selectedAction.action_type] : null;

  const actionModalFooter = selectedAction && unblockInfo ? (
    <div style={{ padding: '18px 22px', display: 'flex', flexDirection: 'column', gap: '12px' }}>
      <div style={{ fontSize: '11px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)', textTransform: 'uppercase', letterSpacing: '0.08em', fontWeight: 700 }}>
        Remediation
      </div>

      {unblockResult && (
        <div style={{
          padding: '10px 14px', borderRadius: '8px', fontSize: '12px',
          background: unblockResult.ok ? 'rgba(195,253,184,0.1)' : 'rgba(255,95,109,0.1)',
          border: `1px solid ${unblockResult.ok ? 'rgba(195,253,184,0.3)' : 'rgba(255,95,109,0.3)'}`,
          color: unblockResult.ok ? 'var(--mint)' : '#ff5f6d',
          display: 'flex', alignItems: 'center', gap: '8px',
        }}>
          {unblockResult.ok ? <CheckCircle size={13} /> : <AlertTriangle size={13} />}
          {unblockResult.msg}
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button
          onClick={() => handleUnblock(selectedAction)}
          disabled={unblocking}
          style={{
            display: 'flex', alignItems: 'center', gap: '7px',
            padding: '9px 20px', borderRadius: '9px',
            background: unblocking ? 'var(--navy-border)' : 'rgba(255,95,109,0.15)',
            color: unblocking ? 'var(--text-secondary)' : '#ff5f6d',
            fontFamily: 'var(--font-mono)', fontWeight: 800, fontSize: '13px',
            cursor: unblocking ? 'not-allowed' : 'pointer',
            transition: 'background var(--transition)',
            border: '1px solid rgba(255,95,109,0.3)',
          } as React.CSSProperties}
        >
          <ShieldOff size={14} />
          {unblocking ? 'Sending…' : unblockInfo.label}
        </button>
      </div>
    </div>
  ) : null;

  // ── Tooltip ───────────────────────────────────────────────────────────────
  const customTooltip = ({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; payload: { color?: string } }> }) => {
    if (!active || !payload?.length) return null;
    return (
      <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '8px', padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: '12px' }}>
        <div style={{ color: payload[0].payload.color || 'var(--mint)', fontWeight: 700 }}>{payload[0].name}</div>
        <div>{payload[0].value}</div>
      </div>
    );
  };

  // ── Render ────────────────────────────────────────────────────────────────
  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader title="Actions" subtitle="Automated response actions log" />
      <div style={{ padding: '0 32px' }}>
        {error && <ErrorBox msg={error} />}
        {loading ? <Spinner /> : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Total"   value={actions.length} icon={<Zap size={18} />}         accent="#C3FDB8" />
              <StatCard label="Success" value={success}        icon={<CheckCircle size={18} />}  accent="#C3FDB8" />
              <StatCard label="Failed"  value={failed}         icon={<XCircle size={18} />}      accent="#ff5f6d" />
              <StatCard label="Pending" value={pending}        icon={<Clock size={18} />}        accent="#ffe066" />
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.6fr', gap: '20px', marginBottom: '28px' }}>
              {/* Pie */}
              <div style={card}>
                <h3 style={cardTitle}>Status Distribution</h3>
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
                  <div style={{ flex: 1, minWidth: '90px' }}>
                    <PieLegend items={pieData} />
                  </div>
                </div>
              </div>

              {/* Bar chart */}
              <div style={card}>
                <h3 style={cardTitle}>Action Types</h3>
                {typeData.length === 0 ? <EmptyChart /> : (
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={typeData} layout="vertical" margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                      <XAxis type="number" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} allowDecimals={false} />
                      <YAxis type="category" dataKey="name" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} width={120} />
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
              <h2 style={{ fontFamily: 'var(--font-mono)', fontWeight: 700, fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <Zap size={16} /> Action Log
                <span style={{ fontSize: '11px', fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)', fontWeight: 400, marginLeft: '4px' }}>
                  — click row to view details
                </span>
              </h2>
              <PaginatedTable
                data={sortedActions}
                columns={columns}
                keyFn={a => a.id}
                onRowClick={handleRowClick}
              />
            </div>
          </>
        )}
      </div>

      {/* JSON Modal for action details */}
      {modal?.type === 'action' && (
        <JsonModal
          title={modal.action.action_type.replace(/_/g, ' ')}
          subtitle={
            <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }}>
              <StatusBadge status={modal.action.status} />
              {UNBLOCK_MAP[modal.action.action_type] && (
                <span style={{ fontSize: '10px', background: 'rgba(255,95,109,0.12)', color: '#ff5f6d', padding: '2px 7px', borderRadius: '4px', fontFamily: 'var(--font-display)', fontWeight: 700 }}>
                  BLOCKED
                </span>
              )}
              {VIEW_ONLY.has(modal.action.action_type) && (
                <span style={{ fontSize: '10px', background: 'var(--mint-glow)', color: 'var(--mint)', padding: '2px 7px', borderRadius: '4px', fontFamily: 'var(--font-display)', fontWeight: 700, display: 'flex', alignItems: 'center', gap: '4px' }}>
                  <Info size={9} /> View Only
                </span>
              )}
            </div>
          }
          data={modal.action}
          onClose={() => setModal(null)}
          footer={actionModalFooter}
        />
      )}

      {/* Agent Mini Modal */}
      {modal?.type === 'agent' && (
        <AgentMiniModal
          hostname={modal.hostname}
          onClose={() => setModal(null)}
        />
      )}
    </div>
  );
}

function fmt(s?: string) { return fmtDate(s ?? undefined); }
function EmptyChart() { return <div style={{ height: '160px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>No data</div>; }
function ErrorBox({ msg }: { msg: string }) {
  return (
    <div style={{ padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' }}>
      <AlertTriangle size={14} /> {msg}
    </div>
  );
}
function Spinner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Loading actions...</span>
      <style>{`@keyframes spin{to{transform:rotate(360deg)}}`}</style>
    </div>
  );
}
const card: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '12px', padding: '20px' };
const cardTitle: React.CSSProperties = { fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '14px', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em' };
