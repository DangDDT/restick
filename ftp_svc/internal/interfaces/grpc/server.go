package grpcx

import (
	"context"
	"crypto/subtle"
	"net"

	"github.com/eykh/restick/ftp_svc/internal/application"
	"github.com/eykh/restick/ftp_svc/internal/domain/objectkey"
	restickv1 "github.com/eykh/restick/contracts/gen/go/restick/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const maxMsg = 16 << 20

type pingServer struct {
	restickv1.UnimplementedPingServiceServer
	serviceName string
}

func (s *pingServer) Ping(_ context.Context, _ *restickv1.PingRequest) (*restickv1.PingResponse, error) {
	return &restickv1.PingResponse{ServiceName: s.serviceName, Message: "pong"}, nil
}

type objectServer struct {
	restickv1.UnimplementedObjectServiceServer
	objects *application.ObjectService
}

func (s *objectServer) PutObject(ctx context.Context, req *restickv1.PutObjectRequest) (*restickv1.PutObjectResponse, error) {
	if err := objectkey.ValidateProcessed(req.GetKey()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.objects.Put(ctx, req.GetKey(), req.GetData(), req.GetContentType()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &restickv1.PutObjectResponse{Key: req.GetKey()}, nil
}

func (s *objectServer) GetObject(ctx context.Context, req *restickv1.GetObjectRequest) (*restickv1.GetObjectResponse, error) {
	data, contentType, err := s.objects.Get(ctx, req.GetKey())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &restickv1.GetObjectResponse{Data: data, ContentType: contentType}, nil
}

func (s *objectServer) StatObject(ctx context.Context, req *restickv1.StatObjectRequest) (*restickv1.StatObjectResponse, error) {
	size, etag, contentType, err := s.objects.Store().Stat(ctx, req.GetKey())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &restickv1.StatObjectResponse{Size: size, Etag: etag, ContentType: contentType}, nil
}

func (s *objectServer) BuildBundle(ctx context.Context, req *restickv1.BuildBundleRequest) (*restickv1.BuildBundleResponse, error) {
	key, err := s.objects.BuildBundle(ctx, req.GetPackId(), int(req.GetVersion()), req.GetKeys())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &restickv1.BuildBundleResponse{BundleKey: key}, nil
}

func Listen(ctx context.Context, addr string, serviceName string, objects *application.ObjectService, internalToken string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsg),
		grpc.MaxSendMsgSize(maxMsg),
		grpc.UnaryInterceptor(internalInterceptor(internalToken)),
	)
	restickv1.RegisterPingServiceServer(server, &pingServer{serviceName: serviceName})
	restickv1.RegisterObjectServiceServer(server, &objectServer{objects: objects})
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
