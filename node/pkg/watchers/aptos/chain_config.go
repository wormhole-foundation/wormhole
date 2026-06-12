package aptos

// This file defines the set of Aptos-derived chains supported by the guardian watcher and their native chain IDs.
// The native chain ID is what the node returns in the `chain_id` field of its `/v1` endpoint.

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// EnvEntry specifies the config data for a given chain / environment.
	EnvEntry struct {
		// AptosChainID is the expected native chain ID (the `chain_id` field returned by the `/v1` endpoint).
		AptosChainID uint64
	}

	// EnvMap defines the config data for a given environment (mainnet or testnet).
	EnvMap map[vaa.ChainID]EnvEntry
)

const (
	aptosMainnetChainID    uint64 = 1
	aptosTestnetChainID    uint64 = 2
	movementMainnetChainID uint64 = 126
	movementTestnetChainID uint64 = 250

	chainIDQueryTimeout = 15 * time.Second
)

var (
	ErrInvalidEnv = errors.New("invalid environment")
	ErrNotFound   = errors.New("not found")

	mainnetChainConfig = EnvMap{
		vaa.ChainIDAptos:    {AptosChainID: aptosMainnetChainID},
		vaa.ChainIDMovement: {AptosChainID: movementMainnetChainID},
	}

	testnetChainConfig = EnvMap{
		vaa.ChainIDAptos:    {AptosChainID: aptosTestnetChainID},
		vaa.ChainIDMovement: {AptosChainID: movementTestnetChainID},
	}
)

// GetAptosChainID returns the configured native chain ID for the specified environment / chain.
func GetAptosChainID(env common.Environment, chainID vaa.ChainID) (uint64, error) {
	m, err := GetChainConfigMap(env)
	if err != nil {
		return 0, err
	}

	entry, exists := m[chainID]
	if !exists {
		return 0, ErrNotFound
	}

	return entry.AptosChainID, nil
}

// GetChainConfigMap returns the configuration map for the specified environment.
func GetChainConfigMap(env common.Environment) (EnvMap, error) {
	if env == common.MainNet {
		return mainnetChainConfig, nil
	}

	if env == common.TestNet {
		return testnetChainConfig, nil
	}

	return EnvMap{}, ErrInvalidEnv
}

// verifyAptosChainID reads the native chain ID from the node and verifies that it matches the expected value
// (making sure we aren't connected to the wrong chain).
func (e *Watcher) verifyAptosChainID(ctx context.Context, logger *zap.Logger, url string) error {
	// Don't bother to check in tilt.
	if e.env == common.UnsafeDevNet {
		return nil
	}

	expected, err := GetAptosChainID(e.env, e.chainID)
	if err != nil {
		return fmt.Errorf("failed to look up aptos chain id: %w", err)
	}

	timeout, cancel := context.WithTimeout(ctx, chainIDQueryTimeout)
	defer cancel()

	actual, err := queryAptosChainID(timeout, url)
	if err != nil {
		return err
	}

	logger.Info("queried aptos chain id", zap.Uint64("expected", expected), zap.Uint64("actual", actual))

	if actual != expected {
		return fmt.Errorf("aptos chain ID mismatch, expected %d, received %d", expected, actual)
	}

	return nil
}

// queryAptosChainID queries the specified RPC for the native Aptos chain ID returned by the `/v1` endpoint.
func queryAptosChainID(ctx context.Context, url string) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/v1", url), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build chain id request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to query aptos chain id: %w", err)
	}
	defer res.Body.Close()

	body, err := common.SafeRead(res.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read aptos chain id response: %w", err)
	}

	if !gjson.Valid(string(body)) {
		return 0, fmt.Errorf("invalid JSON in chain id response: %s", string(body))
	}

	id := gjson.GetBytes(body, "chain_id")
	if !id.Exists() {
		return 0, fmt.Errorf("chain_id field missing from response")
	}

	v := id.Uint()
	if v == 0 || v > math.MaxUint32 {
		return 0, fmt.Errorf("chain_id %d out of expected range", v)
	}
	return v, nil
}
