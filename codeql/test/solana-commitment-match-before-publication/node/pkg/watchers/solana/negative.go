package solana

func negativeAccountChecked(s *SolanaWatcher, messageAccountData MessageAccountData, isReobservation bool) {
	proposal, err := ParseMessagePublicationAccount(messageAccountData)
	if err != nil {
		return
	}
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func negativeShimChecked(s *SolanaWatcher, postMessage *ShimPostMessageData, isReobservation bool) {
	commitment, err := postMessage.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func negativeInstructionScheduleChecked(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, signature Signature, raw []byte, isReobservation bool) {
	data, err := deserializePostMessage(raw)
	if err != nil {
		return
	}
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	RunWithScissors(ctx, nil, "retryFetchMessageAccount", func(ctx Context) error {
		s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, signature)
		return nil
	})
}

func negativeFinalizedWatcherReobservationStillUsesCheck(s *SolanaWatcher, proposal *MessagePublicationAccount) {
	isReobservation := true
	s.commitment = CommitmentFinalized
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func negativePythnetLikeConfirmedAccountSubscriptionStillChecked(s *SolanaWatcher, proposal *MessagePublicationAccount) {
	isReobservation := false
	s.commitment = CommitmentConfirmed
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func negativeUnrelatedRunWithScissorsAfterConversion(s *SolanaWatcher, ctx Context, postMessage *ShimPostMessageData, isReobservation bool) {
	commitment, err := postMessage.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment
	RunWithScissors(ctx, nil, "metrics", func(ctx Context) error {
		return nil
	})
}

func negativeDirectRetryFetchChecked(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, signature Signature, raw []byte, isReobservation bool) {
	data, err := deserializePostMessage(raw)
	if err != nil {
		return
	}
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, signature)
}

func negativeCloseEventDelegatesThroughAccountProof(s *SolanaWatcher, raw []byte, acc PublicKey, signature Signature) bool {
	accountData, err := NewMessageAccountData(raw)
	if err != nil {
		return false
	}
	return s.processMessageAccount(nil, accountData, acc, true, signature, true) > 0
}

func negativeTrueBranchPublication(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if s.checkCommitment(commitment, isReobservation) {
		s.msgC <- &MessagePublication{}
	}
}
