package aptos

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/watchers"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

type (
	// Watcher is responsible for looking over Aptos blockchain and reporting new transactions to the wormhole contract
	Watcher struct {
		chainID   vaa.ChainID
		networkID string

		aptosRPC     string
		aptosAccount string // Wormhole contract address
		aptosHandle  string // Event to subscribe to Aptos RPC

		// Cached canonical event values to compare against, computed at startup.
		accountAddr     [32]byte
		eventTypeAddr   [32]byte
		eventTypeModule string
		eventTypeStruct string

		msgC          chan<- *common.MessagePublication
		obsvReqC      <-chan *gossipv1.ObservationRequest
		readinessSync readiness.Component
	}
)

var (
	aptosMessagesConfirmed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_aptos_observations_confirmed_total",
			Help: "Total number of verified observations found for the chain",
		}, []string{"chain_name"})

	currentAptosHeight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "wormhole_aptos_current_height",
			Help: "Current block height for the chain",
		}, []string{"chain_name"})
)

// NewWatcher creates a new Aptos appid watcher
func NewWatcher(
	chainID vaa.ChainID,
	networkID watchers.NetworkID,
	aptosRPC string,
	aptosAccount string,
	aptosHandle string,
	msgC chan<- *common.MessagePublication,
	obsvReqC <-chan *gossipv1.ObservationRequest,
) (*Watcher, error) {

	/*
		The Aptos smart contracts have two items to consider:
		- WormholeMessageHandle - searchable event
		- WormholeMessage - decoded event data

		During event validation, the code checks that the event is a 'state::WormholeMessage' type.
		This check prevents bugs where the RPC returns an event of the wrong type to the watcher.
		To do this, the text 'Handle' is removed from the `aptosHandle` to derive
		the name of the event. This is done in order to avoid passing another parameter to the watcher.

		This is only by the current convention this optimization can be done. If the Aptos smart contracts or
		watcher event consumption were ever changed, this may no longer work.
	*/
	if !strings.HasSuffix(aptosHandle, "Handle") {
		return nil, fmt.Errorf("aptosHandle %q does not end with 'Handle'", aptosHandle)
	}
	// Validate the handle's structure (<address>::<module>::<struct>) and its embedded address.
	if err := validateAptosHandle(aptosHandle); err != nil {
		return nil, err
	}

	// Validate the configured contract address. parseAptosAddr requires a valid, lowercase Aptos
	// address so that verifyEventType can match it against the (always lowercased) event fields.
	// Fail fast on misconfiguration. The canonical bytes are cached for later comparison.
	accountAddr, ok := parseAptosAddr(aptosAccount)
	if !ok {
		return nil, fmt.Errorf("aptosAccount %q is not a valid lowercase Aptos address", aptosAccount)
	}

	// Cache the canonical pieces of the event type for comparison in verifyEventType.
	eventType := strings.TrimSuffix(aptosHandle, "Handle")
	eventTypeAddr, eventTypeModule, eventTypeStruct, ok := parseAptosEventType(eventType)
	if !ok {
		return nil, fmt.Errorf("derived aptosEventType %q is not a valid Move type tag", eventType)
	}

	return &Watcher{
		chainID:         chainID,
		networkID:       string(networkID),
		aptosRPC:        aptosRPC,
		aptosAccount:    aptosAccount,
		aptosHandle:     aptosHandle,
		accountAddr:     accountAddr,
		eventTypeAddr:   eventTypeAddr,
		eventTypeModule: eventTypeModule,
		eventTypeStruct: eventTypeStruct,
		msgC:            msgC,
		obsvReqC:        obsvReqC,
		readinessSync:   common.MustConvertChainIdToReadinessSyncing(chainID),
	}, nil
}

