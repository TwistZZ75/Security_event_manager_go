import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, HelpCircle, X, Save, AlertTriangle, CheckCircle } from 'lucide-react';
import { createRule, updateRule, getRule } from '../api/api';

const DEFAULT_RULE = {
  name: "Новое правило",
  description: "Описание правила",
  severity: "medium",
  enabled: true,
  tags: [],
  conditions: [{ field: "event_category", operator: "equals", value: "authentication" }],
  aggregation: { type: "count", field: "username", threshold: 5, time_window: "5m" },
  actions: [{ type: "notify", parameters: { message: "Правило сработало", channel: "security-alerts" } }],
};

const HELP_CONTENT = `# Руководство по написанию правил

## Основные поля

| Поле        | Тип     | Описание                                       |
|-------------|---------|------------------------------------------------|
| name        | string  | Уникальное название правила                    |
| description | string  | Описание для людей                             |
| severity    | string  | low · medium · high · critical                 |
| enabled     | boolean | Включено ли правило                            |
| tags        | array   | Метки для фильтрации                           |

---

## Операторы условий

**Строковые операторы:**
- equals / not_equals
- contains / not_contains
- starts_with / ends_with
- regex — регулярное выражение
- in / not_in — значение должно быть массивом

**Числовые операторы:**
- greater_than / less_than
- greater_or_equal / less_or_equal

**Сетевые операторы:**
- ip_equals — точное совпадение IP
- ip_in_range — CIDR (например "192.168.0.0/24")

**Доступные поля событий:**
username, pc_name, ip_address, event_description, event_category, process_name, severity, os, source

---

## Типы агрегации

### count
Срабатывает когда количество событий превышает порог в заданном окне времени.

### threshold
Срабатывает по числовому значению поля.

**Формат временного окна:** 30s, 5m, 1h, 24h

---

## Типы действий

| Тип                | Описание                           |
|--------------------|------------------------------------|
| notify             | Отправить уведомление              |
| block_account      | Заблокировать аккаунт              |
| unblock_account    | Разблокировать аккаунт             |
| block_network      | Заблокировать сеть/IP              |
| unblock_network    | Снять блокировку сети              |
| kill_process       | Завершить процесс                  |

---

## Советы
- Первое тестирование делайте с "enabled": false
- Поле "field" в агрегации группирует счётчики по пользователю или хосту
- Чем точнее условия — тем меньше ложных срабатываний
`;

