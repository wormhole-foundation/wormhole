package connectors

import (
	"context"

	dgAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/delegated_guardians"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"
)

// DelegatedGuardiansConnector provides access to the WormholeDelegatedGuardians contract.
type DelegatedGuardiansConnector struct {
	networkName string
	address     ethCommon.Address
	logger      *zap.Logger
	client      *ethClient.Client
	rawClient   *ethRpc.Client
	caller      *dgAbi.DelegatedguardiansCaller
}

// NewDelegatedGuardiansConnector creates a new connector for the delegated guardians contract.
func NewDelegatedGuardiansConnector(ctx context.Context, networkName, rawUrl string, address ethCommon.Address, logger *zap.Logger) (*DelegatedGuardiansConnector, error) {
	rawClient, err := ethRpc.DialContext(ctx, rawUrl)
	if err != nil {
		return nil, err
	}

	client := ethClient.NewClient(rawClient)

	caller, err := dgAbi.NewDelegatedguardiansCaller(ethCommon.BytesToAddress(address.Bytes()), client)
	if err != nil {
		return nil, err
	}

	return &DelegatedGuardiansConnector{
		networkName: networkName,
		address:     address,
		logger:      logger,
		client:      client,
		rawClient:   rawClient,
		caller:      caller,
	}, nil
}

// GetConfig retrieves the delegated guardian configuration for all chains.
func (d *DelegatedGuardiansConnector) GetConfig(ctx context.Context) ([]dgAbi.WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return d.caller.GetConfig0(&ethBind.CallOpts{Context: ctx})
}

// GetConfigForChain retrieves the delegated guardian configuration for a specific chain.
func (d *DelegatedGuardiansConnector) GetConfigForChain(ctx context.Context, chainId uint16) (dgAbi.WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return d.caller.GetConfig(&ethBind.CallOpts{Context: ctx}, chainId)
}

// NetworkName returns the network name.
func (d *DelegatedGuardiansConnector) NetworkName() string {
	return d.networkName
}

// ContractAddress returns the contract address.
func (d *DelegatedGuardiansConnector) ContractAddress() ethCommon.Address {
	return d.address
}
