export type LogFields = Record<string, unknown>;

export type LogLine = {
  id: number;
  fields: LogFields;
  raw: boolean;
};

export type SourceMode = 'stdin' | 'app-insights';

export type AzureResource = {
  id: string;
  name: string;
  resourceGroup: string;
  subscriptionId: string;
  location: string;
  appId?: string;
};

export type TimeRangePreset = '15m' | '30m' | '1h' | '3h' | '6h' | '12h' | '24h' | '3d' | 'custom';

export type TimeRange = {
  preset: TimeRangePreset;
  start?: string;
  end?: string;
};

export type SourceState = {
  mode: SourceMode;
  appInsightsEnabled: boolean;
  selectedResource?: AzureResource | null;
  query: string;
  total: number;
};

export type LoadingState = {
  active: boolean;
  message?: string;
};

export type QueryError = {
  message: string;
  position: number;
};

export type ServerMessage =
  | { type: 'line'; line: LogLine }
  | { type: 'fields'; fields: string[] }
  | { type: 'source'; source: SourceState }
  | { type: 'timeRange'; timeRange: TimeRange }
  | { type: 'results.reset'; total: number }
  | { type: 'results.stats'; total: number }
  | { type: 'query.error'; error: QueryError }
  | { type: 'azure.resources'; resources: AzureResource[]; errorMessage?: string }
  | { type: 'loading'; loading: LoadingState };

export type ClientCommand =
  | { type: 'query.apply'; query: string }
  | { type: 'timeRange.set'; timeRange: TimeRange }
  | { type: 'source.select'; source: SourceMode }
  | { type: 'azure.resources.list' }
  | { type: 'azure.resource.select'; resourceId: string };
