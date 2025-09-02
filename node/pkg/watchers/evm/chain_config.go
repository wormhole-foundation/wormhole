package evm

// This file defines the set of EVM chains supported by the guardian watcher, including their EVM chain IDs and whether they support finalized and / or safe blocks.
// There is data for both Mainnet and Testnet. A chain should only be populated in the tables if the watcher should be allowed in that environment.
// Wherever possible, a public RPC endpoint is included so the verification tool can confirm that the EVM chain IDs specified here match the values of a known public node.

// NOTE: Whenever changes are made to the config data in this file, you should do the following:
//    node/pkg/watcher/evm$ go test
//    node/pkg/watcher/evm/verify_chain_config$ go run verify.go

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
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

		// ContractAddr specifies the Wormhole core contract address for this chain (starting with 0x).
		// SECURITY: This is for documentation and validation only. Allowing it as a default would provide a single point attack vector.
		ContractAddr string

		// CCLContractAddr specifies the address of the custom consistency level contract for this chain (starting with 0x).
		// SECURITY: This is for documentation and validation only. Allowing it as a default would provide a single point attack vector.
		CCLContractAddr string
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
		vaa.ChainIDEthereum: {Finalized: true, Safe: true, EvmChainID: 1, PublicRPC: "https://ethereum-rpc.publicnode.com", ContractAddr: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"},
		vaa.ChainIDBSC:      {Finalized: true, Safe: true, EvmChainID: 56, PublicRPC: "https://bsc-rpc.publicnode.com", ContractAddr: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"},

		// Polygon supports polling for finalized but not safe: https://forum.polygon.technology/t/optimizing-decentralized-apps-ux-with-milestones-a-significantly-accelerated-finality-solution/13154
		vaa.ChainIDPolygon: {Finalized: true, Safe: false, EvmChainID: 137, PublicRPC: "https://polygon-bor-rpc.publicnode.com", ContractAddr: "0x7A4B5a56256163F07b2C80A7cA55aBE66c4ec4d7"},

		vaa.ChainIDAvalanche: {InstantFinality: true, Finalized: true, Safe: true, EvmChainID: 43114, PublicRPC: "https://avalanche-c-chain-rpc.publicnode.com", ContractAddr: "0x54a8e5f9c4CbA08F9943965859F6c34eAF03E26c"},
		vaa.ChainIDFantom:    {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 250, PublicRPC: "https://fantom-rpc.publicnode.com", ContractAddr: "0x126783A6Cb203a3E35344528B26ca3a0489a1485"},
		vaa.ChainIDKlaytn:    {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 8217, PublicRPC: "https://public-en.node.kaia.io", ContractAddr: "0x0C21603c4f3a6387e241c0091A7EA39E43E90bb7"},
		vaa.ChainIDCelo:      {Finalized: true, Safe: false, EvmChainID: 42220, PublicRPC: "https://celo-rpc.publicnode.com", ContractAddr: "0xa321448d90d4e5b0A732867c18eA198e75CAC48E"},
		vaa.ChainIDMoonbeam:  {Finalized: true, Safe: true, EvmChainID: 1284, PublicRPC: "https://moonbeam-rpc.publicnode.com", ContractAddr: "0xC8e2b0cD52Cf01b0Ce87d389Daa3d414d4cE29f3"},
		vaa.ChainIDArbitrum:  {Finalized: true, Safe: true, EvmChainID: 42161, PublicRPC: "https://arbitrum-one-rpc.publicnode.com", ContractAddr: "0xa5f208e072434bC67592E4C49C1B991BA79BCA46"},
		vaa.ChainIDOptimism:  {Finalized: true, Safe: true, EvmChainID: 10, PublicRPC: "https://optimism-rpc.publicnode.com", ContractAddr: "0xEe91C335eab126dF5fDB3797EA9d6aD93aeC9722"},
		// vaa.ChainIDGnosis:     Not supported in the guardian.
		// vaa.ChainIDBtc:        Not supported in the guardian.
		vaa.ChainIDBase: {Finalized: true, Safe: true, EvmChainID: 8453, PublicRPC: "https://base-rpc.publicnode.com", ContractAddr: "0xbebdb6C8ddC678FfA9f8748f85C815C556Dd8ac6"},
		// vaa.ChainIDFileCoin:   Not supported in the guardian.
		// vaa.ChainIDRootstock:  Not supported in the guardian.

		// As of 11/10/2023 Scroll supports polling for finalized but not safe.
		vaa.ChainIDScroll: {Finalized: true, Safe: false, EvmChainID: 534352, PublicRPC: "https://scroll-rpc.publicnode.com", ContractAddr: "0xbebdb6C8ddC678FfA9f8748f85C815C556Dd8ac6"},

		vaa.ChainIDMantle: {Finalized: true, Safe: true, EvmChainID: 5000, PublicRPC: "https://mantle-rpc.publicnode.com", ContractAddr: "0xbebdb6C8ddC678FfA9f8748f85C815C556Dd8ac6"},
		vaa.ChainIDXLayer: {Finalized: true, Safe: true, EvmChainID: 196, PublicRPC: "https://xlayerrpc.okx.com", ContractAddr: "0x194B123c5E96B9b2E49763619985790Dc241CAC0"},
		// As of 9/06/2024 Linea supports polling for finalized but not safe.
		vaa.ChainIDLinea:     {Finalized: true, Safe: false, EvmChainID: 59144, PublicRPC: "https://rpc.linea.build", ContractAddr: "0x0C56aebD76E6D9e4a1Ec5e94F4162B4CBbf77b32"},
		vaa.ChainIDBerachain: {Finalized: true, Safe: true, EvmChainID: 80094, PublicRPC: "https://berachain-rpc.publicnode.com", ContractAddr: "0xCa1D5a146B03f6303baF59e5AD5615ae0b9d146D"},
		vaa.ChainIDSeiEVM:    {Finalized: true, Safe: true, EvmChainID: 1329, PublicRPC: "https://evm-rpc.sei-apis.com", ContractAddr: "0xCa1D5a146B03f6303baF59e5AD5615ae0b9d146D"},
		// vaa.ChainIDEclipse:    Not supported in the guardian.
		// vaa.ChainIDBOB:        Not supported in the guardian.
		vaa.ChainIDUnichain:   {Finalized: true, Safe: true, EvmChainID: 130, PublicRPC: "https://unichain-rpc.publicnode.com", ContractAddr: "0xCa1D5a146B03f6303baF59e5AD5615ae0b9d146D"},
		vaa.ChainIDWorldchain: {Finalized: true, Safe: true, EvmChainID: 480, PublicRPC: "https://worldchain-mainnet.g.alchemy.com/public", ContractAddr: "0xcbcEe4e081464A15d8Ad5f58BB493954421eB506"},
		vaa.ChainIDInk:        {Finalized: true, Safe: true, EvmChainID: 57073, PublicRPC: "https://rpc-qnd.inkonchain.com", ContractAddr: "0xCa1D5a146B03f6303baF59e5AD5615ae0b9d146D"},
		vaa.ChainIDHyperEVM:   {Finalized: true, Safe: true, EvmChainID: 999, PublicRPC: "https://rpc.hyperliquid.xyz/evm", ContractAddr: "0x7C0faFc4384551f063e05aee704ab943b8B53aB3"},
		// vaa.ChainIDMonad:      Not in Mainnet yet.
		vaa.ChainIDMezo: {Finalized: true, Safe: true, EvmChainID: 31612, PublicRPC: "https://jsonrpc-mezo.boar.network/", ContractAddr: "0xaBf89de706B583424328B54dD05a8fC986750Da8"},
		// vaa.ChainIDConverge: Not in Mainnet yet
		vaa.ChainIDPlume:   {Finalized: true, Safe: true, EvmChainID: 98866, PublicRPC: "https://rpc.plume.org", ContractAddr: "0xaBf89de706B583424328B54dD05a8fC986750Da8"},
		vaa.ChainIDXRPLEVM: {Finalized: true, Safe: true, EvmChainID: 1440000, PublicRPC: "https://rpc.xrplevm.org/", ContractAddr: "0xaBf89de706B583424328B54dD05a8fC986750Da8"},
	}

	// testnetChainConfig specifies the configuration for all chains enabled in Testnet.
	// NOTE: Only add a chain here if the watcher should allow it in Testnet.
	// NOTE: If you change this data, be sure and run the tests described at the top of this file!
	testnetChainConfig = EnvMap{
		// As of 2025 September we use Sepolia as the default Ethereum testnet, given that Holesky is being deprecated.
		vaa.ChainIDEthereum: {Finalized: true, Safe: true, EvmChainID: 11155111, PublicRPC: "https://ethereum-sepolia-rpc.publicnode.com", ContractAddr: "0x4a8bc80Ed5a4067f1CCf107057b8270E0cC11A78"},
		vaa.ChainIDBSC:      {Finalized: true, Safe: true, EvmChainID: 97, PublicRPC: "https://bsc-testnet-rpc.publicnode.com", ContractAddr: "0x68605AD7b15c732a30b1BbC62BE8F2A509D74b4D"},

		// Polygon supports polling for finalized but not safe: https://forum.polygon.technology/t/optimizing-decentralized-apps-ux-with-milestones-a-significantly-accelerated-finality-solution/13154
		vaa.ChainIDPolygon: {Finalized: true, Safe: false, EvmChainID: 80001, ContractAddr: "0x0CBE91CF822c73C2315FB05100C2F714765d5c20"}, // Polygon Mumbai is deprecated.

		vaa.ChainIDAvalanche: {InstantFinality: true, Finalized: true, Safe: true, EvmChainID: 43113, PublicRPC: "https://avalanche-fuji-c-chain-rpc.publicnode.com", ContractAddr: "0x7bbcE28e64B3F8b84d876Ab298393c38ad7aac4C"},
		vaa.ChainIDFantom:    {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 4002, PublicRPC: "https://fantom-testnet-rpc.publicnode.com", ContractAddr: "0x1BB3B4119b7BA9dfad76B0545fb3F531383c3bB7"},
		vaa.ChainIDKlaytn:    {InstantFinality: true, Finalized: false, Safe: false, EvmChainID: 1001, PublicRPC: "https://public-en-kairos.node.kaia.io", ContractAddr: "0x1830CC6eE66c84D2F177B94D544967c774E624cA"},
		vaa.ChainIDCelo:      {Finalized: true, Safe: true, EvmChainID: 44787, PublicRPC: "https://alfajores-forno.celo-testnet.org", ContractAddr: "0x88505117CA88e7dd2eC6EA1E13f0948db2D50D56"},
		vaa.ChainIDMoonbeam:  {Finalized: true, Safe: true, EvmChainID: 1287, PublicRPC: "https://rpc.api.moonbase.moonbeam.network", ContractAddr: "0xa5B7D85a8f27dd7907dc8FdC21FA5657D5E2F901"},
		vaa.ChainIDArbitrum:  {Finalized: true, Safe: true, EvmChainID: 421613, ContractAddr: "0xC7A204bDBFe983FCD8d8E61D02b475D4073fF97e"}, // Arbitrum Goerli is deprecated.
		vaa.ChainIDOptimism:  {Finalized: true, Safe: true, EvmChainID: 420, ContractAddr: "0x6b9C8671cdDC8dEab9c719bB87cBd3e782bA6a35"},    // Optimism Goerli is deprecated.
		// vaa.ChainIDGnosis:      Not supported in the guardian.
		// vaa.ChainIDBtc:         Not supported in the guardian.
		vaa.ChainIDBase: {Finalized: true, Safe: true, EvmChainID: 84531, ContractAddr: "0x23908A62110e21C04F3A4e011d24F901F911744A"}, // Base Goerli is deprecated.
		// vaa.ChainIDFileCoin:    Not supported in the guardian.
		// vaa.ChainIDRootstock:   Not supported in the guardian.

		// As of 11/10/2023 Scroll supports polling for finalized but not safe.
		vaa.ChainIDScroll: {Finalized: true, Safe: false, EvmChainID: 534351, PublicRPC: "https://scroll-sepolia-rpc.publicnode.com", ContractAddr: "0x055F47F1250012C6B20c436570a76e52c17Af2D5"},

		vaa.ChainIDMantle: {Finalized: true, Safe: true, EvmChainID: 5003, PublicRPC: "https://rpc.sepolia.mantle.xyz", ContractAddr: "0x376428e7f26D5867e69201b275553C45B09EE090"},
		vaa.ChainIDXLayer: {Finalized: true, Safe: true, EvmChainID: 195, PublicRPC: "https://xlayertestrpc.okx.com", ContractAddr: "0xA31aa3FDb7aF7Db93d18DDA4e19F811342EDF780"},

		// As of 9/06/2024 Linea supports polling for finalized but not safe.
		vaa.ChainIDLinea: {Finalized: true, Safe: false, EvmChainID: 59141, PublicRPC: "https://rpc.sepolia.linea.build", ContractAddr: "0x79A1027a6A159502049F10906D333EC57E95F083"},

		vaa.ChainIDBerachain: {Finalized: true, Safe: true, EvmChainID: 80069, PublicRPC: "https://bepolia.rpc.berachain.com/", ContractAddr: "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"},
		vaa.ChainIDSeiEVM:    {Finalized: true, Safe: true, EvmChainID: 1328, PublicRPC: "https://evm-rpc-testnet.sei-apis.com/", ContractAddr: "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"},
		// vaa.ChainIDEclipse:     Not supported in the guardian.
		// vaa.ChainIDBOB:         Not supported in the guardian.
		vaa.ChainIDUnichain:        {Finalized: true, Safe: true, EvmChainID: 1301, PublicRPC: "https://unichain-sepolia-rpc.publicnode.com", ContractAddr: "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"},
		vaa.ChainIDWorldchain:      {Finalized: true, Safe: true, EvmChainID: 4801, PublicRPC: "https://worldchain-sepolia.g.alchemy.com/public", ContractAddr: "0xe5E02cD12B6FcA153b0d7fF4bF55730AE7B3C93A"},
		vaa.ChainIDInk:             {Finalized: true, Safe: true, EvmChainID: 763373, PublicRPC: "https://rpc-qnd-sepolia.inkonchain.com", ContractAddr: "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"},
		vaa.ChainIDHyperEVM:        {Finalized: true, Safe: true, EvmChainID: 998, PublicRPC: "https://rpc.hyperliquid-testnet.xyz/evm", ContractAddr: "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"},
		vaa.ChainIDMonad:           {Finalized: true, Safe: true, EvmChainID: 10143, PublicRPC: "https://testnet-rpc.monad.xyz", ContractAddr: "0xBB73cB66C26740F31d1FabDC6b7A46a038A300dd"},
		vaa.ChainIDMezo:            {Finalized: true, Safe: true, EvmChainID: 31611, PublicRPC: "https://rpc.test.mezo.org", ContractAddr: "0x268557122Ffd64c85750d630b716471118F323c8"},
		vaa.ChainIDConverge:        {Finalized: true, Safe: true, EvmChainID: 52085145, PublicRPC: "https://rpc-converge-testnet-1.t.conduit.xyz", ContractAddr: "0x556B259cFaCd9896B2773310080c7c3bcE90Ff01"},
		vaa.ChainIDPlume:           {Finalized: true, Safe: true, EvmChainID: 98867, PublicRPC: "https://testnet-rpc.plume.org", ContractAddr: "0x81705b969cDcc6FbFde91a0C6777bE0EF3A75855"},
		vaa.ChainIDXRPLEVM:         {Finalized: true, Safe: true, EvmChainID: 1449000, PublicRPC: "https://rpc.testnet.xrplevm.org/", ContractAddr: "0xaBf89de706B583424328B54dD05a8fC986750Da8"},
		vaa.ChainIDSepolia:         {Finalized: true, Safe: true, EvmChainID: 11155111, PublicRPC: "https://ethereum-sepolia-rpc.publicnode.com", ContractAddr: "0x4a8bc80Ed5a4067f1CCf107057b8270E0cC11A78"},
		vaa.ChainIDArbitrumSepolia: {Finalized: true, Safe: true, EvmChainID: 421614, PublicRPC: "https://arbitrum-sepolia-rpc.publicnode.com", ContractAddr: "0x6b9C8671cdDC8dEab9c719bB87cBd3e782bA6a35"},
		vaa.ChainIDBaseSepolia:     {Finalized: true, Safe: true, EvmChainID: 84532, PublicRPC: "https://base-sepolia-rpc.publicnode.com", ContractAddr: "0x79A1027a6A159502049F10906D333EC57E95F083"},
		vaa.ChainIDOptimismSepolia: {Finalized: true, Safe: true, EvmChainID: 11155420, PublicRPC: "https://optimism-sepolia-rpc.publicnode.com", ContractAddr: "0x31377888146f3253211EFEf5c676D41ECe7D58Fe"},
		vaa.ChainIDHolesky:         {Finalized: true, Safe: true, EvmChainID: 17000, PublicRPC: "https://1rpc.io/holesky", ContractAddr: "0xa10f2eF61dE1f19f586ab8B6F2EbA89bACE63F7a"},
		vaa.ChainIDPolygonSepolia:  {Finalized: true, Safe: false, EvmChainID: 80002, PublicRPC: "https://polygon-amoy-bor-rpc.publicnode.com", ContractAddr: "0x6b9C8671cdDC8dEab9c719bB87cBd3e782bA6a35"},
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

// GetContractAddr returns the configured contract address for the specified environment / chain.
func GetContractAddrString(env common.Environment, chainID vaa.ChainID) (string, error) {
	m, err := GetChainConfigMap(env)
	if err != nil {
		return "", err
	}

	entry, exists := m[chainID]
	if !exists {
		return "", ErrNotFound
	}

	return entry.ContractAddr, nil
}

// GetContractAddr returns the configured contract address for the specified environment / chain.
func GetContractAddr(env common.Environment, chainID vaa.ChainID) (ethCommon.Address, error) {
	str, err := GetContractAddrString(env, chainID)
	if err != nil {
		return ethCommon.Address{}, err
	}

	return ethCommon.HexToAddress(str), nil
}
