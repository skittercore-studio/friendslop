# syntax=docker/dockerfile:1.7
# Multi-stage build for friendslop.
# Builder needs cgo for mattn/go-sqlite3; runtime is a slim alpine with the
# binary running as a non-root user. The frontend agent's dist bundle is
# copied in via the embed directive at build time — the dist_stub fallback
# keeps this build green even before the frontend ships.

FROM golang:1.23-alpine AS builder
RUN apk add --no-cache build-base git
WORKDIR /src

# Cache deps separately from source for incremental builds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags="-s -w" -o /out/friendslop ./cmd/friendslop

# ----------------------------------------------------------------------------

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata sqlite-libs \
 && addgroup -S slop && adduser -S -G slop -h /home/slop slop \
 && mkdir -p /data && chown slop:slop /data

COPY --from=builder /out/friendslop /usr/local/bin/friendslop

USER slop
ENV SLOP_LISTEN=":8080" SLOP_DB_PATH="/data/slop.db"
EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/friendslop"]
