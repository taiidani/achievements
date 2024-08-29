package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/taiidani/achievements/internal/data"
	"github.com/taiidani/achievements/internal/data/cache"
	"github.com/taiidani/achievements/internal/server"
	"github.com/taiidani/achievements/internal/steam"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Define external clients
	client := steam.NewClient()
	// cache := cache.NewFile()
	cache, err := setupCache()
	if err != nil {
		log.Fatal("Unable to set up cache", "error", err)
	}

	// Begin refreshing data
	go data.Refresher(ctx, client, cache)

	// Serve until interrupted
	if err := serve(ctx, client, cache); err != nil {
		log.Fatal(err)
	}
}

func setupCache() (cache.Cache, error) {
	if addr, ok := os.LookupEnv("REDIS_ADDR"); ok {
		return cache.NewRedis(addr), nil
	} else if host, ok := os.LookupEnv("REDIS_HOST"); ok {
		db := 0
		if dbParsed, err := strconv.ParseInt(os.Getenv("REDIS_DB"), 10, 64); err == nil {
			db = int(dbParsed)
		}

		port := os.Getenv("REDIS_PORT")
		user := os.Getenv("REDIS_USER")
		pass := os.Getenv("REDIS_PASSWORD")

		return cache.NewRedisSecureCache(host, port, user, pass, db), nil
	}

	slog.Warn("No REDIS_ADDR or REDIS_HOST env var set. Falling back upon in-memory store")
	return cache.NewMemory(), nil
}

func serve(ctx context.Context, client *steam.Client, cache cache.Cache) error {
	backend := data.NewData(client, cache)
	srv := server.NewServer(backend)

	go func() {
		slog.Info("Server starting", "dev", server.DevMode)
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			slog.Error("Unclean server shutdown encountered", "error", err)
		}
	}()

	<-ctx.Done()

	// Gracefully shut down over 60 seconds
	slog.Info("Server shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Minute)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	slog.Info("Server exited")
	return nil
}
