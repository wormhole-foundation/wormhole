package node

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGuardianOptionTSS(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	// Create a dummy guardian signer
	signer, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
	require.NoError(t, err)

	selfAddr := ethcommon.HexToAddress("0x1")
	leaderAddr := ethcommon.HexToAddress("0x2")
	address := "localhost:1234"

	t.Run("Insecure", func(t *testing.T) {
		g := &G{
			gst:            common.NewGuardianSetState(nil),
			guardianSigner: signer,
			runnables:      make(map[string]supervisor.Runnable),
		}
		opt := GuardianOptionTSS(selfAddr, leaderAddr, address, "", "")
		err := opt.f(ctx, logger, g)
		assert.NoError(t, err)
		assert.NotNil(t, g.tssEngine)
		assert.NotNil(t, g.runnables["tss"])
	})

	t.Run("MissingKeyPath", func(t *testing.T) {
		g := &G{
			gst:            common.NewGuardianSetState(nil),
			guardianSigner: signer,
			runnables:      make(map[string]supervisor.Runnable),
		}
		opt := GuardianOptionTSS(selfAddr, leaderAddr, address, "/tmp/cert.pem", "")
		err := opt.f(ctx, logger, g)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tss tls key path must be provided")
	})

	t.Run("InvalidCertPath", func(t *testing.T) {
		g := &G{
			gst:            common.NewGuardianSetState(nil),
			guardianSigner: signer,
			runnables:      make(map[string]supervisor.Runnable),
		}
		opt := GuardianOptionTSS(selfAddr, leaderAddr, address, "/nonexistent/cert.pem", "/nonexistent/key.pem")
		err := opt.f(ctx, logger, g)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read tss tls certificate")
	})

	t.Run("ValidMTLS", func(t *testing.T) {
		certPath, keyPath, cleanup := generateTempCertKey(t)
		defer cleanup()

		g := &G{
			gst:            common.NewGuardianSetState(nil),
			guardianSigner: signer,
			runnables:      make(map[string]supervisor.Runnable),
		}
		opt := GuardianOptionTSS(selfAddr, leaderAddr, address, certPath, keyPath)
		err := opt.f(ctx, logger, g)
		assert.NoError(t, err)
		assert.NotNil(t, g.tssEngine)
		assert.NotNil(t, g.runnables["tss"])
	})

	t.Run("InvalidCertContent", func(t *testing.T) {
		f, err := os.CreateTemp("", "garbage-cert")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		_, err = f.WriteString("garbage")
		require.NoError(t, err)
		f.Close()

		g := &G{
			gst:            common.NewGuardianSetState(nil),
			guardianSigner: signer,
			runnables:      make(map[string]supervisor.Runnable),
		}
		opt := GuardianOptionTSS(selfAddr, leaderAddr, address, f.Name(), "somekey")
		err = opt.f(ctx, logger, g)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse tss tls certificate")
	})
}

func generateTempCertKey(t *testing.T) (string, string, func()) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certOut, err := os.CreateTemp("", "cert-*.pem")
	require.NoError(t, err)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certOut.Close()

	keyOut, err := os.CreateTemp("", "key-*.pem")
	require.NoError(t, err)
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})
	keyOut.Close()

	cleanup := func() {
		os.Remove(certOut.Name())
		os.Remove(keyOut.Name())
	}

	return certOut.Name(), keyOut.Name(), cleanup
}
