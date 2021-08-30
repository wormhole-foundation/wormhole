package supervisor

// Supporting infrastructure to allow running some non-Go payloads under supervision.

import (
	"context"
	"net"
	"os/exec"

	"google.golang.org/grpc"
)

// GRPCServer creates a Runnable that serves gRPC requests as longs as it's not canceled.
// If graceful is set to true, the server will be gracefully stopped instead of plain stopped. This means all pending
// RPCs will finish, but also requires streaming gRPC handlers to check their context liveliness and exit accordingly.
// If the server code does not support this, `graceful` should be false and the server will be killed violently instead.
func GRPCServer(srv *grpc.Server, lis net.Listener, graceful bool) Runnable {
	return func(ctx context.Context) error {
		Signal(ctx, SignalHealthy)
		errC := make(chan error)
		go func() {
			errC <- srv.Serve(lis)
		}()
		select {
		case <-ctx.Done():
			if graceful {
				srv.GracefulStop()
			} else {
				srv.Stop()
			}
			return ctx.Err()
		case err := <-errC:
			return err
		}
	}
}

// Command will create a Runnable that starts a long-running command, whose exit is determined to be a failure.
func Command(name string, arg ...string) Runnable {
	return func(ctx context.Context) error {
		Signal(ctx, SignalHealthy)

		cmd := exec.CommandContext(ctx, name, arg...)
		return cmd.Run()
	}
}
