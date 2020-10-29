package processor

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// handleCleanup handles periodic retransmissions and cleanup of VAAs
func (p *Processor) handleCleanup(ctx context.Context) {
	p.logger.Info("aggregation state summary", zap.Int("pending", len(p.state.vaaSignatures)))

	for hash, s := range p.state.vaaSignatures {
		delta := time.Now().Sub(s.firstObserved)

		switch {
		case s.submitted && delta.Hours() >= 1:
			// We could delete submitted VAAs right away, but then we'd lose context about additional (late)
			// observation that come in. Therefore, keep it for a reasonable amount of time.
			// If a very late observation arrives after cleanup, a nil aggregation state will be created
			// and then expired after a while (as noted in observation.go, this can be abused by a byzantine guardian).
			p.logger.Info("expiring submitted VAA", zap.String("digest", hash), zap.Duration("delta", delta))
			delete(p.state.vaaSignatures, hash)
		case !s.submitted && s.retryCount >= 10:
			// Clearly, this horse is dead and continued beatings won't bring it closer to quorum.
			p.logger.Info("expiring unsubmitted VAA after exhausting retries", zap.String("digest", hash), zap.Duration("delta", delta))
			delete(p.state.vaaSignatures, hash)
		case !s.submitted && delta.Minutes() >= 5:
			// Poor VAA has been unsubmitted for five minutes - clearly, something went wrong.
			// If we have previously submitted an observation, we can make another attempt to get it over
			// the finish line by rebroadcasting our sig. If we do not have a VAA, it means we either never observed it,
			// or it got revived by a malfunctioning guardian node, in which case, we can't do anything
			// about it and just delete it to keep our state nice and lean.
			if s.ourMsg != nil {
				p.logger.Info("resubmitting VAA observation",
					zap.String("digest", hash),
					zap.Duration("delta", delta),
					zap.Int("retry", 1))
				p.sendC <- s.ourMsg
				s.retryCount += 1
			} else {
				p.logger.Info("expiring unsubmitted nil VAA", zap.String("digest", hash), zap.Duration("delta", delta))
				delete(p.state.vaaSignatures, hash)
			}
		}
	}
}
