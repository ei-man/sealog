import type { LogLine } from './types';

export const TIMESTAMP_FIELDS = new Set(['timestamp', '@t', 'Timestamp', 'time', 'Time', 'ingested_at']);
export const LEVEL_FIELDS = ['level', 'Level', 'severity', 'severityLevel', 'LogLevel', '@l'];
export const MESSAGE_FIELDS = new Set(['message', 'Message', '@m', 'msg']);

export type LevelTone = 'error' | 'warn' | 'info' | 'debug' | 'trace' | 'none';

export function valueToText(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

export function getPathValue(fields: Record<string, unknown>, path: string): unknown {
  if (Object.prototype.hasOwnProperty.call(fields, path)) return fields[path];

  const parts = path.split('.');
  let current: unknown = fields;
  for (const part of parts) {
    if (typeof current !== 'object' || current === null) return undefined;
    const record = current as Record<string, unknown>;
    if (!Object.prototype.hasOwnProperty.call(record, part)) return undefined;
    current = record[part];
  }
  return current;
}

export function formatCellValue(value: unknown, field: string, includeDate: boolean): string {
  if (TIMESTAMP_FIELDS.has(field)) {
    const formatted = formatTimestamp(value, includeDate);
    if (formatted) return formatted;
  }
  return valueToText(value);
}

export function parseTimestamp(value: unknown): Date | null {
  if (typeof value === 'string') {
    const parsed = new Date(value);
    if (!Number.isNaN(parsed.getTime())) return parsed;
    return null;
  }

  if (typeof value === 'number') {
    const millis = Math.abs(value) < 10_000_000_000 ? value * 1000 : value;
    const parsed = new Date(millis);
    if (!Number.isNaN(parsed.getTime())) return parsed;
  }

  return null;
}

export function formatTimestamp(value: unknown, includeDate: boolean): string | null {
  const date = parseTimestamp(value);
  if (!date) return null;
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  const seconds = String(date.getSeconds()).padStart(2, '0');
  const millis = String(date.getMilliseconds()).padStart(3, '0');
  const time = `${hours}:${minutes}:${seconds}.${millis}`;
  if (!includeDate) return time;

  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day} ${time}`;
}

export function timestampsSpanMultipleDays(lines: LogLine[]): boolean {
  let firstDay: string | null = null;
  for (const line of lines) {
    for (const field of TIMESTAMP_FIELDS) {
      const date = parseTimestamp(getPathValue(line.fields, field));
      if (!date) continue;
      const day = `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
      if (!firstDay) firstDay = day;
      if (firstDay !== day) return true;
      break;
    }
  }
  return false;
}

export function isLevelField(field: string): boolean {
  return LEVEL_FIELDS.includes(field);
}

export function levelToneFromValue(value: unknown): LevelTone {
  const text = valueToText(value).toLocaleLowerCase();
  if (!text) return 'none';
  if (['error', 'fatal', 'critical'].some((level) => text.includes(level))) return 'error';
  if (text.includes('warn')) return 'warn';
  if (text === 'info' || text === 'information' || text.includes('info')) return 'info';
  if (text.includes('debug')) return 'debug';
  if (text.includes('trace') || text.includes('verbose')) return 'trace';
  return 'none';
}

export function levelTone(line: LogLine): LevelTone {
  let value = '';
  for (const field of LEVEL_FIELDS) {
    const fieldValue = getPathValue(line.fields, field);
    if (fieldValue !== undefined) {
      value = valueToText(fieldValue);
      break;
    }
  }

  return levelToneFromValue(value);
}

export function quoteFilterValue(value: unknown): string {
  return valueToText(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
}
