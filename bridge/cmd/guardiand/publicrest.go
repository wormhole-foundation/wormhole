package guardiand

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net/http"
)

func publicrestServiceRunnable(
	logger *zap.Logger,
	listenAddr string,
	upstreamAddr string,
) (supervisor.Runnable, error) {
	return func(ctx context.Context) error {
		conn, err := grpc.DialContext(
			ctx,
			fmt.Sprintf("unix:///%s", upstreamAddr),
			grpc.WithBlock(),
			grpc.WithInsecure())
		if err != nil {
			return fmt.Errorf("failed to dial upstream: %s", err)
		}

		gwmux := runtime.NewServeMux()
		err = publicrpcv1.RegisterPublicrpcHandler(ctx, gwmux, conn)
		if err != nil {
			panic(err)
		}
		srv := &http.Server{
			Addr:    listenAddr,
			Handler: gwmux,
		}

		supervisor.Signal(ctx, supervisor.SignalHealthy)
		errC := make(chan error)
		go func() {
			logger.Info("publicrest server listening", zap.String("addr", srv.Addr))
			errC <- srv.ListenAndServe()
		}()
		select {
		case <-ctx.Done():
			// non-graceful shutdown
			if err := srv.Close(); err != nil {
				return err
			}
			return ctx.Err()
		case err := <-errC:
			return err
		}
	}, nil
}