export default function CreateRulePage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id?: string }>();
  const isEdit = !!id;

  const [json, setJson]           = useState(JSON.stringify(DEFAULT_RULE, null, 2));
  const [showHelp, setShowHelp]   = useState(false);
  const [jsonError, setJsonError] = useState('');
  const [saving, setSaving]       = useState(false);
  const [saved, setSaved]         = useState(false);
  const [saveError, setSaveError] = useState('');
  const [loadingRule, setLoading] = useState(isEdit);

  useEffect(() => {
    if (!isEdit || !id) return;
    setLoading(true);
    getRule(id)
      .then(rule => setJson(JSON.stringify(rule, null, 2)))
      .catch(e => setSaveError('Ошибка загрузки правила: ' + e.message))
      .finally(() => setLoading(false));
  }, [id, isEdit]);

  const validateJson = (val: string) => {
    try { JSON.parse(val); setJsonError(''); return true; }
    catch (e) { setJsonError(e instanceof Error ? e.message : 'Неверный JSON'); return false; }
  };

  const handleChange = (val: string) => {
    setJson(val);
    if (val.trim()) validateJson(val);
    else setJsonError('');
  };

  const handleSave = async () => {
    if (!validateJson(json)) return;
    setSaving(true);
    setSaveError('');
    try {
      const parsed = JSON.parse(json);
      if (isEdit && id) { await updateRule(id, parsed); }
      else { await createRule(parsed); }
      setSaved(true);
      setTimeout(() => navigate('/rules'), 1200);
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Ошибка сохранения';
      if (msg.toLowerCase().includes('already exists') || msg.toLowerCase().includes('duplicate')) {
        const name = (() => { try { return JSON.parse(json).name ?? ''; } catch { return ''; } })();
        setSaveError(`Правило с названием "${name}" уже существует. Выберите другое название.`);
      } else {
        setSaveError(msg);
      }
    } finally {
      setSaving(false);
    }
  };

  const handleFormat = () => {
    try { setJson(JSON.stringify(JSON.parse(json), null, 2)); setJsonError(''); } catch {}
  };

  if (loadingRule) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '400px', flexDirection: 'column', gap: '12px' }}>
        <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
        <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Загрузка правила...</span>
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
      </div>
    );
  }

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      <div style={{ padding: '28px 32px 0', display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
          <button onClick={() => navigate('/rules')}
            style={{ background: 'transparent', border: '1px solid var(--navy-border)', color: 'var(--mint)', borderRadius: '8px', padding: '8px', cursor: 'pointer', display: 'flex', alignItems: 'center' }}>
            <ArrowLeft size={16} />
          </button>
          <div>
            <h1 style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '24px', letterSpacing: '-0.01em' }}>
              {isEdit ? 'Редактировать правило' : 'Создать правило'}
            </h1>
            <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>
              {isEdit ? `Редактирование: ${id}` : 'Создание нового правила обнаружения'}
            </p>
          </div>
        </div>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={() => setShowHelp(true)} style={headerBtn}><HelpCircle size={15} /> Справка</button>
          <button onClick={handleFormat} style={{ ...headerBtn, opacity: jsonError ? 0.5 : 1 }}>⌥ Форматировать JSON</button>
          <button onClick={handleSave} disabled={saving || !!jsonError || saved}
            style={{ display: 'flex', alignItems: 'center', gap: '7px', padding: '9px 18px', borderRadius: '9px', border: 'none', background: saved ? '#2a4a2a' : (saving || jsonError ? 'var(--mint-dim)' : 'var(--mint)'), color: saved ? '#C3FDB8' : 'var(--navy)', cursor: saving || !!jsonError ? 'not-allowed' : 'pointer', fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '13px', transition: 'background var(--transition)' }}>
            {saved ? <><CheckCircle size={15} /> Сохранено!</> : saving ? 'Сохранение...' : <><Save size={15} /> {isEdit ? 'Обновить правило' : 'Сохранить правило'}</>}
          </button>
        </div>
      </div>

      <div style={{ padding: '0 32px' }}>
        {saveError && (
          <div style={{ padding: '12px 16px', borderRadius: '8px', marginBottom: '16px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' }}>
            <AlertTriangle size={14} /> {saveError}
          </div>
        )}

        <div style={{ background: 'var(--navy-light)', border: `1px solid ${jsonError ? '#ff5f6d44' : 'var(--navy-border)'}`, borderRadius: '12px', overflow: 'hidden' }}>
          <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'var(--navy-lighter)' }}>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' }}>
                {isEdit ? `rule-${id}.json` : 'new-rule.json'}
              </span>
              {!jsonError && json.trim() && <span style={{ fontSize: '10px', color: '#C3FDB8', background: 'rgba(195,253,184,0.1)', padding: '2px 8px', borderRadius: '10px', fontFamily: 'var(--font-display)', fontWeight: 700 }}>✓ Валидный JSON</span>}
            </div>
            {jsonError && <span style={{ fontSize: '11px', color: '#ff5f6d', fontFamily: 'var(--font-mono)' }}>⚠ {jsonError}</span>}
          </div>

          <div style={{ position: 'relative', display: 'flex' }}>
            <div style={{ padding: '16px 10px 16px 8px', minWidth: '48px', textAlign: 'right', borderRight: '1px solid var(--navy-border)', background: 'var(--navy)', fontFamily: 'var(--font-mono)', fontSize: '13px', lineHeight: '22px', color: 'var(--navy-border)', userSelect: 'none' }}>
              {json.split('\n').map((_, i) => <div key={i}>{i + 1}</div>)}
            </div>
            <textarea value={json} onChange={e => handleChange(e.target.value)} spellCheck={false}
              style={{ flex: 1, padding: '16px', background: 'var(--navy)', border: 'none', color: 'var(--mint)', fontFamily: 'var(--font-mono)', fontSize: '13px', lineHeight: '22px', resize: 'vertical', outline: 'none', minHeight: '500px', tabSize: 2 }}
              onKeyDown={e => {
                if (e.key === 'Tab') {
                  e.preventDefault();
                  const start = e.currentTarget.selectionStart;
                  const end = e.currentTarget.selectionEnd;
                  const newVal = json.substring(0, start) + '  ' + json.substring(end);
                  setJson(newVal);
                  setTimeout(() => { e.currentTarget.selectionStart = e.currentTarget.selectionEnd = start + 2; });
                }
              }} />
          </div>
        </div>
      </div>

      {showHelp && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(10,14,30,0.85)', zIndex: 1000, display: 'flex', alignItems: 'flex-start', justifyContent: 'center', padding: '40px 20px', overflowY: 'auto' }}
          onClick={e => { if (e.target === e.currentTarget) setShowHelp(false); }}>
          <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '16px', width: '100%', maxWidth: '720px', maxHeight: '80vh', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
            <div style={{ padding: '20px 24px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <HelpCircle size={18} color="var(--mint)" />
                <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '16px' }}>Справка по правилам</h2>
              </div>
              <button onClick={() => setShowHelp(false)} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', display: 'flex' }}><X size={18} /></button>
            </div>
            <div style={{ padding: '24px', overflowY: 'auto', flex: 1 }}>
              <HelpContent content={HELP_CONTENT} />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function HelpContent({ content }: { content: string }) {
  const lines = content.split('\n');
  return (
    <div style={{ fontFamily: 'var(--font-mono)', fontSize: '13px', lineHeight: '1.7' }}>
      {lines.map((line, i) => {
        if (line.startsWith('# '))   return <h2 key={i} style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '18px', marginBottom: '16px', marginTop: i > 0 ? '24px' : 0 }}>{line.slice(2)}</h2>;
        if (line.startsWith('## '))  return <h3 key={i} style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '10px', marginTop: '20px', color: 'var(--mint)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{line.slice(3)}</h3>;
        if (line.startsWith('### ')) return <h4 key={i} style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '13px', marginBottom: '8px', marginTop: '16px' }}>{line.slice(4)}</h4>;
        if (line.startsWith('---')) return <hr key={i} style={{ border: 'none', borderTop: '1px solid var(--navy-border)', margin: '20px 0' }} />;
        if (line.startsWith('| ')) return (
          <div key={i} style={{ padding: '5px 0', borderBottom: '1px solid var(--navy-border)', display: 'flex', gap: '16px', fontSize: '12px' }}>
            {line.split('|').filter((_, ci) => ci > 0 && ci < line.split('|').length - 1).map((cell, ci) => (
              <span key={ci} style={{ minWidth: ci === 0 ? '140px' : '80px', color: ci === 0 ? 'var(--mint)' : 'var(--text-secondary)' }}>{cell.trim()}</span>
            ))}
          </div>
        );
        if (line.startsWith('- ')) return <div key={i} style={{ padding: '2px 0 2px 16px', color: 'var(--text-secondary)', fontSize: '12px' }}>· {line.slice(2)}</div>;
        if (line.trim() === '') return <div key={i} style={{ height: '8px' }} />;
        return <p key={i} style={{ color: 'var(--text-secondary)', fontSize: '12px', marginBottom: '4px' }}>{line}</p>;
      })}
    </div>
  );
}

const headerBtn: React.CSSProperties = { display: 'flex', alignItems: 'center', gap: '7px', padding: '9px 16px', borderRadius: '9px', border: '1px solid var(--navy-border)', background: 'transparent', color: 'var(--mint)', cursor: 'pointer', fontFamily: 'var(--font-display)', fontWeight: 600, fontSize: '13px' };
