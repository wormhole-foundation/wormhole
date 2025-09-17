package sui

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func (e *Watcher) verify(
	ctx context.Context,
	msg *common.MessagePublication,
	txDigest string,
	logger *zap.Logger,
) (common.MessagePublication, error) {

	if msg == nil {
		return common.MessagePublication{}, fmt.Errorf("MessagePublication is nil")
	}

	if msg.VerificationState() != common.NotVerified {
		return common.MessagePublication{}, fmt.Errorf("MessagePublication already has a non-default verification state")
	}

	if e.suiTxVerifier == nil {
		return common.MessagePublication{}, fmt.Errorf("transfer verifier is nil")
	}

	localMsg := *msg

	var verificationState common.VerificationState

	// If the payload does not represent a transfer, or if the emitter address of the message does
	// not match the token bridge emitter, mark the message's verification state as NotApplicable.
	if !vaa.IsTransfer(msg.Payload) || "0x"+localMsg.EmitterAddress.String() != e.suiTxVerifier.GetTokenBridgeEmitter() {
		verificationState = common.NotApplicable
	} else {
		// Validate the transfers in the transaction block associated with the
		// transaction digest.
		valid, err := e.suiTxVerifier.ProcessDigest(ctx, txDigest, localMsg.MessageIDString(), logger)

		if err != nil {
			logger.Error("an internal Sui tx verifier error occurred: ", zap.Error(err))
			verificationState = common.CouldNotVerify
		} else if valid {
			verificationState = common.Valid
		} else {
			// If no error and validation failed, mark as Anomalous. For Sui transfers, Anomalous is used, since there are
			// more edge cases that need to be considered and covered before outright rejecting a transfer.
			verificationState = common.Anomalous
		}
	}

	// Update the state of the message.
	updateErr := localMsg.SetVerificationState(verificationState)
	if updateErr != nil {
		errMsg := fmt.Sprintf("could not set verification state for message with txID %s", localMsg.TxIDString())
		return common.MessagePublication{}, fmt.Errorf("%s %w", errMsg, updateErr)
	}

	return localMsg, nil
}
