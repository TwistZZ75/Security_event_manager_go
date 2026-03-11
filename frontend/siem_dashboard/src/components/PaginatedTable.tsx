import { useState } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface Column<T> {
  key: string;
  header: string;
  render: (row: T) => React.ReactNode;
  width?: string;
}

interface PaginatedTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyFn: (row: T) => string | number;
  onRowClick?: (row: T) => void;
}

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100];

export default function PaginatedTable<T>({
  data,
  columns,
  keyFn,
  onRowClick,
}: PaginatedTableProps<T>) {
  const [pageSize, setPageSize] = useState(10);
  const [page, setPage] = useState(0);

  const totalPages = Math.ceil(data.length / pageSize);
  const startIdx = page * pageSize;
  const pageData = data.slice(startIdx, startIdx + pageSize);

  const handlePageSize = (val: number) => {
    setPageSize(val);
    setPage(0);
  };

  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '14px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>Show:</span>
          <select
            value={pageSize}
            onChange={e => handlePageSize(Number(e.target.value))}
            style={{ background: 'var(--navy-light)', border: '1px solid var(--navy-border)', color: 'var(--mint)', borderRadius: '6px', padding: '5px 10px', fontSize: '12px', fontFamily: 'var(--font-mono)', cursor: 'pointer', outline: 'none' }}
          >
            {PAGE_SIZE_OPTIONS.map(s => (
              <option key={s} value={s}>{s} rows</option>
            ))}
          </select>
        </div>
        <span style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>
          {data.length === 0 ? '0 records' : `${startIdx + 1}–${Math.min(startIdx + pageSize, data.length)} of ${data.length}`}
        </span>
      </div>

      <div style={{ border: '1px solid var(--navy-border)', borderRadius: '10px', overflow: 'hidden' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: 'var(--navy-lighter)' }}>
              {columns.map(col => (
                <th key={col.key} style={{
                  padding: '11px 14px', textAlign: 'left', fontSize: '11px',
                  fontFamily: 'var(--font-display)', fontWeight: 700, letterSpacing: '0.08em',
                  color: 'var(--text-secondary)', width: col.width, borderBottom: '1px solid var(--navy-border)',
                  textTransform: 'uppercase',
                }}>
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {pageData.length === 0 ? (
              <tr>
                <td colSpan={columns.length} style={{ padding: '32px', textAlign: 'center', color: 'var(--text-secondary)', fontSize: '13px' }}>
                  No records found
                </td>
              </tr>
            ) : (
              pageData.map((row, i) => (
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
              ))
            )}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '6px', marginTop: '14px' }}>
          <button onClick={() => setPage(p => Math.max(0, p - 1))} disabled={page === 0} style={btnStyle(page === 0)}>
            <ChevronLeft size={14} />
          </button>
          {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
            let pageIdx = i;
            if (totalPages > 7) {
              if (page < 4) pageIdx = i;
              else if (page > totalPages - 5) pageIdx = totalPages - 7 + i;
              else pageIdx = page - 3 + i;
            }
            return (
              <button key={pageIdx} onClick={() => setPage(pageIdx)} style={{ ...btnStyle(false), background: pageIdx === page ? 'var(--mint)' : 'transparent', color: pageIdx === page ? 'var(--navy)' : 'var(--mint)', fontWeight: pageIdx === page ? 700 : 400 }}>
                {pageIdx + 1}
              </button>
            );
          })}
          <button onClick={() => setPage(p => Math.min(totalPages - 1, p + 1))} disabled={page >= totalPages - 1} style={btnStyle(page >= totalPages - 1)}>
            <ChevronRight size={14} />
          </button>
        </div>
      )}
    </div>
  );
}

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
