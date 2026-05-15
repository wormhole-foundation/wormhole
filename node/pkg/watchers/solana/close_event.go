package solana

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap"
)

const (
	closePostedMessageInstructionID     = 0x09
	closePostedMessageMinNumAccounts    = 6
	closePostedMessageMessageAccountIdx = 1
	closeEventTagStr                    = "e445a52e51cb9a1d" // EVENT_IX_TAG_LE: SHA256("anchor:event")[0..8] as LE u64
	closeEventDiscriminatorStr          = "9ef61bc2241428b9" // SHA256("event:MessageAccountClosed")[0..8]
	closeEventMinDataLen                = 16 + 3             // 16-byte event header + 3-byte account prefix
)

// closeEventTag is the Anchor event CPI selector (8 bytes).
var closeEventTag []byte

// closeEventDiscriminator identifies the MessageAccountClosed event (8 bytes).
var closeEventDiscriminator []byte

func init() {
	var err error
	closeEventTag, err = hex.DecodeString(closeEventTagStr)
	if err != nil {
		panic("invalid closeEventTag: " + err.Error())
	}
	closeEventDiscriminator, err = hex.DecodeString(closeEventDiscriminatorStr)
	if err != nil {
		panic("invalid closeEventDiscriminator: " + err.Error())
	}
}

// isCloseEventInstruction returns true if the instruction data starts with the
// close event tag and discriminator.
func isCloseEventInstruction(programIndex uint16, inst solana.CompiledInstruction) bool {
	return inst.ProgramIDIndex == programIndex &&
		len(inst.Data) >= 16 &&
		bytes.Equal(inst.Data[:8], closeEventTag) &&
		bytes.Equal(inst.Data[8:16], closeEventDiscriminator)
}

// resolveCloseEventMessageAccount extracts the message account pubkey from the
// close instruction and returns the account data portion of the CPI event
// (everything after the 16-byte event header, i.e. the full account data
// including the 3-byte prefix).
func resolveCloseEventMessageAccount(
	tx *solana.Transaction,
	closeInst solana.CompiledInstruction,
	eventData []byte,
) (solana.PublicKey, []byte, error) {
	if len(eventData) < closeEventMinDataLen {
		return solana.PublicKey{}, nil, fmt.Errorf("close event data too short: %d bytes", len(eventData))
	}
	if len(closeInst.Accounts) < closePostedMessageMinNumAccounts {
		return solana.PublicKey{}, nil, fmt.Errorf("close instruction has insufficient accounts: %d", len(closeInst.Accounts))
	}
	messageAccountIdx := closeInst.Accounts[closePostedMessageMessageAccountIdx]
	if int(messageAccountIdx) >= len(tx.Message.AccountKeys) {
		return solana.PublicKey{}, nil, fmt.Errorf("message account index %d out of bounds (account keys len %d)", messageAccountIdx, len(tx.Message.AccountKeys))
	}
	return tx.Message.AccountKeys[messageAccountIdx], eventData[16:], nil
}

// processClosePostedMessageEvent handles a top-level close_posted_message
// instruction by scanning its inner instructions for the CPI event.
func (s *SolanaWatcher) processClosePostedMessageEvent(
	logger *zap.Logger,
	programIndex uint16,
	tx *solana.Transaction,
	innerInstructions []rpc.InnerInstruction,
	topLevelIndex int,
	topLevelInst solana.CompiledInstruction,
	alreadyProcessed ShimAlreadyProcessed,
	signature solana.Signature,
) (bool, error) {
	topLevelIdx := uint16(topLevelIndex) // #nosec G115 -- Solana max tx size (1232 bytes) bounds instruction count well within uint16.
	for outerIdx, innerSet := range innerInstructions {
		if innerSet.Index != topLevelIdx {
			continue
		}
		for innerIdx, inst := range innerSet.Instructions {
			if isCloseEventInstruction(programIndex, inst) {
				messageAccount, accountData, err := resolveCloseEventMessageAccount(tx, topLevelInst, inst.Data)
				if err != nil {
					return false, err
				}
				alreadyProcessed.add(outerIdx, innerIdx)
				return s.processMessageAccount(logger, accountData, messageAccount, true, signature, true) > 0, nil
			}
		}
	}
	return false, nil
}

// processInnerClosePostedMessageEvent handles a close_posted_message that
// appears as an inner instruction (called via CPI from another program). The
// close instruction and its CPI event are sibling inner instructions within
// the same inner instruction set. Both are marked in alreadyProcessed so the
// outer loop does not re-process them (same pattern as the shim).
func (s *SolanaWatcher) processInnerClosePostedMessageEvent(
	logger *zap.Logger,
	programIndex uint16,
	tx *solana.Transaction,
	innerInstructions []solana.CompiledInstruction,
	outerIdx int,
	closeIdx int,
	closeInst solana.CompiledInstruction,
	alreadyProcessed ShimAlreadyProcessed,
	signature solana.Signature,
) (bool, error) {
	alreadyProcessed.add(outerIdx, closeIdx)

	// The CPI event follows the close instruction in the same inner set.
	for i := closeIdx + 1; i < len(innerInstructions); i++ {
		inst := innerInstructions[i]
		if isCloseEventInstruction(programIndex, inst) {
			messageAccount, accountData, err := resolveCloseEventMessageAccount(tx, closeInst, inst.Data)
			if err != nil {
				return false, err
			}
			alreadyProcessed.add(outerIdx, i)
			return s.processMessageAccount(logger, accountData, messageAccount, true, signature, true) > 0, nil
		}
	}
	return false, nil
}
