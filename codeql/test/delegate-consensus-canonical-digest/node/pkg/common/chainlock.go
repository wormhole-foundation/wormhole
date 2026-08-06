package common

type Address [32]byte

func (a Address) Hex() string { return "addr" }

type MessagePublication struct {
	Timestamp        uint32
	Nonce            uint32
	Sequence         uint64
	ConsistencyLevel uint8
	EmitterChain     uint16
	EmitterAddress   Address
	Payload          []byte
	IsReobservation  bool
	Unreliable       bool
	verificationState string
	TxID             []byte
	TxHash           []byte
}

type Hash struct{ b []byte }

func (h Hash) Bytes() []byte { return h.b }
func (h Hash) Hex() string   { return "hash" }

type VAA struct{ GuardianSetIndex uint32 }

func (v *VAA) SigningDigest() Hash { return Hash{} }

func (m *MessagePublication) CreateVAA(index uint32) *VAA { return &VAA{GuardianSetIndex: index} }
func (m *MessagePublication) CreateDigest() string         { return "digest" }
func (m *MessagePublication) MarshalBinary() ([]byte, error) { return m.Payload, nil }
func (m *MessagePublication) Marshal() ([]byte, error)       { return m.Payload, nil }
func (m *MessagePublication) MessageIDString() string        { return "chain/address/sequence" }
func (m *MessagePublication) NormalizeForDelegateConsensus() {}
