package watchers

import (
	"errors"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// ErrNilObservationRequest indicates that validation received no observation request.
	ErrNilObservationRequest = errors.New("observation request is nil")

	// ErrNilMessagePublication indicates that publishing received no message publication.
	ErrNilMessagePublication = errors.New("message publication is nil")
)

// InvalidChainIDError reports an observation request chain ID that is not known to the SDK.
func InvalidChainIDError(chainID uint32, err error) error {
	return fmt.Errorf("invalid chain id %d: %w", chainID, err)
}

// UnexpectedChainIDError reports an observation request for a chain other than the watcher's chain.
func UnexpectedChainIDError(got vaa.ChainID, want vaa.ChainID) error {
	return fmt.Errorf("unexpected chain id %v, expected %v", got, want)
}

// UnexpectedTxHashLengthError reports a transaction identifier with an unexpected byte length.
func UnexpectedTxHashLengthError(got int) error {
	return fmt.Errorf("unexpected tx hash length %d", got)
}

// MessagePublicationTxIDMismatchError reports a message publication that does not match its validated observation request.
func MessagePublicationTxIDMismatchError(msg *common.MessagePublication, observation ValidObservation) error {
	return fmt.Errorf("message publication TxID %v does not match validated observation txHash %v", msg.TxIDString(), observation.TxHash())
}

// MessagePublicationChainMismatchError reports a message publication for a chain other than the expected chain.
func MessagePublicationChainMismatchError(got vaa.ChainID, want vaa.ChainID) error {
	return fmt.Errorf("message publication emitter chain %v does not match watcher chain %v", got, want)
}

// ReobservedMessageChainMismatchError reports a reobserved message for a chain other than the validated observation chain.
func ReobservedMessageChainMismatchError(got vaa.ChainID, want vaa.ChainID) error {
	return fmt.Errorf("message publication emitter chain %v does not match validated observation chain %v", got, want)
}
