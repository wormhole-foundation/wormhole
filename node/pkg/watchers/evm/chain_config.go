package evm

// This file defines the set of EVM chains supported by the guardian watcher, including their EVM chain IDs and whether they support finalized and / or safe blocks.
// There is data for both Mainnet and Testnet. A chain should only be populated in the tables if the watcher should be allowed in that environment.
// Wherever possible, a public RPC endpoint is included so the verification tool can confirm that the EVM chain IDs specified here match the values of a known public node.

// NOTE: Whenever changes are made to the config data in this file, you should do the following:
//    node/pkg/watcher/evm$ go test
//    node/pkg/watcher/evm/verify_chain_config$ go run verify.go

// TODO: In a future PR we could consider merging the data in `node/hack/repair_eth/repair_eth.go` into here.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// EnvEntry specifies the config data for a given chain / environment.
	EnvEntry struct {
		// InstantFinality indicates that the chain has instant finality, meaning finalized and safe blocks can be generated along with the latest block.
		// Note that if InstantFinality is set to true, Finalized and Safe are for documentation purposes only.
		InstantFinality bool

		// Finalized indicates if the chain supports querying for finalized blocks.
		Finalized bool

		// Safe indicates if the chain supports querying for safe blocks.
		Safe bool

		// EvmChainID is the expected EVM chain ID (what is returned by the `eth_chainId` RPC call).
		EvmChainID uint64

		// PublicRPC is not actually used by the watcher. It's used to verify that the EvmChainID specified here is correct.
		PublicRPC string
	}

	// EnvMap defines the config data for a given environment (mainet or testnet).
	EnvMap map[vaa.ChainID]EnvEntry
)

