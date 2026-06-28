import type { AzureResource, LoadingState, LogLine, SourceState, TimeRange } from '../types';

const DEFAULT_SOURCE: SourceState = {
  mode: 'stdin',
  appInsightsEnabled: false,
  selectedResource: null,
  query: '',
  total: 0
};

const DEFAULT_TIME_RANGE: TimeRange = {
  preset: '1h'
};

class LogStore {
  lines = $state<LogLine[]>([]);
  fields = $state<string[]>([]);
  connected = $state(false);
  total = $state(0);
  queryError = $state('');
  source = $state<SourceState>(DEFAULT_SOURCE);
  timeRange = $state<TimeRange>(DEFAULT_TIME_RANGE);
  azureResources = $state<AzureResource[]>([]);
  azureError = $state('');
  loading = $state<LoadingState>({ active: false });

  resetResults(total: number) {
    this.lines = [];
    this.total = total;
    this.queryError = '';
  }

  setConnected(value: boolean) {
    this.connected = value;
  }

  setFields(fields: string[]) {
    this.fields = [...fields].sort((a, b) => a.localeCompare(b));
  }

  setSource(source: SourceState) {
    this.source = source;
    this.total = source.total;
  }

  setTimeRange(timeRange: TimeRange) {
    this.timeRange = timeRange;
  }

  setTotal(total: number) {
    this.total = total;
  }

  setQueryError(message: string) {
    this.queryError = message;
  }

  setAzureResources(resources: AzureResource[], error = '') {
    this.azureResources = [...resources].sort((a, b) => {
      const left = `${a.subscriptionId}/${a.resourceGroup}/${a.name}`;
      const right = `${b.subscriptionId}/${b.resourceGroup}/${b.name}`;
      return left.localeCompare(right);
    });
    this.azureError = error;
  }

  setLoading(loading: LoadingState) {
    this.loading = loading;
  }

  addLine(line: LogLine) {
    this.lines.push(line);
  }
}

export const logs = new LogStore();
