import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, HelpCircle, X, Save, AlertTriangle, CheckCircle } from 'lucide-react';
import { createRule, updateRule, getRule } from '../api/api';

const DEFAULT_RULE = {
  name: "New Rule",
  description: "Describe what this rule detects",
  severity: "medium",
  enabled: true,
  tags: [],
  conditions: [
    {
      field: "event_category",
      operator: "equals",
      value: "authentication"
    }
  ],
  aggregation: {
    type: "count",
    threshold: 5,
    time_window: "5m",
    group_by: ["username"]
  },
  actions: [
    {
      type: "notify",
      parameters: {
        message: "Rule triggered",
        channel: "security-alerts"
      }
    }
  ]
};

const HELP_CONTENT = `# Rule Writing Guide

## Top-Level Fields

| Field       | Type    | Description                                      |
|-------------|---------|--------------------------------------------------|
| name        | string  | Unique rule name                                 |
| description | string  | Human-readable description                       |
| severity    | string  | low · medium · high · critical                   |
| enabled     | boolean | Whether the rule is active                       |
| tags        | array   | Optional labels for filtering                    |

---

## Condition Operators

**String operators:**
- equals / not_equals
- contains / not_contains
- starts_with / ends_with
- regex — full regex pattern
- in / not_in — value must be array

**Numeric operators:**
- greater_than / less_than
- greater_or_equal / less_or_equal

**Network operators:**
- ip_equals — exact IP match
- ip_in_range — CIDR notation (e.g. "192.168.0.0/24")

**Available event fields:**
username, pc_name, ip_address, event_description, event_category, process_name, severity, os, source

---

## Aggregation Types

### count
Triggers when event count exceeds threshold within time window.

### threshold
Triggers based on a numeric field value exceeding a threshold.

### sequence
Triggers when events occur in sequence within time window.

**Time window format:** 30s, 5m, 1h, 24h

---

## Action Types

| Type               | Description                            |
|--------------------|----------------------------------------|
| notify             | Send notification                      |
| block_account      | Disable user account                   |
| unblock_account    | Re-enable user account                 |
| block_network      | Block IP/network at firewall           |
| unblock_network    | Unblock IP/network                     |
| isolate_host       | Isolate host from network              |
| kill_process       | Terminate a running process            |
| quarantine_file    | Move file to quarantine                |
| run_script         | Execute a script on agent              |

Use {{field_name}} in parameters to interpolate event fields, e.g. {{username}}.

---

## Tips
- Always test rules with "enabled": false first
- Use group_by in aggregation to correlate per-user or per-host
- Keep conditions specific to reduce false positives
`;

