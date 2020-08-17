package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/libp2p/go-libp2p-core/crypto"
	"go.uber.org/zap"
)

func getOrCreateNodeKey(logger *zap.Logger, path string) (crypto.PrivKey, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("No node key found, generating a new one...", zap.String("path", path))

			// TODO(leo): what does -1 mean?
			priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
			if err != nil {
				panic(err)
			}

			s, err := crypto.MarshalPrivateKey(priv)
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

	priv, err := crypto.UnmarshalPrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node key: %w", err)
	}

	logger.Info("Found existing node key", zap.String("path", path))

	return priv, nil
}

// deterministicNodeKey returns a non-nil value if we have a deterministic key on file for the current host.
func deterministicNodeKey() crypto.PrivKey {
	idx, err := getDevnetIndex()
	if err != nil {
		panic(err)
	}

	if idx == 0 {
		// node ID: 12D3KooWQ1sV2kowPY1iJX1hJcVTysZjKv3sfULTGwhdpUGGZ1VF
		b, err := base64.StdEncoding.DecodeString("CAESQGlv6OJOMXrZZVTCC0cgCv7goXr6QaSVMZIndOIXKNh80vYnG+EutVlZK20Nx9cLkUG5ymKB\n88LXi/vPBwP8zfY=")
		if err != nil {
			panic(err)
		}

		priv, err := crypto.UnmarshalPrivateKey(b)
		if err != nil {
			panic(err)
		}

		return priv
	}

	return nil
}
