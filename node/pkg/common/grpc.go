package common

import (
	"context"
	"fmt"
	"sync"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type GrpcLogDetail string

// initMutex is used during initialization to prevent concurrent writes to the prometheus registry.
// This is only relevant during testing of node/node.go where multiple guardians are created in the same process.
var initMutex sync.Mutex

const (
	GrpcLogDetailNone    GrpcLogDetail = "none"
	GrpcLogDetailMinimal GrpcLogDetail = "minimal"
	GrpcLogDetailFull    GrpcLogDetail = "full"
)

func truncateStr(str string, maxLen int) string {
	if len(str) > maxLen {
		return str[:maxLen] + "..."
	}
	return str
}

func addDetail(ctx context.Context, logDetail GrpcLogDetail) {
	if logDetail == GrpcLogDetailNone {
		return
	}
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		tags := grpc_ctxtags.Extract(ctx)
		tags.Set("x-forwarded-for", md.Get("x-forwarded-for"))

		if logDetail == GrpcLogDetailFull {
			if len(md.Get("grpcgateway-user-agent")) > 0 {
				tags.Set("user-agent", truncateStr(md.Get("grpcgateway-user-agent")[0], 200))
			}
		}
	}
}

// newMetadataStreamServerInterceptor returns stream interceptor that
// adds the `x-forwarded-for` and, if logDetail == "full", the `user-agent` metadata as a tag to cause it to be logged by grpc_zap.StreamServerInterceptor().
// Note that `x-forwarded-for` can only be trusted if the latest hop in the proxy chain is trusted.
// For JSON-Web requests, the latest hop is the guardian itself (`grpc-gateway`), which is listening on TCP and forwarding to the gRPC publicrpc UNIX socket.
// This can be identified by `"peer.address": "@"` in the logs and `grpc-gateway` correctly sets the `x-forwarded-for` metadata.
func newMetadataStreamServerInterceptor(logDetail GrpcLogDetail) func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		addDetail(ctx, logDetail)
		return handler(srv, stream)
	}
}

// newMetadataServerInterceptor returns a unary interceptor that
// adds the `x-forwarded-for` and, if logDetail == "full", the `user-agent` metadata as a tag to cause it to be logged by grpc_zap.StreamServerInterceptor().
// Note that `x-forwarded-for` can only be trusted if the latest hop in the proxy chain is trusted.
// For JSON-Web requests, the latest hop is the guardian itself (`grpc-gateway`), which is listening on TCP and forwarding to the gRPC publicrpc UNIX socket.
// This can be identified by `"peer.address": "@"` in the logs and `grpc-gateway` correctly sets the `x-forwarded-for` metadata.
func newMetadataServerInterceptor(logDetail GrpcLogDetail) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		addDetail(ctx, logDetail)
		return handler(ctx, req)
	}
}

func requestPayloadServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if p, ok := req.(protoreflect.ProtoMessage); ok {
		if b, err := protojson.Marshal(p); err == nil {
			if len(b) <= 200 {
				grpc_ctxtags.Extract(ctx).Set("grpc.requestbody", &protojsonObjectMarshaler{pb: p})
			} else {
				grpc_ctxtags.Extract(ctx).Set("grpc.requestbody", "too long")
			}
		}
	}
	return handler(ctx, req)
}

func NewInstrumentedGRPCServer(logger *zap.Logger, rpcLogDetail GrpcLogDetail) *grpc.Server {
	initMutex.Lock()
	defer initMutex.Unlock()

	streamInterceptors := []grpc.StreamServerInterceptor{
		grpc_ctxtags.StreamServerInterceptor(),
		grpc_prometheus.StreamServerInterceptor,
	}

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		grpc_ctxtags.UnaryServerInterceptor(),
		grpc_prometheus.UnaryServerInterceptor,
	}

	if rpcLogDetail != GrpcLogDetailNone {
		logger = logger.With(zap.Bool("_privateLogEntry", true))
		streamInterceptors = append(streamInterceptors,
			newMetadataStreamServerInterceptor(rpcLogDetail),
			grpc_zap.StreamServerInterceptor(logger),
		)

		// if logging detail is "full", also log the request payload (only applicable to unary)
		if rpcLogDetail == GrpcLogDetailFull {
			unaryInterceptors = append(unaryInterceptors, requestPayloadServerInterceptor)
		}

		unaryInterceptors = append(unaryInterceptors,
			newMetadataServerInterceptor(rpcLogDetail),
			grpc_zap.UnaryServerInterceptor(logger),
		)
	}

	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamInterceptors...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryInterceptors...)),
	)

	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(server)

	return server
}

// this helper type and associated functions are such that the ZAP jsonEncoder will properly encode the gRPC request payload.
// We could instead just encode the payload to a string here, but then that string will be encoded again by the ZAP jsonEncoder, making downstream processing more difficult
type protojsonObjectMarshaler struct {
	pb protoreflect.ProtoMessage
}

func (j *protojsonObjectMarshaler) MarshalLogObject(e zapcore.ObjectEncoder) error {
	// ZAP jsonEncoder deals with AddReflect by using json.MarshalObject. The same thing applies for consoleEncoder.
	return e.AddReflected("msg", j)
}

func (j *protojsonObjectMarshaler) MarshalJSON() ([]byte, error) {
	b, err := protojson.Marshal(j.pb)
	if err != nil {
		return nil, fmt.Errorf("jsonpb serializer failed: %v", err)
	}
	return b, nil
}
