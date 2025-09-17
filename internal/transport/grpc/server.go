package grpctransport

import (
	"context"
	"log/slog"
	"net"

	"github.com/Shyyw1e/ozon-bank-url-test/internal/api/shortener/v1"
	"github.com/Shyyw1e/ozon-bank-url-test/internal/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	health "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type server struct {
	shortenerv1.UnimplementedShortenerServer
	log *slog.Logger
	svc *core.Shortener
}

func NewGRPCServer(log *slog.Logger, svc *core.Shortener) *grpc.Server {
	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			loggingInterceptor(log),
		),
	)
	s := &server{log: log, svc: svc}
	shortenerv1.RegisterShortenerServer(grpcSrv, s)

	// Health + Reflection
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcSrv, hs)
	reflection.Register(grpcSrv)
	return grpcSrv
}

func (s *server) Shorten(ctx context.Context, req *shortenerv1.ShortenRequest) (*shortenerv1.ShortenResponse, error) {
	if req == nil || req.Url == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}
	code, err := s.svc.Create(ctx, req.Url)
	if err != nil {
		switch err {
		case core.ErrInvalidURL:
			return nil, status.Error(codes.InvalidArgument, "invalid url")
		case core.ErrConflict:
			return nil, status.Error(codes.Aborted, "too many collisions")
		default:
			s.log.Error("Shorten failed", "err", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
	}
	return &shortenerv1.ShortenResponse{Code: code}, nil
}

func (s *server) Resolve(ctx context.Context, req *shortenerv1.ResolveRequest) (*shortenerv1.ResolveResponse, error) {
	if req == nil || !core.IsValidCode(req.Code) {
		return nil, status.Error(codes.InvalidArgument, "invalid code")
	}
	orig, err := s.svc.Resolve(ctx, req.Code)
	if err != nil {
		if err == core.ErrNotFound {
			return nil, status.Error(codes.NotFound, "not found")
		}
		s.log.Error("Resolve failed", "code", req.Code, "err", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &shortenerv1.ResolveResponse{Url: orig}, nil
}

// ---- interceptors ----

func loggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			st, _ := status.FromError(err)
			log.Info("grpc_request",
				"method", info.FullMethod,
				"code", st.Code().String(),
				"msg", st.Message(),
			)
			return resp, err
		}
		log.Info("grpc_request", "method", info.FullMethod, "code", codes.OK.String())
		return resp, nil
	}
}

func recoveryInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r)
				err = status.Error(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// Утилита для запуска (можно использовать из main)
func ListenAndServe(grpcSrv *grpc.Server, addr string) (net.Listener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() { _ = grpcSrv.Serve(lis) }()
	return lis, nil
}
