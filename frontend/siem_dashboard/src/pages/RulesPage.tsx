import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts';
import { BookOpen, Plus, AlertTriangle, Pencil, Trash2, ToggleLeft, ToggleRight, X } from 'lucide-react';
import { getRules, deleteRule, setRuleEnabled } from '../api/api';
import type { Rule } from '../types';
import { parseDate } from '../utils/date';
import StatCard from '../components/StatCard';
import PaginatedTable from '../components/PaginatedTable';
import SeverityBadge from '../components/SeverityBadge';
import StatusBadge from '../components/StatusBadge';
import PageHeader from '../components/PageHeader';
import PieLegend from '../components/PieLegend';

const SEV_COLORS: Record<string, string> = { critical: '#ff3055', high: '#ff5f6d', medium: '#ffa94d', low: '#C3FDB8' };
const SEV_LABELS: Record<string, string> = { critical: 'Критический', high: 'Высокий', medium: 'Средний', low: 'Низкий' };

export default function RulesPage() {
  const [rules, setRules]           = useState<Rule[]>([]);
  const [loading, setLoading]       = useState(true);
  const [error, setError]           = useState('');
  const [selectedRule, setSelected] = useState<Rule | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    setLoading(true);
    getRules().then(setRules).catch(e => setError(e.message)).finally(() => setLoading(false));
  }, []);

  const enabled  = rules.filter(r => r.enabled).length;
  const disabled = rules.filter(r => !r.enabled).length;

  const sevMap: Record<string, number> = {};
  rules.forEach(r => { sevMap[r.severity] = (sevMap[r.severity] || 0) + 1; });
  const sevData = Object.entries(sevMap).map(([name, value]) => ({ name: SEV_LABELS[name] ?? name, value, color: SEV_COLORS[name] || '#888' }));

  const triggerData = [...rules].filter(r => (r.trigger_count ?? 0) > 0)
    .sort((a, b) => (b.trigger_count ?? 0) - (a.trigger_count ?? 0))
    .slice(0, 8)
    .map(r => ({ name: r.name.length > 18 ? r.name.slice(0, 16) + '…' : r.name, value: r.trigger_count ?? 0 }));

  const sortedRules = [...rules].sort((a, b) => {
    if (a.enabled !== b.enabled) return a.enabled ? -1 : 1;
    return a.name.localeCompare(b.name, 'ru');
  });

  const handleToggle = async (e: React.MouseEvent, rule: Rule) => {
    e.stopPropagation();
    try {
      await setRuleEnabled(rule.id, !rule.enabled);
      setRules(prev => prev.map(r => r.id === rule.id ? { ...r, enabled: !r.enabled } : r));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка обновления правила');
    }
  };

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    if (!confirm('Удалить это правило?')) return;
    try {
      await deleteRule(id);
      setRules(prev => prev.filter(r => r.id !== id));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка удаления');
    }
  };

  const columns = [
    { key: 'name', header: 'Название правила', sortValue: (r: Rule) => r.name,
      render: (r: Rule) => (
        <div>
          <div style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '13px' }}>{r.name}</div>
          {r.tags && r.tags.length > 0 && (
            <div style={{ display: 'flex', gap: '4px', marginTop: '4px', flexWrap: 'wrap' }}>
              {r.tags.map(t => <span key={t} style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '10px', background: 'var(--mint-glow)', color: 'var(--text-secondary)' }}>{t}</span>)}
            </div>
          )}
        </div>
      ) },
    { key: 'severity', header: 'Важность', width: '120px',
      sortValue: (r: Rule) => ['critical','high','medium','low'].indexOf(r.severity),
      render: (r: Rule) => <SeverityBadge severity={r.severity} /> },
    { key: 'status', header: 'Статус', width: '130px',
      render: (r: Rule) => <StatusBadge status={r.enabled ? 'enabled' : 'disabled'} /> },
    { key: 'triggers', header: 'Срабатываний', width: '130px',
      sortValue: (r: Rule) => r.trigger_count ?? 0,
      render: (r: Rule) => <span style={{ fontFamily: 'var(--font-mono)', fontSize: '13px', color: (r.trigger_count ?? 0) > 0 ? 'var(--mint)' : 'var(--text-secondary)' }}>{r.trigger_count ?? 0}</span> },
    { key: 'created_at', header: 'Создано', width: 'auto',
      sortValue: (r: Rule) => parseDate(r.created_at) ?? new Date(0),
      render: (r: Rule) => <span style={{ color: 'var(--text-secondary)', fontSize: '11px' }}>{fmtDate(r.created_at)}</span> },
    { key: 'actions_col', header: 'Действия', width: '120px',
      render: (r: Rule) => (
        <div style={{ display: 'flex', gap: '6px', alignItems: 'center' }} onClick={e => e.stopPropagation()}>
          <button onClick={e => handleToggle(e, r)} title={r.enabled ? 'Отключить правило' : 'Включить правило'} style={{ ...btnStyle, color: r.enabled ? 'var(--mint)' : 'var(--text-secondary)', background: r.enabled ? 'rgba(195,253,184,0.1)' : 'transparent' }}>
            {r.enabled ? <ToggleRight size={18} strokeWidth={2} /> : <ToggleLeft size={18} strokeWidth={2} />}
          </button>
          <button onClick={e => { e.stopPropagation(); navigate(`/rules/edit/${r.id}`); }} title="Редактировать" style={btnStyle}><Pencil size={15} strokeWidth={2} /></button>
          <button onClick={e => handleDelete(e, r.id)} title="Удалить" style={{ ...btnStyle, color: '#ff5f6d' }}><Trash2 size={15} strokeWidth={2} /></button>
        </div>
      ) },
  ];

  const tt = ({ active, payload }: { active?: boolean; payload?: Array<{ name: string; value: number; payload: { color?: string } }> }) => {
    if (!active || !payload?.length) return null;
    return <div style={ttStyle}><div style={{ color: payload[0].payload.color || 'var(--mint)', fontWeight: 700 }}>{payload[0].name}</div><div>{payload[0].value}</div></div>;
  };

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <PageHeader title="Правила" subtitle="Правила обнаружения и автоматические реакции"
        action={
          <button onClick={() => navigate('/rules/create')} style={createBtnStyle}
            onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.background = 'var(--mint-dim)'; }}
            onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.background = 'var(--mint)'; }}>
            <Plus size={16} strokeWidth={2.5} /> Создать правило
          </button>
        }
      />
      <div style={{ padding: '0 32px' }}>
        {error && (
          <div style={errorBox}>
            <AlertTriangle size={14} /> {error}
            <button onClick={() => setError('')} style={{ marginLeft: 'auto', background: 'none', border: 'none', color: '#ff5f6d', cursor: 'pointer' }}>✕</button>
          </div>
        )}
        {loading ? <Spinner /> : (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '14px', marginBottom: '28px' }}>
              <StatCard label="Всего правил"    value={rules.length} icon={<BookOpen size={18} />}    accent="#C3FDB8" />
              <StatCard label="Включено"         value={enabled}      icon={<ToggleRight size={18} />} accent="#C3FDB8" />
              <StatCard label="Отключено"        value={disabled}     icon={<ToggleLeft size={18} />}  accent="#888bac" />
              <StatCard label="Срабатывало"      value={rules.filter(r => (r.trigger_count ?? 0) > 0).length} icon={<AlertTriangle size={18} />} accent="#ffa94d" sub="хотя бы раз" />
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1.6fr', gap: '20px', marginBottom: '28px' }}>
              <div style={card}>
                <h3 style={cardTitle}>По важности</h3>
                <div style={{ display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
                  <div style={{ flex: '0 0 150px' }}>
                    {sevData.length === 0 ? <EmptyChart /> : (
                      <ResponsiveContainer width="100%" height={150}>
                        <PieChart>
                          <Pie data={sevData} dataKey="value" nameKey="name" cx="50%" cy="50%" innerRadius={38} outerRadius={62} paddingAngle={3}>
                            {sevData.map((e, i) => <Cell key={i} fill={e.color} stroke="transparent" />)}
                          </Pie>
                          <Tooltip content={tt as React.FC} />
                        </PieChart>
                      </ResponsiveContainer>
                    )}
                  </div>
                  <div style={{ flex: 1 }}><PieLegend items={sevData} /></div>
                </div>
              </div>
              <div style={card}>
                <h3 style={cardTitle}>Топ по срабатываниям</h3>
                {triggerData.length === 0 ? <EmptyChart /> : (
                  <ResponsiveContainer width="100%" height={220}>
                    <BarChart data={triggerData} layout="vertical" margin={{ top: 5, right: 10, left: 10, bottom: 0 }}>
                      <XAxis type="number" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} allowDecimals={false} />
                      <YAxis type="category" dataKey="name" tick={{ fill: 'var(--text-secondary)', fontSize: 10, fontFamily: 'var(--font-mono)' }} axisLine={false} tickLine={false} width={110} />
                      <Tooltip content={tt as React.FC} />
                      <Bar dataKey="value" fill="#C3FDB8" radius={[0, 4, 4, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                )}
              </div>
            </div>
            <div>
              <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '16px', marginBottom: '16px', display: 'flex', alignItems: 'center', gap: '8px' }}><BookOpen size={16} /> Список правил</h2>
              <PaginatedTable data={sortedRules} columns={columns} keyFn={r => r.id} onRowClick={r => setSelected(r)} />
            </div>
          </>
        )}
      </div>

      {selectedRule && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(10,14,30,0.85)', zIndex: 1000, display: 'flex', alignItems: 'flex-start', justifyContent: 'center', padding: '40px 20px', overflowY: 'auto' }}
          onClick={e => { if (e.target === e.currentTarget) setSelected(null); }}>
          <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '16px', width: '100%', maxWidth: '700px', maxHeight: '80vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', animation: 'fadeIn 0.2s ease' }}>
            <div style={{ padding: '18px 22px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <BookOpen size={16} color="var(--mint)" />
                <span style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '15px' }}>{selectedRule.name}</span>
                <SeverityBadge severity={selectedRule.severity} />
              </div>
              <button onClick={() => setSelected(null)} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', display: 'flex' }}><X size={18} /></button>
            </div>
            <div style={{ overflowY: 'auto', flex: 1 }}>
              <pre style={{ padding: '20px', margin: 0, fontFamily: 'var(--font-mono)', fontSize: '12px', lineHeight: '1.7', color: 'var(--mint)', background: 'var(--navy)', overflowX: 'auto' }}>
                {JSON.stringify(selectedRule, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function fmtDate(str?: string) {
  if (!str) return '—';
  const d = new Date(str);
  return isNaN(d.getTime()) ? '—' : d.toLocaleDateString('ru-RU');
}
function EmptyChart() { return <div style={{ height: '220px', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>Нет данных</div>; }
function Spinner() {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '300px', flexDirection: 'column', gap: '12px' }}>
      <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
      <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Загрузка правил...</span>
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  );
}

const btnStyle: React.CSSProperties = { background: 'transparent', border: '1px solid var(--navy-border)', color: 'var(--text-secondary)', cursor: 'pointer', padding: '5px 7px', borderRadius: '6px', display: 'flex', alignItems: 'center', transition: 'color var(--transition), background var(--transition)' };
const createBtnStyle: React.CSSProperties = { display: 'flex', alignItems: 'center', gap: '8px', padding: '10px 18px', borderRadius: '10px', border: 'none', background: 'var(--mint)', color: 'var(--navy)', fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '13px', cursor: 'pointer', letterSpacing: '0.02em', transition: 'background var(--transition)' };
const card: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '12px', padding: '20px' };
const cardTitle: React.CSSProperties = { fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '14px', color: 'var(--text-secondary)', textTransform: 'uppercase', letterSpacing: '0.08em' };
const errorBox: React.CSSProperties = { padding: '12px 16px', borderRadius: '8px', marginBottom: '20px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' };
const ttStyle: React.CSSProperties = { background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '8px', padding: '10px 14px', fontFamily: 'var(--font-mono)', fontSize: '12px' };
