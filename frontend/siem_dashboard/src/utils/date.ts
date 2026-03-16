export function fmtDate(s?: string | null): string {
  if (!s) return '—';
  // Убираем TZ-суффикс: Z, +03:00, -05:00 и т.п.
  const noTZ = s.replace(/([Zz]|[+-]\d{2}:?\d{2})$/, '');
  const d = new Date(noTZ);
  if (isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', {
    day: '2-digit', month: '2-digit', year: 'numeric',
    hour: '2-digit', minute: '2-digit', second: '2-digit',
  });
}
 
export function fmtDateShort(s?: string | null): string {
  if (!s) return '—';
  const noTZ = s.replace(/([Zz]|[+-]\d{2}:?\d{2})$/, '');
  const d = new Date(noTZ);
  if (isNaN(d.getTime())) return '—';
  return d.toLocaleString('ru-RU', {
    day: '2-digit', month: '2-digit',
    hour: '2-digit', minute: '2-digit',
  });
}
 
export function parseDate(s?: string | null): Date {
  if (!s) return new Date(0);
  const noTZ = s.replace(/([Zz]|[+-]\d{2}:?\d{2})$/, '');
  const d = new Date(noTZ);
  return isNaN(d.getTime()) ? new Date(0) : d;
}