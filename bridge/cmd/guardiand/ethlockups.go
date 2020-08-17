package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
)

func ethLockupProcessor(ec chan *common.ChainLock, gk *ecdsa.PrivateKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case k := <-ec:
				supervisor.Logger(ctx).Info("lockup confirmed",
					zap.String("source", hex.EncodeToString(k.SourceAddress[:])),
					zap.String("target", hex.EncodeToString(k.TargetAddress[:])),
					zap.String("amount", k.Amount.String()),
					zap.String("hash", hex.EncodeToString(k.Hash())),
				)
			}
		}
	}
}
