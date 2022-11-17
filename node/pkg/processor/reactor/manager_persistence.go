package reactor

import "github.com/wormhole-foundation/wormhole/sdk/vaa"

// ConsensusStorage is used to store observations and consensus reactor state
type ConsensusStorage[K Observation] interface {
	// StoreSignedObservation stores a signed observation
	StoreSignedObservation(observation K, signatures []*vaa.Signature) error
	// GetSignedObservation retrieves a signed observation from the DB using the message id as key
	GetSignedObservation(id string) (observation K, signatures []*vaa.Signature, found bool, err error)
}
