import { useState, useMemo } from 'react';
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react';

// ─── Types ────────────────────────────────────────────────────────────────────

export interface Column<T> {
  key: string;
  header: string;
  render: (row: T) => React.ReactNode;
  width?: string;
  /**
   * Функция для сортировки. Если указана — заголовок колонки становится
   * кликабельным. Возвращает значение для сравнения (string | number | Date).
   */
  sortValue?: (row: T) => string | number | Date;
}

interface PaginatedTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyFn: (row: T) => string | number;
  onRowClick?: (row: T) => void;
  /** Размер страницы по умолчанию (default 20) */
  defaultPageSize?: number;
}

type SortDir = 'asc' | 'desc';

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

// ─── Component ────────────────────────────────────────────────────────────────

export default function PaginatedTable<T>({
  data,
  columns,
  keyFn,
  onRowClick,
  defaultPageSize = 20,
}: PaginatedTableProps<T>) {
  const [pageSize, setPageSize] = useState(defaultPageSize);
  const [page, setPage] = useState(0);
  const [sortKey, setSortKey] = useState<string | null>(null);
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const [goToInput, setGoToInput] = useState('');

  // ── Sorting ──────────────────────────────────────────────────────────────
  const sortedData = useMemo(() => {
    if (!sortKey) return data;
    const col = columns.find(c => c.key === sortKey);
    if (!col?.sortValue) return data;
    return [...data].sort((a, b) => {
      const va = col.sortValue!(a);
      const vb = col.sortValue!(b);
      let cmp = 0;
      if (va instanceof Date && vb instanceof Date) {
        cmp = va.getTime() - vb.getTime();
      } else if (typeof va === 'number' && typeof vb === 'number') {
        cmp = va - vb;
      } else {
        cmp = String(va).localeCompare(String(vb), 'ru');
      }
      return sortDir === 'asc' ? cmp : -cmp;
    });
  }, [data, sortKey, sortDir, columns]);

  const totalPages = Math.max(1, Math.ceil(sortedData.length / pageSize));
  const safePage = Math.min(page, totalPages - 1);
  const startIdx = safePage * pageSize;
  const pageData = sortedData.slice(startIdx, startIdx + pageSize);

  const handlePageSize = (val: number) => { setPageSize(val); setPage(0); };
  const goTo = (p: number) => setPage(Math.max(0, Math.min(totalPages - 1, p)));

  const handleSortClick = (col: Column<T>) => {
    if (!col.sortValue) return;
    if (sortKey === col.key) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortKey(col.key);
      setSortDir('asc');
    }
    setPage(0);
  };

  const handleGoTo = () => {
    const n = parseInt(goToInput, 10);
    if (!isNaN(n)) { goTo(n - 1); setGoToInput(''); }
  };

  // ─── Render ───────────────────────────────────────────────────────────────
  return (
    <div>
      {/* Top bar */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '12px', flexWrap: 'wrap', gap: '8px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Rows:</span>
          <select
            value={pageSize}
            onChange={e => handlePageSize(Number(e.target.value))}
            style={selectStyle}
          >
            {PAGE_SIZE_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <span style={{ fontSize: '12px', color: 'var(--text-secondary)', fontFamily: 'var(--font-mono)' }}>
          {sortedData.length === 0 ? '0 записей' : `${startIdx + 1}–${Math.min(startIdx + pageSize, sortedData.length)} из ${sortedData.length}`}
        </span>
      </div>

      {/* Table */}
      <div style={{ border: '1px solid var(--navy-border)', borderRadius: '10px', overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', tableLayout: 'fixed'  }}>
          <thead>
            <tr style={{ background: 'var(--navy-lighter)' }}>
              {columns.map(col => {
                const sortable = !!col.sortValue;
                const isActive = sortKey === col.key;
                return (
                  <th
                    key={col.key}
                    onClick={() => handleSortClick(col)}
                    title={sortable ? 'Кликните для сортировки' : undefined}
                    style={{
                      padding: '11px 14px', textAlign: 'left',
                      fontSize: '11px', fontFamily: 'var(--font-display)',
                      fontWeight: 700, letterSpacing: '0.08em',
                      color: isActive ? 'var(--mint)' : 'var(--text-secondary)',
                      width: col.width, borderBottom: '1px solid var(--navy-border)',
                      textTransform: 'uppercase',
                      cursor: sortable ? 'pointer' : 'default',
                      userSelect: 'none',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
                      {col.header}
                      {sortable && (
                        <span style={{ opacity: isActive ? 1 : 0.35, display: 'flex' }}>
                          {isActive
                            ? (sortDir === 'asc' ? <ArrowUp size={11} /> : <ArrowDown size={11} />)
                            : <ArrowUpDown size={11} />}
                        </span>
                      )}
                    </div>
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {pageData.length === 0 ? (
              <tr>
                <td colSpan={columns.length} style={{ padding: '32px', textAlign: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>
                  Нет записей
                </td>
              </tr>
            ) : pageData.map((row, i) => (
              <tr
                key={keyFn(row)}
                onClick={() => onRowClick?.(row)}
                style={{
                  borderBottom: i < pageData.length - 1 ? '1px solid var(--navy-border)' : 'none',
                  transition: 'background var(--transition)',
                  cursor: onRowClick ? 'pointer' : 'default',
                }}
                onMouseEnter={e => (e.currentTarget.style.background = 'var(--mint-glow)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
              >
                {columns.map(col => (
                  <td key={col.key} style={{ padding: '11px 14px', fontSize: '13px', verticalAlign: 'middle' }}>
                    {col.render(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '5px', marginTop: '14px', flexWrap: 'wrap' }}>
          {/* First */}
          <button onClick={() => goTo(0)} disabled={safePage === 0} title="Первая страница" style={btnStyle(safePage === 0)}>
            <ChevronsLeft size={14} />
          </button>
          {/* Prev */}
          <button onClick={() => goTo(safePage - 1)} disabled={safePage === 0} title="Предыдущая" style={btnStyle(safePage === 0)}>
            <ChevronLeft size={14} />
          </button>

          {/* Page numbers */}
          {buildPageRange(safePage, totalPages).map((item, idx) =>
            item === '…' ? (
              <span key={`ellipsis-${idx}`} style={{ color: 'var(--text-secondary)', padding: '0 4px', fontSize: '12px' }}>…</span>
            ) : (
              <button
                key={item}
                onClick={() => goTo(item as number)}
                style={{
                  ...btnStyle(false),
                  background: item === safePage ? 'var(--mint)' : 'transparent',
                  color: item === safePage ? 'var(--navy)' : 'var(--mint)',
                  fontWeight: item === safePage ? 700 : 400,
                  minWidth: '30px',
                }}
              >
                {(item as number) + 1}
              </button>
            )
          )}

          {/* Next */}
          <button onClick={() => goTo(safePage + 1)} disabled={safePage >= totalPages - 1} title="Следующая" style={btnStyle(safePage >= totalPages - 1)}>
            <ChevronRight size={14} />
          </button>
          {/* Last */}
          <button onClick={() => goTo(totalPages - 1)} disabled={safePage >= totalPages - 1} title="Последняя страница" style={btnStyle(safePage >= totalPages - 1)}>
            <ChevronsRight size={14} />
          </button>

          {/* Go-to page */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '5px', marginLeft: '8px' }}>
            <span style={{ fontSize: '11px', color: 'var(--text-secondary)' }}>стр.</span>
            <input
              type="number"
              min={1}
              max={totalPages}
              value={goToInput}
              onChange={e => setGoToInput(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleGoTo()}
              placeholder={String(safePage + 1)}
              style={{
                width: '48px', padding: '4px 6px', borderRadius: '6px',
                background: 'var(--navy-light)', border: '1px solid var(--navy-border)',
                color: 'var(--mint)', fontFamily: 'var(--font-mono)', fontSize: '12px',
                outline: 'none', textAlign: 'center',
              }}
            />
            <button onClick={handleGoTo} style={{ ...btnStyle(false), fontSize: '11px', width: 'auto', padding: '0 8px' }}>
              →
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

/** Строит массив номеров страниц с многоточием */
function buildPageRange(current: number, total: number): (number | '…')[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i);
  const pages: (number | '…')[] = [];
  const addPage = (p: number) => { if (!pages.includes(p)) pages.push(p); };
  const addEllipsis = () => { if (pages[pages.length - 1] !== '…') pages.push('…'); };

  addPage(0);
  if (current > 3) addEllipsis();
  for (let i = Math.max(1, current - 2); i <= Math.min(total - 2, current + 2); i++) addPage(i);
  if (current < total - 4) addEllipsis();
  addPage(total - 1);
  return pages;
}

const selectStyle: React.CSSProperties = {
  background: 'var(--navy-light)', border: '1px solid var(--navy-border)',
  color: 'var(--mint)', borderRadius: '6px', padding: '5px 10px',
  fontSize: '12px', fontFamily: 'var(--font-mono)', cursor: 'pointer', outline: 'none',
};

function btnStyle(disabled: boolean): React.CSSProperties {
  return {
    display: 'flex', alignItems: 'center', justifyContent: 'center',
    width: '30px', height: '30px', borderRadius: '6px',
    border: '1px solid var(--navy-border)', background: 'transparent',
    color: disabled ? 'var(--navy-border)' : 'var(--mint)',
    cursor: disabled ? 'not-allowed' : 'pointer',
    fontFamily: 'var(--font-mono)', fontSize: '12px',
    transition: 'background var(--transition)', opacity: disabled ? 0.5 : 1,
  };
}
