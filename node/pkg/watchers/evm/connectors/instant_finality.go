package connectors

import (
	"context"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"

	ethereum "github.com/ethereum/go-ethereum"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
)

// InstantFinalityConnector is used for chains that support instant finality. It uses the standard geth head sink to read blocks
// and publishes each block as latest, safe and finalized.
type InstantFinalityConnector struct {
	Connector
	logger *zap.Logger
}

func NewInstantFinalityConnector(baseConnector Connector, logger *zap.Logger) (*InstantFinalityConnector, error) {
	connector := &InstantFinalityConnector{
		Connector: baseConnector,
		logger:    logger,
	}
	return connector, nil
}

func (c *InstantFinalityConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	headSink := make(chan *ethTypes.Header, 2)
	headerSubscription, err := c.Connector.Client().SubscribeNewHead(ctx, headSink)
	if err != nil {
		return nil, err
	}

	// The purpose of this is to map events from the geth event channel to the new block event channel.
	common.RunWithScissors(ctx, errC, "eth_connector_subscribe_for_block", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case ev := <-headSink:
				if ev == nil {
					c.logger.Error("new header event is nil")
					continue
				}
				if ev.Number == nil {
					c.logger.Error("new header block number is nil")
					continue
				}
				block := &NewBlock{
					Number:   ev.Number,
					Time:     ev.Time,
					Hash:     ev.Hash(),
					Finality: Finalized,
				}
				sink <- block
				sink <- block.Copy(Safe)
				sink <- block.Copy(Latest)
			}
		}
	})

	return headerSubscription, err
}
