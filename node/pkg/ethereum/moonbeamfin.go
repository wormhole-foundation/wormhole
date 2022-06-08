// This implements the finality check for Moonbeam.
//
// Moonbeam can publish blocks before they are marked final. This means we need to sit on the block until a special "is finalized"
// query returns true. The assumption is that every block number will eventually be published and finalized, it's just that the contents
// of the block (and therefore the hash) might change if there is a rollback.

package ethereum

import (
	"context"
	"time"

	common "github.com/certusone/wormhole/node/pkg/common"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"
)

type MoonbeamFinalizer struct {
	logger      *zap.Logger
	networkName string
	client      *ethRpc.Client
}

func (f *MoonbeamFinalizer) SetLogger(l *zap.Logger, netName string) {
	f.logger = l
	f.networkName = netName
	f.logger.Info("using Moonbeam specific finality check", zap.String("eth_network", f.networkName))
}

func (f *MoonbeamFinalizer) DialContext(ctx context.Context, rawurl string) (err error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	f.client, err = ethRpc.DialContext(timeout, rawurl)
	return err
}

func (f *MoonbeamFinalizer) IsBlockFinalized(ctx context.Context, block *common.NewBlock) (bool, error) {
	var finalized bool
	err := f.client.CallContext(ctx, &finalized, "moon_isBlockFinalized", block.Hash.Hex())
	if err != nil {
		f.logger.Error("failed to check for finality", zap.String("eth_network", f.networkName), zap.Error(err))
		return false, err
	}

	return finalized, nil
}
