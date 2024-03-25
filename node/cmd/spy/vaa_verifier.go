package spy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

// VaaVerifier is an object that can be used to validate VAA signatures.
// It reads the guardian set on chain whenever a new guardian set index is detected.
type VaaVerifier struct {
	logger       *zap.Logger
	rpcUrl       string
	coreAddr     ethCommon.Address
	lock         sync.Mutex
	guardianSets map[uint32]*common.GuardianSet
}

// RpcTimeout is the context timeout on RPC calls.
const RpcTimeout = time.Second * 5

// NewVaaVerifier creates a VaaVerifier.
func NewVaaVerifier(logger *zap.Logger, rpcUrl string, coreAddr string) *VaaVerifier {
	return &VaaVerifier{
		logger:       logger,
		rpcUrl:       rpcUrl,
		coreAddr:     ethCommon.HexToAddress(coreAddr),
		guardianSets: make(map[uint32]*common.GuardianSet),
	}
}

// GetInitialGuardianSet gets the current guardian set and adds it to the map. It is not necessary
// to call this function, but doing so will allow you to verify that the RPC endpoint works on start up,
// rather than having it fail the first VAA is received.
func (v *VaaVerifier) GetInitialGuardianSet() error {
	timeout, cancel := context.WithTimeout(context.Background(), RpcTimeout)
	defer cancel()

	rawClient, err := ethRpc.DialContext(timeout, v.rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(v.coreAddr, client)
	if err != nil {
		return fmt.Errorf("failed to create caller: %w", err)
	}

	gsIndex, err := caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: timeout})
	if err != nil {
		return fmt.Errorf("error requesting current guardian set index: %w", err)
	}

	result, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: timeout}, gsIndex)
	if err != nil {
		return fmt.Errorf("error requesting guardian set for index %d: %w", gsIndex, err)
	}

	gs := &common.GuardianSet{
		Keys:  result.Keys,
		Index: gsIndex,
	}

	v.logger.Warn("read current guardian set", zap.Uint32("index", gsIndex), zap.Any("gs", *gs))
	v.guardianSets[gsIndex] = gs
	return nil
}

// VerifySignatures verifies that the signature on a VAA is valid, based on the guardian set contained in the VAA.
// If the guardian set is not currently in our map, it queries that guardian set and adds it.
func (v *VaaVerifier) VerifySignatures(vv *vaa.VAA) (bool, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	gs, exists := v.guardianSets[vv.GuardianSetIndex]
	if !exists {
		var err error
		gs, err = v.fetchGuardianSet(vv.GuardianSetIndex)
		if err != nil {
			return false, fmt.Errorf("failed to fetch guardian set for index %d: %w", vv.GuardianSetIndex, err)
		}

		v.logger.Warn("read guardian set", zap.Uint32("index", gs.Index), zap.Any("gs", *gs))
		v.guardianSets[gs.Index] = gs
	}

	if err := vv.Verify(gs.Keys); err != nil {
		return false, nil
	}

	return true, nil
}

// fetchGuardianSet reads the guardian set for the index passed in.
func (v *VaaVerifier) fetchGuardianSet(gsIndex uint32) (*common.GuardianSet, error) {
	timeout, cancel := context.WithTimeout(context.Background(), RpcTimeout)
	defer cancel()

	rawClient, err := ethRpc.DialContext(timeout, v.rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	client := ethClient.NewClient(rawClient)
	caller, err := ethAbi.NewAbiCaller(v.coreAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create caller: %w", err)
	}

	gs, err := caller.GetGuardianSet(&ethBind.CallOpts{Context: timeout}, gsIndex)
	if err != nil {
		return nil, fmt.Errorf("error requesting current guardian set for index %d: %w", gsIndex, err)
	}

	return &common.GuardianSet{
		Keys:  gs.Keys,
		Index: gsIndex,
	}, nil
}
