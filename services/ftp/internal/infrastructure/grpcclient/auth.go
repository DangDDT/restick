package grpcclient

import (
	"context"
	"sync"
	"time"

	restickv1 "github.com/eykh/restick/contracts/gen/go/restick/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type cacheEntry struct {
	valid     bool
	expiresAt time.Time
}

type AuthClient struct {
	addr          string
	internalToken string
	ttl           time.Duration
	cache         sync.Map
}

func NewAuthClient(addr string, internalToken string, ttl time.Duration) *AuthClient {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &AuthClient{
		addr:          addr,
		internalToken: internalToken,
		ttl:           ttl,
	}
}

func (c *AuthClient) Validate(ctx context.Context, apiKey string) bool {
	if apiKey == "" || c == nil || c.addr == "" {
		return false
	}
	if val, ok := c.cache.Load(apiKey); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.valid
		}
		c.cache.Delete(apiKey)
	}

	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(c.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()

	client := restickv1.NewAuthServiceClient(conn)
	md := metadata.Pairs("x-internal-token", c.internalToken)
	callCtx := metadata.NewOutgoingContext(dialCtx, md)

	resp, err := client.ValidateAPIKey(callCtx, &restickv1.ValidateAPIKeyRequest{ApiKey: apiKey})
	if err != nil {
		return false
	}
	valid := resp.GetValid()
	c.cache.Store(apiKey, cacheEntry{
		valid:     valid,
		expiresAt: time.Now().Add(c.ttl),
	})
	return valid
}
