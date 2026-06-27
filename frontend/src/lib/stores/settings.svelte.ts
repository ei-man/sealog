const DEFAULT_WIDTH = 180;
const MIN_WIDTH = 80;
const COMMON_FIELD_GROUPS = [
  ['timestamp', '@t', 'time', 'Timestamp', 'Time'],
  ['level', 'Level', 'severity', 'LogLevel', '@l'],
  ['message', 'Message', '@m', 'msg']
];

function defaultWidth(field: string): number {
  const lower = field.toLowerCase();
  if (lower.includes('message') || field === '@m' || lower === 'msg') return 420;
  if (lower.includes('timestamp') || lower === 'time' || field === '@t') return 190;
  if (lower === 'level' || lower === 'severity' || field === '@l') return 120;
  return DEFAULT_WIDTH;
}

function pickCommonFields(fields: string[]): string[] {
  const available = new Set(fields);
  const selected: string[] = [];
  for (const group of COMMON_FIELD_GROUPS) {
    const match = group.find((field) => available.has(field));
    if (match) selected.push(match);
  }
  return selected;
}

function arraysEqual(left: string[], right: string[]): boolean {
  if (left.length !== right.length) return false;
  return left.every((value, index) => value === right[index]);
}

class SettingsStore {
  selectedFields = $state<string[]>([]);
  columnWidths = $state<Record<string, number>>({});
  private autoSelectCommonFields = true;

  syncAvailableFields(fields: string[]) {
    const currentWidths = this.columnWidths;
    let nextWidths = currentWidths;
    for (const field of fields) {
      if (!nextWidths[field]) {
        if (nextWidths === currentWidths) {
          nextWidths = { ...currentWidths };
        }
        nextWidths[field] = defaultWidth(field);
      }
    }
    if (nextWidths !== currentWidths) this.columnWidths = nextWidths;

    if (this.autoSelectCommonFields) {
      const common = pickCommonFields(fields);
      if (!arraysEqual(this.selectedFields, common)) {
        this.selectedFields = common;
      }
    } else {
      const available = new Set(fields);
      const selectedFields = this.selectedFields.filter((field) => available.has(field));
      if (!arraysEqual(this.selectedFields, selectedFields)) {
        this.selectedFields = selectedFields;
      }
    }
  }

  toggleField(field: string) {
    this.autoSelectCommonFields = false;
    if (this.selectedFields.includes(field)) {
      this.selectedFields = this.selectedFields.filter((item) => item !== field);
    } else {
      this.selectedFields = [...this.selectedFields, field];
      if (!this.columnWidths[field]) this.setColumnWidth(field, defaultWidth(field));
    }
  }

  moveField(dragged: string, target: string, placement: 'before' | 'after' = 'before') {
    if (dragged === target) return;
    const current = this.selectedFields.filter((field) => field !== dragged);
    const targetIndex = current.indexOf(target);
    if (targetIndex === -1) return;
    current.splice(placement === 'after' ? targetIndex + 1 : targetIndex, 0, dragged);
    if (arraysEqual(this.selectedFields, current)) return;
    this.autoSelectCommonFields = false;
    this.selectedFields = current;
  }

  setColumnWidth(field: string, width: number) {
    const nextWidth = Math.max(MIN_WIDTH, Math.round(width));
    if (this.columnWidths[field] === nextWidth) return;

    this.columnWidths = {
      ...this.columnWidths,
      [field]: nextWidth
    };
  }

  widthFor(field: string): number {
    return this.columnWidths[field] ?? defaultWidth(field);
  }
}

export const settings = new SettingsStore();
