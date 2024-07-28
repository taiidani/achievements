package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/taiidani/achievements/internal/data"
	"github.com/taiidani/achievements/internal/server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	// Begin refreshing data
	go data.Refresher(ctx)

	// Serve until interrupted
	if err := serve(ctx); err != nil {
		log.Fatal(err)
	}
}

func serve(ctx context.Context) error {
	srv := server.NewServer()
	go func() {
		slog.Info("Server starting")
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