var (
	ErrInvalidEnv = errors.New("invalid environment")
	ErrNotFound   = errors.New("not found")

	// mainnetChainConfig specifies the configuration for all chains enabled in Mainnet.
	// NOTE: Only add a chain here if the watcher should allow it in Mainnet!
	// NOTE: If you change this data, be sure and run the tests described at the top of this file!
	mainnetChainConfig = EnvMap{
		vaa.ChainIDEthereum: {Finalized: true, Safe: true, EvmChainID: 1, PublicRPC: "https://ethereum-rpc.publicnode.com"},
		vaa.ChainIDBSC:      {Finalized: true, Safe: true, EvmChainID: 56, PublicRPC: "https://bsc-rpc.publicnode.com"},

		// Polygon supports polling for finalized but not safe: https://forum.polygon.technology/t/optimizing-decentralized-apps-ux-with-milestones-a-significantly-accelerated-finality-solution/13154
		vaa.ChainIDPolygon: {Finalized: true, Safe: false, EvmChainID: 137, PublicRPC: "https://polygon-bor-rpc.publicnode.com"},

		vaa.ChainIDAvalanche: {InstantFinality: true, Finalized: true, Safe: true, EvmChainID: 43114, PublicRPC: "https://avalanche-c-chain-rpc.publicnode.com"},
		vaa.ChainIDOasis:     {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 42262, PublicRPC: "https://emerald.oasis.dev/"},
		// vaa.ChainIDAurora:    Not supported in the guardian.
		vaa.ChainIDFantom:   {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 250, PublicRPC: "https://fantom-rpc.publicnode.com"},
		vaa.ChainIDKarura:   {Finalized: true, Safe: true, EvmChainID: 686, PublicRPC: "https://eth-rpc-karura.aca-api.network/"},
		vaa.ChainIDAcala:    {Finalized: true, Safe: true, EvmChainID: 787, PublicRPC: "https://eth-rpc-acala.aca-api.network/"},
		vaa.ChainIDKlaytn:   {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 8217, PublicRPC: "https://public-en.node.kaia.io"},
		vaa.ChainIDCelo:     {Finalized: true, Safe: false, EvmChainID: 42220, PublicRPC: "https://celo-rpc.publicnode.com"},
		vaa.ChainIDMoonbeam: {Finalized: true, Safe: true, EvmChainID: 1284, PublicRPC: "https://moonbeam-rpc.publicnode.com"},
		vaa.ChainIDArbitrum: {Finalized: true, Safe: true, EvmChainID: 42161, PublicRPC: "https://arbitrum-one-rpc.publicnode.com"},
		vaa.ChainIDOptimism: {Finalized: true, Safe: true, EvmChainID: 10, PublicRPC: "https://optimism-rpc.publicnode.com"},
		// vaa.ChainIDGnosis:     Not supported in the guardian.
		// vaa.ChainIDBtc:        Not supported in the guardian.
		vaa.ChainIDBase: {Finalized: true, Safe: true, EvmChainID: 8453, PublicRPC: "https://base-rpc.publicnode.com"},
		// vaa.ChainIDFileCoin:   Not supported in the guardian.
		// vaa.ChainIDRootstock:  Not supported in the guardian.

		// As of 11/10/2023 Scroll supports polling for finalized but not safe.
		vaa.ChainIDScroll: {Finalized: true, Safe: false, EvmChainID: 534352, PublicRPC: "https://scroll-rpc.publicnode.com"},

		vaa.ChainIDMantle: {Finalized: true, Safe: true, EvmChainID: 5000, PublicRPC: "https://mantle-rpc.publicnode.com"},
		vaa.ChainIDBlast:  {Finalized: true, Safe: true, EvmChainID: 81457, PublicRPC: "https://blast-rpc.publicnode.com"},
		vaa.ChainIDXLayer: {Finalized: true, Safe: true, EvmChainID: 196, PublicRPC: "https://xlayerrpc.okx.com"},

		// As of 9/06/2024 Linea supports polling for finalized but not safe.
		vaa.ChainIDLinea: {Finalized: true, Safe: false, EvmChainID: 59144, PublicRPC: "https://rpc.linea.build"},

		vaa.ChainIDBerachain: {InstantFinality: true, Finalized: true, Safe: true, EvmChainID: 80094, PublicRPC: "https://berachain-rpc.publicnode.com"},
		// vaa.ChainIDSeiEVM:     Not in Mainnet yet.
		// vaa.ChainIDEclipse:    Not supported in the guardian.
		// vaa.ChainIDBOB:        Not supported in the guardian.
		vaa.ChainIDSnaxchain:  {Finalized: true, Safe: true, EvmChainID: 2192, PublicRPC: "https://mainnet.snaxchain.io"},
		vaa.ChainIDUnichain:   {Finalized: true, Safe: true, EvmChainID: 130, PublicRPC: "https://unichain-rpc.publicnode.com"},
		vaa.ChainIDWorldchain: {Finalized: true, Safe: true, EvmChainID: 480, PublicRPC: "https://worldchain-mainnet.g.alchemy.com/public"},
		// vaa.ChainIDInk:        Not in Mainnet yet.
		// vaa.ChainIDHyperEVM:   Not in Mainnet yet.
		// vaa.ChainIDMonad:      Not in Mainnet yet.
	}

	// testnetChainConfig specifies the configuration for all chains enabled in Testnet.
	// NOTE: Only add a chain here if the watcher should allow it in Testnet.
	// NOTE: If you change this data, be sure and run the tests described at the top of this file!
	testnetChainConfig = EnvMap{
		// For Ethereum testnet we actually use Holeksy since Goerli is deprecated.
		vaa.ChainIDEthereum: {Finalized: true, Safe: true, EvmChainID: 17000, PublicRPC: "https://1rpc.io/holesky"},
		vaa.ChainIDBSC:      {Finalized: true, Safe: true, EvmChainID: 97, PublicRPC: "https://bsc-testnet-rpc.publicnode.com"},

		// Polygon supports polling for finalized but not safe: https://forum.polygon.technology/t/optimizing-decentralized-apps-ux-with-milestones-a-significantly-accelerated-finality-solution/13154
		vaa.ChainIDPolygon: {Finalized: true, Safe: false, EvmChainID: 80001}, // Polygon Mumbai is deprecated.

		vaa.ChainIDAvalanche: {InstantFinality: true, Finalized: true, Safe: true, EvmChainID: 43113, PublicRPC: "https://avalanche-fuji-c-chain-rpc.publicnode.com"},
		vaa.ChainIDOasis:     {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 42261, PublicRPC: "https://testnet.emerald.oasis.dev"},
		// vaa.ChainIDAurora:    Not supported in the guardian.
		vaa.ChainIDFantom:   {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 4002, PublicRPC: "https://fantom-testnet-rpc.publicnode.com"},
		vaa.ChainIDKarura:   {Finalized: true, Safe: true, EvmChainID: 596, PublicRPC: "https://eth-rpc-karura-testnet.aca-staging.network"},
		vaa.ChainIDAcala:    {Finalized: true, Safe: true, EvmChainID: 597, PublicRPC: "https://eth-rpc-acala-testnet.aca-staging.network"},
		vaa.ChainIDKlaytn:   {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 1001, PublicRPC: "https://public-en-kairos.node.kaia.io"},
		vaa.ChainIDCelo:     {Finalized: true, Safe: true, EvmChainID: 44787, PublicRPC: "https://alfajores-forno.celo-testnet.org"},
		vaa.ChainIDMoonbeam: {Finalized: true, Safe: true, EvmChainID: 1287, PublicRPC: "https://rpc.api.moonbase.moonbeam.network"},
		vaa.ChainIDArbitrum: {Finalized: true, Safe: true, EvmChainID: 421613}, // Arbitrum Goerli is deprecated.
		vaa.ChainIDOptimism: {Finalized: true, Safe: true, EvmChainID: 420},    // Optimism Goerli is deprecated.
		// vaa.ChainIDGnosis:      Not supported in the guardian.
		// vaa.ChainIDBtc:         Not supported in the guardian.
		vaa.ChainIDBase: {Finalized: true, Safe: true, EvmChainID: 84531}, // Base Goerli is deprecated.
		// vaa.ChainIDFileCoin:    Not supported in the guardian.
		// vaa.ChainIDRootstock:   Not supported in the guardian.

		// As of 11/10/2023 Scroll supports polling for finalized but not safe.
		vaa.ChainIDScroll: {Finalized: true, Safe: false, EvmChainID: 534351, PublicRPC: "https://scroll-sepolia-rpc.publicnode.com"},

		vaa.ChainIDMantle: {Finalized: true, Safe: true, EvmChainID: 5003, PublicRPC: "https://rpc.sepolia.mantle.xyz"},
		vaa.ChainIDBlast:  {Finalized: true, Safe: true, EvmChainID: 168587773, PublicRPC: "https://sepolia.blast.io"},
		vaa.ChainIDXLayer: {Finalized: true, Safe: true, EvmChainID: 195, PublicRPC: "https://xlayertestrpc.okx.com"},

		// As of 9/06/2024 Linea supports polling for finalized but not safe.
		vaa.ChainIDLinea: {Finalized: true, Safe: false, EvmChainID: 59141, PublicRPC: "https://rpc.sepolia.linea.build"},

		vaa.ChainIDBerachain: {InstantFinality: true, Finalized: true, Safe: true, EvmChainID: 80084, PublicRPC: "https://bartio.rpc.berachain.com"},
		vaa.ChainIDSeiEVM:    {Finalized: true, Safe: true, EvmChainID: 1328, PublicRPC: "https://evm-rpc-testnet.sei-apis.com/"},
		// vaa.ChainIDEclipse:     Not supported in the guardian.
		// vaa.ChainIDBOB:         Not supported in the guardian.
		vaa.ChainIDSnaxchain:       {Finalized: true, Safe: true, EvmChainID: 13001, PublicRPC: "https://testnet.snaxchain.io"},
		vaa.ChainIDUnichain:        {Finalized: true, Safe: true, EvmChainID: 1301, PublicRPC: "https://unichain-sepolia-rpc.publicnode.com"},
		vaa.ChainIDWorldchain:      {Finalized: true, Safe: true, EvmChainID: 4801, PublicRPC: "https://worldchain-sepolia.g.alchemy.com/public"},
		vaa.ChainIDInk:             {Finalized: true, Safe: true, EvmChainID: 763373, PublicRPC: "https://rpc-qnd-sepolia.inkonchain.com"},
		vaa.ChainIDHyperEVM:        {Finalized: true, Safe: true, EvmChainID: 998, PublicRPC: "https://rpc.hyperliquid-testnet.xyz/evm"},
		vaa.ChainIDMonad:           {Finalized: true, Safe: true, EvmChainID: 10143, PublicRPC: "https://testnet-rpc.monad.xyz"},
		vaa.ChainIDSepolia:         {Finalized: true, Safe: true, EvmChainID: 11155111, PublicRPC: "https://ethereum-sepolia-rpc.publicnode.com"},
		vaa.ChainIDArbitrumSepolia: {Finalized: true, Safe: true, EvmChainID: 421614, PublicRPC: "https://arbitrum-sepolia-rpc.publicnode.com"},
		vaa.ChainIDBaseSepolia:     {Finalized: true, Safe: true, EvmChainID: 84532, PublicRPC: "https://base-sepolia-rpc.publicnode.com"},
		vaa.ChainIDOptimismSepolia: {Finalized: true, Safe: true, EvmChainID: 11155420, PublicRPC: "https://optimism-sepolia-rpc.publicnode.com"},
		vaa.ChainIDHolesky:         {Finalized: true, Safe: true, EvmChainID: 17000, PublicRPC: "https://1rpc.io/holesky"},
		vaa.ChainIDPolygonSepolia:  {Finalized: true, Safe: false, EvmChainID: 80002, PublicRPC: "https://polygon-amoy-bor-rpc.publicnode.com"},
	}
)