func (e *Watcher) Run(ctx context.Context) error {
	p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
		ContractAddress: e.aptosAccount,
	})

	logger := supervisor.Logger(ctx)

	logger.Info("Starting watcher",
		zap.String("watcher_name", e.networkID),
		zap.String("rpc", e.aptosRPC),
		zap.String("account", e.aptosAccount),
		zap.String("handle", e.aptosHandle),
	)

	// Get the node version for troubleshooting
	e.logVersion(logger)

	// SECURITY: the API guarantees that we only get the events from the right contract.
	// Additional defense-in-depth check to verify that this is the case.
	var eventsEndpoint = fmt.Sprintf(`%s/v1/accounts/%s/events/%s/event`, e.aptosRPC, e.aptosAccount, e.aptosHandle)
	var aptosHealth = fmt.Sprintf(`%s/v1`, e.aptosRPC)

	logger.Info("watcher connecting to RPC node ",
		zap.String("url", e.aptosRPC),
		zap.String("eventsQuery", eventsEndpoint),
		zap.String("healthQuery", aptosHealth),
	)

	// the events have sequence numbers associated with them in the aptos API
	// (NOTE: this is not the same as the wormhole sequence id). The event
	// endpoint is paginated, so we use this variable to keep track of which
	// sequence number to look up next.
	var nextSequence uint64 = 0

	timer := time.NewTicker(time.Second * 1)
	defer timer.Stop()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-e.obsvReqC:
			// node/pkg/node/reobserve.go already enforces the chain id is a valid uint16
			// and only writes to the channel for this chain id.
			// If either of the below cases are true, something has gone wrong
			if r.ChainId > math.MaxUint16 || vaa.ChainID(r.ChainId) != e.chainID {
				panic("invalid chain ID")
			}

			// Aptos's TxID is a uint64. Historically, all TxIDs used a fixed 32-byte hash type.
			// This parsing is leftover from that time period. It should be possible to refactor
			// this code such that the TxID received from p2p is exactly 8 bytes, which would
			// obviate the need for the below bounds check and parsing.
			//
			// SECURITY: This acts as a bounds check for the BigEndian.Unint64 call below.
			const AptosTxIDExpectedLen = 32
			if len(r.TxHash) < AptosTxIDExpectedLen {
				logger.Error("invalid TxID: too short")
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// uint64 will read the *first* 8 bytes, but the sequence is stored in the *last* 8.
			nativeSeq := binary.BigEndian.Uint64(r.TxHash[24:])

			logger.Info("Received obsv request", zap.Uint64("tx_hash", nativeSeq))

			s := fmt.Sprintf(`%s?start=%d&limit=1`, eventsEndpoint, nativeSeq)

			body, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(body)) {
				logger.Error("InvalidJson: " + string(body))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				break

			}

			outcomes := gjson.ParseBytes(body)

			e.processReobservationBatch(logger, outcomes, nativeSeq)

		case <-timer.C:
			s := ""

			if nextSequence == 0 {
				// if nextSequence is 0, we look up the most recent event
				s = fmt.Sprintf(`%s?limit=1`, eventsEndpoint)
			} else {
				// otherwise just look up events starting at nextSequence.
				// this will potentially return multiple events (whatever
				// the default limit is per page), so we'll handle all of them.
				s = fmt.Sprintf(`%s?start=%d`, eventsEndpoint, nextSequence)
			}

			eventsJson, err := e.retrievePayload(s)
			if err != nil {
				logger.Error("retrievePayload", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			// data doesn't exist yet. skip, and try again later
			// this happens when the sequence id we're looking up hasn't
			// been used yet.
			if string(eventsJson) == "" {
				continue
			}

			if !gjson.Valid(string(eventsJson)) {
				logger.Error("InvalidJson: " + string(eventsJson))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue

			}

			events := gjson.ParseBytes(eventsJson)

			e.processPollingBatch(logger, events, &nextSequence)

			health, err := e.retrievePayload(aptosHealth)
			if err != nil {
				logger.Error("health", zap.Error(err))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue
			}

			if !gjson.Valid(string(health)) {
				logger.Error("Invalid JSON in health response: " + string(health))
				p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
				continue

			}

			// TODO: Make this log more useful for humans
			logger.Debug(string(health) + string(eventsJson))

			pHealth := gjson.ParseBytes(health)

			blockHeight := pHealth.Get("block_height")

			if blockHeight.Uint() > math.MaxInt64 {
				logger.Error("Block height not a valid uint64: ", zap.Uint64("blockHeight", blockHeight.Uint()))
				p2p.DefaultRegistry.AddErrorCount(vaa.ChainIDAptos, 1)
				continue
			}

			if blockHeight.Exists() {
				currentAptosHeight.WithLabelValues(e.networkID).Set(float64(blockHeight.Uint()))
				p2p.DefaultRegistry.SetNetworkStats(e.chainID, &gossipv1.Heartbeat_Network{
					Height:          int64(blockHeight.Uint()), // #nosec G115 -- This is validated above
					ContractAddress: e.aptosAccount,
				})

				readiness.SetReady(e.readinessSync)
			}
		}
	}
}

