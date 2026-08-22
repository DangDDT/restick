package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/eykh/restick/api_svc/internal/infrastructure/config"
	"github.com/eykh/restick/api_svc/internal/infrastructure/grpcclient"
	"github.com/eykh/restick/api_svc/internal/infrastructure/postgres"
	grpcx "github.com/eykh/restick/api_svc/internal/interfaces/grpc"
	httpx "github.com/eykh/restick/api_svc/internal/interfaces/http"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := postgres.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	if err := store.Seed(ctx, cfg.BootstrapAdminEmail, cfg.BootstrapAdminPass, cfg.BootstrapAPIKey); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
	httpServer := httpx.NewServer(store, cfg.JWTSecret, cfg.WebOrigin, cfg.PublicFTPOrigin)
	errCh := make(chan error, 2)
	go func() {
		errCh <- httpx.Listen(ctx, cfg.HTTPAddr, httpServer.Handler())
	}()
	go func() {
		errCh <- grpcx.Listen(ctx, cfg.GRPCAddr, cfg.ServiceName, store, cfg.InternalToken)
	}()
	go grpcclient.PingPeers(ctx, cfg.ServiceName, map[string]string{
		"ftp_svc":  cfg.FTPSvcGRPCAddr,
		"img_tool": cfg.ImgToolGRPCAddr,
	})
	slog.Info("api_svc listening", "http", cfg.HTTPAddr, "grpc", cfg.GRPCAddr)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("server stopped", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
}
