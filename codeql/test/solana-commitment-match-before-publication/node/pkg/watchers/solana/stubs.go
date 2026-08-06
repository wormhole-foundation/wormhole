package solana

type CommitmentType string

const (
	CommitmentConfirmed CommitmentType = "confirmed"
	CommitmentFinalized CommitmentType = "finalized"
)

type ConsistencyLevel uint8

func (c ConsistencyLevel) Commitment() (CommitmentType, error) { return CommitmentType(""), nil }

func accountConsistencyLevelToCommitment(c uint8) (CommitmentType, error) { return CommitmentType(""), nil }

type MessagePublicationAccount struct {
	ConsistencyLevel uint8
}

type MessageAccountData struct{}

func NewMessageAccountData(data []byte) (MessageAccountData, error) { return MessageAccountData{}, nil }

func ParseMessagePublicationAccount(messageAccountData MessageAccountData) (*MessagePublicationAccount, error) {
	return nil, nil
}

type ShimPostMessageData struct {
	ConsistencyLevel ConsistencyLevel
	Nonce            uint32
	Payload          []byte
}

type PostMessageData struct {
	ConsistencyLevel ConsistencyLevel
}

type MessagePublication struct{}

type SolanaWatcher struct {
	commitment CommitmentType
	msgC       chan<- *MessagePublication
}

type CommitmentHelper struct{}

func (h *CommitmentHelper) checkCommitment(commitment CommitmentType, isReobservation bool) bool {
	return true
}

func (s *SolanaWatcher) checkCommitment(commitment CommitmentType, isReobservation bool) bool {
	if commitment == s.commitment {
		return true
	}
	return isReobservation && s.commitment == CommitmentFinalized
}

type PublicKey struct{}

type Signature struct{}

type Transaction struct{}

type RPCClient struct{}

type Context struct{}

func RunWithScissors(ctx Context, errC chan error, name string, runnable func(Context) error) {}

func (s *SolanaWatcher) retryFetchMessageAccount(ctx Context, rpcClient *RPCClient, acc PublicKey, slot uint64, retry uint, isReobservation bool, signature Signature) {
}

func (s *SolanaWatcher) processMessageAccount(logger any, messageAccountData MessageAccountData, acc PublicKey, isReobservation bool, signature Signature, useSignatureAsTxID bool) uint32 {
	proposal, err := ParseMessagePublicationAccount(messageAccountData)
	if err != nil {
		return 0
	}
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return 0
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return 0
	}
	s.msgC <- &MessagePublication{}
	return 1
}

func deserializePostMessage(data []byte) (PostMessageData, error) { return PostMessageData{}, nil }
