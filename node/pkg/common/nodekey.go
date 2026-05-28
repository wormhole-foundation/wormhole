package common

import (
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

func GetOrCreateNodeKey(logger *zap.Logger, path string) (crypto.PrivKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("No node key found, generating a new one...", zap.String("path", path))

			priv, _, genErr := crypto.GenerateKeyPair(crypto.Ed25519, -1)
			if genErr != nil {
				panic(genErr)
			}

			s, marshalErr := crypto.MarshalPrivateKey(priv)
			if marshalErr != nil {
				panic(marshalErr)
			}

			err = os.WriteFile(path, s, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to write node key: %w", err)
			}

			return priv, nil
		} else {
			return nil, fmt.Errorf("failed to read node key: %w", err)
		}
	}

	priv, err := crypto.UnmarshalPrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node key: %w", err)
	}

	peerID, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		panic(err)
	}

	logger.Info("Found existing node key",
		zap.String("path", path),
		zap.Stringer("peerID", peerID))

	return priv, nil
}
