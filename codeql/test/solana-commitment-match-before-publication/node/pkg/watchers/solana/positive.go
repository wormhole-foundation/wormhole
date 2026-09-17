package solana

func positiveAccountNoCheck(s *SolanaWatcher, messageAccountData MessageAccountData, isReobservation bool) {
	proposal, err := ParseMessagePublicationAccount(messageAccountData)
	if err != nil {
		return
	}
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	_ = commitment
	observation := &MessagePublication{}
	s.msgC <- observation
}

func positiveAccountCheckAfterSend(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	observation := &MessagePublication{}
	s.msgC <- observation
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
}

func positiveAccountWrongCommitmentChecked(s *SolanaWatcher, proposal *MessagePublicationAccount, other *ShimPostMessageData, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	otherCommitment, err := other.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	if !s.checkCommitment(otherCommitment, isReobservation) {
		return
	}
	_ = commitment
	s.msgC <- &MessagePublication{}
}

func positiveIgnoredCheckResult(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	_ = s.checkCommitment(commitment, isReobservation)
	s.msgC <- &MessagePublication{}
}

func positiveNonDominatingCheck(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool, maybe bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if maybe {
		if !s.checkCommitment(commitment, isReobservation) {
			return
		}
	}
	s.msgC <- &MessagePublication{}
}

func positiveShimNoCheck(s *SolanaWatcher, postMessage *ShimPostMessageData, isReobservation bool) {
	commitment, err := postMessage.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment
	s.msgC <- &MessagePublication{}
}

func positiveInstructionScheduleNoCheck(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, signature Signature, raw []byte, isReobservation bool) {
	data, err := deserializePostMessage(raw)
	if err != nil {
		return
	}
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment
	RunWithScissors(ctx, nil, "retryFetchMessageAccount", func(ctx Context) error {
		s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, signature)
		return nil
	})
}

func positiveConversionErrorIgnored(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, _ := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func positiveCommitmentReassignedAfterCheck(s *SolanaWatcher, proposal *MessagePublicationAccount, other *ShimPostMessageData, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	commitment, err = other.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment
	s.msgC <- &MessagePublication{}
}

func positiveInstructionScheduleNoLocalCheckEvenWhenAccountSideChecked(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, signature Signature, raw []byte, proposal *MessagePublicationAccount, isReobservation bool) {
	data, err := deserializePostMessage(raw)
	if err != nil {
		return
	}
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment
	RunWithScissors(ctx, nil, "retryFetchMessageAccount", func(ctx Context) error {
		s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, signature)
		return nil
	})

	accountCommitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(accountCommitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func positiveDirectRetryFetchNoLocalCheck(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, signature Signature, raw []byte, isReobservation bool) {
	data, err := deserializePostMessage(raw)
	if err != nil {
		return
	}
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment
	s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, signature)
}

func positiveConfirmedReobservationStillNeedsCanonicalCheck(s *SolanaWatcher, proposal *MessagePublicationAccount) {
	isReobservation := true
	s.commitment = CommitmentConfirmed
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if commitment != CommitmentConfirmed {
		return
	}
	_ = isReobservation
	s.msgC <- &MessagePublication{}
}

func positiveInvertedGuardReturnsOnCheckSuccess(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func positiveOtherReceiverSameNameHelper(s *SolanaWatcher, helper *CommitmentHelper, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !helper.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func positiveDifferentWatcherChecked(s *SolanaWatcher, other *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !other.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func positiveNestedConditionalReturnInCommitmentFailureBranch(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool, maybe bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		if maybe {
			return
		}
	}
	s.msgC <- &MessagePublication{}
}

func positiveNestedConditionalReturnInConversionErrorBranch(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool, maybe bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		if maybe {
			return
		}
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}

func positiveStalePreConversionErrGuard(s *SolanaWatcher, messageAccountData MessageAccountData, isReobservation bool) {
	proposal, err := ParseMessagePublicationAccount(messageAccountData)
	if err != nil {
		return
	}
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}
	s.msgC <- &MessagePublication{}
}
