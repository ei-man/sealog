import type { LogLine } from '../types';

class LogStore {
  lines = $state<LogLine[]>([]);
  fields = $state<string[]>([]);
  connected = $state(false);

  beginSnapshot() {
    this.lines = [];
  }

  setConnected(value: boolean) {
    this.connected = value;
  }

  setFields(fields: string[]) {
    this.fields = [...fields].sort((a, b) => a.localeCompare(b));
  }

  addLine(line: LogLine) {
    this.lines.push(line);
  }
}

export const logs = new LogStore();
