package processor

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// readerChannels is the set of channels read by a single worker.
type readerChannels struct {
	setC            <-chan *common.GuardianSet // TODO: Drop this if we get rid of single threaded mode.
	msgC            <-chan *common.MessagePublication
	acctReadC       <-chan *common.MessagePublication // TODO: Drop this if we get rid of single threaded mode.
	injectC         <-chan *vaa.VAA
	obsvC           <-chan *common.MsgWithTimeStamp[gossipv1.SignedObservation]
	signedInC       <-chan *gossipv1.SignedVAAWithQuorum // TODO: Drop this if we get rid of single threaded mode.
	parsedSignedInC <-chan *vaa.VAA
}

// writerChannels is the set of channels used to write to a single worker.
type writerChannels struct {
	setC            chan<- *common.GuardianSet // TODO: Drop this if we get rid of single threaded mode.
	msgC            chan<- *common.MessagePublication
	acctReadC       chan<- *common.MessagePublication // TODO: Drop this if we get rid of single threaded mode.
	injectC         chan<- *vaa.VAA
	obsvC           chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservation]
	signedInC       chan<- *gossipv1.SignedVAAWithQuorum // TODO: Drop this if we get rid of single threaded mode.
	parsedSignedInC chan<- *vaa.VAA
}

// runDispatcher starts the dispatcher and workers. It creates a readerChannels struct for each worker and an array of writerChannels structs for the dispatcher.
func (p *Processor) runDispatcher(ctx context.Context) {
	// Compute the number of workers based on the number of CPUs and the worker factor.
	numWorkers := int(math.Ceil(float64(runtime.NumCPU()) * p.workerFactor))
	p.numWorkers = uint64(numWorkers)

	// Compute the size of the message channels based on incoming channel size and the number of workers.
	msgCSize := p.computeMsgChanSize(cap(p.msgC))
	signedInCSize := p.computeMsgChanSize(cap(p.signedInC))
	obsvCSize := p.computeMsgChanSize(cap(p.obsvC))
	injectCSize := p.computeMsgChanSize(cap(p.injectC))
	acctCSize := p.computeMsgChanSize(cap(p.acctReadC))

	p.logger.Info("processor configured to use workers",
		zap.Int("numWorkers", numWorkers),
		zap.Float64("workerFactor", p.workerFactor),
		zap.Int("msgCSize", msgCSize),
		zap.Int("signedInCSize", signedInCSize),
		zap.Int("obsvCSize", obsvCSize),
		zap.Int("injectCSize", injectCSize),
		zap.Int("acctCSize", acctCSize),
	)

	var w sync.WaitGroup
	w.Add(numWorkers)

	p.workerChans = make([]*writerChannels, 0)

	for workerIdx := 0; workerIdx < numWorkers; workerIdx++ {
		setC := makeChannelPair[*common.GuardianSet](1)
		msgC := makeChannelPair[*common.MessagePublication](msgCSize)
		signedInC := makeChannelPair[*gossipv1.SignedVAAWithQuorum](signedInCSize)
		parsedSignedInC := makeChannelPair[*vaa.VAA](signedInCSize)
		obsvC := makeChannelPair[*common.MsgWithTimeStamp[gossipv1.SignedObservation]](obsvCSize)
		injectC := makeChannelPair[*vaa.VAA](injectCSize)
		acctC := makeChannelPair[*common.MessagePublication](acctCSize)

		readerChans := &readerChannels{
			setC:            setC.readC,
			msgC:            msgC.readC,
			acctReadC:       acctC.readC,
			injectC:         injectC.readC,
			obsvC:           obsvC.readC,
			signedInC:       signedInC.readC,
			parsedSignedInC: parsedSignedInC.readC,
		}

		writerChans := &writerChannels{
			setC:            setC.writeC,
			msgC:            msgC.writeC,
			acctReadC:       acctC.writeC,
			injectC:         injectC.writeC,
			obsvC:           obsvC.writeC,
			signedInC:       signedInC.writeC,
			parsedSignedInC: parsedSignedInC.writeC,
		}

		p.workerChans = append(p.workerChans, writerChans)

		go func(ctx context.Context, workerIdx int, chans *readerChannels) {
			p.logger.Info("processor worker started", zap.Int("workerIdx", workerIdx))
			err := p.runWorker(ctx, chans, false)
			if err != nil {
				p.logger.Error("processor worker failed", zap.Int("workerIdx", workerIdx), zap.Error(err))
			}
			p.logger.Info("processor worker done", zap.Int("workerIdx", workerIdx))
			w.Done()
		}(ctx, workerIdx, readerChans)
	}

	go func() { _ = p.dispatcher(ctx, p.workerChans) }()

	w.Wait()
}

// computeMsgChanSize computes the size of a per-worker message channel based on the base size and the number of workers.
func (p *Processor) computeMsgChanSize(baseSize int) int {
	return int(math.Ceil(float64(baseSize)/float64(p.numWorkers))) * 2
}