export default function CreateRulePage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id?: string }>();
  const isEditMode = !!id;

  const [json, setJson] = useState(JSON.stringify(DEFAULT_RULE, null, 2));
  const [showHelp, setShowHelp] = useState(false);
  const [jsonError, setJsonError] = useState('');
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [loadingRule, setLoadingRule] = useState(isEditMode);

  // Load rule for edit mode
  useEffect(() => {
    if (!isEditMode || !id) return;
    setLoadingRule(true);
    getRule(id)
      .then(rule => {
        setJson(JSON.stringify(rule, null, 2));
      })
      .catch(e => {
        setSaveError('Failed to load rule: ' + e.message);
      })
      .finally(() => setLoadingRule(false));
  }, [id, isEditMode]);

  const validateJson = (val: string) => {
    try {
      JSON.parse(val);
      setJsonError('');
      return true;
    } catch (e) {
      setJsonError(e instanceof Error ? e.message : 'Invalid JSON');
      return false;
    }
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
      if (isEditMode && id) {
        await updateRule(id, parsed);
      } else {
        await createRule(parsed);
      }
      setSaved(true);
      setTimeout(() => navigate('/rules'), 1200);
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const handleFormat = () => {
    try {
      setJson(JSON.stringify(JSON.parse(json), null, 2));
      setJsonError('');
    } catch {
      // Keep current value if invalid JSON
    }
  };

  if (loadingRule) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '400px', flexDirection: 'column', gap: '12px' }}>
        <div style={{ width: '36px', height: '36px', borderRadius: '50%', border: '3px solid var(--navy-border)', borderTopColor: 'var(--mint)', animation: 'spin 0.8s linear infinite' }} />
        <span style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>Loading rule...</span>
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
      </div>
    );
  }

  return (
    <div style={{ padding: '0 0 40px' }} className="animate-in">
      {/* Header */}
      <div style={{ padding: '28px 32px 0', display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '24px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '14px' }}>
          <button
            onClick={() => navigate('/rules')}
            style={{ background: 'transparent', border: '1px solid var(--navy-border)', color: 'var(--mint)', borderRadius: '8px', padding: '8px', cursor: 'pointer', display: 'flex', alignItems: 'center' }}
          >
            <ArrowLeft size={16} />
          </button>
          <div>
            <h1 style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '24px', letterSpacing: '-0.01em' }}>
              {isEditMode ? 'Edit Rule' : 'Create Rule'}
            </h1>
            <p style={{ color: 'var(--text-secondary)', fontSize: '13px' }}>
              {isEditMode ? `Editing rule: ${id}` : 'Define a new detection rule'}
            </p>
          </div>
        </div>

        <div style={{ display: 'flex', gap: '10px' }}>
          <button onClick={() => setShowHelp(true)} style={headerBtnStyle}>
            <HelpCircle size={15} /> Help
          </button>
          <button
            onClick={handleFormat}
            title="Prettify / re-indent JSON"
            style={{ ...headerBtnStyle, opacity: jsonError ? 0.5 : 1 }}
          >
            ⌥ Format JSON
          </button>
          <button
            onClick={handleSave}
            disabled={saving || !!jsonError || saved}
            style={{
              display: 'flex', alignItems: 'center', gap: '7px',
              padding: '9px 18px', borderRadius: '9px', border: 'none',
              background: saved ? '#2a4a2a' : (saving || jsonError ? 'var(--mint-dim)' : 'var(--mint)'),
              color: saved ? '#C3FDB8' : 'var(--navy)',
              cursor: saving || !!jsonError ? 'not-allowed' : 'pointer',
              fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '13px',
              transition: 'background var(--transition)',
            }}
          >
            {saved
              ? <><CheckCircle size={15} /> Saved!</>
              : saving ? 'Saving...' : <><Save size={15} /> {isEditMode ? 'Update Rule' : 'Save Rule'}</>
            }
          </button>
        </div>
      </div>

      <div style={{ padding: '0 32px' }}>
        {saveError && (
          <div style={{ padding: '12px 16px', borderRadius: '8px', marginBottom: '16px', background: 'rgba(255,95,109,0.1)', border: '1px solid rgba(255,95,109,0.25)', color: '#ff5f6d', fontSize: '13px', display: 'flex', gap: '8px', alignItems: 'center' }}>
            <AlertTriangle size={14} /> {saveError}
          </div>
        )}

        {/* Editor */}
        <div style={{
          background: 'var(--navy-light)',
          border: `1px solid ${jsonError ? '#ff5f6d44' : 'var(--navy-border)'}`,
          borderRadius: '12px', overflow: 'hidden',
        }}>
          {/* Toolbar */}
          <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: 'var(--navy-lighter)' }}>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <span style={{ fontSize: '11px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' }}>
                {isEditMode ? `rule-${id}.json` : 'new-rule.json'}
              </span>
              {!jsonError && json.trim() && (
                <span style={{ fontSize: '10px', color: '#C3FDB8', background: 'rgba(195,253,184,0.1)', padding: '2px 8px', borderRadius: '10px', fontFamily: 'var(--font-display)', fontWeight: 700 }}>
                  ✓ Valid JSON
                </span>
              )}
            </div>
            {jsonError && (
              <span style={{ fontSize: '11px', color: '#ff5f6d', fontFamily: 'var(--font-mono)' }}>⚠ {jsonError}</span>
            )}
          </div>

          {/* Editor body */}
          <div style={{ position: 'relative', display: 'flex' }}>
            <div style={{
              padding: '16px 0', minWidth: '48px', textAlign: 'right',
              borderRight: '1px solid var(--navy-border)', background: 'var(--navy)',
              fontFamily: 'var(--font-mono)', fontSize: '13px', lineHeight: '22px',
              color: 'var(--navy-border)', userSelect: 'none', paddingRight: '10px', paddingLeft: '8px',
            }}>
              {json.split('\n').map((_, i) => <div key={i}>{i + 1}</div>)}
            </div>
            <textarea
              value={json}
              onChange={e => handleChange(e.target.value)}
              spellCheck={false}
              style={{
                flex: 1, padding: '16px', background: 'var(--navy)', border: 'none',
                color: 'var(--mint)', fontFamily: 'var(--font-mono)', fontSize: '13px',
                lineHeight: '22px', resize: 'vertical', outline: 'none',
                minHeight: '500px', tabSize: 2,
              }}
              onKeyDown={e => {
                if (e.key === 'Tab') {
                  e.preventDefault();
                  const start = e.currentTarget.selectionStart;
                  const end = e.currentTarget.selectionEnd;
                  const newVal = json.substring(0, start) + '  ' + json.substring(end);
                  setJson(newVal);
                  setTimeout(() => { e.currentTarget.selectionStart = e.currentTarget.selectionEnd = start + 2; });
                }
              }}
            />
          </div>
        </div>
      </div>

      {/* Help Modal */}
      {showHelp && (
        <div
          style={{ position: 'fixed', inset: 0, background: 'rgba(10,14,30,0.85)', zIndex: 1000, display: 'flex', alignItems: 'flex-start', justifyContent: 'center', padding: '40px 20px', overflowY: 'auto' }}
          onClick={e => { if (e.target === e.currentTarget) setShowHelp(false); }}
        >
          <div style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', borderRadius: '16px', width: '100%', maxWidth: '720px', maxHeight: '80vh', overflow: 'hidden', display: 'flex', flexDirection: 'column', animation: 'fadeIn 0.2s ease' }}>
            <div style={{ padding: '20px 24px', borderBottom: '1px solid var(--navy-border)', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <HelpCircle size={18} color="var(--mint)" />
                <h2 style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '16px' }}>Rule Writing Guide</h2>
              </div>
              <button onClick={() => setShowHelp(false)} style={{ background: 'transparent', border: 'none', color: 'var(--text-secondary)', cursor: 'pointer', display: 'flex' }}>
                <X size={18} />
              </button>
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

const headerBtnStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', gap: '7px',
  padding: '9px 16px', borderRadius: '9px',
  border: '1px solid var(--navy-border)', background: 'transparent',
  color: 'var(--mint)', cursor: 'pointer',
  fontFamily: 'var(--font-display)', fontWeight: 600, fontSize: '13px',
};

function HelpContent({ content }: { content: string }) {
  const lines = content.split('\n');
  return (
    <div style={{ fontFamily: 'var(--font-mono)', fontSize: '13px', lineHeight: '1.7' }}>
      {lines.map((line, i) => {
        if (line.startsWith('# ')) return <h2 key={i} style={{ fontFamily: 'var(--font-display)', fontWeight: 800, fontSize: '18px', marginBottom: '16px', marginTop: i > 0 ? '24px' : 0 }}>{line.slice(2)}</h2>;
        if (line.startsWith('## ')) return <h3 key={i} style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '14px', marginBottom: '10px', marginTop: '20px', color: 'var(--mint)', textTransform: 'uppercase', letterSpacing: '0.06em' }}>{line.slice(3)}</h3>;
        if (line.startsWith('### ')) return <h4 key={i} style={{ fontFamily: 'var(--font-display)', fontWeight: 700, fontSize: '13px', marginBottom: '8px', marginTop: '16px' }}>{line.slice(4)}</h4>;
        if (line.startsWith('---')) return <hr key={i} style={{ border: 'none', borderTop: '1px solid var(--navy-border)', margin: '20px 0' }} />;
        if (line.startsWith('| ')) return (
          <div key={i} style={{ padding: '5px 0', borderBottom: '1px solid var(--navy-border)', display: 'flex', gap: '16px', fontSize: '12px' }}>
            {line.split('|').filter((_, ci) => ci > 0 && ci < line.split('|').length - 1).map((cell, ci) => (
              <span key={ci} style={{ minWidth: ci === 0 ? '140px' : '80px', color: ci === 0 ? 'var(--mint)' : 'var(--text-secondary)' }}>{cell.trim()}</span>
            ))}
          </div>
        );
        if (line.startsWith('- ')) return <div key={i} style={{ padding: '2px 0 2px 16px', color: 'var(--text-secondary)', fontSize: '12px' }}>· {renderInline(line.slice(2))}</div>;
        if (line.trim() === '') return <div key={i} style={{ height: '8px' }} />;
        return <p key={i} style={{ color: 'var(--text-secondary)', fontSize: '12px', marginBottom: '4px' }}>{renderInline(line)}</p>;
      })}
    </div>
  );
}

function renderInline(text: string): React.ReactNode {
  const parts = text.split(/(`[^`]+`)/g);
  return parts.map((part, i) => {
    if (part.startsWith('`') && part.endsWith('`')) {
      return <code key={i} style={{ background: 'var(--navy)', color: 'var(--mint)', padding: '1px 6px', borderRadius: '4px', fontSize: '11px' }}>{part.slice(1, -1)}</code>;
    }
    const boldParts = part.split(/(\*\*[^*]+\*\*)/g);
    return boldParts.map((bp, bi) => bp.startsWith('**') && bp.endsWith('**')
      ? <strong key={bi} style={{ color: 'var(--mint)' }}>{bp.slice(2, -2)}</strong>
      : bp
    );
  });
}
