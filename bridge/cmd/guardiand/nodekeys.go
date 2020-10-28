package main

import (
	"crypto/ecdsa"
	"fmt"
	"io/ioutil"
	"os"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
)

func loadGuardianKey(logger *zap.Logger) *ecdsa.PrivateKey {
	var gk *ecdsa.PrivateKey

	if *unsafeDevMode {
		// Figure out our devnet index
		idx, err := devnet.GetDevnetIndex()
		if err != nil {
			logger.Fatal("Failed to parse hostname - are we running in devnet?")
		}

		// Generate guardian key
		gk = devnet.DeterministicEcdsaKeyByIndex(ethcrypto.S256(), uint64(idx))
	} else {
		panic("not implemented") // TODO
	}

	logger.Info("Loaded guardian key", zap.String(
		"address", ethcrypto.PubkeyToAddress(gk.PublicKey).String()))

	return gk
}

func getOrCreateNodeKey(logger *zap.Logger, path string) (p2pcrypto.PrivKey, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("No node key found, generating a new one...", zap.String("path", path))

			priv, _, err := p2pcrypto.GenerateKeyPair(p2pcrypto.Ed25519, -1)
			if err != nil {
				panic(err)
			}

			s, err := p2pcrypto.MarshalPrivateKey(priv)
			if err != nil {
				panic(err)
			}

			err = ioutil.WriteFile(path, s, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to write node key: %w", err)
			}

			return priv, nil
		} else {
			return nil, fmt.Errorf("failed to read node key: %w", err)
		}
	}

	priv, err := p2pcrypto.UnmarshalPrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node key: %w", err)
	}

	logger.Info("Found existing node key", zap.String("path", path))

	return priv, nil
}