// The dispatcher receives events from outside the processor and dispatches to a deterministic worker based on the VAA digest.
// It also performs periodic cleanup. This is okay because cleanup locks the entire state map anyway.
func (p *Processor) dispatcher(ctx context.Context, workerChans []*writerChannels) error {
	// Always start the timers to avoid nil pointer dereferences below. They will only be rearmed on worker 1.
	cleanup := time.NewTimer(CleanupInterval)
	defer cleanup.Stop()

	// Always initialize the timer so don't have a nil pointer in the case below. It won't get rearmed after that.
	govTimer := time.NewTimer(GovInterval)
	defer govTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			if p.acct != nil {
				p.acct.Close()
			}
			return ctx.Err()
		case gs := <-p.setC:
			p.gs.Store(gs)
			p.logger.Info("guardian set updated",
				zap.Strings("set", gs.KeysAsHexStrings()),
				zap.Uint32("index", gs.Index))
			p.gst.Set(gs)
		case k := <-p.msgC:
			p.dispatchMessage(workerChans, k)
		case k := <-p.acctReadC:
			if p.acct == nil {
				return fmt.Errorf("received an accountant event when accountant is not configured")
			}
			// SECURITY defense-in-depth: Make sure the accountant did not generate an unexpected message.
			if !p.acct.IsMessageCoveredByAccountant(k) {
				return fmt.Errorf("accountant published a message that is not covered by it: `%s`", k.MessageIDString())
			}
			p.dispatchMessage(workerChans, k)
		case v := <-p.injectC:
			digest := v.SigningDigest().Bytes()
			workerIdx := p.workerIdxFromDigest(digest)
			if workerIdx > len(workerChans) {
				panic(fmt.Sprintf(`failed to compute worker idx on injected VAA for digest "%v", numWorkers: %d`, hex.EncodeToString(digest), len(workerChans)))
			}
			workerChans[workerIdx].injectC <- v
		case m := <-p.obsvC:
			workerIdx := p.workerIdxFromDigest(m.Msg.Hash)
			if workerIdx > len(workerChans) {
				panic(fmt.Sprintf(`failed to compute worker idx on observation for digest "%v", numWorkers: %d`, hex.EncodeToString(m.Msg.Hash), len(workerChans)))
			}
			workerChans[workerIdx].obsvC <- m
		case m := <-p.signedInC:
			v, alreadyInDB, err := p.unmarshalSignedVaaWithQuorum(m)
			if err != nil {
				p.logger.Error("failed to parse incoming signed VAA with quorum", zap.Error(err))
				continue
			}
			if !alreadyInDB {
				digest := v.SigningDigest().Bytes()
				workerIdx := p.workerIdxFromDigest(digest)
				if workerIdx > len(workerChans) {
					panic(fmt.Sprintf(`failed to compute worker idx on incoming signed VAA with quorum for digest "%v", numWorkers: %d`, hex.EncodeToString(digest), len(workerChans)))
				}
				workerChans[workerIdx].parsedSignedInC <- v
			}
		case <-cleanup.C:
			cleanup.Reset(CleanupInterval)
			p.handleCleanup(ctx)
		case <-govTimer.C:
			if p.governor != nil {
				toBePublished, err := p.governor.CheckPending()
				if err != nil {
					return err
				}
				if len(toBePublished) != 0 {
					for _, k := range toBePublished {
						// SECURITY defense-in-depth: Make sure the governor did not generate an unexpected message.
						if msgIsGoverned, err := p.governor.IsGovernedMsg(k); err != nil {
							return fmt.Errorf("governor failed to determine if message should be governed: `%s`: %w", k.MessageIDString(), err)
						} else if !msgIsGoverned {
							return fmt.Errorf("governor published a message that should not be governed: `%s`", k.MessageIDString())
						}
						if p.acct != nil {
							shouldPub, err := p.acct.SubmitObservation(k)
							if err != nil {
								return fmt.Errorf("failed to process message released by governor `%s`: %w", k.MessageIDString(), err)
							}
							if !shouldPub {
								continue
							}
						}
						p.dispatchMessage(workerChans, k)
					}
				}
				govTimer.Reset(GovInterval)
			}
		}
	}
}

// dispatchMessage dispatches a message publication to the appropriate worker.
func (p *Processor) dispatchMessage(workerChans []*writerChannels, k *common.MessagePublication) {
	digest := digestFromMsg(k)
	workerIdx := p.workerIdxFromDigest(digest)
	if workerIdx >= len(workerChans) {
		panic(fmt.Sprintf(`failed to compute worker idx for digest "%v", numWorkers: %d`, hex.EncodeToString(digest), len(workerChans)))
	}
	workerChans[workerIdx].msgC <- k
}

// DispatchObservation allows P2P to directly submit an observation to a worker, bypassing the dispatcher and eliminating a channel hop.
func (p *Processor) DispatchObservation(m *common.MsgWithTimeStamp[gossipv1.SignedObservation]) bool {
	workerIdx := p.workerIdxFromDigest(m.Msg.Hash)
	if workerIdx >= len(p.workerChans) {
		p.logger.Error("failed to compute worker idx on observation", zap.String("digest", hex.EncodeToString(m.Msg.Hash)), zap.Int("numWorkers", len(p.workerChans)))
		return false
	}
	select {
	case p.workerChans[workerIdx].obsvC <- m:
		return true
	default:
		return false
	}
}

// digestFromMsg returns the digest of the message publication by creating a VAA. TODO: We could pass this VAA to handleMessage() so we don't have to create the VAA twice.
func digestFromMsg(msg *common.MessagePublication) []byte {
	v := msg.CreateVAA(0) // We can pass zero in as the guardian set index because it is not part of the digest.
	return v.SigningDigest().Bytes()
}

// workerIdxFromDigest generates the worker index from the digest by doing a modulo operation.
func (p *Processor) workerIdxFromDigest(digest []byte) int {
	return int(binary.BigEndian.Uint64(digest) % p.numWorkers)
}

// TODO: Move this to common.
type channelPair[T any] struct {
	readC  <-chan T
	writeC chan<- T
}

func makeChannelPair[T any](cap int) channelPair[T] {
	out := make(chan T, cap)
	return channelPair[T]{out, out}
}
