package watchers

import (
	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// ObservationValidator validates a raw gossiped observation request before any
// watcher-specific processing code is allowed to use it.
//
// Implementations should treat Validate as the admission control boundary for
// reobservation requests. The returned ValidObservation should be the canonical
// representation used by downstream watcher logic instead of the raw protobuf
// request.
//
// Validate should perform checks that are fundamental to the safety of turning a
// gossiped observation request into a MessagePublication. Typical checks include:
//
//   - the request belongs to this watcher's chain
//   - the request's chain ID is a known SDK chain ID
//   - the transaction identifier has the watcher-specific shape required by the
//     subsequent processing logic
//
// Validate should not publish messages, mutate global state, or perform logging
// as a side effect. It should return a rich error and let the caller decide how
// to report it.
type ObservationValidator interface {
	Validate(req *gossipv1.ObservationRequest) (ValidObservation, error)
}

// MessagePublisher publishes message publications after watcher-specific logic
// has decided that they are ready for the processor.
//
// PublishMessage is the normal publication path for already-built
// MessagePublications.
//
// PublishReobservation is the guarded reobservation publication path. It should
// only be used when a MessagePublication is being emitted in response to a
// previously validated ValidObservation. Implementations should ensure that the
// message remains consistent with the validated observation before delegating to
// PublishMessage.
type MessagePublisher interface {
	PublishMessage(msg *common.MessagePublication) error
	PublishReobservation(observation ValidObservation, msg *common.MessagePublication) error
}

// Watcher is the common watcher-side security contract for the
// observation-request to message-publication path.
//
// The purpose of this interface is not to model every behavior of a chain
// watcher. Watchers differ substantially in how they subscribe to chain data,
// fetch transactions, parse logs, and reconstruct message publications. Those
// chain-specific concerns should remain local to each watcher implementation.
//
// Instead, this interface captures the narrow set of behaviors that are shared
// across watchers and are security-sensitive:
//
//   - identifying the watcher chain with ChainID
//   - validating raw gossiped observation requests with Validate
//   - publishing MessagePublications with PublishMessage
//   - publishing reobservations through the validated path with
//     PublishReobservation
//
// In practical terms, a chain-specific watcher should implement this interface
// on its concrete watcher type and use it to enforce a clean separation between:
//
//   - raw protobuf observation requests received from p2p
//   - validated observation requests represented as ValidObservation
//   - MessagePublications ready to be sent to the processor
//
// Implementers should follow these rules:
//
//   - ChainID should return the actual chain handled by this watcher instance,
//     not a hard-coded unrelated constant.
//   - Validate should be the primary place for watcher-specific request shape
//     checks that are required before processing can safely continue.
//   - PublishMessage should own the final write to the watcher's publication
//     channel.
//   - PublishReobservation should only be used for requests that have already
//     passed Validate and should delegate into PublishMessage after applying any
//     reobservation-specific checks or flags.
//   - Watchers should also keep MessagePublication construction in a dedicated
//     watcher-local helper or method, for example a BuildMessagePublication
//     function. That build step should consume a previously validated
//     ValidObservation together with watcher-specific parsed chain data and
//     return a MessagePublication ready for publication. Because the parsed
//     chain data differs substantially across watchers, this build step is
//     intentionally documented here as guidance rather than encoded as a shared
//     interface method.
//
// The interface is intentionally small. If a new method does not apply cleanly
// to all watchers without forcing chain-specific parsing or construction details
// into a shared abstraction, it likely does not belong here.
type Watcher interface {
	ChainID() vaa.ChainID
	ObservationValidator
	MessagePublisher
}
