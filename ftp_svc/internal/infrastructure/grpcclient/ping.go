package grpcclient

import (
	"context"
	"log/slog"
	"time"

	restickv1 "github.com/eykh/restick/contracts/gen/go/restick/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func PingAddr(ctx context.Context, addr string, fromService string) error {
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	client := restickv1.NewPingServiceClient(conn)
	_, err = client.Ping(dialCtx, &restickv1.PingRequest{FromService: fromService})
	return err
}

func PingPeers(ctx context.Context, fromService string, peers map[string]string) {
	pending := map[string]string{}
	for name, addr := range peers {
		if addr != "" {
			pending[name] = addr
		}
	}
	for attempt := 1; attempt <= 12; attempt++ {
		if ctx.Err() != nil {
			return
		}
		time.Sleep(time.Second)
		for name, addr := range pending {
			if err := PingAddr(ctx, addr, fromService); err != nil {
				slog.Warn("grpc peer unreachable", "peer", name, "addr", addr, "attempt", attempt, "err", err)
				continue
			}
			slog.Info("grpc peer ok", "peer", name, "addr", addr)
			delete(pending, name)
		}
		if len(pending) == 0 {
			return
		}
	}
}
