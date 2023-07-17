package processor3

import (
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

// state represents the local view of the accumulation state of observations for forming a VAA
type state struct {
	// First time this digest was seen (possibly even before we observed it ourselves).
	firstObserved time.Time
	// The most recent time that a re-observation request was sent to the guardian network.
	lastRetry time.Time
	// msg is the MessagePublication for which we are collecting observations
	msg *common.MessagePublication
	// Map of signatures seen by guardian. During guardian set updates, this may contain signatures belonging
	// to either the old or new guardian set.
	signatures map[ethcommon.Address][]byte
	// Flag set after reaching quorum and submitting the VAA.
	submitted bool
	// guardian set valid at observation/injection time.
	gs *common.GuardianSet
}

func (p *ConcurrentProcessor) NewStateFromSelfObserved() *state {
	return &state{
		firstObserved: time.Now(),
		gs:            p.gst.Get(),
		signatures:    make(map[ethcommon.Address][]byte),
	}
}

func (p *ConcurrentProcessor) NewStateFromForeignObserved() *state {
	return &state{
		firstObserved: time.Now(),
		gs:            p.gst.Get(),
		signatures:    make(map[ethcommon.Address][]byte),
	}
}

func (s *state) SelfObserved(m *common.MessagePublication) {
	s.msg = m
}
