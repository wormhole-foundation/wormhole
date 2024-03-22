package spy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethBind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
)

// VaaVerifier is an object that can be used to verify VAAs are valid by calling `parseAndVerifyVM` on chain.
// It reads the guardian set on chain and periodically polls for guardian set updates.
type VaaVerifier struct {
	ctx                context.Context
	logger             *zap.Logger
	rpcUrl             string
	coreAddr           string
	caller             *ethAbi.AbiCaller
	lock               sync.Mutex
	currentIndex       uint32
	currentGuardianSet *common.GuardianSet
}

// GuardianSetIndexRefreshInterval specifies how often to poll for a guardian set update.
const GuardianSetIndexRefreshInterval = time.Second * 60

// RpcTimeout is the context timeout on RPC calls.
const RpcTimeout = time.Second * 5

// NewVaaVerifier creates a VaaVerifier.
func NewVaaVerifier(ctx context.Context, logger *zap.Logger, rpcUrl string, coreAddr string) *VaaVerifier {
	return &VaaVerifier{
		ctx:      ctx,
		logger:   logger,
		rpcUrl:   rpcUrl,
		coreAddr: coreAddr,
	}
}

// Start reads the initial guardian set and then starts a go routine to periodically poll for guardian set updates.
func (v *VaaVerifier) Start(errC chan error) error {
	err := v.connect()
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	v.currentIndex, err = v.fetchCurrentGuardianSetIndex()
	if err != nil {
		return fmt.Errorf("failed to fetch guardian set index: %w", err)
	}

	v.currentGuardianSet, err = v.fetchGuardianSet(v.currentIndex)
	if err != nil {
		return fmt.Errorf("failed to fetch guardian set index: %w", err)
	}

	// Log this as a warning in case they have info logging disabled.
	v.logger.Warn("read initial guardian set", zap.Uint32("index", v.currentIndex), zap.Any("gs", *v.currentGuardianSet))

	common.RunWithScissors(v.ctx, errC, "gs_poller", func(ctx context.Context) error {
		t := time.NewTicker(GuardianSetIndexRefreshInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				err := v.checkForGuardianSetUpdate()
				if err != nil {
					errC <- err
					return nil
				}
			}
		}
	})

	return nil
}

// connect creates the RPC connection and ABI caller to be used for reading the guardian set information.
func (v *VaaVerifier) connect() error {
	timeout, cancel := context.WithTimeout(v.ctx, 15*time.Second)
	defer cancel()

	ethContract := ethCommon.HexToAddress(v.coreAddr)
	rawClient, err := ethRpc.DialContext(timeout, v.rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	client := ethClient.NewClient(rawClient)
	v.caller, err = ethAbi.NewAbiCaller(ethContract, client)
	if err != nil {
		return fmt.Errorf("failed to create caller: %w", err)
	}

	return nil
}

// fetchCurrentGuardianSetIndex reads the current guardian set index.
func (v *VaaVerifier) fetchCurrentGuardianSetIndex() (uint32, error) {
	timeout, cancel := context.WithTimeout(v.ctx, RpcTimeout)
	defer cancel()
	currentIndex, err := v.caller.GetCurrentGuardianSetIndex(&ethBind.CallOpts{Context: timeout})
	if err != nil {
		return 0, fmt.Errorf("error requesting current guardian set index: %w", err)
	}
	return currentIndex, nil
}

// fetchGuardianSet reads the guardian set for the index passed in.
func (v *VaaVerifier) fetchGuardianSet(gsIndex uint32) (*common.GuardianSet, error) {
	timeout, cancel := context.WithTimeout(v.ctx, RpcTimeout)
	defer cancel()
	gs, err := v.caller.GetGuardianSet(&ethBind.CallOpts{Context: timeout}, gsIndex)
	if err != nil {
		return nil, fmt.Errorf("error requesting current guardian set for index %d: %w", gsIndex, err)
	}
	return &common.GuardianSet{
		Keys:  gs.Keys,
		Index: gsIndex,
	}, nil
}

// checkForGuardianSetUpdate checks to see if the guardian set has changed. It does this by reading the guardian set index
// and comparing it to the last seen value. If it is the same, then we're done. If not, it reads the new guardian set.
// Since this function is only called from a single go routine, it can access the stored current index without the lock.
// It only grabs the lock if it needs to update the guardian set (since the `VerifyVAA` function accesses it).
func (v *VaaVerifier) checkForGuardianSetUpdate() error {
	newIndex, err := v.fetchCurrentGuardianSetIndex()
	if err != nil {
		return fmt.Errorf("failed to read guardian set index: %w", err)
	}

	if newIndex != v.currentIndex {
		v.logger.Info("guardian set index has changed, rereading guardian set", zap.Uint32("oldIndex", v.currentIndex), zap.Uint32("newIndex", newIndex))
		gs, err := v.fetchGuardianSet(newIndex)
		if err != nil {
			return fmt.Errorf("failed to read guardian set: %w", err)
		}

		// Log this as a warning in case they have info logging disabled.
		v.logger.Warn("guardian set index has changed, switching to new guardian set",
			zap.Uint32("oldIndex", v.currentIndex),
			zap.Uint32("newIndex", newIndex),
			zap.Any("oldGS", *v.currentGuardianSet),
			zap.Any("newGS", *gs),
		)

		v.lock.Lock()
		v.currentIndex = newIndex
		v.currentGuardianSet = gs
		v.lock.Unlock()
	}

	return nil
}

func (v *VaaVerifier) VerifyVAA(vaaBytes []byte) (bool, string, error) {
	timeout, cancel := context.WithTimeout(v.ctx, RpcTimeout)
	defer cancel()
	result, err := v.caller.ParseAndVerifyVM(&ethBind.CallOpts{Context: timeout}, vaaBytes)
	if err != nil {
		return false, "", err
	}

	return result.Valid, result.Reason, nil
}
