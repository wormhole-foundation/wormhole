package interfaces

import (
	"context"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type Reobserver interface {
	Reobserve(ctx context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error)
}

type Reobservers map[vaa.ChainID]Reobserver
