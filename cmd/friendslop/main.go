// friendslop server entrypoint.
//
// Reads SLOP_LISTEN (default ":8080") and SLOP_DB_PATH (default
// "/data/slop.db"), opens the SQLite store with WAL + migrations, builds the
// HTTP server, and serves until SIGTERM/SIGINT. On shutdown we close the
// listener and give in-flight requests up to 10s to drain before closing the
// DB.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/skittercore-studio/friendslop/internal/db"
	"github.com/skittercore-studio/friendslop/internal/server"
)

const (
	defaultListen = ":8080"
	defaultDB     = "/data/slop.db"
	shutdownGrace = 10 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	listenAddr := envDefault("SLOP_LISTEN", defaultListen)
	dbPath := envDefault("SLOP_DB_PATH", defaultDB)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	d, err := db.Open(rootCtx, dbPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer func() {
		if err := d.Close(); err != nil {
			log.Printf("db close: %v", err)
		}
	}()

	r := chi.NewRouter()
	s := server.NewServer(d)
	s.Mount(r)

	httpSrv := &http.Server{
		Addr:              listenAddr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("friendslop listening on %s (db=%s)", listenAddr, dbPath)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-rootCtx.Done()
	log.Println("shutdown signal received")

	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := httpSrv.Shutdown(shutCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	log.Println("bye")
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
