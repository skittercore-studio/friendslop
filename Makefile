.PHONY: build test run docker tidy assets frontend clean

BIN ?= ./bin/friendslop
LISTEN ?= :8080
DB_PATH ?= ./local.db

# `make build` always rebuilds the SPA + mirrors it into web/frontend-dist
# before invoking go build, so a fresh checkout yields a self-contained
# binary in one command.
build: assets
	go build -trimpath -o $(BIN) ./cmd/friendslop

# `make frontend` produces frontend/dist/ via vite (canonical SPA build).
frontend:
	cd frontend && npm install --silent && npm run build

# `make assets` mirrors frontend/dist into web/frontend-dist where the Go
# embed directive can pick it up. web/frontend-dist/ is gitignored.
assets: frontend
	rm -rf web/frontend-dist
	cp -r frontend/dist web/frontend-dist

test:
	go test ./... -race -count=1

run: build
	SLOP_LISTEN=$(LISTEN) SLOP_DB_PATH=$(DB_PATH) $(BIN)

docker:
	docker build -t friendslop .

tidy:
	go mod tidy

clean:
	rm -rf bin/ web/frontend-dist/ frontend/dist/
