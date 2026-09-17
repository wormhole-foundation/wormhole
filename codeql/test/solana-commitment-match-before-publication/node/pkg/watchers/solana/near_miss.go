package solana

func nearMissNonFinalizedReobservationStillNeedsCheck(s *SolanaWatcher, proposal *MessagePublicationAccount) {
	isReobservation := true
	_ = isReobservation
	s.commitment = CommitmentConfirmed
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	_ = commitment
	s.msgC <- &MessagePublication{}
}

func nearMissPythnetCarveoutIsNotCommitmentCarveout(s *SolanaWatcher, proposal *MessagePublicationAccount) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	_ = commitment
	// Simulates the unrelated Pythnet finalized-account-field carve-out. It must not suppress commitment checking.
	if true {
		s.msgC <- &MessagePublication{}
	}
}
