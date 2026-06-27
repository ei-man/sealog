export type LogFields = Record<string, unknown>;

export type LogLine = {
  id: number;
  fields: LogFields;
  raw: boolean;
};

export type ServerMessage =
  | { type: 'line'; line: LogLine }
  | { type: 'fields'; fields: string[] };
