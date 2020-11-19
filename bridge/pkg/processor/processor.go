package processor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/terra"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
)

type (
	// vaaState represents the local view of a given VAA
	vaaState struct {
		firstObserved time.Time
		ourVAA        *vaa.VAA
		signatures    map[ethcommon.Address][]byte
		submitted     bool
		retryCount    uint
		ourMsg        []byte
	}

	vaaMap map[string]*vaaState

	// aggregationState represents the node's aggregation of guardian signatures.
	aggregationState struct {
		vaaSignatures vaaMap
	}
)

type Processor struct {
	// lockC is a channel of observed chain lockups
	lockC chan *common.ChainLock
	// setC is a channel of guardian set updates
	setC chan *common.GuardianSet

	// sendC is a channel of outbound messages to broadcast on p2p
	sendC chan []byte
	// obsvC is a channel of inbound decoded observations from p2p
	obsvC chan *gossipv1.LockupObservation

	// vaaC is a channel of VAAs to submit to store on Solana (either as target, or for data availability)
	vaaC chan *vaa.VAA

	// injectC is a channel of VAAs injected locally.
	injectC chan *vaa.VAA

	// gk is the node's guardian private key
	gk *ecdsa.PrivateKey

	// devnetMode specified whether to submit transactions to the hardcoded Ethereum devnet
	devnetMode         bool
	devnetNumGuardians uint
	devnetEthRPC       string

	terraLCD      string
	terraChaidID  string
	terraContract string
	terraFeePayer string

	logger *zap.Logger

	// Runtime state

	// gs is the currently valid guardian set
	gs *common.GuardianSet
	// state is the current runtime VAA view
	state *aggregationState
	// gk pk as eth address
	ourAddr ethcommon.Address
	// cleanup triggers periodic state cleanup
	cleanup *time.Ticker
}

func NewProcessor(
	ctx context.Context,
	lockC chan *common.ChainLock,
	setC chan *common.GuardianSet,
	sendC chan []byte,
	obsvC chan *gossipv1.LockupObservation,
	vaaC chan *vaa.VAA,
	injectC chan *vaa.VAA,
	gk *ecdsa.PrivateKey,
	devnetMode bool,
	devnetNumGuardians uint,
	devnetEthRPC string,
	terraLCD string,
	terraChaidID string,
	terraContract string,
	terraFeePayer string) *Processor {

	return &Processor{
		lockC:              lockC,
		setC:               setC,
		sendC:              sendC,
		obsvC:              obsvC,
		vaaC:               vaaC,
		injectC:            injectC,
		gk:                 gk,
		devnetMode:         devnetMode,
		devnetNumGuardians: devnetNumGuardians,
		devnetEthRPC:       devnetEthRPC,

		terraLCD:      terraLCD,
		terraChaidID:  terraChaidID,
		terraContract: terraContract,
		terraFeePayer: terraFeePayer,

		logger:  supervisor.Logger(ctx),
		state:   &aggregationState{vaaMap{}},
		ourAddr: crypto.PubkeyToAddress(gk.PublicKey),
	}
}

func (p *Processor) Run(ctx context.Context) error {
	p.cleanup = time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p.gs = <-p.setC:
			p.logger.Info("guardian set updated",
				zap.Strings("set", p.gs.KeysAsHexStrings()),
				zap.Uint32("index", p.gs.Index))

			// Dev mode guardian set update check (no-op in production)
			err := p.checkDevModeGuardianSetUpdate(ctx)
			if err != nil {
				return err
			}
		case k := <-p.lockC:
			p.handleLockup(ctx, k)
		case v := <-p.injectC:
			p.handleInjection(ctx, v)
		case m := <-p.obsvC:
			p.handleObservation(ctx, m)
		case <-p.cleanup.C:
			p.handleCleanup(ctx)
		}
	}
}

func (p *Processor) checkDevModeGuardianSetUpdate(ctx context.Context) error {
	if p.devnetMode {
		if uint(len(p.gs.Keys)) != p.devnetNumGuardians {
			v := devnet.DevnetGuardianSetVSS(p.devnetNumGuardians)

			p.logger.Info(fmt.Sprintf("guardian set has %d members, expecting %d - submitting VAA",
				len(p.gs.Keys), p.devnetNumGuardians),
				zap.Any("v", v))

			timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
			defer cancel()
			trx, err := devnet.SubmitVAA(timeout, p.devnetEthRPC, v)
			if err != nil {
				return fmt.Errorf("failed to submit devnet guardian set change: %v", err)
			}

			p.logger.Info("devnet guardian set change submitted to Ethereum", zap.Any("trx", trx), zap.Any("vaa", v))

			if p.terraChaidID != "" {
				// Submit to Terra
				trxResponse, err := terra.SubmitVAA(timeout, p.terraLCD, p.terraChaidID, p.terraContract, p.terraFeePayer, v)
				if err != nil {
					return fmt.Errorf("failed to submit devnet guardian set change: %v", err)
				}
				p.logger.Info("devnet guardian set change submitted to Terra", zap.Any("trxResponse", trxResponse), zap.Any("vaa", v))
			}

			// Submit VAA to Solana as well. This is asynchronous and can fail, leading to inconsistent devnet state.
			p.vaaC <- v
		}
	}

	return nil
}
