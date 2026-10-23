.PHONY: prepare dev-api dev-web build test lint create-admin

# go:embed needs web/dist to exist before any Go command runs. In development the
# frontend is served by Vite, so a placeholder index.html is enough.
prepare:
	@mkdir -p web/dist
	@test -f web/dist/index.html || echo '<!doctype html><html lang="id"><head><meta charset="utf-8"><title>SiTukang</title></head><body><p>Frontend belum dibangun. Jalankan <code>make build</code>.</p></body></html>' > web/dist/index.html

dev-api: prepare
	go run ./cmd/situkang

dev-web:
	cd web && bun run dev

build:
	cd web && bun install && bun run build
	go build -o bin/situkang ./cmd/situkang

test: prepare
	go test ./...
	cd web && bun run typecheck

lint: prepare
	go vet ./...
	cd web && bun run lint

create-admin: prepare
	go run ./cmd/situkang create-admin
