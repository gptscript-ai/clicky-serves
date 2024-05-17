package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/rs/cors"
)

type Config struct {
	Port         string
	GPTScriptBin string
}

func Start(ctx context.Context, config Config) error {
	sigCtx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)
	defer cancel()

	addRoutes(http.DefaultServeMux, config)

	server := http.Server{
		Addr: ":" + config.Port,
		Handler: apply(http.DefaultServeMux,
			addRequestID,
			addLogger,
			logRequest,
			cors.Default().Handler,
			contentType("application/json"),
		),
	}

	slog.Info("Starting server", "addr", server.Addr)
	errChan := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errChan <- fmt.Errorf("failed to start server: %w", err):
			default:
			}
		}
	}()

	select {
	case <-sigCtx.Done():
	case err := <-errChan:
		return err
	}

	slog.Info("Shutting down server")
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}