// SupportedInMainnet returns true if the chain is configured in Mainnet.
func SupportedInMainnet(chainID vaa.ChainID) bool {
	_, exists := mainnetChainConfig[chainID]
	return exists
}

// GetFinality returns the finalized and safe flags for the specified environment / chain. These are used to configure the watcher at run time.
func GetFinality(env common.Environment, chainID vaa.ChainID) (finalized bool, safe bool, err error) {
	// Tilt supports polling for both finalized and safe.
	if env == common.UnsafeDevNet {
		return true, true, nil
	}

	m, err := GetChainConfigMap(env)
	if err != nil {
		return false, false, err
	}

	entry, exists := m[chainID]
	if !exists {
		return false, false, ErrNotFound
	}

	if entry.InstantFinality {
		return false, false, nil
	}

	return entry.Finalized, entry.Safe, nil
}

// GetEvmChainID returns the configured EVM chain ID for the specified environment / chain.
func GetEvmChainID(env common.Environment, chainID vaa.ChainID) (uint64, error) {
	m, err := GetChainConfigMap(env)
	if err != nil {
		return 0, err
	}

	entry, exists := m[chainID]
	if !exists {
		return 0, ErrNotFound
	}

	return entry.EvmChainID, nil
}

// GetChainConfigMap is a helper that returns the configuration for the specified environment.
// This is public so the chain verify utility can use it.
func GetChainConfigMap(env common.Environment) (EnvMap, error) {
	if env == common.MainNet {
		return mainnetChainConfig, nil
	}

	if env == common.TestNet {
		return testnetChainConfig, nil
	}

	return EnvMap{}, ErrInvalidEnv
}

