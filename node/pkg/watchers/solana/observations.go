package solana

import (
	"errors"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/gagliardetto/solana-go"
)

// buildMessagePublicationFromAccountData is the deterministic account-message parser/builder boundary.
// It must not perform network requests; callers are responsible for fetching account bytes, proving account ownership, and converting bytes to MessageAccountData before calling it.
func (s *SolanaWatcher) buildMessagePublicationFromAccountData(messageAccount solana.PublicKey, messageAccountData MessageAccountData, txSignature solana.Signature, isReobservation bool, useSignatureAsTxID bool) (*common.MessagePublication, error) {
	proposal, err := ParseMessagePublicationAccount(messageAccountData)
	if err != nil {
		return nil, err
	}

	var txID []byte
	if useSignatureAsTxID && !txSignature.IsZero() {
		// Close event path: use the Solana transaction signature as TxID,
		// matching the shim convention.
		txID = txSignature[:]
	} else {
		// Account-based path: use the message account pubkey as TxID.
		txID = messageAccount[:]
	}

	return &common.MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(proposal.SubmissionTime), 0),
		Nonce:            proposal.Nonce,
		Sequence:         proposal.Sequence,
		EmitterChain:     s.chainID, // SECURITY: The message must be emitted from the chain this watcher is observing. This prevents mix-ups between different SVM chains.
		EmitterAddress:   proposal.EmitterAddress,
		Payload:          proposal.Payload,
		ConsistencyLevel: proposal.ConsistencyLevel,
		IsReobservation:  isReobservation,
		Unreliable:       !messageAccountData.IsReliable(),
	}, nil
}

// buildMessagePublicationFromShimData is the deterministic shim parser/builder boundary.
// It must not perform network requests; callers are responsible for extracting the relevant instruction bytes from a transaction before calling it.
func (s *SolanaWatcher) buildMessagePublicationFromShimData(txSignature solana.Signature, postMessageInstructionData []byte, messageEventInstructionData []byte, isReobservation bool) (*common.MessagePublication, error) {
	postMessage, err := shimParsePostMessage(s.shimPostMessageDiscriminator, postMessageInstructionData)
	if err != nil {
		return nil, err
	}
	if postMessage == nil {
		return nil, errors.New("instruction is not a shim post message")
	}

	messageEvent, err := shimParseMessageEvent(s.shimMessageEventDiscriminator, messageEventInstructionData)
	if err != nil {
		return nil, err
	}
	if messageEvent == nil {
		return nil, errors.New("instruction is not a shim message event")
	}

	return &common.MessagePublication{
		TxID:             txSignature[:],
		Timestamp:        time.Unix(int64(messageEvent.Timestamp), 0),
		Nonce:            postMessage.Nonce,
		Sequence:         messageEvent.Sequence,
		EmitterChain:     s.chainID,
		EmitterAddress:   messageEvent.EmitterAddress,
		Payload:          postMessage.Payload,
		ConsistencyLevel: uint8(postMessage.ConsistencyLevel),
		IsReobservation:  isReobservation,

		// Shim messages are always reliable.
		Unreliable: false,
	}, nil
}
