package ethereum

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/dfuse-io/solana-go"
	"github.com/dfuse-io/solana-go/rpc"
	"github.com/dfuse-io/solana-go/rpc/ws"
	eth_common "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"time"
)

type SolanaWatcher struct {
	bridge    solana.PublicKey
	url       string
	lockEvent chan *common.ChainLock
}

func NewSolanaWatcher(wsUrl string, bridgeAddress solana.PublicKey, lockEvents chan *common.ChainLock) *SolanaWatcher {
	return &SolanaWatcher{bridge: bridgeAddress, url: wsUrl, lockEvent: lockEvents}
}

func (s *SolanaWatcher) Run(ctx context.Context) error {
	c, err := ws.Dial(ctx, s.url)
	if err != nil {
		return fmt.Errorf("failed to connect to solana ws: %w", err)
	}
	defer c.Close()

	logger := supervisor.Logger(ctx)

	sub, err := c.ProgramSubscribe(s.bridge, rpc.CommitmentRecent)
	if err != nil {
		return fmt.Errorf("failed to subscribe to program: %w", err)
	}
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
	}()

	logger.Info("watching for on-chain events")
	for {
		updateRaw, err := sub.Recv()
		if err != nil {
			return err
		}
		programUpdate := updateRaw.(*ws.ProgramResult)
		data := programUpdate.Value.Account.Data

		if len(data) != 1184 {
			logger.Debug(
				"saw update to non-transfer-proposal wormhole account",
				zap.Stringer("account", programUpdate.Value.PubKey),
				zap.Uint64("slot", programUpdate.Context.Slot),
				zap.Error(err),
			)
			continue
		}

		proposal, err := ParseTransferOutProposal(data)
		if err != nil {
			logger.Warn(
				"failed to parse transfer proposal",
				zap.Stringer("account", programUpdate.Value.PubKey),
				zap.Uint64("slot", programUpdate.Context.Slot),
				zap.Error(err),
			)
			continue
		}

		// VAA submitted
		if proposal.VaaTime.Unix() == 0 {
			continue
		}

		var txHash eth_common.Hash
		copy(txHash[:], programUpdate.Value.PubKey[:])

		lock := &common.ChainLock{
			TxHash:        txHash,
			Timestamp:     proposal.LockupTime,
			Nonce:         proposal.Nonce,
			SourceAddress: proposal.SourceAddress,
			TargetAddress: proposal.ForeignAddress,
			SourceChain:   vaa.ChainIDSolana,
			TargetChain:   proposal.ToChainID,
			TokenChain:    proposal.Asset.Chain,
			TokenAddress:  proposal.Asset.Address,
			TokenDecimals: proposal.Asset.Decimals,
			Amount:        proposal.Amount,
		}
		logger.Info("found new lockup transaction", zap.Stringer("lockup_address", programUpdate.Value.PubKey))
		s.lockEvent <- lock
	}
}

type (
	TransferOutProposal struct {
		Amount           *big.Int
		ToChainID        vaa.ChainID
		SourceAddress    vaa.Address
		ForeignAddress   vaa.Address
		Asset            vaa.AssetMeta
		Nonce            uint32
		VAA              [1001]byte
		VaaTime          time.Time
		LockupTime       time.Time
		PokeCounter      uint8
		SignatureAccount solana.PublicKey
	}
)

