package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gonka/exporter/internal/collector"
	"github.com/gonka/exporter/internal/config"
	"github.com/gonka/exporter/internal/fetcher"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	slog.Info("gonka-exporter starting",
		"port", cfg.Port,
		"participant", cfg.Participant,
		"refresh_interval", cfg.RefreshInterval,
		"node_rpc", cfg.NodeRPCURL,
		"node_rest", cfg.NodeRESTURL,
		"api_url", cfg.APIURL,
		"admin_api", cfg.AdminAPIURL,
		"state_file", cfg.StateFile,
		"history_file", cfg.HistoryFile,
	)

	if cfg.Participant == "" {
		slog.Warn("PARTICIPANT_ADDRESS is not set — participant metrics will be skipped")
	}

	c := collector.New(cfg, fetcher.NewHTTPFetcher(), prometheus.DefaultRegisterer)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	slog.Info("running initial collection")
	c.Collect()
	slog.Info("initial collection done")

	ticker := time.NewTicker(time.Duration(cfg.RefreshInterval) * time.Second)
	go func() {
		for range ticker.C {
			c.Collect()
		}
	}()

	go func() {
		slog.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("shutting down", "signal", sig)

	ticker.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("exporter stopped")
}
