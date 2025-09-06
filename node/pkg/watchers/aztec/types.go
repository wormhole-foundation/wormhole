package aztec

import (
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// WatcherConfig is the configuration used by node.go
type WatcherConfig struct {
	NetworkID watchers.NetworkID
	ChainID   vaa.ChainID
	Rpc       string
	Contract  string
}

// LogParameters encapsulates the core parameters from a log
type LogParameters struct {
	SenderAddress    vaa.Address
	Sequence         uint64
	Nonce            uint32
	ConsistencyLevel uint8
	ArbitrumAddress  []byte
	ArbitrumChainID  uint16
	Amount           uint64
	TxID             string
}

// BlockInfo enhanced to include block hash and parent hash
type BlockInfo struct {
	TxHash            string
	Timestamp         uint64
	archiveRoot       string
	parentArchiveRoot string
	TxHashesByIndex   map[int]string
}

// FinalizedBlock represents a finalized block's information
type FinalizedBlock struct {
	Number int    `json:"number"`
	Hash   string `json:"hash,omitempty"`
}

// L2Tips represents the response from the node_getL2Tips RPC method
type L2Tips struct {
	Latest struct {
		Number int    `json:"number"`
		Hash   string `json:"hash"`
	} `json:"latest"`
	Proven struct {
		Number int    `json:"number"`
		Hash   string `json:"hash"`
	} `json:"proven"`
	Finalized struct {
		Number int    `json:"number"`
		Hash   string `json:"hash,omitempty"`
	} `json:"finalized"`
}

type L2TipsResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  L2Tips `json:"result"`
}

// JSON-RPC related structures
type JsonRpcResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Logs       []ExtendedPublicLog `json:"logs"`
		MaxLogsHit bool                `json:"maxLogsHit"`
	} `json:"result"`
}

type BlockResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  BlockResult `json:"result"`
}

type BlockResult struct {
	Archive BlockArchive `json:"archive"`
	Header  BlockHeader  `json:"header"`
	Body    BlockBody    `json:"body"`
}

type BlockArchive struct {
	Root                   string `json:"root"`
	NextAvailableLeafIndex int    `json:"nextAvailableLeafIndex"`
}

type BlockHeader struct {
	LastArchive       BlockArchive      `json:"lastArchive"`
	ContentCommitment ContentCommitment `json:"contentCommitment"`
	State             State             `json:"state"`
	GlobalVariables   GlobalVariables   `json:"globalVariables"`
	TotalFees         string            `json:"totalFees"`
	TotalManaUsed     string            `json:"totalManaUsed"`
}

type ContentCommitment struct {
	BlobsHash string `json:"blobsHash"`
	InHash    string `json:"inHash"`
	OutHash   string `json:"outHash"`
}

type State struct {
	L1ToL2MessageTree MerkleTree   `json:"l1ToL2MessageTree"`
	Partial           PartialState `json:"partial"`
}

type PartialState struct {
	NoteHashTree   MerkleTree `json:"noteHashTree"`
	NullifierTree  MerkleTree `json:"nullifierTree"`
	PublicDataTree MerkleTree `json:"publicDataTree"`
}

type MerkleTree struct {
	Root                   string `json:"root"`
	NextAvailableLeafIndex int    `json:"nextAvailableLeafIndex"`
}

type GlobalVariables struct {
	ChainID      string  `json:"chainId"`
	Version      string  `json:"version"`
	BlockNumber  int     `json:"blockNumber"`
	SlotNumber   string  `json:"slotNumber"`
	Timestamp    string  `json:"timestamp"`
	Coinbase     string  `json:"coinbase"`
	FeeRecipient string  `json:"feeRecipient"`
	GasFees      GasFees `json:"gasFees"`
}

type GasFees struct {
	FeePerDaGas string `json:"feePerDaGas"`
	FeePerL2Gas string `json:"feePerL2Gas"`
}

type BlockBody struct {
	TxEffects []TxEffect `json:"txEffects"`
}

type TxEffect struct {
	RevertCode        int                `json:"revertCode"`
	TxHash            string             `json:"txHash"`
	TransactionFee    string             `json:"transactionFee"`
	NoteHashes        []string           `json:"noteHashes"`
	Nullifiers        []string           `json:"nullifiers"`
	L2ToL1Msgs        []string           `json:"l2ToL1Msgs"`
	PublicDataWrites  []PublicDataWrite  `json:"publicDataWrites"`
	PrivateLogs       []PrivateLog       `json:"privateLogs"`
	PublicLogs        []PublicLog        `json:"publicLogs"`
	ContractClassLogs []ContractClassLog `json:"contractClassLogs"`
}

type PublicDataWrite struct {
	LeafSlot string `json:"leafSlot"`
	Value    string `json:"value"`
}

type PrivateLog struct {
	Fields        []string `json:"fields"`
	EmittedLength int      `json:"emittedLength"`
}

type ContractClassLog struct {
	ContractAddress string `json:"contractAddress"`
	Fields          struct {
		Fields []string `json:"fields"`
	} `json:"fields"`
	EmittedLength int `json:"emittedLength"`
}

type LogId struct {
	BlockNumber int `json:"blockNumber"`
	TxIndex     int `json:"txIndex"`
	LogIndex    int `json:"logIndex"`
}

type PublicLog struct {
	ContractAddress string   `json:"contractAddress"`
	Fields          []string `json:"fields"`
	EmittedLength   int      `json:"emittedLength"`
}

type ExtendedPublicLog struct {
	ID  LogId     `json:"id"`
	Log PublicLog `json:"log"`
}

type ErrRPCError struct {
	Method string
	Code   int
	Msg    string
}

func (e ErrRPCError) Error() string {
	return "RPC error calling " + e.Method + ": " + e.Msg
}

type ErrMaxRetriesExceeded struct {
	Method string
}

func (e ErrMaxRetriesExceeded) Error() string {
	return "max retries exceeded for " + e.Method
}

type ErrParsingFailed struct {
	What string
	Err  error
}

func (e ErrParsingFailed) Error() string {
	return "failed parsing " + e.What + ": " + e.Err.Error()
}
