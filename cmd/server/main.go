package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Shyyw1e/ozon-bank-url-test/internal/config"
	"github.com/Shyyw1e/ozon-bank-url-test/internal/core"
	"github.com/Shyyw1e/ozon-bank-url-test/internal/storage/memory"
	pgstore "github.com/Shyyw1e/ozon-bank-url-test/internal/storage/postgres"
	httptransport "github.com/Shyyw1e/ozon-bank-url-test/internal/transport/http"
	"github.com/Shyyw1e/ozon-bank-url-test/pkg/logger"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Println("config load:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)
	log.Info("config", "httpAddr", cfg.HTTPAddr, "storage", cfg.StorageBackend)

	var store core.Store
	var closer func() error
	switch cfg.StorageBackend {
	case "postgres":
		dsn := os.Getenv("DATABASE_URL")
		if dsn == "" {
			log.Error("DATABASE_URL is required for postgres")
			os.Exit(1)
		}
		ps, err := pgstore.New(dsn)
		if err != nil {
			log.Error("postgres connect failed", "err", err)
			os.Exit(1)
		}
		store = ps
		closer = ps.Close
	default:
		store = memory.New()
		closer = func() error { return nil }
	}

	svc := core.NewShortener(store, core.NewCode)
	handler := httptransport.NewRouter(log, svc)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Info("http listen", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http serve error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("shutting down...")
	_ = srv.Shutdown(ctx)
	if err := closer(); err != nil {
		log.Error("store close error", "err", err)
	}
	log.Info("bye")
}
