# syntax=docker/dockerfile:1.7
# Multi-stage build for friendslop.
#
# Stages:
#   1. spa     — node 22, builds frontend/dist via vite
#   2. builder — golang 1.25 alpine, embeds spa output into static binary
#   3. runtime — alpine 3.20, runs as non-root, /data volume for sqlite
#
# Builder needs cgo for mattn/go-sqlite3 (sqlite3 driver). Runtime keeps
# only the binary + sqlite-libs.

FROM node:22-alpine AS spa
WORKDIR /spa
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --silent
COPY frontend/. ./
RUN npm run build

# ----------------------------------------------------------------------------

FROM golang:1.25-alpine AS builder
RUN apk add --no-cache build-base git
WORKDIR /src

# Cache deps separately from source for incremental builds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Mirror the vite output into web/frontend-dist where the embed directive
# expects to find it. web/frontend-dist is gitignored — it's a build artifact.
RUN rm -rf web/frontend-dist
COPY --from=spa /spa/dist ./web/frontend-dist

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
