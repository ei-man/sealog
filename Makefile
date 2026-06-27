.PHONY: dev build check clean

dev:
	cd frontend && npm install
	@set -e; \
	(cd frontend && npm run dev) & \
	vite_pid=$$!; \
	trap 'kill $$vite_pid' EXIT INT TERM; \
	go run . --no-open

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