// QueryEvmChainID queries the specified RPC for the EVM chain ID.
func QueryEvmChainID(ctx context.Context, url string) (uint64, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c, err := rpc.DialContext(timeout, url)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to endpoint: %w", err)
	}

	var str string
	err = c.CallContext(ctx, &str, "eth_chainId")
	if err != nil {
		return 0, fmt.Errorf("failed to read evm chain id: %w", err)
	}

	evmChainID, err := strconv.ParseUint(strings.TrimPrefix(str, "0x"), 16, 64)
	if err != nil {
		return 0, fmt.Errorf(`eth_chainId returned an invalid int: "%s"`, str)
	}

	return evmChainID, nil
}

// verifyEvmChainID reads the EVM chain ID from the node and verifies that it matches the expected value (making sure we aren't connected to the wrong chain).
func (w *Watcher) verifyEvmChainID(ctx context.Context, logger *zap.Logger, url string) error {
	// Don't bother to check in tilt.
	if w.env == common.UnsafeDevNet {
		return nil
	}

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	c, err := rpc.DialContext(timeout, url)
	if err != nil {
		return fmt.Errorf("failed to connect to endpoint: %w", err)
	}

	var str string
	err = c.CallContext(ctx, &str, "eth_chainId")
	if err != nil {
		return fmt.Errorf("failed to read evm chain id: %w", err)
	}

	evmChainID, err := strconv.ParseUint(strings.TrimPrefix(str, "0x"), 16, 64)
	if err != nil {
		return fmt.Errorf(`eth_chainId returned an invalid int: "%s"`, str)
	}

	expectedEvmChainID, err := GetEvmChainID(w.env, w.chainID)
	if err != nil {
		return fmt.Errorf("failed to look up evm chain id: %w", err)
	}

	logger.Info("queried evm chain id", zap.Uint64("expected", expectedEvmChainID), zap.Uint64("actual", evmChainID))

	if evmChainID != uint64(expectedEvmChainID) {
		return fmt.Errorf("evm chain ID miss match, expected %d, received %d", expectedEvmChainID, evmChainID)
	}

	return nil
}
