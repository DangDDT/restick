package grpcx

import (
	"context"
	"crypto/subtle"
	"net"

	"github.com/eykh/restick/api_svc/internal/domain/pack"
	"github.com/eykh/restick/api_svc/internal/infrastructure/postgres"
	restickv1 "github.com/eykh/restick/contracts/gen/go/restick/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type pingServer struct {
	restickv1.UnimplementedPingServiceServer
	serviceName string
}

func (s *pingServer) Ping(_ context.Context, _ *restickv1.PingRequest) (*restickv1.PingResponse, error) {
	return &restickv1.PingResponse{ServiceName: s.serviceName, Message: "pong"}, nil
}

type jobServer struct {
	restickv1.UnimplementedJobServiceServer
	store *postgres.Store
}

func (s *jobServer) ClaimJob(ctx context.Context, _ *restickv1.ClaimJobRequest) (*restickv1.ClaimJobResponse, error) {
	job, p, _, ok, err := s.store.ClaimJob(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !ok {
		return &restickv1.ClaimJobResponse{Empty: true}, nil
	}
	return &restickv1.ClaimJobResponse{
		Empty: false, JobId: job.ID, PackId: p.ID, SourceKind: p.SourceKind, SourcePrefix: p.SourcePrefix,
		Name: p.Name, Slug: p.Slug, Author: p.Author, Tags: p.Tags,
	}, nil
}

func (s *jobServer) CompleteJob(ctx context.Context, req *restickv1.CompleteJobRequest) (*restickv1.CompleteJobResponse, error) {
	p, u, err := s.store.PackUserForJob(ctx, req.GetJobId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	stickers := metaToStickers(req.GetStickers())
	next := pack.AfterProcessStatus(u.IsAdmin())
	version := int(req.GetVersion())
	if version < 1 {
		version = p.CurrentVersion
	}
	if err := s.store.CompleteJob(ctx, req.GetJobId(), req.GetChecksumSha256(), req.GetThumbnailKey(), req.GetBundleKey(), version, stickers, next); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &restickv1.CompleteJobResponse{}, nil
}

func (s *jobServer) FailJob(ctx context.Context, req *restickv1.FailJobRequest) (*restickv1.FailJobResponse, error) {
	if err := s.store.FailJob(ctx, req.GetJobId(), req.GetError()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &restickv1.FailJobResponse{}, nil
}

func (s *jobServer) UpsertCLIPack(ctx context.Context, req *restickv1.UpsertCLIPackRequest) (*restickv1.UpsertCLIPackResponse, error) {
	p := pack.Pack{
		ID: req.GetPackId(), Slug: req.GetSlug(), Name: req.GetName(), Author: req.GetAuthor(),
		Tags: req.GetTags(), ChecksumSHA256: req.GetChecksumSha256(), ThumbnailKey: req.GetThumbnailKey(),
		BundleKey: req.GetBundleKey(), CurrentVersion: int(req.GetVersion()),
	}
	if err := s.store.UpsertCLI(ctx, p, metaToStickers(req.GetStickers())); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &restickv1.UpsertCLIPackResponse{}, nil
}

type authServer struct {
	restickv1.UnimplementedAuthServiceServer
	store *postgres.Store
}

func (s *authServer) ValidateAPIKey(ctx context.Context, req *restickv1.ValidateAPIKeyRequest) (*restickv1.ValidateAPIKeyResponse, error) {
	key := req.GetApiKey()
	if key == "" {
		return &restickv1.ValidateAPIKeyResponse{Valid: false}, nil
	}
	valid, userID, keyID, name, err := s.store.ValidateAPIKeyDetailed(ctx, key)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &restickv1.ValidateAPIKeyResponse{
		Valid:  valid,
		UserId: userID,
		KeyId:  keyID,
		Name:   name,
	}, nil
}

func metaToStickers(items []*restickv1.StickerMeta) []pack.Sticker {
	out := make([]pack.Sticker, 0, len(items))
	for i, item := range items {
		out = append(out, pack.Sticker{
			Filename: item.GetFile(), EmojiHint: item.GetEmojiHint(), Width: int(item.GetWidth()), Height: int(item.GetHeight()),
			ObjectKey: item.GetObjectKey(), ThumbKey: item.GetThumbKey(), SortOrder: i,
		})
	}
	return out
}

func Listen(ctx context.Context, addr string, serviceName string, store *postgres.Store, internalToken string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(16<<20),
		grpc.MaxSendMsgSize(16<<20),
		grpc.UnaryInterceptor(internalInterceptor(internalToken)),
	)
	restickv1.RegisterPingServiceServer(server, &pingServer{serviceName: serviceName})
	restickv1.RegisterJobServiceServer(server, &jobServer{store: store})
	restickv1.RegisterAuthServiceServer(server, &authServer{store: store})
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	return server.Serve(listener)
}

func internalInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == restickv1.PingService_Ping_FullMethodName {
			return handler(ctx, req)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		values := md.Get("x-internal-token")
		if len(values) == 0 || token == "" || subtle.ConstantTimeCompare([]byte(values[0]), []byte(token)) != 1 {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}
		return handler(ctx, req)
	}
}
