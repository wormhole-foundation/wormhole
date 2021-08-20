package guardiand

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"log"
	"net/http"
	"strings"
)

func allowCORSWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				corsPreflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func corsPreflightHandler(w http.ResponseWriter, r *http.Request) {
	headers := []string{
		"content-type",
		"accept",
		"x-user-agent",
		"x-grpc-web",
		"grpc-status",
		"grpc-message",
		"authorization",
	}
	w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
}

func publicrestServiceRunnable(
	logger *zap.Logger,
	listenAddr string,
	upstreamAddr string,
	grpcServer *grpc.Server,
	tlsHostname string,
	tlsProd bool,
	tlsCacheDir string,
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

		mux := http.NewServeMux()
		grpcWebServer := grpcweb.WrapServer(grpcServer)
		mux.Handle("/", allowCORSWrapper(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if grpcWebServer.IsGrpcWebRequest(req) {
				grpcWebServer.ServeHTTP(resp, req)
			} else {
				gwmux.ServeHTTP(resp, req)
			}
		})))

		srv := &http.Server{
			Addr:     listenAddr,
			ErrorLog: log.Default(),
			Handler:  mux,
		}

		// TLS setup
		if tlsHostname != "" {
			logger.Info("provisioning Let's Encrypt certificate", zap.String("hostname", tlsHostname))

			var acmeApi string
			if tlsProd {
				logger.Info("using production Let's Encrypt server")
				acmeApi = autocert.DefaultACMEDirectory
			} else {
				logger.Info("using staging Let's Encrypt server")
				acmeApi = "https://acme-staging-v02.api.letsencrypt.org/directory"
			}

			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(tlsHostname),
				Cache:      autocert.DirCache(tlsCacheDir),
				Client:     &acme.Client{DirectoryURL: acmeApi},
			}

			srv.TLSConfig = certManager.TLSConfig()
			logger.Info("certificate provisioning configured")
		}

		supervisor.Signal(ctx, supervisor.SignalHealthy)
		errC := make(chan error)
		go func() {
			logger.Info("publicrest server listening", zap.String("addr", srv.Addr))
			if tlsHostname != "" {
				errC <- srv.ListenAndServeTLS("", "")
			} else {
				errC <- srv.ListenAndServe()
			}
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
