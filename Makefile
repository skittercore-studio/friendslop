.PHONY: build test run docker tidy

BIN ?= ./bin/friendslop
LISTEN ?= :8080
DB_PATH ?= ./local.db

build:
	go build -trimpath -o $(BIN) ./cmd/friendslop

test:
	go test ./... -race -count=1

run: build
	SLOP_LISTEN=$(LISTEN) SLOP_DB_PATH=$(DB_PATH) $(BIN)

docker:
	docker build -t friendslop .

tidy:
	go mod tidy
