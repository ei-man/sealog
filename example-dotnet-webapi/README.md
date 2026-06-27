# Example .NET Web API

This is a small ASP.NET Core Web API that writes structured JSON logs to stdout using the built-in JSON console logger.

## Production-style run

The `../sealog` binary contains a frontend snapshot embedded at build time. If you changed anything under `frontend/src`, rebuild Sealog first:

```bash
cd ..
make build
cd example-dotnet-webapi
dotnet run | ../sealog
```

Open:

```text
http://localhost:7100
```

## Frontend development run

For live Svelte changes without rebuilding the Go binary each time, run this from the repo root:

```bash
make dev-example
```

Open the Vite dev UI, not the embedded production UI:

```text
http://127.0.0.1:5173
```

Vite proxies `/ws` and `/api/*` to the Go server on port `7100`.

## Example requests

```bash
curl http://localhost:5056/
curl http://localhost:5056/weatherforecast
```
