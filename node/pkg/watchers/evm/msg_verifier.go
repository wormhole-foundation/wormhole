package evm

import (
	"context"
	"fmt"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/txverifier"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// verify evaluates a MessagePublication using the Transfer Verifier.
// On success, this function returns a copy of the MessagePublication with its verificationState set to the result of
// the transfer verifier's evaluation.
// On error, it returns an empty MessagePublication struct.
func verify(
	ctx context.Context,
	msg *common.MessagePublication,
	txHash eth_common.Hash,
	receipt *types.Receipt,
	txVerifier txverifier.TransferVerifierInterface,
) (common.MessagePublication, error) {

	if msg == nil {
		return common.MessagePublication{}, fmt.Errorf("MessagePublication is nil")
	}

	if msg.VerificationState() != common.NotVerified {
		return common.MessagePublication{}, fmt.Errorf("MessagePublication already has a non-default verification state")
	}

	if txVerifier == nil {
		return common.MessagePublication{}, fmt.Errorf("transfer verifier is nil")
	}

	// Create a local copy of the MessagePublication.
	localMsg := msg

	var newState common.VerificationState

	// Only involve the transfer verifier for core messages sent
	// from the token bridge. This check is also done in the
	// transfer verifier package, but this helps us skip useless
	// computation.
	if txverifier.Cmp(localMsg.EmitterAddress, txVerifier.Addrs().TokenBridgeAddr) != 0 {
		newState = common.NotApplicable
	} else {
		newState = state(ctx, txHash, receipt, txVerifier)
	}

	// Update the state of the message.
	updateErr := localMsg.SetVerificationState(newState)
	if updateErr != nil {
		errMsg := fmt.Sprintf("could not set verification state for message with txID %s", localMsg.TxIDString())
		return common.MessagePublication{}, fmt.Errorf("%s %w", errMsg, updateErr)
	}

	return *localMsg, nil
}

// state returns a verification state based on the results of querying the Transfer Verifier.
func state(ctx context.Context, txHash eth_common.Hash, receipt *types.Receipt, tv txverifier.TransferVerifierInterface) common.VerificationState {
	// Verify the transfer by analyzing the transaction receipt. This is a defense-in-depth mechanism
	// to protect against fraudulent message emissions.
	valid, err := tv.ProcessEvent(ctx, txHash, receipt)

	// The receipt couldn't be processed properly for some reason.
	if err != nil {
		return common.CouldNotVerify
	}

	if !valid {
		return common.Rejected
	}

	return common.Valid
}