func ParseTransferOutProposal(data []byte) (*TransferOutProposal, error) {
	prop := &TransferOutProposal{}
	r := bytes.NewBuffer(data)

	var amountBytes [32]byte
	if n, err := r.Read(amountBytes[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read amount: %w", err)
	}
	// Reverse (little endian -> big endian)
	for i := 0; i < len(amountBytes)/2; i++ {
		amountBytes[i], amountBytes[len(amountBytes)-i-1] = amountBytes[len(amountBytes)-i-1], amountBytes[i]
	}
	prop.Amount = new(big.Int).SetBytes(amountBytes[:])

	if err := binary.Read(r, binary.LittleEndian, &prop.ToChainID); err != nil {
		return nil, fmt.Errorf("failed to read to chain id: %w", err)
	}

	if n, err := r.Read(prop.SourceAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read source address: %w", err)
	}

	if n, err := r.Read(prop.ForeignAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read source address: %w", err)
	}

	assetMeta := vaa.AssetMeta{}
	if n, err := r.Read(assetMeta.Address[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read asset meta address: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &assetMeta.Chain); err != nil {
		return nil, fmt.Errorf("failed to read asset meta chain: %w", err)
	}

	if err := binary.Read(r, binary.LittleEndian, &assetMeta.Decimals); err != nil {
		return nil, fmt.Errorf("failed to read asset meta decimals: %w", err)
	}
	prop.Asset = assetMeta

	// Skip alignment byte
	r.Next(1)

	if err := binary.Read(r, binary.LittleEndian, &prop.Nonce); err != nil {
		return nil, fmt.Errorf("failed to read nonce: %w", err)
	}

	if n, err := r.Read(prop.VAA[:]); err != nil || n != 1001 {
		return nil, fmt.Errorf("failed to read vaa: %w", err)
	}

	// Skip alignment bytes
	r.Next(3)

	var vaaTime uint32
	if err := binary.Read(r, binary.LittleEndian, &vaaTime); err != nil {
		return nil, fmt.Errorf("failed to read vaa time: %w", err)
	}
	prop.VaaTime = time.Unix(int64(vaaTime), 0)

	var lockupTime uint32
	if err := binary.Read(r, binary.LittleEndian, &lockupTime); err != nil {
		return nil, fmt.Errorf("failed to read lockup time: %w", err)
	}
	prop.LockupTime = time.Unix(int64(lockupTime), 0)

	if err := binary.Read(r, binary.LittleEndian, &prop.PokeCounter); err != nil {
		return nil, fmt.Errorf("failed to read poke counter: %w", err)
	}

	if n, err := r.Read(prop.SignatureAccount[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read signature account: %w", err)
	}

	return prop, nil
}

//0000:   80 96 98 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0010:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0020:   02 bd 84 f9  6d c4 95 5d  6c 7f 87 6d  e1 15 73 84   ....m..]l..m..s.
//0030:   76 dd d3 43  fe 10 19 d1  39 53 4a dd  c9 07 01 8c   v..C....9SJ.....
//0040:   fb 00 00 00  00 00 00 00  00 00 00 00  00 8d 68 94   ..............h.
//0050:   76 eb 44 6a  1f b0 06 5b  ff ac 32 39  8e d7 f8 91   v.Dj...[..29....
//0060:   65 00 00 00  00 00 00 00  00 00 00 00  00 a0 b8 69   e..............i
//0070:   91 c6 21 8b  36 c1 d1 9d  4a 2e 9e b0  ce 36 06 eb   ..!.6...J....6..
//0080:   48 02 06 00  26 3a 00 00  01 00 00 00  00 00 60 07   H...&:........`.
//0090:   5f e0 10 00  00 3a 26 01  02 bd 84 f9  6d c4 95 5d   _....:&.....m..]
//00a0:   6c 7f 87 6d  e1 15 73 84  76 dd d3 43  fe 10 19 d1   l..m..s.v..C....
//00b0:   39 53 4a dd  c9 07 01 8c  fb 00 00 00  00 00 00 00   9SJ.............
//00c0:   00 00 00 00  00 8d 68 94  76 eb 44 6a  1f b0 06 5b   ......h.v.Dj...[
//00d0:   ff ac 32 39  8e d7 f8 91  65 02 00 00  00 00 00 00   ..29....e.......
//00e0:   00 00 00 00  00 00 a0 b8  69 91 c6 21  8b 36 c1 d1   ........i..!.6..
//00f0:   9d 4a 2e 9e  b0 ce 36 06  eb 48 06 00  00 00 00 00   .J....6..H......
//0100:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0110:   00 00 00 00  00 00 00 00  98 96 80 ff  00 00 00 00   ................
//0120:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0130:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0140:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0150:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0160:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0170:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0180:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0190:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//01a0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//01b0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//01c0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//01d0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//01e0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//01f0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0200:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0210:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0220:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0230:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0240:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0250:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0260:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0270:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0280:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0290:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//02a0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//02b0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//02c0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//02d0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//02e0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//02f0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0300:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0310:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0320:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0330:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0340:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0350:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0360:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0370:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0380:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0390:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//03a0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//03b0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//03c0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//03d0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//03e0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//03f0:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0400:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0410:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0420:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0430:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0440:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0450:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0460:   00 00 00 00  00 00 00 00  00 00 00 00  00 00 00 00   ................
//0470:   00 00 00 00  e0 5f 07 60  e0 5f 07 60  03 a4 f2 e0   ....._.`._.`....
//0480:   22 fe c8 5b  8b cb fe c1  92 d7 1a 5d  38 e4 82 f1   "..[.......]8...
//0490:   32 8d a6 bf  4d 14 a9 2f  77 55 fc 2e  cc 01 00 00   2...M../wU......
