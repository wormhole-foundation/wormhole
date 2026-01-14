package tss

import (
	"context"
	"errors"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	tsscommon "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
)

var (
	errNilGuardianSetState      = errors.New("tss' guardianSetState nil")
	errNilVaa                   = errors.New("nil VAAv1")
	errNilGuardianSet           = errors.New("nil guardian set")
	errNotVaaV1                 = errors.New("not a VAAv1")
	errNetworkOutputChannelFull = errors.New("network output channel buffer is full")
	errCouldntGossipVaa         = errors.New("couldn't gossip VAAv1")
	errNilGuardianSigner        = errors.New("guardianSigner is nil")
)

func (s *signerClient) WitnessNewVaaV1(ctx context.Context, v *vaa.VAA) error {
	if s == nil {
		return ErrSignerClientNil
	}

	if v == nil {
		return errNilVaa
	}

	if !s.vaaData.isLeader {
		return nil
	}

	if s.vaaData.gst == nil {
		return errNilGuardianSetState
	}

	if s.vaaData.GuardianSigner == nil {
		return errNilGuardianSigner
	}

	if v.Version != vaa.VaaVersion1 {
		// not an error. but will not accept.
		return nil
	}

	gs := s.vaaData.gst.Get()
	if err := v.Verify(gs.Keys); err != nil {
		return nil // won't send invalid VAAs.
	}

	bts, err := v.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal VAA: %w", err)
	}

	// signing the content the leader will gossip.
	sig, err := s.vaaData.GuardianSigner.Sign(ctx, ethcrypto.Keccak256(bts))
	if err != nil {
		return fmt.Errorf("failed to sign VAA: %w", err)
	}
	m := &gossipv1.TSSGossipMessage{
		Message:      bts,
		Signature:    sig,
		GuardianAddr: nil, // TODO.
	}

	// send to network.
	select {
	case s.vaaData.gossipOutput <- m:
		return nil
	default:
		return errNetworkOutputChannelFull
	}
}

func (s *signerClient) Outbound() <-chan *gossipv1.TSSGossipMessage {
	if s == nil {
		return nil
	}

	// nothing to publish if not leader.
	if !s.vaaData.isLeader {
		return nil
	}

	return s.vaaData.gossipOutput
}

// Inform is used to inform the TSS signer of a new incoming gossip message.
// it returns an error if the message couldn't be delivered.
func (s *signerClient) Inform(v *gossipv1.TSSGossipMessage) error {
	if s == nil {
		return ErrSignerClientNil
	}

	if s.vaaData.isLeader { // leader doesn't need to be informed.
		return nil
	}

	select {
	case s.vaaData.incomingGossip <- v:
		return nil
	default:
		return errCouldntGossipVaa
	}
}

// the gossipListener listens for incoming gossip messages and processes them.
// closes when the context is done.
func (s *signerClient) gossipListener(ctx context.Context, logger *zap.Logger) {
	logger = logger.Named("gossipListener")
	dt := s.vaaData
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-dt.incomingGossip:
			gs := dt.gst.Get()
			if gs == nil {
				// log.Warn("gossipListener: nil guardian set")
				continue
			}

			if err := dt.verifyGossipSig(msg, gs); err != nil {
				logger.Warn("invalid gossip signature", zap.Error(err))
				continue
			}

			newVaa, err := vaa.Unmarshal(msg.Message)
			if err != nil {
				logger.Warn("malformed VAA", zap.Error(err))
				continue
			}

			if newVaa.Version != vaa.VaaVersion1 {
				continue
			}

			if err := newVaa.Verify(gs.Keys); err != nil {
				logger.Warn("invalid VAA", zap.Error(err))
				continue
			}

			if err := s.vaaToSignRequest(newVaa, gs); err != nil {
				logger.Error("failed to create sign request from VAA", zap.Error(err))
				continue
			}
		}
	}
}

func (s *signerClient) vaaToSignRequest(newVaa *vaa.VAA, gs *common.GuardianSet) error {
	rq := &signer.SignRequest{
		Digest:    newVaa.SigningDigest().Bytes(),
		Committee: [][]byte{},
		Protocol:  string(tsscommon.ProtocolFROSTSign), // TODO: what protocol do we use here?
	}

	// set committee members to be the current guardian set.
	for _, sig := range newVaa.Signatures {
		if sig == nil {
			continue
		}

		if sig.Index >= uint8(len(gs.Keys)) {
			continue
		}

		addr := gs.Keys[sig.Index]
		rq.Committee = append(rq.Committee, addr.Bytes())
	}

	return s.AsyncSign(rq)
}

func (dt vaaHandling) verifyGossipSig(msg *gossipv1.TSSGossipMessage, gs *common.GuardianSet) error {
	pubKey, err := ethcrypto.Ecrecover(ethcrypto.Keccak256(msg.Message), msg.Signature)
	if err != nil {
		return fmt.Errorf("failed to recover public key: %w", err)
	}

	signerAddr := ethcommon.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
	leaderAddr := gs.Keys[dt.leaderIndex]

	if signerAddr != leaderAddr {
		return fmt.Errorf("signature not from leader: got %s, want %s", signerAddr.Hex(), leaderAddr.Hex())
	}

	return nil
}