//nolint:noctx // TODO: this function should use a context. (Also this line flags nolintlint unless placed here.)
func (e *Watcher) retrievePayload(s string) ([]byte, error) {
	//nolint:gosec // the URL is hard-coded to the Aptos RPC endpoint.
	res, err := http.Get(s)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := common.SafeRead(res.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

// processPollingBatch iterates the events returned by the polling endpoint,
// advancing nextSequence as events are consumed.
func (e *Watcher) processPollingBatch(logger *zap.Logger, events gjson.Result, nextSequence *uint64) {
	// the endpoint returns an array of events, ordered by sequence id (ASC)
	for _, event := range events.Array() {
		eventSequence := event.Get("sequence_number")
		if !eventSequence.Exists() {
			continue
		}
		eventSeq := eventSequence.Uint()

		// On startup nextSequence is 0; a non-zero first event is pre-existing backlog we
		// should not republish. Capture this before advancing, since it reads the old value.
		isStartupBacklog := *nextSequence == 0 && eventSeq != 0

		// Consume this slot regardless of outcome so a skipped event is never re-fetched.
		*nextSequence = eventSeq + 1

		// SECURITY: Aptos event type validation
		if err := e.verifyEventType(event); err != nil {
			logger.Error("aptos event failed verification",
				zap.Error(err),
				zap.Uint64("eventSequence", eventSeq),
				zap.String("eventType", event.Get("type").String()),
				zap.String("guidAccountAddress", event.Get("guid.account_address").String()),
				zap.String("guidCreationNumber", event.Get("guid.creation_number").String()),
				zap.String("account", e.aptosAccount),
				zap.String("handle", e.aptosHandle),
			)
			p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
			continue
		}

		if isStartupBacklog {
			// Avoid publishing an old observation on startup. This does not block the first message on a new chain (when eventSeq would be zero).
			continue
		}

		data := event.Get("data")
		if !data.Exists() {
			continue
		}

		// Validates and publishes the message to the processor
		e.observeData(logger, data, eventSeq, false)
	}
}

// processReobsBatch handles the response to a reobservation lookup. The query
// uses limit=1 so outcomes is expected to be a zero- or one-element array.
func (e *Watcher) processReobservationBatch(logger *zap.Logger, outcomes gjson.Result, nativeSeq uint64) {
	for _, aptosEvent := range outcomes.Array() {
		newSeq := aptosEvent.Get("sequence_number")
		if !newSeq.Exists() {
			break
		}

		if newSeq.Uint() != nativeSeq {
			logger.Error("newSeq != nativeSeq")
			break
		}

		// SECURITY: Aptos event type validation
		if err := e.verifyEventType(aptosEvent); err != nil {
			logger.Error("aptos event failed verification",
				zap.Error(err),
				zap.Uint64("eventSequence", nativeSeq),
				zap.String("eventType", aptosEvent.Get("type").String()),
				zap.String("guidAccountAddress", aptosEvent.Get("guid.account_address").String()),
				zap.String("guidCreationNumber", aptosEvent.Get("guid.creation_number").String()),
				zap.String("account", e.aptosAccount),
				zap.String("handle", e.aptosHandle),
			)
			p2p.DefaultRegistry.AddErrorCount(e.chainID, 1)
			continue
		}

		data := aptosEvent.Get("data")
		if !data.Exists() {
			break
		}

		// Validates and publishes the message to the processor
		e.observeData(logger, data, nativeSeq, true)
	}
}

// stripHexadecimalPrefix strips an optional leading "0x" prefix so a configured value without it still
// matches the chain's 0x-prefixed representation.
func stripHexadecimalPrefix(s string) string {
	return strings.TrimPrefix(s, "0x")
}

// validateAptosHandle checks that the event handle has the canonical
// "<address>::<module>::<struct>" shape and that its address segment is a valid, lowercase Aptos
// address. Splitting on "::" must yield exactly three segments; the explicit length check guards the
// parts[0] access below from an out-of-bounds read.
func validateAptosHandle(handle string) error {
	parts := strings.Split(handle, "::")
	if len(parts) != 3 {
		return fmt.Errorf("aptosHandle %q must have the form <address>::<module>::<struct>", handle)
	}
	if _, ok := parseAptosAddr(parts[0]); !ok {
		return fmt.Errorf("aptosHandle address segment %q is not a valid lowercase Aptos address", parts[0])
	}
	return nil
}

// parseAptosAddr decodes a hex Aptos address into its canonical 32-byte form. The address must be
// lowercase with an optional lowercase "0x" prefix; uppercase hex (or a "0X" prefix) is rejected so
// that a configured address compares equal to the always-lowercased values returned by the Aptos API.
func parseAptosAddr(addr string) ([32]byte, bool) {
	var out [32]byte
	addr = stripHexadecimalPrefix(addr)
	if len(addr) == 0 || len(addr) > 64 {
		return out, false
	}
	if addr != strings.ToLower(addr) {
		return out, false
	}
	// The Aptos API returns special addresses in short form on a 4-bit nibble boundary
	// (e.g. "0x1", "0x0"), which is odd-length hex. Pad so it decodes.
	if len(addr)%2 == 1 {
		addr = "0" + addr
	}
	decoded, err := hex.DecodeString(addr)
	if err != nil {
		return out, false
	}
	copy(out[32-len(decoded):], decoded)
	return out, true
}

// parseAptosEventType splits a Move type tag "<address>::<module>::<struct>" into its canonical
// address bytes and its module/struct names. ok is false if the tag does not have exactly three
// "::"-separated segments or its address segment is not a valid Aptos address.
func parseAptosEventType(t string) (addr [32]byte, module, structName string, ok bool) {
	parts := strings.Split(t, "::")
	if len(parts) != 3 {
		return addr, "", "", false
	}
	addr, ok = parseAptosAddr(parts[0])
	if !ok {
		return addr, "", "", false
	}
	return addr, parts[1], parts[2], true
}

// Verify that the event on the response lines up with the
// event on the subscription URL. Defense-in-depth check.
func (e *Watcher) verifyEventType(event gjson.Result) error {
	t := event.Get("type")
	if !t.Exists() {
		return fmt.Errorf("event missing 'type' field")
	}

	// Compare the address segment by canonical bytes, since the API may strip leading zeros.
	// Move module/struct names are case-sensitive and must match exactly.
	addr, module, structName, ok := parseAptosEventType(t.String())
	if !ok || addr != e.eventTypeAddr || module != e.eventTypeModule || structName != e.eventTypeStruct {
		return fmt.Errorf("event type mismatch: got %q, want handle %q", t.String(), e.aptosHandle)
	}

	// The GUID identifies the on-chain EventHandle that emitted the event;
	// account_address must equal the configured core bridge account.
	guidAddr := event.Get("guid.account_address")
	if !guidAddr.Exists() {
		return fmt.Errorf("event missing 'guid.account_address' field")
	}
	gAddr, ok := parseAptosAddr(guidAddr.String())
	if !ok || gAddr != e.accountAddr {
		return fmt.Errorf("event guid.account_address mismatch: got %q, want %q", guidAddr.String(), e.aptosAccount)
	}

	return nil
}

func (e *Watcher) observeData(logger *zap.Logger, data gjson.Result, nativeSeq uint64, isReobservation bool) {
	em := data.Get("sender")
	if !em.Exists() {
		logger.Error("sender field missing")
		return
	}

	emitter := make([]byte, 8)
	binary.BigEndian.PutUint64(emitter, em.Uint())

	// SECURITY: vaa.Address is guaranteed to be 32 bytes so copy's slice into `a` is safe.
	// The maximum value is u64 because of the incrementing ID of the emitter's type in the
	// smart contract. Thus, this is a safe conversion.
	var a vaa.Address
	copy(a[24:], emitter)

	id := make([]byte, 8)
	binary.BigEndian.PutUint64(id, nativeSeq)

	var txHash = eth_common.BytesToHash(id) // 32 bytes = d3b136a6a182a40554b2fafbc8d12a7a22737c10c81e33b33d1dcb74c532708b

	v := data.Get("payload")
	if !v.Exists() {
		logger.Error("payload field missing")
		return
	}

	s := v.String()
	if !strings.HasPrefix(s, "0x") {
		logger.Error("payload missing 0x prefix", zap.String("payload", s))
		return
	}

	pl, err := hex.DecodeString(stripHexadecimalPrefix(s))
	if err != nil {
		logger.Error("payload decode", zap.Error(err))
		return
	}

	ts := data.Get("timestamp")
	if !ts.Exists() {
		logger.Error("timestamp field missing")
		return
	}

	nonce := data.Get("nonce")
	if !nonce.Exists() {
		logger.Error("nonce field missing")
		return
	}

	sequence := data.Get("sequence")
	if !sequence.Exists() {
		logger.Error("sequence field missing")
		return
	}

	consistencyLevel := data.Get("consistency_level")
	if !consistencyLevel.Exists() {
		logger.Error("consistencyLevel field missing")
		return
	}

	if nonce.Uint() > math.MaxUint32 {
		logger.Error("nonce is larger than expected MaxUint32")
		return
	}

	if consistencyLevel.Uint() > math.MaxUint8 {
		logger.Error("consistency level is larger than expected MaxUint8")
		return
	}

	observation := &common.MessagePublication{
		TxID:             txHash.Bytes(),
		Timestamp:        time.Unix(int64(ts.Uint()), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            uint32(nonce.Uint()),           // #nosec G115 -- This is validated above
		Sequence:         sequence.Uint(),
		EmitterChain:     e.chainID,
		EmitterAddress:   a,
		Payload:          pl,
		ConsistencyLevel: uint8(consistencyLevel.Uint()), // #nosec G115 -- This is validated above
		IsReobservation:  isReobservation,
		Unreliable:       false,
	}

	aptosMessagesConfirmed.WithLabelValues(e.networkID).Inc()
	if isReobservation {
		watchers.ReobservationsByChain.WithLabelValues(e.chainID.String(), "std").Inc()
	}

	logger.Info("message observed",
		zap.String("txHash", observation.TxIDString()),
		zap.Time("timestamp", observation.Timestamp),
		zap.Uint32("nonce", observation.Nonce),
		zap.Uint64("sequence", observation.Sequence),
		zap.Stringer("emitter_chain", observation.EmitterChain),
		zap.Stringer("emitter_address", observation.EmitterAddress),
		zap.Binary("payload", observation.Payload),
		zap.Uint8("consistencyLevel", observation.ConsistencyLevel),
	)

	e.msgC <- observation // Note on channel capacity: The channel to the processor is buffered and shared across chains, if it backs up we should stop processing new observations
}

// logVersion retrieves the Aptos node version and logs it
func (e *Watcher) logVersion(logger *zap.Logger) {
	// From https://www.alchemy.com/docs/node/aptos/aptos-api-endpoints/aptos-api-endpoints/v-1
	networkName := "aptos"
	versionsEndpoint := fmt.Sprintf("%s/v1", e.aptosRPC)

	body, err := e.retrievePayload(versionsEndpoint)
	if err != nil {
		logger.Error("problem retrieving node version",
			zap.Error(err),
			zap.String("network", networkName),
		)
		return
	}

	if !gjson.Valid(string(body)) {
		logger.Error("problem retrieving node version",
			zap.String("invalid json", string(body)),
			zap.String("network", networkName),
		)
		return
	}

	version := gjson.GetBytes(body, "git_hash").String()

	if version == "" {
		logger.Error("problem retrieving node version",
			zap.String("empty version", version),
			zap.String("network", networkName),
		)
		return
	}

	logger.Info("node version",
		zap.String("network", networkName),
		zap.String("version", version),
	)
}
