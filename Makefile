.PHONY: dev dev-example build check clean

dev:
	cd frontend && npm install
	@set -e; \
	(cd frontend && npm run dev) & \
	vite_pid=$$!; \
	trap 'kill $$vite_pid' EXIT INT TERM; \
	printf '\nOpen the live frontend at http://127.0.0.1:5173\n\n'; \
	go run . --no-open

dev-example:
	cd frontend && npm install
	@set -e; \
	(cd frontend && npm run dev) & \
	vite_pid=$$!; \
	trap 'kill $$vite_pid 2>/dev/null || true' EXIT INT TERM; \
	printf '\nOpen the live frontend at http://127.0.0.1:5173\n'; \
	printf 'The embedded production UI remains at http://localhost:7100\n\n'; \
	(cd example-dotnet-webapi && dotnet run) | go run . --no-open

build:
	cd frontend && npm install
	cd frontend && npm run build
	go build -o sealog .

check:
	cd frontend && npm install
	cd frontend && npm run check
	go test ./...

clean:
	rm -rf frontend/dist sealog
