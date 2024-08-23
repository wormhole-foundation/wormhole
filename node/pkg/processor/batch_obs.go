package processor

import (
	"context"
	"errors"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"google.golang.org/protobuf/proto"
)

// postObservationToBatch posts an individual observation to the batch processor.
func (p *Processor) postObservationToBatch(obs *gossipv1.Observation) {
	select {
	case p.batchObsvPubC <- obs:
	default:
		batchObservationChannelOverflow.WithLabelValues("batchObsvPub").Inc()
	}
}

// batchProcessor is the entry point for the batch processor, which is responsible for taking individual
// observations and publishing them as batches. It limits the size of a batch and the delay before publishing.
func (p *Processor) batchProcessor(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := p.handleBatch(ctx); err != nil {
				return err
			}
		}
	}
}

// handleBatch reads observations from the channel, either until a timeout occurs or the batch is full.
// Then it builds a `SendObservationBatch` gossip message and posts it to p2p.
func (p *Processor) handleBatch(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, p2p.MaxObservationBatchDelay)
	defer cancel()

	observations, err := common.ReadFromChannelWithTimeout(ctx, p.batchObsvPubC, p2p.MaxObservationBatchSize)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("failed to read observations from the internal observation batch channel: %w", err)
	}

	if len(observations) != 0 {
		_ = p.publishBatch(observations)
	}

	return nil
}

// publishBatch formats the set of observations into a gossip message, publishes it, and returns the message bytes.
func (p *Processor) publishBatch(observations []*gossipv1.Observation) []byte {
	batchMsg := gossipv1.SignedObservationBatch{
		Addr:         p.ourAddr.Bytes(),
		Observations: observations,
	}

	gossipMsg := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_SignedObservationBatch{SignedObservationBatch: &batchMsg}}
	msg, err := proto.Marshal(&gossipMsg)
	if err != nil {
		panic(err)
	}

	select {
	case p.gossipAttestationSendC <- msg:
	default:
		batchObservationChannelOverflow.WithLabelValues("gossipSend").Inc()
	}

	return msg
}

// shouldPublishImmediately returns true if the observation should be published immediately, rather than waiting for the next batch.
func (p *Processor) shouldPublishImmediately(v *vaa.VAA) bool {
	return v.EmitterChain == vaa.ChainIDPythNet
}

// publishImmediately formats a single observation into a `SignedObservationBatch` gossip message and publishes it. It the returns the message bytes.
func (p *Processor) publishImmediately(obs *gossipv1.Observation) []byte {
	return p.publishBatch([]*gossipv1.Observation{obs})
}
