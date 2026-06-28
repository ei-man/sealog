# Sealog

Sealog is a small local log viewer for structured application logs. It turns JSON log streams or Azure Application Insights traces into a searchable table with selectable columns, a detail view, and time-range filtering.

## Use Cases

- Inspect local development logs without losing structured fields.
- Pipe command output into a browser UI instead of scrolling in a terminal.
- Filter nested JSON fields with simple expressions like `level="error" AND message~="timeout"`.
- Query recent Application Insights traces with the same filter language used for local logs.

## Usage

```bash
some-command | sealog
```

Open `http://localhost:7100`.

For Application Insights:

```bash
az login
./sealog --app-insights
```

Then select an Application Insights resource in the UI.

## Development

```bash
make dev
make build
make check
```

The example .NET API can be run with:

```bash
make dev-example
```

Sealog is licensed under AGPLv3. See `LICENSE`.
