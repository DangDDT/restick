package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/eykh/restick/ftp_svc/internal/application"
	"github.com/eykh/restick/ftp_svc/internal/infrastructure/config"
	"github.com/eykh/restick/ftp_svc/internal/infrastructure/grpcclient"
	miniostore "github.com/eykh/restick/ftp_svc/internal/infrastructure/minio"
	grpcx "github.com/eykh/restick/ftp_svc/internal/interfaces/grpc"
	httpx "github.com/eykh/restick/ftp_svc/internal/interfaces/http"
)

func main() {
	cfg := config.Load()
	store, err := miniostore.New(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket, cfg.MinIOUseSSL)
	if err != nil {
		slog.Error("minio client", "err", err)
		os.Exit(1)
	}
	objects := application.NewObjectService(store)
	authClient := grpcclient.NewAuthClient(cfg.APISvcGRPCAddr, cfg.InternalToken, 0)
	httpServer := httpx.NewServer(objects, httpx.Auth{
		InternalToken: cfg.InternalToken,
		JWTSecret:     []byte(cfg.JWTSecret),
		APIKeys:       cfg.APIKeys,
		KeyValidator:  authClient,
	}, cfg.WebOrigin)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 2)
	go func() {
		errCh <- httpx.Listen(ctx, cfg.HTTPAddr, httpServer.Handler())
	}()
	go func() {
		errCh <- grpcx.Listen(ctx, cfg.GRPCAddr, cfg.ServiceName, objects, cfg.InternalToken)
	}()
	go grpcclient.PingPeers(ctx, cfg.ServiceName, map[string]string{
		"api_svc":  cfg.APISvcGRPCAddr,
		"img_tool": cfg.ImgToolGRPCAddr,
	})
	slog.Info("ftp_svc listening", "http", cfg.HTTPAddr, "grpc", cfg.GRPCAddr)
	select {
	case err := <-errCh:
		if err != nil {
			slog.Error("server stopped", "err", err)
			os.Exit(1)
		}
	case <-ctx.Done():
	}
}
