// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package lineaabi

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// IL1MessageServiceClaimMessageWithProofParams is an auto generated low-level Go binding around an user-defined struct.
type IL1MessageServiceClaimMessageWithProofParams struct {
	Proof         [][32]byte
	MessageNumber *big.Int
	LeafIndex     uint32
	From          common.Address
	To            common.Address
	Fee           *big.Int
	Value         *big.Int
	FeeRecipient  common.Address
	MerkleRoot    [32]byte
	Data          []byte
}

// ILineaRollupFinalizationData is an auto generated low-level Go binding around an user-defined struct.
type ILineaRollupFinalizationData struct {
	ParentStateRootHash        [32]byte
	DataHashes                 [][32]byte
	DataParentHash             [32]byte
	FinalBlockNumber           *big.Int
	LastFinalizedTimestamp     *big.Int
	FinalTimestamp             *big.Int
	L1RollingHash              [32]byte
	L1RollingHashMessageNumber *big.Int
	L2MerkleRoots              [][32]byte
	L2MerkleTreesDepth         *big.Int
	L2MessagingBlocksOffsets   []byte
}

// ILineaRollupSubmissionData is an auto generated low-level Go binding around an user-defined struct.
type ILineaRollupSubmissionData struct {
	ParentStateRootHash [32]byte
	DataParentHash      [32]byte
	FinalStateRootHash  [32]byte
	FirstBlockInData    *big.Int
	FinalBlockInData    *big.Int
	SnarkHash           [32]byte
	CompressedData      []byte
}

// ILineaRollupSupportingSubmissionData is an auto generated low-level Go binding around an user-defined struct.
type ILineaRollupSupportingSubmissionData struct {
	ParentStateRootHash [32]byte
	DataParentHash      [32]byte
	FinalStateRootHash  [32]byte
	FirstBlockInData    *big.Int
	FinalBlockInData    *big.Int
	SnarkHash           [32]byte
}

// LineaabiMetaData contains all meta data concerning the Lineaabi contract.
var LineaabiMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"BytesLengthNotMultipleOf32\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"bytesLength\",\"type\":\"uint256\"}],\"name\":\"BytesLengthNotMultipleOfTwo\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"currentDataHash\",\"type\":\"bytes32\"}],\"name\":\"DataAlreadySubmitted\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"actual\",\"type\":\"uint256\"}],\"name\":\"DataEndingBlockDoesNotMatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"expected\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"value\",\"type\":\"bytes32\"}],\"name\":\"DataHashesNotInSequence\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"DataParentHasEmptyShnarf\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"actual\",\"type\":\"uint256\"}],\"name\":\"DataStartingBlockDoesNotMatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptyBlobData\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"EmptySubmissionData\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"}],\"name\":\"FeePaymentFailed\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FeeTooLow\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"finalBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lastFinalizedBlock\",\"type\":\"uint256\"}],\"name\":\"FinalBlockNumberLessThanOrEqualToLastFinalizedBlock\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FinalBlockStateEqualsZeroHash\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"firstHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"secondHash\",\"type\":\"bytes32\"}],\"name\":\"FinalStateRootHashDoesNotMatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FinalizationDataMissing\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"l2BlockTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"currentBlockTimestamp\",\"type\":\"uint256\"}],\"name\":\"FinalizationInTheFuture\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"firstBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"finalBlockNumber\",\"type\":\"uint256\"}],\"name\":\"FirstBlockGreaterThanFinalBlock\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"firstBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lastFinalizedBlock\",\"type\":\"uint256\"}],\"name\":\"FirstBlockLessThanOrEqualToLastFinalizedBlock\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FirstByteIsNotZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidMerkleProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidProof\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"InvalidProofType\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"pauseType\",\"type\":\"uint256\"}],\"name\":\"IsNotPaused\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"pauseType\",\"type\":\"uint256\"}],\"name\":\"IsPaused\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"L1L2MessageNotSent\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"messageNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"rollingHash\",\"type\":\"bytes32\"}],\"name\":\"L1RollingHashDoesNotExistOnL1\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"name\":\"L2MerkleRootAlreadyAnchored\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"L2MerkleRootDoesNotExist\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"expected\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"actual\",\"type\":\"bytes32\"}],\"name\":\"LastFinalizedShnarfWrong\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"LimitIsZero\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"messageIndex\",\"type\":\"uint256\"}],\"name\":\"MessageAlreadyClaimed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"MessageAlreadyReceived\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"MessageDoesNotExistOrHasAlreadyBeenClaimed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"destination\",\"type\":\"address\"}],\"name\":\"MessageSendingFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"rollingHash\",\"type\":\"bytes32\"}],\"name\":\"MissingMessageNumberForRollingHash\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"messageNumber\",\"type\":\"uint256\"}],\"name\":\"MissingRollingHashForMessageNumber\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"firstHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"secondHash\",\"type\":\"bytes32\"}],\"name\":\"ParentHashesDoesNotMatch\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PeriodIsZero\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"PointEvaluationFailed\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"fieldElements\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"blsCurveModulus\",\"type\":\"uint256\"}],\"name\":\"PointEvaluationResponseInvalid\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"actual\",\"type\":\"uint256\"}],\"name\":\"PrecompileReturnDataLengthWrong\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ProofIsEmpty\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"actual\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"}],\"name\":\"ProofLengthDifferentThanMerkleDepth\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"RateLimitExceeded\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"StartingRootHashDoesNotMatch\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"expected\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"actual\",\"type\":\"bytes32\"}],\"name\":\"StateRootHashInvalid\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"expected\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"TimestampsNotInSequence\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ValueSentTooLow\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"YPointGreaterThanCurveModulus\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"ZeroAddressNotAllowed\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"resettingAddress\",\"type\":\"address\"}],\"name\":\"AmountUsedInPeriodReset\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"stateRootHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bool\",\"name\":\"finalizedWithProof\",\"type\":\"bool\"}],\"name\":\"BlockFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"lastBlockFinalized\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"startingRootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"finalRootHash\",\"type\":\"bytes32\"}],\"name\":\"BlocksVerificationDone\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"lastBlockFinalized\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"startingRootHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"finalRootHash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"withProof\",\"type\":\"bool\"}],\"name\":\"DataFinalized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"startBlock\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"endBlock\",\"type\":\"uint256\"}],\"name\":\"DataSubmitted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"version\",\"type\":\"uint8\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32[]\",\"name\":\"messageHashes\",\"type\":\"bytes32[]\"}],\"name\":\"L1L2MessagesReceivedOnL2\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"L2L1MessageHashAddedToInbox\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"l2MerkleRoot\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"treeDepth\",\"type\":\"uint256\"}],\"name\":\"L2MerkleRootAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"l2Block\",\"type\":\"uint256\"}],\"name\":\"L2MessagingBlockAnchored\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"amountChangeBy\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"amountUsedLoweredToLimit\",\"type\":\"bool\"},{\"indexed\":false,\"internalType\":\"bool\",\"name\":\"usedAmountResetToZero\",\"type\":\"bool\"}],\"name\":\"LimitAmountChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_messageHash\",\"type\":\"bytes32\"}],\"name\":\"MessageClaimed\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"bytes\",\"name\":\"_calldata\",\"type\":\"bytes\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"_messageHash\",\"type\":\"bytes32\"}],\"name\":\"MessageSent\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"messageSender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"pauseType\",\"type\":\"uint256\"}],\"name\":\"Paused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"periodInSeconds\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"limitInWei\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"currentPeriodEnd\",\"type\":\"uint256\"}],\"name\":\"RateLimitInitialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"previousAdminRole\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"newAdminRole\",\"type\":\"bytes32\"}],\"name\":\"RoleAdminChanged\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleGranted\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"}],\"name\":\"RoleRevoked\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"messageNumber\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"rollingHash\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"RollingHashUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"systemMigrationBlock\",\"type\":\"uint256\"}],\"name\":\"SystemMigrationBlockInitialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"address\",\"name\":\"messageSender\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"pauseType\",\"type\":\"uint256\"}],\"name\":\"UnPaused\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"verifierAddress\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"proofType\",\"type\":\"uint256\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"verifierSetBy\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"oldVerifierAddress\",\"type\":\"address\"}],\"name\":\"VerifierAddressChanged\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEFAULT_ADMIN_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"GENERAL_PAUSE_TYPE\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"GENESIS_SHNARF\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"INBOX_STATUS_RECEIVED\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"INBOX_STATUS_UNKNOWN\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L1_L2_PAUSE_TYPE\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"L2_L1_PAUSE_TYPE\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OPERATOR_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OUTBOX_STATUS_RECEIVED\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OUTBOX_STATUS_SENT\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"OUTBOX_STATUS_UNKNOWN\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PAUSE_MANAGER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PROVING_SYSTEM_PAUSE_TYPE\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"RATE_LIMIT_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"VERIFIER_SETTER_ROLE\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_value\",\"type\":\"uint256\"},{\"internalType\":\"addresspayable\",\"name\":\"_feeRecipient\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"_calldata\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_nonce\",\"type\":\"uint256\"}],\"name\":\"claimMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32[]\",\"name\":\"proof\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"messageNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"leafIndex\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"fee\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"addresspayable\",\"name\":\"feeRecipient\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"data\",\"type\":\"bytes\"}],\"internalType\":\"structIL1MessageService.ClaimMessageWithProofParams\",\"name\":\"_params\",\"type\":\"tuple\"}],\"name\":\"claimMessageWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentFinalizedShnarf\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentL2BlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentL2StoredL1MessageNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentL2StoredL1RollingHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentPeriodAmountInWei\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentPeriodEnd\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"currentTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"dataEndingBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"endingBlock\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"dataFinalStateRootHashes\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"finalStateRootHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"dataParents\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"parentHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"dataShnarfHashes\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"shnarfHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"dataHash\",\"type\":\"bytes32\"}],\"name\":\"dataStartingBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"startingBlock\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes\",\"name\":\"_aggregatedProof\",\"type\":\"bytes\"},{\"internalType\":\"uint256\",\"name\":\"_proofType\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"parentStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"dataHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"dataParentHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"finalBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lastFinalizedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"finalTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"l1RollingHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"l1RollingHashMessageNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"l2MerkleRoots\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"l2MerkleTreesDepth\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"l2MessagingBlocksOffsets\",\"type\":\"bytes\"}],\"internalType\":\"structILineaRollup.FinalizationData\",\"name\":\"_finalizationData\",\"type\":\"tuple\"}],\"name\":\"finalizeCompressedBlocksWithProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"parentStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32[]\",\"name\":\"dataHashes\",\"type\":\"bytes32[]\"},{\"internalType\":\"bytes32\",\"name\":\"dataParentHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"finalBlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lastFinalizedTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"finalTimestamp\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"l1RollingHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"l1RollingHashMessageNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32[]\",\"name\":\"l2MerkleRoots\",\"type\":\"bytes32[]\"},{\"internalType\":\"uint256\",\"name\":\"l2MerkleTreesDepth\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"l2MessagingBlocksOffsets\",\"type\":\"bytes\"}],\"internalType\":\"structILineaRollup.FinalizationData\",\"name\":\"_finalizationData\",\"type\":\"tuple\"}],\"name\":\"finalizeCompressedBlocksWithoutProof\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"}],\"name\":\"getRoleAdmin\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"grantRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"hasRole\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"inboxL2L1MessageStatus\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"messageStatus\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_initialStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"_initialL2BlockNumber\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"_defaultVerifier\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_securityCouncil\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"_operators\",\"type\":\"address[]\"},{\"internalType\":\"uint256\",\"name\":\"_rateLimitPeriodInSeconds\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_rateLimitAmountInWei\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"_genesisTimestamp\",\"type\":\"uint256\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_lastFinalizedShnarf\",\"type\":\"bytes32\"}],\"name\":\"initializeLastFinalizedShnarf\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_messageNumber\",\"type\":\"uint256\"}],\"name\":\"isMessageClaimed\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"_pauseType\",\"type\":\"uint8\"}],\"name\":\"isPaused\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"merkleRoot\",\"type\":\"bytes32\"}],\"name\":\"l2MerkleRootsDepths\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"treeDepth\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"limitInWei\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"nextMessageNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"messageHash\",\"type\":\"bytes32\"}],\"name\":\"outboxL1L2MessageStatus\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"messageStatus\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"_pauseType\",\"type\":\"uint8\"}],\"name\":\"pauseByType\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"pauseType\",\"type\":\"bytes32\"}],\"name\":\"pauseTypeStatuses\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"pauseStatus\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"periodInSeconds\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"renounceRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"resetAmountUsedInPeriod\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"resetRateLimitAmount\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"role\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"revokeRole\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"messageNumber\",\"type\":\"uint256\"}],\"name\":\"rollingHashes\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"rollingHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_fee\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_calldata\",\"type\":\"bytes\"}],\"name\":\"sendMessage\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"sender\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_newVerifierAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"_proofType\",\"type\":\"uint256\"}],\"name\":\"setVerifierAddress\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"stateRootHashes\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"stateRootHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"parentStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"dataParentHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"finalStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"firstBlockInData\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"finalBlockInData\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"snarkHash\",\"type\":\"bytes32\"}],\"internalType\":\"structILineaRollup.SupportingSubmissionData\",\"name\":\"_submissionData\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"_dataEvaluationClaim\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"_kzgCommitment\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"_kzgProof\",\"type\":\"bytes\"}],\"name\":\"submitBlobData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"parentStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"dataParentHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"finalStateRootHash\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"firstBlockInData\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"finalBlockInData\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"snarkHash\",\"type\":\"bytes32\"},{\"internalType\":\"bytes\",\"name\":\"compressedData\",\"type\":\"bytes\"}],\"internalType\":\"structILineaRollup.SubmissionData\",\"name\":\"_submissionData\",\"type\":\"tuple\"}],\"name\":\"submitData\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes4\",\"name\":\"interfaceId\",\"type\":\"bytes4\"}],\"name\":\"supportsInterface\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"systemMigrationBlock\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint8\",\"name\":\"_pauseType\",\"type\":\"uint8\"}],\"name\":\"unPauseByType\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"proofType\",\"type\":\"uint256\"}],\"name\":\"verifiers\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"verifierAddress\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// LineaabiABI is the input ABI used to generate the binding from.
// Deprecated: Use LineaabiMetaData.ABI instead.
var LineaabiABI = LineaabiMetaData.ABI

// Lineaabi is an auto generated Go binding around an Ethereum contract.
type Lineaabi struct {
	LineaabiCaller     // Read-only binding to the contract
	LineaabiTransactor // Write-only binding to the contract
	LineaabiFilterer   // Log filterer for contract events
}

// LineaabiCaller is an auto generated read-only Go binding around an Ethereum contract.
type LineaabiCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LineaabiTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LineaabiTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LineaabiFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LineaabiFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LineaabiSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LineaabiSession struct {
	Contract     *Lineaabi         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// LineaabiCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LineaabiCallerSession struct {
	Contract *LineaabiCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// LineaabiTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LineaabiTransactorSession struct {
	Contract     *LineaabiTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// LineaabiRaw is an auto generated low-level Go binding around an Ethereum contract.
type LineaabiRaw struct {
	Contract *Lineaabi // Generic contract binding to access the raw methods on
}

// LineaabiCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LineaabiCallerRaw struct {
	Contract *LineaabiCaller // Generic read-only contract binding to access the raw methods on
}

// LineaabiTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LineaabiTransactorRaw struct {
	Contract *LineaabiTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLineaabi creates a new instance of Lineaabi, bound to a specific deployed contract.
func NewLineaabi(address common.Address, backend bind.ContractBackend) (*Lineaabi, error) {
	contract, err := bindLineaabi(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Lineaabi{LineaabiCaller: LineaabiCaller{contract: contract}, LineaabiTransactor: LineaabiTransactor{contract: contract}, LineaabiFilterer: LineaabiFilterer{contract: contract}}, nil
}

// NewLineaabiCaller creates a new read-only instance of Lineaabi, bound to a specific deployed contract.
func NewLineaabiCaller(address common.Address, caller bind.ContractCaller) (*LineaabiCaller, error) {
	contract, err := bindLineaabi(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LineaabiCaller{contract: contract}, nil
}

// NewLineaabiTransactor creates a new write-only instance of Lineaabi, bound to a specific deployed contract.
func NewLineaabiTransactor(address common.Address, transactor bind.ContractTransactor) (*LineaabiTransactor, error) {
	contract, err := bindLineaabi(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LineaabiTransactor{contract: contract}, nil
}

// NewLineaabiFilterer creates a new log filterer instance of Lineaabi, bound to a specific deployed contract.
func NewLineaabiFilterer(address common.Address, filterer bind.ContractFilterer) (*LineaabiFilterer, error) {
	contract, err := bindLineaabi(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LineaabiFilterer{contract: contract}, nil
}

// bindLineaabi binds a generic wrapper to an already deployed contract.
func bindLineaabi(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LineaabiMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Lineaabi *LineaabiRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Lineaabi.Contract.LineaabiCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Lineaabi *LineaabiRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Lineaabi.Contract.LineaabiTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Lineaabi *LineaabiRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Lineaabi.Contract.LineaabiTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Lineaabi *LineaabiCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Lineaabi.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Lineaabi *LineaabiTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Lineaabi.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Lineaabi *LineaabiTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Lineaabi.Contract.contract.Transact(opts, method, params...)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) DEFAULTADMINROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "DEFAULT_ADMIN_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Lineaabi.Contract.DEFAULTADMINROLE(&_Lineaabi.CallOpts)
}

// DEFAULTADMINROLE is a free data retrieval call binding the contract method 0xa217fddf.
//
// Solidity: function DEFAULT_ADMIN_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) DEFAULTADMINROLE() ([32]byte, error) {
	return _Lineaabi.Contract.DEFAULTADMINROLE(&_Lineaabi.CallOpts)
}

// GENERALPAUSETYPE is a free data retrieval call binding the contract method 0x6a637967.
//
// Solidity: function GENERAL_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCaller) GENERALPAUSETYPE(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "GENERAL_PAUSE_TYPE")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GENERALPAUSETYPE is a free data retrieval call binding the contract method 0x6a637967.
//
// Solidity: function GENERAL_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiSession) GENERALPAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.GENERALPAUSETYPE(&_Lineaabi.CallOpts)
}

// GENERALPAUSETYPE is a free data retrieval call binding the contract method 0x6a637967.
//
// Solidity: function GENERAL_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) GENERALPAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.GENERALPAUSETYPE(&_Lineaabi.CallOpts)
}

// GENESISSHNARF is a free data retrieval call binding the contract method 0xe97a1e9e.
//
// Solidity: function GENESIS_SHNARF() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) GENESISSHNARF(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "GENESIS_SHNARF")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GENESISSHNARF is a free data retrieval call binding the contract method 0xe97a1e9e.
//
// Solidity: function GENESIS_SHNARF() view returns(bytes32)
func (_Lineaabi *LineaabiSession) GENESISSHNARF() ([32]byte, error) {
	return _Lineaabi.Contract.GENESISSHNARF(&_Lineaabi.CallOpts)
}

// GENESISSHNARF is a free data retrieval call binding the contract method 0xe97a1e9e.
//
// Solidity: function GENESIS_SHNARF() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) GENESISSHNARF() ([32]byte, error) {
	return _Lineaabi.Contract.GENESISSHNARF(&_Lineaabi.CallOpts)
}

// INBOXSTATUSRECEIVED is a free data retrieval call binding the contract method 0x48922ab7.
//
// Solidity: function INBOX_STATUS_RECEIVED() view returns(uint8)
func (_Lineaabi *LineaabiCaller) INBOXSTATUSRECEIVED(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "INBOX_STATUS_RECEIVED")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// INBOXSTATUSRECEIVED is a free data retrieval call binding the contract method 0x48922ab7.
//
// Solidity: function INBOX_STATUS_RECEIVED() view returns(uint8)
func (_Lineaabi *LineaabiSession) INBOXSTATUSRECEIVED() (uint8, error) {
	return _Lineaabi.Contract.INBOXSTATUSRECEIVED(&_Lineaabi.CallOpts)
}

// INBOXSTATUSRECEIVED is a free data retrieval call binding the contract method 0x48922ab7.
//
// Solidity: function INBOX_STATUS_RECEIVED() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) INBOXSTATUSRECEIVED() (uint8, error) {
	return _Lineaabi.Contract.INBOXSTATUSRECEIVED(&_Lineaabi.CallOpts)
}

// INBOXSTATUSUNKNOWN is a free data retrieval call binding the contract method 0x7d1e8c55.
//
// Solidity: function INBOX_STATUS_UNKNOWN() view returns(uint8)
func (_Lineaabi *LineaabiCaller) INBOXSTATUSUNKNOWN(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "INBOX_STATUS_UNKNOWN")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// INBOXSTATUSUNKNOWN is a free data retrieval call binding the contract method 0x7d1e8c55.
//
// Solidity: function INBOX_STATUS_UNKNOWN() view returns(uint8)
func (_Lineaabi *LineaabiSession) INBOXSTATUSUNKNOWN() (uint8, error) {
	return _Lineaabi.Contract.INBOXSTATUSUNKNOWN(&_Lineaabi.CallOpts)
}

// INBOXSTATUSUNKNOWN is a free data retrieval call binding the contract method 0x7d1e8c55.
//
// Solidity: function INBOX_STATUS_UNKNOWN() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) INBOXSTATUSUNKNOWN() (uint8, error) {
	return _Lineaabi.Contract.INBOXSTATUSUNKNOWN(&_Lineaabi.CallOpts)
}

// L1L2PAUSETYPE is a free data retrieval call binding the contract method 0x11314d0f.
//
// Solidity: function L1_L2_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCaller) L1L2PAUSETYPE(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "L1_L2_PAUSE_TYPE")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// L1L2PAUSETYPE is a free data retrieval call binding the contract method 0x11314d0f.
//
// Solidity: function L1_L2_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiSession) L1L2PAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.L1L2PAUSETYPE(&_Lineaabi.CallOpts)
}

// L1L2PAUSETYPE is a free data retrieval call binding the contract method 0x11314d0f.
//
// Solidity: function L1_L2_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) L1L2PAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.L1L2PAUSETYPE(&_Lineaabi.CallOpts)
}

// L2L1PAUSETYPE is a free data retrieval call binding the contract method 0xabd6230d.
//
// Solidity: function L2_L1_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCaller) L2L1PAUSETYPE(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "L2_L1_PAUSE_TYPE")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// L2L1PAUSETYPE is a free data retrieval call binding the contract method 0xabd6230d.
//
// Solidity: function L2_L1_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiSession) L2L1PAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.L2L1PAUSETYPE(&_Lineaabi.CallOpts)
}

// L2L1PAUSETYPE is a free data retrieval call binding the contract method 0xabd6230d.
//
// Solidity: function L2_L1_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) L2L1PAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.L2L1PAUSETYPE(&_Lineaabi.CallOpts)
}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) OPERATORROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "OPERATOR_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiSession) OPERATORROLE() ([32]byte, error) {
	return _Lineaabi.Contract.OPERATORROLE(&_Lineaabi.CallOpts)
}

// OPERATORROLE is a free data retrieval call binding the contract method 0xf5b541a6.
//
// Solidity: function OPERATOR_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) OPERATORROLE() ([32]byte, error) {
	return _Lineaabi.Contract.OPERATORROLE(&_Lineaabi.CallOpts)
}

// OUTBOXSTATUSRECEIVED is a free data retrieval call binding the contract method 0x73bd07b7.
//
// Solidity: function OUTBOX_STATUS_RECEIVED() view returns(uint8)
func (_Lineaabi *LineaabiCaller) OUTBOXSTATUSRECEIVED(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "OUTBOX_STATUS_RECEIVED")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// OUTBOXSTATUSRECEIVED is a free data retrieval call binding the contract method 0x73bd07b7.
//
// Solidity: function OUTBOX_STATUS_RECEIVED() view returns(uint8)
func (_Lineaabi *LineaabiSession) OUTBOXSTATUSRECEIVED() (uint8, error) {
	return _Lineaabi.Contract.OUTBOXSTATUSRECEIVED(&_Lineaabi.CallOpts)
}

// OUTBOXSTATUSRECEIVED is a free data retrieval call binding the contract method 0x73bd07b7.
//
// Solidity: function OUTBOX_STATUS_RECEIVED() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) OUTBOXSTATUSRECEIVED() (uint8, error) {
	return _Lineaabi.Contract.OUTBOXSTATUSRECEIVED(&_Lineaabi.CallOpts)
}

// OUTBOXSTATUSSENT is a free data retrieval call binding the contract method 0x5b7eb4bd.
//
// Solidity: function OUTBOX_STATUS_SENT() view returns(uint8)
func (_Lineaabi *LineaabiCaller) OUTBOXSTATUSSENT(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "OUTBOX_STATUS_SENT")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// OUTBOXSTATUSSENT is a free data retrieval call binding the contract method 0x5b7eb4bd.
//
// Solidity: function OUTBOX_STATUS_SENT() view returns(uint8)
func (_Lineaabi *LineaabiSession) OUTBOXSTATUSSENT() (uint8, error) {
	return _Lineaabi.Contract.OUTBOXSTATUSSENT(&_Lineaabi.CallOpts)
}

// OUTBOXSTATUSSENT is a free data retrieval call binding the contract method 0x5b7eb4bd.
//
// Solidity: function OUTBOX_STATUS_SENT() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) OUTBOXSTATUSSENT() (uint8, error) {
	return _Lineaabi.Contract.OUTBOXSTATUSSENT(&_Lineaabi.CallOpts)
}

// OUTBOXSTATUSUNKNOWN is a free data retrieval call binding the contract method 0x986fcddd.
//
// Solidity: function OUTBOX_STATUS_UNKNOWN() view returns(uint8)
func (_Lineaabi *LineaabiCaller) OUTBOXSTATUSUNKNOWN(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "OUTBOX_STATUS_UNKNOWN")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// OUTBOXSTATUSUNKNOWN is a free data retrieval call binding the contract method 0x986fcddd.
//
// Solidity: function OUTBOX_STATUS_UNKNOWN() view returns(uint8)
func (_Lineaabi *LineaabiSession) OUTBOXSTATUSUNKNOWN() (uint8, error) {
	return _Lineaabi.Contract.OUTBOXSTATUSUNKNOWN(&_Lineaabi.CallOpts)
}

// OUTBOXSTATUSUNKNOWN is a free data retrieval call binding the contract method 0x986fcddd.
//
// Solidity: function OUTBOX_STATUS_UNKNOWN() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) OUTBOXSTATUSUNKNOWN() (uint8, error) {
	return _Lineaabi.Contract.OUTBOXSTATUSUNKNOWN(&_Lineaabi.CallOpts)
}

// PAUSEMANAGERROLE is a free data retrieval call binding the contract method 0xd84f91e8.
//
// Solidity: function PAUSE_MANAGER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) PAUSEMANAGERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "PAUSE_MANAGER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// PAUSEMANAGERROLE is a free data retrieval call binding the contract method 0xd84f91e8.
//
// Solidity: function PAUSE_MANAGER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiSession) PAUSEMANAGERROLE() ([32]byte, error) {
	return _Lineaabi.Contract.PAUSEMANAGERROLE(&_Lineaabi.CallOpts)
}

// PAUSEMANAGERROLE is a free data retrieval call binding the contract method 0xd84f91e8.
//
// Solidity: function PAUSE_MANAGER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) PAUSEMANAGERROLE() ([32]byte, error) {
	return _Lineaabi.Contract.PAUSEMANAGERROLE(&_Lineaabi.CallOpts)
}

// PROVINGSYSTEMPAUSETYPE is a free data retrieval call binding the contract method 0xb4a5a4b7.
//
// Solidity: function PROVING_SYSTEM_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCaller) PROVINGSYSTEMPAUSETYPE(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "PROVING_SYSTEM_PAUSE_TYPE")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// PROVINGSYSTEMPAUSETYPE is a free data retrieval call binding the contract method 0xb4a5a4b7.
//
// Solidity: function PROVING_SYSTEM_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiSession) PROVINGSYSTEMPAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.PROVINGSYSTEMPAUSETYPE(&_Lineaabi.CallOpts)
}

// PROVINGSYSTEMPAUSETYPE is a free data retrieval call binding the contract method 0xb4a5a4b7.
//
// Solidity: function PROVING_SYSTEM_PAUSE_TYPE() view returns(uint8)
func (_Lineaabi *LineaabiCallerSession) PROVINGSYSTEMPAUSETYPE() (uint8, error) {
	return _Lineaabi.Contract.PROVINGSYSTEMPAUSETYPE(&_Lineaabi.CallOpts)
}

// RATELIMITSETTERROLE is a free data retrieval call binding the contract method 0xbf3e7505.
//
// Solidity: function RATE_LIMIT_SETTER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) RATELIMITSETTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "RATE_LIMIT_SETTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RATELIMITSETTERROLE is a free data retrieval call binding the contract method 0xbf3e7505.
//
// Solidity: function RATE_LIMIT_SETTER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiSession) RATELIMITSETTERROLE() ([32]byte, error) {
	return _Lineaabi.Contract.RATELIMITSETTERROLE(&_Lineaabi.CallOpts)
}

// RATELIMITSETTERROLE is a free data retrieval call binding the contract method 0xbf3e7505.
//
// Solidity: function RATE_LIMIT_SETTER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) RATELIMITSETTERROLE() ([32]byte, error) {
	return _Lineaabi.Contract.RATELIMITSETTERROLE(&_Lineaabi.CallOpts)
}

// VERIFIERSETTERROLE is a free data retrieval call binding the contract method 0x6e673843.
//
// Solidity: function VERIFIER_SETTER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) VERIFIERSETTERROLE(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "VERIFIER_SETTER_ROLE")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// VERIFIERSETTERROLE is a free data retrieval call binding the contract method 0x6e673843.
//
// Solidity: function VERIFIER_SETTER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiSession) VERIFIERSETTERROLE() ([32]byte, error) {
	return _Lineaabi.Contract.VERIFIERSETTERROLE(&_Lineaabi.CallOpts)
}

// VERIFIERSETTERROLE is a free data retrieval call binding the contract method 0x6e673843.
//
// Solidity: function VERIFIER_SETTER_ROLE() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) VERIFIERSETTERROLE() ([32]byte, error) {
	return _Lineaabi.Contract.VERIFIERSETTERROLE(&_Lineaabi.CallOpts)
}

// CurrentFinalizedShnarf is a free data retrieval call binding the contract method 0xcd9b9e9a.
//
// Solidity: function currentFinalizedShnarf() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) CurrentFinalizedShnarf(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentFinalizedShnarf")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CurrentFinalizedShnarf is a free data retrieval call binding the contract method 0xcd9b9e9a.
//
// Solidity: function currentFinalizedShnarf() view returns(bytes32)
func (_Lineaabi *LineaabiSession) CurrentFinalizedShnarf() ([32]byte, error) {
	return _Lineaabi.Contract.CurrentFinalizedShnarf(&_Lineaabi.CallOpts)
}

// CurrentFinalizedShnarf is a free data retrieval call binding the contract method 0xcd9b9e9a.
//
// Solidity: function currentFinalizedShnarf() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) CurrentFinalizedShnarf() ([32]byte, error) {
	return _Lineaabi.Contract.CurrentFinalizedShnarf(&_Lineaabi.CallOpts)
}

// CurrentL2BlockNumber is a free data retrieval call binding the contract method 0x695378f5.
//
// Solidity: function currentL2BlockNumber() view returns(uint256)
func (_Lineaabi *LineaabiCaller) CurrentL2BlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentL2BlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentL2BlockNumber is a free data retrieval call binding the contract method 0x695378f5.
//
// Solidity: function currentL2BlockNumber() view returns(uint256)
func (_Lineaabi *LineaabiSession) CurrentL2BlockNumber() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentL2BlockNumber(&_Lineaabi.CallOpts)
}

// CurrentL2BlockNumber is a free data retrieval call binding the contract method 0x695378f5.
//
// Solidity: function currentL2BlockNumber() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) CurrentL2BlockNumber() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentL2BlockNumber(&_Lineaabi.CallOpts)
}

// CurrentL2StoredL1MessageNumber is a free data retrieval call binding the contract method 0x05861180.
//
// Solidity: function currentL2StoredL1MessageNumber() view returns(uint256)
func (_Lineaabi *LineaabiCaller) CurrentL2StoredL1MessageNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentL2StoredL1MessageNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentL2StoredL1MessageNumber is a free data retrieval call binding the contract method 0x05861180.
//
// Solidity: function currentL2StoredL1MessageNumber() view returns(uint256)
func (_Lineaabi *LineaabiSession) CurrentL2StoredL1MessageNumber() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentL2StoredL1MessageNumber(&_Lineaabi.CallOpts)
}

// CurrentL2StoredL1MessageNumber is a free data retrieval call binding the contract method 0x05861180.
//
// Solidity: function currentL2StoredL1MessageNumber() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) CurrentL2StoredL1MessageNumber() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentL2StoredL1MessageNumber(&_Lineaabi.CallOpts)
}

// CurrentL2StoredL1RollingHash is a free data retrieval call binding the contract method 0xd5d4b835.
//
// Solidity: function currentL2StoredL1RollingHash() view returns(bytes32)
func (_Lineaabi *LineaabiCaller) CurrentL2StoredL1RollingHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentL2StoredL1RollingHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// CurrentL2StoredL1RollingHash is a free data retrieval call binding the contract method 0xd5d4b835.
//
// Solidity: function currentL2StoredL1RollingHash() view returns(bytes32)
func (_Lineaabi *LineaabiSession) CurrentL2StoredL1RollingHash() ([32]byte, error) {
	return _Lineaabi.Contract.CurrentL2StoredL1RollingHash(&_Lineaabi.CallOpts)
}

// CurrentL2StoredL1RollingHash is a free data retrieval call binding the contract method 0xd5d4b835.
//
// Solidity: function currentL2StoredL1RollingHash() view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) CurrentL2StoredL1RollingHash() ([32]byte, error) {
	return _Lineaabi.Contract.CurrentL2StoredL1RollingHash(&_Lineaabi.CallOpts)
}

// CurrentPeriodAmountInWei is a free data retrieval call binding the contract method 0xc0729ab1.
//
// Solidity: function currentPeriodAmountInWei() view returns(uint256)
func (_Lineaabi *LineaabiCaller) CurrentPeriodAmountInWei(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentPeriodAmountInWei")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentPeriodAmountInWei is a free data retrieval call binding the contract method 0xc0729ab1.
//
// Solidity: function currentPeriodAmountInWei() view returns(uint256)
func (_Lineaabi *LineaabiSession) CurrentPeriodAmountInWei() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentPeriodAmountInWei(&_Lineaabi.CallOpts)
}

// CurrentPeriodAmountInWei is a free data retrieval call binding the contract method 0xc0729ab1.
//
// Solidity: function currentPeriodAmountInWei() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) CurrentPeriodAmountInWei() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentPeriodAmountInWei(&_Lineaabi.CallOpts)
}

// CurrentPeriodEnd is a free data retrieval call binding the contract method 0x58794456.
//
// Solidity: function currentPeriodEnd() view returns(uint256)
func (_Lineaabi *LineaabiCaller) CurrentPeriodEnd(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentPeriodEnd")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentPeriodEnd is a free data retrieval call binding the contract method 0x58794456.
//
// Solidity: function currentPeriodEnd() view returns(uint256)
func (_Lineaabi *LineaabiSession) CurrentPeriodEnd() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentPeriodEnd(&_Lineaabi.CallOpts)
}

// CurrentPeriodEnd is a free data retrieval call binding the contract method 0x58794456.
//
// Solidity: function currentPeriodEnd() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) CurrentPeriodEnd() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentPeriodEnd(&_Lineaabi.CallOpts)
}

// CurrentTimestamp is a free data retrieval call binding the contract method 0x1e2ff94f.
//
// Solidity: function currentTimestamp() view returns(uint256)
func (_Lineaabi *LineaabiCaller) CurrentTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "currentTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CurrentTimestamp is a free data retrieval call binding the contract method 0x1e2ff94f.
//
// Solidity: function currentTimestamp() view returns(uint256)
func (_Lineaabi *LineaabiSession) CurrentTimestamp() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentTimestamp(&_Lineaabi.CallOpts)
}

// CurrentTimestamp is a free data retrieval call binding the contract method 0x1e2ff94f.
//
// Solidity: function currentTimestamp() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) CurrentTimestamp() (*big.Int, error) {
	return _Lineaabi.Contract.CurrentTimestamp(&_Lineaabi.CallOpts)
}

// DataEndingBlock is a free data retrieval call binding the contract method 0x5ed73ceb.
//
// Solidity: function dataEndingBlock(bytes32 dataHash) view returns(uint256 endingBlock)
func (_Lineaabi *LineaabiCaller) DataEndingBlock(opts *bind.CallOpts, dataHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "dataEndingBlock", dataHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DataEndingBlock is a free data retrieval call binding the contract method 0x5ed73ceb.
//
// Solidity: function dataEndingBlock(bytes32 dataHash) view returns(uint256 endingBlock)
func (_Lineaabi *LineaabiSession) DataEndingBlock(dataHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.DataEndingBlock(&_Lineaabi.CallOpts, dataHash)
}

// DataEndingBlock is a free data retrieval call binding the contract method 0x5ed73ceb.
//
// Solidity: function dataEndingBlock(bytes32 dataHash) view returns(uint256 endingBlock)
func (_Lineaabi *LineaabiCallerSession) DataEndingBlock(dataHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.DataEndingBlock(&_Lineaabi.CallOpts, dataHash)
}

// DataFinalStateRootHashes is a free data retrieval call binding the contract method 0x6078bfd8.
//
// Solidity: function dataFinalStateRootHashes(bytes32 dataHash) view returns(bytes32 finalStateRootHash)
func (_Lineaabi *LineaabiCaller) DataFinalStateRootHashes(opts *bind.CallOpts, dataHash [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "dataFinalStateRootHashes", dataHash)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DataFinalStateRootHashes is a free data retrieval call binding the contract method 0x6078bfd8.
//
// Solidity: function dataFinalStateRootHashes(bytes32 dataHash) view returns(bytes32 finalStateRootHash)
func (_Lineaabi *LineaabiSession) DataFinalStateRootHashes(dataHash [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.DataFinalStateRootHashes(&_Lineaabi.CallOpts, dataHash)
}

// DataFinalStateRootHashes is a free data retrieval call binding the contract method 0x6078bfd8.
//
// Solidity: function dataFinalStateRootHashes(bytes32 dataHash) view returns(bytes32 finalStateRootHash)
func (_Lineaabi *LineaabiCallerSession) DataFinalStateRootHashes(dataHash [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.DataFinalStateRootHashes(&_Lineaabi.CallOpts, dataHash)
}

// DataParents is a free data retrieval call binding the contract method 0x4cdd389b.
//
// Solidity: function dataParents(bytes32 dataHash) view returns(bytes32 parentHash)
func (_Lineaabi *LineaabiCaller) DataParents(opts *bind.CallOpts, dataHash [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "dataParents", dataHash)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DataParents is a free data retrieval call binding the contract method 0x4cdd389b.
//
// Solidity: function dataParents(bytes32 dataHash) view returns(bytes32 parentHash)
func (_Lineaabi *LineaabiSession) DataParents(dataHash [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.DataParents(&_Lineaabi.CallOpts, dataHash)
}

// DataParents is a free data retrieval call binding the contract method 0x4cdd389b.
//
// Solidity: function dataParents(bytes32 dataHash) view returns(bytes32 parentHash)
func (_Lineaabi *LineaabiCallerSession) DataParents(dataHash [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.DataParents(&_Lineaabi.CallOpts, dataHash)
}

// DataShnarfHashes is a free data retrieval call binding the contract method 0x66f96e98.
//
// Solidity: function dataShnarfHashes(bytes32 dataHash) view returns(bytes32 shnarfHash)
func (_Lineaabi *LineaabiCaller) DataShnarfHashes(opts *bind.CallOpts, dataHash [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "dataShnarfHashes", dataHash)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// DataShnarfHashes is a free data retrieval call binding the contract method 0x66f96e98.
//
// Solidity: function dataShnarfHashes(bytes32 dataHash) view returns(bytes32 shnarfHash)
func (_Lineaabi *LineaabiSession) DataShnarfHashes(dataHash [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.DataShnarfHashes(&_Lineaabi.CallOpts, dataHash)
}

// DataShnarfHashes is a free data retrieval call binding the contract method 0x66f96e98.
//
// Solidity: function dataShnarfHashes(bytes32 dataHash) view returns(bytes32 shnarfHash)
func (_Lineaabi *LineaabiCallerSession) DataShnarfHashes(dataHash [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.DataShnarfHashes(&_Lineaabi.CallOpts, dataHash)
}

// DataStartingBlock is a free data retrieval call binding the contract method 0x1f443da0.
//
// Solidity: function dataStartingBlock(bytes32 dataHash) view returns(uint256 startingBlock)
func (_Lineaabi *LineaabiCaller) DataStartingBlock(opts *bind.CallOpts, dataHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "dataStartingBlock", dataHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DataStartingBlock is a free data retrieval call binding the contract method 0x1f443da0.
//
// Solidity: function dataStartingBlock(bytes32 dataHash) view returns(uint256 startingBlock)
func (_Lineaabi *LineaabiSession) DataStartingBlock(dataHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.DataStartingBlock(&_Lineaabi.CallOpts, dataHash)
}

// DataStartingBlock is a free data retrieval call binding the contract method 0x1f443da0.
//
// Solidity: function dataStartingBlock(bytes32 dataHash) view returns(uint256 startingBlock)
func (_Lineaabi *LineaabiCallerSession) DataStartingBlock(dataHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.DataStartingBlock(&_Lineaabi.CallOpts, dataHash)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Lineaabi *LineaabiCaller) GetRoleAdmin(opts *bind.CallOpts, role [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "getRoleAdmin", role)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Lineaabi *LineaabiSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.GetRoleAdmin(&_Lineaabi.CallOpts, role)
}

// GetRoleAdmin is a free data retrieval call binding the contract method 0x248a9ca3.
//
// Solidity: function getRoleAdmin(bytes32 role) view returns(bytes32)
func (_Lineaabi *LineaabiCallerSession) GetRoleAdmin(role [32]byte) ([32]byte, error) {
	return _Lineaabi.Contract.GetRoleAdmin(&_Lineaabi.CallOpts, role)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Lineaabi *LineaabiCaller) HasRole(opts *bind.CallOpts, role [32]byte, account common.Address) (bool, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "hasRole", role, account)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Lineaabi *LineaabiSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Lineaabi.Contract.HasRole(&_Lineaabi.CallOpts, role, account)
}

// HasRole is a free data retrieval call binding the contract method 0x91d14854.
//
// Solidity: function hasRole(bytes32 role, address account) view returns(bool)
func (_Lineaabi *LineaabiCallerSession) HasRole(role [32]byte, account common.Address) (bool, error) {
	return _Lineaabi.Contract.HasRole(&_Lineaabi.CallOpts, role, account)
}

// InboxL2L1MessageStatus is a free data retrieval call binding the contract method 0x5c721a0c.
//
// Solidity: function inboxL2L1MessageStatus(bytes32 messageHash) view returns(uint256 messageStatus)
func (_Lineaabi *LineaabiCaller) InboxL2L1MessageStatus(opts *bind.CallOpts, messageHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "inboxL2L1MessageStatus", messageHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// InboxL2L1MessageStatus is a free data retrieval call binding the contract method 0x5c721a0c.
//
// Solidity: function inboxL2L1MessageStatus(bytes32 messageHash) view returns(uint256 messageStatus)
func (_Lineaabi *LineaabiSession) InboxL2L1MessageStatus(messageHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.InboxL2L1MessageStatus(&_Lineaabi.CallOpts, messageHash)
}

// InboxL2L1MessageStatus is a free data retrieval call binding the contract method 0x5c721a0c.
//
// Solidity: function inboxL2L1MessageStatus(bytes32 messageHash) view returns(uint256 messageStatus)
func (_Lineaabi *LineaabiCallerSession) InboxL2L1MessageStatus(messageHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.InboxL2L1MessageStatus(&_Lineaabi.CallOpts, messageHash)
}

// IsMessageClaimed is a free data retrieval call binding the contract method 0x9ee8b211.
//
// Solidity: function isMessageClaimed(uint256 _messageNumber) view returns(bool)
func (_Lineaabi *LineaabiCaller) IsMessageClaimed(opts *bind.CallOpts, _messageNumber *big.Int) (bool, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "isMessageClaimed", _messageNumber)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsMessageClaimed is a free data retrieval call binding the contract method 0x9ee8b211.
//
// Solidity: function isMessageClaimed(uint256 _messageNumber) view returns(bool)
func (_Lineaabi *LineaabiSession) IsMessageClaimed(_messageNumber *big.Int) (bool, error) {
	return _Lineaabi.Contract.IsMessageClaimed(&_Lineaabi.CallOpts, _messageNumber)
}

// IsMessageClaimed is a free data retrieval call binding the contract method 0x9ee8b211.
//
// Solidity: function isMessageClaimed(uint256 _messageNumber) view returns(bool)
func (_Lineaabi *LineaabiCallerSession) IsMessageClaimed(_messageNumber *big.Int) (bool, error) {
	return _Lineaabi.Contract.IsMessageClaimed(&_Lineaabi.CallOpts, _messageNumber)
}

// IsPaused is a free data retrieval call binding the contract method 0xbc61e733.
//
// Solidity: function isPaused(uint8 _pauseType) view returns(bool)
func (_Lineaabi *LineaabiCaller) IsPaused(opts *bind.CallOpts, _pauseType uint8) (bool, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "isPaused", _pauseType)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsPaused is a free data retrieval call binding the contract method 0xbc61e733.
//
// Solidity: function isPaused(uint8 _pauseType) view returns(bool)
func (_Lineaabi *LineaabiSession) IsPaused(_pauseType uint8) (bool, error) {
	return _Lineaabi.Contract.IsPaused(&_Lineaabi.CallOpts, _pauseType)
}

// IsPaused is a free data retrieval call binding the contract method 0xbc61e733.
//
// Solidity: function isPaused(uint8 _pauseType) view returns(bool)
func (_Lineaabi *LineaabiCallerSession) IsPaused(_pauseType uint8) (bool, error) {
	return _Lineaabi.Contract.IsPaused(&_Lineaabi.CallOpts, _pauseType)
}

// L2MerkleRootsDepths is a free data retrieval call binding the contract method 0x60e83cf3.
//
// Solidity: function l2MerkleRootsDepths(bytes32 merkleRoot) view returns(uint256 treeDepth)
func (_Lineaabi *LineaabiCaller) L2MerkleRootsDepths(opts *bind.CallOpts, merkleRoot [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "l2MerkleRootsDepths", merkleRoot)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// L2MerkleRootsDepths is a free data retrieval call binding the contract method 0x60e83cf3.
//
// Solidity: function l2MerkleRootsDepths(bytes32 merkleRoot) view returns(uint256 treeDepth)
func (_Lineaabi *LineaabiSession) L2MerkleRootsDepths(merkleRoot [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.L2MerkleRootsDepths(&_Lineaabi.CallOpts, merkleRoot)
}

// L2MerkleRootsDepths is a free data retrieval call binding the contract method 0x60e83cf3.
//
// Solidity: function l2MerkleRootsDepths(bytes32 merkleRoot) view returns(uint256 treeDepth)
func (_Lineaabi *LineaabiCallerSession) L2MerkleRootsDepths(merkleRoot [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.L2MerkleRootsDepths(&_Lineaabi.CallOpts, merkleRoot)
}

// LimitInWei is a free data retrieval call binding the contract method 0xad422ff0.
//
// Solidity: function limitInWei() view returns(uint256)
func (_Lineaabi *LineaabiCaller) LimitInWei(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "limitInWei")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// LimitInWei is a free data retrieval call binding the contract method 0xad422ff0.
//
// Solidity: function limitInWei() view returns(uint256)
func (_Lineaabi *LineaabiSession) LimitInWei() (*big.Int, error) {
	return _Lineaabi.Contract.LimitInWei(&_Lineaabi.CallOpts)
}

// LimitInWei is a free data retrieval call binding the contract method 0xad422ff0.
//
// Solidity: function limitInWei() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) LimitInWei() (*big.Int, error) {
	return _Lineaabi.Contract.LimitInWei(&_Lineaabi.CallOpts)
}

// NextMessageNumber is a free data retrieval call binding the contract method 0xb837dbe9.
//
// Solidity: function nextMessageNumber() view returns(uint256)
func (_Lineaabi *LineaabiCaller) NextMessageNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "nextMessageNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NextMessageNumber is a free data retrieval call binding the contract method 0xb837dbe9.
//
// Solidity: function nextMessageNumber() view returns(uint256)
func (_Lineaabi *LineaabiSession) NextMessageNumber() (*big.Int, error) {
	return _Lineaabi.Contract.NextMessageNumber(&_Lineaabi.CallOpts)
}

// NextMessageNumber is a free data retrieval call binding the contract method 0xb837dbe9.
//
// Solidity: function nextMessageNumber() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) NextMessageNumber() (*big.Int, error) {
	return _Lineaabi.Contract.NextMessageNumber(&_Lineaabi.CallOpts)
}

// OutboxL1L2MessageStatus is a free data retrieval call binding the contract method 0x3fc08b65.
//
// Solidity: function outboxL1L2MessageStatus(bytes32 messageHash) view returns(uint256 messageStatus)
func (_Lineaabi *LineaabiCaller) OutboxL1L2MessageStatus(opts *bind.CallOpts, messageHash [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "outboxL1L2MessageStatus", messageHash)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// OutboxL1L2MessageStatus is a free data retrieval call binding the contract method 0x3fc08b65.
//
// Solidity: function outboxL1L2MessageStatus(bytes32 messageHash) view returns(uint256 messageStatus)
func (_Lineaabi *LineaabiSession) OutboxL1L2MessageStatus(messageHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.OutboxL1L2MessageStatus(&_Lineaabi.CallOpts, messageHash)
}

// OutboxL1L2MessageStatus is a free data retrieval call binding the contract method 0x3fc08b65.
//
// Solidity: function outboxL1L2MessageStatus(bytes32 messageHash) view returns(uint256 messageStatus)
func (_Lineaabi *LineaabiCallerSession) OutboxL1L2MessageStatus(messageHash [32]byte) (*big.Int, error) {
	return _Lineaabi.Contract.OutboxL1L2MessageStatus(&_Lineaabi.CallOpts, messageHash)
}

// PauseTypeStatuses is a free data retrieval call binding the contract method 0xcc5782f6.
//
// Solidity: function pauseTypeStatuses(bytes32 pauseType) view returns(bool pauseStatus)
func (_Lineaabi *LineaabiCaller) PauseTypeStatuses(opts *bind.CallOpts, pauseType [32]byte) (bool, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "pauseTypeStatuses", pauseType)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// PauseTypeStatuses is a free data retrieval call binding the contract method 0xcc5782f6.
//
// Solidity: function pauseTypeStatuses(bytes32 pauseType) view returns(bool pauseStatus)
func (_Lineaabi *LineaabiSession) PauseTypeStatuses(pauseType [32]byte) (bool, error) {
	return _Lineaabi.Contract.PauseTypeStatuses(&_Lineaabi.CallOpts, pauseType)
}

// PauseTypeStatuses is a free data retrieval call binding the contract method 0xcc5782f6.
//
// Solidity: function pauseTypeStatuses(bytes32 pauseType) view returns(bool pauseStatus)
func (_Lineaabi *LineaabiCallerSession) PauseTypeStatuses(pauseType [32]byte) (bool, error) {
	return _Lineaabi.Contract.PauseTypeStatuses(&_Lineaabi.CallOpts, pauseType)
}

// PeriodInSeconds is a free data retrieval call binding the contract method 0xc1dc0f07.
//
// Solidity: function periodInSeconds() view returns(uint256)
func (_Lineaabi *LineaabiCaller) PeriodInSeconds(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "periodInSeconds")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PeriodInSeconds is a free data retrieval call binding the contract method 0xc1dc0f07.
//
// Solidity: function periodInSeconds() view returns(uint256)
func (_Lineaabi *LineaabiSession) PeriodInSeconds() (*big.Int, error) {
	return _Lineaabi.Contract.PeriodInSeconds(&_Lineaabi.CallOpts)
}

// PeriodInSeconds is a free data retrieval call binding the contract method 0xc1dc0f07.
//
// Solidity: function periodInSeconds() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) PeriodInSeconds() (*big.Int, error) {
	return _Lineaabi.Contract.PeriodInSeconds(&_Lineaabi.CallOpts)
}

// RollingHashes is a free data retrieval call binding the contract method 0x914e57eb.
//
// Solidity: function rollingHashes(uint256 messageNumber) view returns(bytes32 rollingHash)
func (_Lineaabi *LineaabiCaller) RollingHashes(opts *bind.CallOpts, messageNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "rollingHashes", messageNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// RollingHashes is a free data retrieval call binding the contract method 0x914e57eb.
//
// Solidity: function rollingHashes(uint256 messageNumber) view returns(bytes32 rollingHash)
func (_Lineaabi *LineaabiSession) RollingHashes(messageNumber *big.Int) ([32]byte, error) {
	return _Lineaabi.Contract.RollingHashes(&_Lineaabi.CallOpts, messageNumber)
}

// RollingHashes is a free data retrieval call binding the contract method 0x914e57eb.
//
// Solidity: function rollingHashes(uint256 messageNumber) view returns(bytes32 rollingHash)
func (_Lineaabi *LineaabiCallerSession) RollingHashes(messageNumber *big.Int) ([32]byte, error) {
	return _Lineaabi.Contract.RollingHashes(&_Lineaabi.CallOpts, messageNumber)
}

// Sender is a free data retrieval call binding the contract method 0x67e404ce.
//
// Solidity: function sender() view returns(address)
func (_Lineaabi *LineaabiCaller) Sender(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "sender")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Sender is a free data retrieval call binding the contract method 0x67e404ce.
//
// Solidity: function sender() view returns(address)
func (_Lineaabi *LineaabiSession) Sender() (common.Address, error) {
	return _Lineaabi.Contract.Sender(&_Lineaabi.CallOpts)
}

// Sender is a free data retrieval call binding the contract method 0x67e404ce.
//
// Solidity: function sender() view returns(address)
func (_Lineaabi *LineaabiCallerSession) Sender() (common.Address, error) {
	return _Lineaabi.Contract.Sender(&_Lineaabi.CallOpts)
}

// StateRootHashes is a free data retrieval call binding the contract method 0x8be745d1.
//
// Solidity: function stateRootHashes(uint256 blockNumber) view returns(bytes32 stateRootHash)
func (_Lineaabi *LineaabiCaller) StateRootHashes(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "stateRootHashes", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// StateRootHashes is a free data retrieval call binding the contract method 0x8be745d1.
//
// Solidity: function stateRootHashes(uint256 blockNumber) view returns(bytes32 stateRootHash)
func (_Lineaabi *LineaabiSession) StateRootHashes(blockNumber *big.Int) ([32]byte, error) {
	return _Lineaabi.Contract.StateRootHashes(&_Lineaabi.CallOpts, blockNumber)
}

// StateRootHashes is a free data retrieval call binding the contract method 0x8be745d1.
//
// Solidity: function stateRootHashes(uint256 blockNumber) view returns(bytes32 stateRootHash)
func (_Lineaabi *LineaabiCallerSession) StateRootHashes(blockNumber *big.Int) ([32]byte, error) {
	return _Lineaabi.Contract.StateRootHashes(&_Lineaabi.CallOpts, blockNumber)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Lineaabi *LineaabiCaller) SupportsInterface(opts *bind.CallOpts, interfaceId [4]byte) (bool, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "supportsInterface", interfaceId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Lineaabi *LineaabiSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Lineaabi.Contract.SupportsInterface(&_Lineaabi.CallOpts, interfaceId)
}

// SupportsInterface is a free data retrieval call binding the contract method 0x01ffc9a7.
//
// Solidity: function supportsInterface(bytes4 interfaceId) view returns(bool)
func (_Lineaabi *LineaabiCallerSession) SupportsInterface(interfaceId [4]byte) (bool, error) {
	return _Lineaabi.Contract.SupportsInterface(&_Lineaabi.CallOpts, interfaceId)
}

// SystemMigrationBlock is a free data retrieval call binding the contract method 0x2c70645c.
//
// Solidity: function systemMigrationBlock() view returns(uint256)
func (_Lineaabi *LineaabiCaller) SystemMigrationBlock(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "systemMigrationBlock")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SystemMigrationBlock is a free data retrieval call binding the contract method 0x2c70645c.
//
// Solidity: function systemMigrationBlock() view returns(uint256)
func (_Lineaabi *LineaabiSession) SystemMigrationBlock() (*big.Int, error) {
	return _Lineaabi.Contract.SystemMigrationBlock(&_Lineaabi.CallOpts)
}

// SystemMigrationBlock is a free data retrieval call binding the contract method 0x2c70645c.
//
// Solidity: function systemMigrationBlock() view returns(uint256)
func (_Lineaabi *LineaabiCallerSession) SystemMigrationBlock() (*big.Int, error) {
	return _Lineaabi.Contract.SystemMigrationBlock(&_Lineaabi.CallOpts)
}

// Verifiers is a free data retrieval call binding the contract method 0xac1eff68.
//
// Solidity: function verifiers(uint256 proofType) view returns(address verifierAddress)
func (_Lineaabi *LineaabiCaller) Verifiers(opts *bind.CallOpts, proofType *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Lineaabi.contract.Call(opts, &out, "verifiers", proofType)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Verifiers is a free data retrieval call binding the contract method 0xac1eff68.
//
// Solidity: function verifiers(uint256 proofType) view returns(address verifierAddress)
func (_Lineaabi *LineaabiSession) Verifiers(proofType *big.Int) (common.Address, error) {
	return _Lineaabi.Contract.Verifiers(&_Lineaabi.CallOpts, proofType)
}

// Verifiers is a free data retrieval call binding the contract method 0xac1eff68.
//
// Solidity: function verifiers(uint256 proofType) view returns(address verifierAddress)
func (_Lineaabi *LineaabiCallerSession) Verifiers(proofType *big.Int) (common.Address, error) {
	return _Lineaabi.Contract.Verifiers(&_Lineaabi.CallOpts, proofType)
}

// ClaimMessage is a paid mutator transaction binding the contract method 0x491e0936.
//
// Solidity: function claimMessage(address _from, address _to, uint256 _fee, uint256 _value, address _feeRecipient, bytes _calldata, uint256 _nonce) returns()
func (_Lineaabi *LineaabiTransactor) ClaimMessage(opts *bind.TransactOpts, _from common.Address, _to common.Address, _fee *big.Int, _value *big.Int, _feeRecipient common.Address, _calldata []byte, _nonce *big.Int) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "claimMessage", _from, _to, _fee, _value, _feeRecipient, _calldata, _nonce)
}

// ClaimMessage is a paid mutator transaction binding the contract method 0x491e0936.
//
// Solidity: function claimMessage(address _from, address _to, uint256 _fee, uint256 _value, address _feeRecipient, bytes _calldata, uint256 _nonce) returns()
func (_Lineaabi *LineaabiSession) ClaimMessage(_from common.Address, _to common.Address, _fee *big.Int, _value *big.Int, _feeRecipient common.Address, _calldata []byte, _nonce *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.ClaimMessage(&_Lineaabi.TransactOpts, _from, _to, _fee, _value, _feeRecipient, _calldata, _nonce)
}

// ClaimMessage is a paid mutator transaction binding the contract method 0x491e0936.
//
// Solidity: function claimMessage(address _from, address _to, uint256 _fee, uint256 _value, address _feeRecipient, bytes _calldata, uint256 _nonce) returns()
func (_Lineaabi *LineaabiTransactorSession) ClaimMessage(_from common.Address, _to common.Address, _fee *big.Int, _value *big.Int, _feeRecipient common.Address, _calldata []byte, _nonce *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.ClaimMessage(&_Lineaabi.TransactOpts, _from, _to, _fee, _value, _feeRecipient, _calldata, _nonce)
}

// ClaimMessageWithProof is a paid mutator transaction binding the contract method 0x6463fb2a.
//
// Solidity: function claimMessageWithProof((bytes32[],uint256,uint32,address,address,uint256,uint256,address,bytes32,bytes) _params) returns()
func (_Lineaabi *LineaabiTransactor) ClaimMessageWithProof(opts *bind.TransactOpts, _params IL1MessageServiceClaimMessageWithProofParams) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "claimMessageWithProof", _params)
}

// ClaimMessageWithProof is a paid mutator transaction binding the contract method 0x6463fb2a.
//
// Solidity: function claimMessageWithProof((bytes32[],uint256,uint32,address,address,uint256,uint256,address,bytes32,bytes) _params) returns()
func (_Lineaabi *LineaabiSession) ClaimMessageWithProof(_params IL1MessageServiceClaimMessageWithProofParams) (*types.Transaction, error) {
	return _Lineaabi.Contract.ClaimMessageWithProof(&_Lineaabi.TransactOpts, _params)
}

// ClaimMessageWithProof is a paid mutator transaction binding the contract method 0x6463fb2a.
//
// Solidity: function claimMessageWithProof((bytes32[],uint256,uint32,address,address,uint256,uint256,address,bytes32,bytes) _params) returns()
func (_Lineaabi *LineaabiTransactorSession) ClaimMessageWithProof(_params IL1MessageServiceClaimMessageWithProofParams) (*types.Transaction, error) {
	return _Lineaabi.Contract.ClaimMessageWithProof(&_Lineaabi.TransactOpts, _params)
}

// FinalizeCompressedBlocksWithProof is a paid mutator transaction binding the contract method 0xd630280f.
//
// Solidity: function finalizeCompressedBlocksWithProof(bytes _aggregatedProof, uint256 _proofType, (bytes32,bytes32[],bytes32,uint256,uint256,uint256,bytes32,uint256,bytes32[],uint256,bytes) _finalizationData) returns()
func (_Lineaabi *LineaabiTransactor) FinalizeCompressedBlocksWithProof(opts *bind.TransactOpts, _aggregatedProof []byte, _proofType *big.Int, _finalizationData ILineaRollupFinalizationData) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "finalizeCompressedBlocksWithProof", _aggregatedProof, _proofType, _finalizationData)
}

// FinalizeCompressedBlocksWithProof is a paid mutator transaction binding the contract method 0xd630280f.
//
// Solidity: function finalizeCompressedBlocksWithProof(bytes _aggregatedProof, uint256 _proofType, (bytes32,bytes32[],bytes32,uint256,uint256,uint256,bytes32,uint256,bytes32[],uint256,bytes) _finalizationData) returns()
func (_Lineaabi *LineaabiSession) FinalizeCompressedBlocksWithProof(_aggregatedProof []byte, _proofType *big.Int, _finalizationData ILineaRollupFinalizationData) (*types.Transaction, error) {
	return _Lineaabi.Contract.FinalizeCompressedBlocksWithProof(&_Lineaabi.TransactOpts, _aggregatedProof, _proofType, _finalizationData)
}

// FinalizeCompressedBlocksWithProof is a paid mutator transaction binding the contract method 0xd630280f.
//
// Solidity: function finalizeCompressedBlocksWithProof(bytes _aggregatedProof, uint256 _proofType, (bytes32,bytes32[],bytes32,uint256,uint256,uint256,bytes32,uint256,bytes32[],uint256,bytes) _finalizationData) returns()
func (_Lineaabi *LineaabiTransactorSession) FinalizeCompressedBlocksWithProof(_aggregatedProof []byte, _proofType *big.Int, _finalizationData ILineaRollupFinalizationData) (*types.Transaction, error) {
	return _Lineaabi.Contract.FinalizeCompressedBlocksWithProof(&_Lineaabi.TransactOpts, _aggregatedProof, _proofType, _finalizationData)
}

// FinalizeCompressedBlocksWithoutProof is a paid mutator transaction binding the contract method 0xf9f48284.
//
// Solidity: function finalizeCompressedBlocksWithoutProof((bytes32,bytes32[],bytes32,uint256,uint256,uint256,bytes32,uint256,bytes32[],uint256,bytes) _finalizationData) returns()
func (_Lineaabi *LineaabiTransactor) FinalizeCompressedBlocksWithoutProof(opts *bind.TransactOpts, _finalizationData ILineaRollupFinalizationData) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "finalizeCompressedBlocksWithoutProof", _finalizationData)
}

// FinalizeCompressedBlocksWithoutProof is a paid mutator transaction binding the contract method 0xf9f48284.
//
// Solidity: function finalizeCompressedBlocksWithoutProof((bytes32,bytes32[],bytes32,uint256,uint256,uint256,bytes32,uint256,bytes32[],uint256,bytes) _finalizationData) returns()
func (_Lineaabi *LineaabiSession) FinalizeCompressedBlocksWithoutProof(_finalizationData ILineaRollupFinalizationData) (*types.Transaction, error) {
	return _Lineaabi.Contract.FinalizeCompressedBlocksWithoutProof(&_Lineaabi.TransactOpts, _finalizationData)
}

// FinalizeCompressedBlocksWithoutProof is a paid mutator transaction binding the contract method 0xf9f48284.
//
// Solidity: function finalizeCompressedBlocksWithoutProof((bytes32,bytes32[],bytes32,uint256,uint256,uint256,bytes32,uint256,bytes32[],uint256,bytes) _finalizationData) returns()
func (_Lineaabi *LineaabiTransactorSession) FinalizeCompressedBlocksWithoutProof(_finalizationData ILineaRollupFinalizationData) (*types.Transaction, error) {
	return _Lineaabi.Contract.FinalizeCompressedBlocksWithoutProof(&_Lineaabi.TransactOpts, _finalizationData)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiTransactor) GrantRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "grantRole", role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.Contract.GrantRole(&_Lineaabi.TransactOpts, role, account)
}

// GrantRole is a paid mutator transaction binding the contract method 0x2f2ff15d.
//
// Solidity: function grantRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiTransactorSession) GrantRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.Contract.GrantRole(&_Lineaabi.TransactOpts, role, account)
}

// Initialize is a paid mutator transaction binding the contract method 0x5355420e.
//
// Solidity: function initialize(bytes32 _initialStateRootHash, uint256 _initialL2BlockNumber, address _defaultVerifier, address _securityCouncil, address[] _operators, uint256 _rateLimitPeriodInSeconds, uint256 _rateLimitAmountInWei, uint256 _genesisTimestamp) returns()
func (_Lineaabi *LineaabiTransactor) Initialize(opts *bind.TransactOpts, _initialStateRootHash [32]byte, _initialL2BlockNumber *big.Int, _defaultVerifier common.Address, _securityCouncil common.Address, _operators []common.Address, _rateLimitPeriodInSeconds *big.Int, _rateLimitAmountInWei *big.Int, _genesisTimestamp *big.Int) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "initialize", _initialStateRootHash, _initialL2BlockNumber, _defaultVerifier, _securityCouncil, _operators, _rateLimitPeriodInSeconds, _rateLimitAmountInWei, _genesisTimestamp)
}

// Initialize is a paid mutator transaction binding the contract method 0x5355420e.
//
// Solidity: function initialize(bytes32 _initialStateRootHash, uint256 _initialL2BlockNumber, address _defaultVerifier, address _securityCouncil, address[] _operators, uint256 _rateLimitPeriodInSeconds, uint256 _rateLimitAmountInWei, uint256 _genesisTimestamp) returns()
func (_Lineaabi *LineaabiSession) Initialize(_initialStateRootHash [32]byte, _initialL2BlockNumber *big.Int, _defaultVerifier common.Address, _securityCouncil common.Address, _operators []common.Address, _rateLimitPeriodInSeconds *big.Int, _rateLimitAmountInWei *big.Int, _genesisTimestamp *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.Initialize(&_Lineaabi.TransactOpts, _initialStateRootHash, _initialL2BlockNumber, _defaultVerifier, _securityCouncil, _operators, _rateLimitPeriodInSeconds, _rateLimitAmountInWei, _genesisTimestamp)
}

// Initialize is a paid mutator transaction binding the contract method 0x5355420e.
//
// Solidity: function initialize(bytes32 _initialStateRootHash, uint256 _initialL2BlockNumber, address _defaultVerifier, address _securityCouncil, address[] _operators, uint256 _rateLimitPeriodInSeconds, uint256 _rateLimitAmountInWei, uint256 _genesisTimestamp) returns()
func (_Lineaabi *LineaabiTransactorSession) Initialize(_initialStateRootHash [32]byte, _initialL2BlockNumber *big.Int, _defaultVerifier common.Address, _securityCouncil common.Address, _operators []common.Address, _rateLimitPeriodInSeconds *big.Int, _rateLimitAmountInWei *big.Int, _genesisTimestamp *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.Initialize(&_Lineaabi.TransactOpts, _initialStateRootHash, _initialL2BlockNumber, _defaultVerifier, _securityCouncil, _operators, _rateLimitPeriodInSeconds, _rateLimitAmountInWei, _genesisTimestamp)
}

// InitializeLastFinalizedShnarf is a paid mutator transaction binding the contract method 0x3631b669.
//
// Solidity: function initializeLastFinalizedShnarf(bytes32 _lastFinalizedShnarf) returns()
func (_Lineaabi *LineaabiTransactor) InitializeLastFinalizedShnarf(opts *bind.TransactOpts, _lastFinalizedShnarf [32]byte) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "initializeLastFinalizedShnarf", _lastFinalizedShnarf)
}

// InitializeLastFinalizedShnarf is a paid mutator transaction binding the contract method 0x3631b669.
//
// Solidity: function initializeLastFinalizedShnarf(bytes32 _lastFinalizedShnarf) returns()
func (_Lineaabi *LineaabiSession) InitializeLastFinalizedShnarf(_lastFinalizedShnarf [32]byte) (*types.Transaction, error) {
	return _Lineaabi.Contract.InitializeLastFinalizedShnarf(&_Lineaabi.TransactOpts, _lastFinalizedShnarf)
}

// InitializeLastFinalizedShnarf is a paid mutator transaction binding the contract method 0x3631b669.
//
// Solidity: function initializeLastFinalizedShnarf(bytes32 _lastFinalizedShnarf) returns()
func (_Lineaabi *LineaabiTransactorSession) InitializeLastFinalizedShnarf(_lastFinalizedShnarf [32]byte) (*types.Transaction, error) {
	return _Lineaabi.Contract.InitializeLastFinalizedShnarf(&_Lineaabi.TransactOpts, _lastFinalizedShnarf)
}

// PauseByType is a paid mutator transaction binding the contract method 0xe196fb5d.
//
// Solidity: function pauseByType(uint8 _pauseType) returns()
func (_Lineaabi *LineaabiTransactor) PauseByType(opts *bind.TransactOpts, _pauseType uint8) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "pauseByType", _pauseType)
}

// PauseByType is a paid mutator transaction binding the contract method 0xe196fb5d.
//
// Solidity: function pauseByType(uint8 _pauseType) returns()
func (_Lineaabi *LineaabiSession) PauseByType(_pauseType uint8) (*types.Transaction, error) {
	return _Lineaabi.Contract.PauseByType(&_Lineaabi.TransactOpts, _pauseType)
}

// PauseByType is a paid mutator transaction binding the contract method 0xe196fb5d.
//
// Solidity: function pauseByType(uint8 _pauseType) returns()
func (_Lineaabi *LineaabiTransactorSession) PauseByType(_pauseType uint8) (*types.Transaction, error) {
	return _Lineaabi.Contract.PauseByType(&_Lineaabi.TransactOpts, _pauseType)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiTransactor) RenounceRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "renounceRole", role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.Contract.RenounceRole(&_Lineaabi.TransactOpts, role, account)
}

// RenounceRole is a paid mutator transaction binding the contract method 0x36568abe.
//
// Solidity: function renounceRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiTransactorSession) RenounceRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.Contract.RenounceRole(&_Lineaabi.TransactOpts, role, account)
}

// ResetAmountUsedInPeriod is a paid mutator transaction binding the contract method 0xaea4f745.
//
// Solidity: function resetAmountUsedInPeriod() returns()
func (_Lineaabi *LineaabiTransactor) ResetAmountUsedInPeriod(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "resetAmountUsedInPeriod")
}

// ResetAmountUsedInPeriod is a paid mutator transaction binding the contract method 0xaea4f745.
//
// Solidity: function resetAmountUsedInPeriod() returns()
func (_Lineaabi *LineaabiSession) ResetAmountUsedInPeriod() (*types.Transaction, error) {
	return _Lineaabi.Contract.ResetAmountUsedInPeriod(&_Lineaabi.TransactOpts)
}

// ResetAmountUsedInPeriod is a paid mutator transaction binding the contract method 0xaea4f745.
//
// Solidity: function resetAmountUsedInPeriod() returns()
func (_Lineaabi *LineaabiTransactorSession) ResetAmountUsedInPeriod() (*types.Transaction, error) {
	return _Lineaabi.Contract.ResetAmountUsedInPeriod(&_Lineaabi.TransactOpts)
}

// ResetRateLimitAmount is a paid mutator transaction binding the contract method 0x557eac73.
//
// Solidity: function resetRateLimitAmount(uint256 _amount) returns()
func (_Lineaabi *LineaabiTransactor) ResetRateLimitAmount(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "resetRateLimitAmount", _amount)
}

// ResetRateLimitAmount is a paid mutator transaction binding the contract method 0x557eac73.
//
// Solidity: function resetRateLimitAmount(uint256 _amount) returns()
func (_Lineaabi *LineaabiSession) ResetRateLimitAmount(_amount *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.ResetRateLimitAmount(&_Lineaabi.TransactOpts, _amount)
}

// ResetRateLimitAmount is a paid mutator transaction binding the contract method 0x557eac73.
//
// Solidity: function resetRateLimitAmount(uint256 _amount) returns()
func (_Lineaabi *LineaabiTransactorSession) ResetRateLimitAmount(_amount *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.ResetRateLimitAmount(&_Lineaabi.TransactOpts, _amount)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiTransactor) RevokeRole(opts *bind.TransactOpts, role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "revokeRole", role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.Contract.RevokeRole(&_Lineaabi.TransactOpts, role, account)
}

// RevokeRole is a paid mutator transaction binding the contract method 0xd547741f.
//
// Solidity: function revokeRole(bytes32 role, address account) returns()
func (_Lineaabi *LineaabiTransactorSession) RevokeRole(role [32]byte, account common.Address) (*types.Transaction, error) {
	return _Lineaabi.Contract.RevokeRole(&_Lineaabi.TransactOpts, role, account)
}

// SendMessage is a paid mutator transaction binding the contract method 0x9f3ce55a.
//
// Solidity: function sendMessage(address _to, uint256 _fee, bytes _calldata) payable returns()
func (_Lineaabi *LineaabiTransactor) SendMessage(opts *bind.TransactOpts, _to common.Address, _fee *big.Int, _calldata []byte) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "sendMessage", _to, _fee, _calldata)
}

// SendMessage is a paid mutator transaction binding the contract method 0x9f3ce55a.
//
// Solidity: function sendMessage(address _to, uint256 _fee, bytes _calldata) payable returns()
func (_Lineaabi *LineaabiSession) SendMessage(_to common.Address, _fee *big.Int, _calldata []byte) (*types.Transaction, error) {
	return _Lineaabi.Contract.SendMessage(&_Lineaabi.TransactOpts, _to, _fee, _calldata)
}

// SendMessage is a paid mutator transaction binding the contract method 0x9f3ce55a.
//
// Solidity: function sendMessage(address _to, uint256 _fee, bytes _calldata) payable returns()
func (_Lineaabi *LineaabiTransactorSession) SendMessage(_to common.Address, _fee *big.Int, _calldata []byte) (*types.Transaction, error) {
	return _Lineaabi.Contract.SendMessage(&_Lineaabi.TransactOpts, _to, _fee, _calldata)
}

// SetVerifierAddress is a paid mutator transaction binding the contract method 0xc2116974.
//
// Solidity: function setVerifierAddress(address _newVerifierAddress, uint256 _proofType) returns()
func (_Lineaabi *LineaabiTransactor) SetVerifierAddress(opts *bind.TransactOpts, _newVerifierAddress common.Address, _proofType *big.Int) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "setVerifierAddress", _newVerifierAddress, _proofType)
}

// SetVerifierAddress is a paid mutator transaction binding the contract method 0xc2116974.
//
// Solidity: function setVerifierAddress(address _newVerifierAddress, uint256 _proofType) returns()
func (_Lineaabi *LineaabiSession) SetVerifierAddress(_newVerifierAddress common.Address, _proofType *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.SetVerifierAddress(&_Lineaabi.TransactOpts, _newVerifierAddress, _proofType)
}

// SetVerifierAddress is a paid mutator transaction binding the contract method 0xc2116974.
//
// Solidity: function setVerifierAddress(address _newVerifierAddress, uint256 _proofType) returns()
func (_Lineaabi *LineaabiTransactorSession) SetVerifierAddress(_newVerifierAddress common.Address, _proofType *big.Int) (*types.Transaction, error) {
	return _Lineaabi.Contract.SetVerifierAddress(&_Lineaabi.TransactOpts, _newVerifierAddress, _proofType)
}

// SubmitBlobData is a paid mutator transaction binding the contract method 0x2d3c12e5.
//
// Solidity: function submitBlobData((bytes32,bytes32,bytes32,uint256,uint256,bytes32) _submissionData, uint256 _dataEvaluationClaim, bytes _kzgCommitment, bytes _kzgProof) returns()
func (_Lineaabi *LineaabiTransactor) SubmitBlobData(opts *bind.TransactOpts, _submissionData ILineaRollupSupportingSubmissionData, _dataEvaluationClaim *big.Int, _kzgCommitment []byte, _kzgProof []byte) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "submitBlobData", _submissionData, _dataEvaluationClaim, _kzgCommitment, _kzgProof)
}

// SubmitBlobData is a paid mutator transaction binding the contract method 0x2d3c12e5.
//
// Solidity: function submitBlobData((bytes32,bytes32,bytes32,uint256,uint256,bytes32) _submissionData, uint256 _dataEvaluationClaim, bytes _kzgCommitment, bytes _kzgProof) returns()
func (_Lineaabi *LineaabiSession) SubmitBlobData(_submissionData ILineaRollupSupportingSubmissionData, _dataEvaluationClaim *big.Int, _kzgCommitment []byte, _kzgProof []byte) (*types.Transaction, error) {
	return _Lineaabi.Contract.SubmitBlobData(&_Lineaabi.TransactOpts, _submissionData, _dataEvaluationClaim, _kzgCommitment, _kzgProof)
}

// SubmitBlobData is a paid mutator transaction binding the contract method 0x2d3c12e5.
//
// Solidity: function submitBlobData((bytes32,bytes32,bytes32,uint256,uint256,bytes32) _submissionData, uint256 _dataEvaluationClaim, bytes _kzgCommitment, bytes _kzgProof) returns()
func (_Lineaabi *LineaabiTransactorSession) SubmitBlobData(_submissionData ILineaRollupSupportingSubmissionData, _dataEvaluationClaim *big.Int, _kzgCommitment []byte, _kzgProof []byte) (*types.Transaction, error) {
	return _Lineaabi.Contract.SubmitBlobData(&_Lineaabi.TransactOpts, _submissionData, _dataEvaluationClaim, _kzgCommitment, _kzgProof)
}

// SubmitData is a paid mutator transaction binding the contract method 0x7a776315.
//
// Solidity: function submitData((bytes32,bytes32,bytes32,uint256,uint256,bytes32,bytes) _submissionData) returns()
func (_Lineaabi *LineaabiTransactor) SubmitData(opts *bind.TransactOpts, _submissionData ILineaRollupSubmissionData) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "submitData", _submissionData)
}

// SubmitData is a paid mutator transaction binding the contract method 0x7a776315.
//
// Solidity: function submitData((bytes32,bytes32,bytes32,uint256,uint256,bytes32,bytes) _submissionData) returns()
func (_Lineaabi *LineaabiSession) SubmitData(_submissionData ILineaRollupSubmissionData) (*types.Transaction, error) {
	return _Lineaabi.Contract.SubmitData(&_Lineaabi.TransactOpts, _submissionData)
}

// SubmitData is a paid mutator transaction binding the contract method 0x7a776315.
//
// Solidity: function submitData((bytes32,bytes32,bytes32,uint256,uint256,bytes32,bytes) _submissionData) returns()
func (_Lineaabi *LineaabiTransactorSession) SubmitData(_submissionData ILineaRollupSubmissionData) (*types.Transaction, error) {
	return _Lineaabi.Contract.SubmitData(&_Lineaabi.TransactOpts, _submissionData)
}

// UnPauseByType is a paid mutator transaction binding the contract method 0x1065a399.
//
// Solidity: function unPauseByType(uint8 _pauseType) returns()
func (_Lineaabi *LineaabiTransactor) UnPauseByType(opts *bind.TransactOpts, _pauseType uint8) (*types.Transaction, error) {
	return _Lineaabi.contract.Transact(opts, "unPauseByType", _pauseType)
}

// UnPauseByType is a paid mutator transaction binding the contract method 0x1065a399.
//
// Solidity: function unPauseByType(uint8 _pauseType) returns()
func (_Lineaabi *LineaabiSession) UnPauseByType(_pauseType uint8) (*types.Transaction, error) {
	return _Lineaabi.Contract.UnPauseByType(&_Lineaabi.TransactOpts, _pauseType)
}

// UnPauseByType is a paid mutator transaction binding the contract method 0x1065a399.
//
// Solidity: function unPauseByType(uint8 _pauseType) returns()
func (_Lineaabi *LineaabiTransactorSession) UnPauseByType(_pauseType uint8) (*types.Transaction, error) {
	return _Lineaabi.Contract.UnPauseByType(&_Lineaabi.TransactOpts, _pauseType)
}

// LineaabiAmountUsedInPeriodResetIterator is returned from FilterAmountUsedInPeriodReset and is used to iterate over the raw logs and unpacked data for AmountUsedInPeriodReset events raised by the Lineaabi contract.
type LineaabiAmountUsedInPeriodResetIterator struct {
	Event *LineaabiAmountUsedInPeriodReset // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiAmountUsedInPeriodResetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiAmountUsedInPeriodReset)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiAmountUsedInPeriodReset)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiAmountUsedInPeriodResetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiAmountUsedInPeriodResetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiAmountUsedInPeriodReset represents a AmountUsedInPeriodReset event raised by the Lineaabi contract.
type LineaabiAmountUsedInPeriodReset struct {
	ResettingAddress common.Address
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterAmountUsedInPeriodReset is a free log retrieval operation binding the contract event 0xba88c025b0cbb77022c0c487beef24f759f1e4be2f51a205bc427cee19c2eaa6.
//
// Solidity: event AmountUsedInPeriodReset(address indexed resettingAddress)
func (_Lineaabi *LineaabiFilterer) FilterAmountUsedInPeriodReset(opts *bind.FilterOpts, resettingAddress []common.Address) (*LineaabiAmountUsedInPeriodResetIterator, error) {

	var resettingAddressRule []interface{}
	for _, resettingAddressItem := range resettingAddress {
		resettingAddressRule = append(resettingAddressRule, resettingAddressItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "AmountUsedInPeriodReset", resettingAddressRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiAmountUsedInPeriodResetIterator{contract: _Lineaabi.contract, event: "AmountUsedInPeriodReset", logs: logs, sub: sub}, nil
}

// WatchAmountUsedInPeriodReset is a free log subscription operation binding the contract event 0xba88c025b0cbb77022c0c487beef24f759f1e4be2f51a205bc427cee19c2eaa6.
//
// Solidity: event AmountUsedInPeriodReset(address indexed resettingAddress)
func (_Lineaabi *LineaabiFilterer) WatchAmountUsedInPeriodReset(opts *bind.WatchOpts, sink chan<- *LineaabiAmountUsedInPeriodReset, resettingAddress []common.Address) (event.Subscription, error) {

	var resettingAddressRule []interface{}
	for _, resettingAddressItem := range resettingAddress {
		resettingAddressRule = append(resettingAddressRule, resettingAddressItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "AmountUsedInPeriodReset", resettingAddressRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiAmountUsedInPeriodReset)
				if err := _Lineaabi.contract.UnpackLog(event, "AmountUsedInPeriodReset", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAmountUsedInPeriodReset is a log parse operation binding the contract event 0xba88c025b0cbb77022c0c487beef24f759f1e4be2f51a205bc427cee19c2eaa6.
//
// Solidity: event AmountUsedInPeriodReset(address indexed resettingAddress)
func (_Lineaabi *LineaabiFilterer) ParseAmountUsedInPeriodReset(log types.Log) (*LineaabiAmountUsedInPeriodReset, error) {
	event := new(LineaabiAmountUsedInPeriodReset)
	if err := _Lineaabi.contract.UnpackLog(event, "AmountUsedInPeriodReset", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiBlockFinalizedIterator is returned from FilterBlockFinalized and is used to iterate over the raw logs and unpacked data for BlockFinalized events raised by the Lineaabi contract.
type LineaabiBlockFinalizedIterator struct {
	Event *LineaabiBlockFinalized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiBlockFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiBlockFinalized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiBlockFinalized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiBlockFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiBlockFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiBlockFinalized represents a BlockFinalized event raised by the Lineaabi contract.
type LineaabiBlockFinalized struct {
	BlockNumber        *big.Int
	StateRootHash      [32]byte
	FinalizedWithProof bool
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterBlockFinalized is a free log retrieval operation binding the contract event 0x047c6ce79802b16b6527cedd89156bb59f2da26867b4f218fa60c9521ddcce55.
//
// Solidity: event BlockFinalized(uint256 indexed blockNumber, bytes32 indexed stateRootHash, bool indexed finalizedWithProof)
func (_Lineaabi *LineaabiFilterer) FilterBlockFinalized(opts *bind.FilterOpts, blockNumber []*big.Int, stateRootHash [][32]byte, finalizedWithProof []bool) (*LineaabiBlockFinalizedIterator, error) {

	var blockNumberRule []interface{}
	for _, blockNumberItem := range blockNumber {
		blockNumberRule = append(blockNumberRule, blockNumberItem)
	}
	var stateRootHashRule []interface{}
	for _, stateRootHashItem := range stateRootHash {
		stateRootHashRule = append(stateRootHashRule, stateRootHashItem)
	}
	var finalizedWithProofRule []interface{}
	for _, finalizedWithProofItem := range finalizedWithProof {
		finalizedWithProofRule = append(finalizedWithProofRule, finalizedWithProofItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "BlockFinalized", blockNumberRule, stateRootHashRule, finalizedWithProofRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiBlockFinalizedIterator{contract: _Lineaabi.contract, event: "BlockFinalized", logs: logs, sub: sub}, nil
}

// WatchBlockFinalized is a free log subscription operation binding the contract event 0x047c6ce79802b16b6527cedd89156bb59f2da26867b4f218fa60c9521ddcce55.
//
// Solidity: event BlockFinalized(uint256 indexed blockNumber, bytes32 indexed stateRootHash, bool indexed finalizedWithProof)
func (_Lineaabi *LineaabiFilterer) WatchBlockFinalized(opts *bind.WatchOpts, sink chan<- *LineaabiBlockFinalized, blockNumber []*big.Int, stateRootHash [][32]byte, finalizedWithProof []bool) (event.Subscription, error) {

	var blockNumberRule []interface{}
	for _, blockNumberItem := range blockNumber {
		blockNumberRule = append(blockNumberRule, blockNumberItem)
	}
	var stateRootHashRule []interface{}
	for _, stateRootHashItem := range stateRootHash {
		stateRootHashRule = append(stateRootHashRule, stateRootHashItem)
	}
	var finalizedWithProofRule []interface{}
	for _, finalizedWithProofItem := range finalizedWithProof {
		finalizedWithProofRule = append(finalizedWithProofRule, finalizedWithProofItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "BlockFinalized", blockNumberRule, stateRootHashRule, finalizedWithProofRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiBlockFinalized)
				if err := _Lineaabi.contract.UnpackLog(event, "BlockFinalized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBlockFinalized is a log parse operation binding the contract event 0x047c6ce79802b16b6527cedd89156bb59f2da26867b4f218fa60c9521ddcce55.
//
// Solidity: event BlockFinalized(uint256 indexed blockNumber, bytes32 indexed stateRootHash, bool indexed finalizedWithProof)
func (_Lineaabi *LineaabiFilterer) ParseBlockFinalized(log types.Log) (*LineaabiBlockFinalized, error) {
	event := new(LineaabiBlockFinalized)
	if err := _Lineaabi.contract.UnpackLog(event, "BlockFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiBlocksVerificationDoneIterator is returned from FilterBlocksVerificationDone and is used to iterate over the raw logs and unpacked data for BlocksVerificationDone events raised by the Lineaabi contract.
type LineaabiBlocksVerificationDoneIterator struct {
	Event *LineaabiBlocksVerificationDone // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiBlocksVerificationDoneIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiBlocksVerificationDone)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiBlocksVerificationDone)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiBlocksVerificationDoneIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiBlocksVerificationDoneIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiBlocksVerificationDone represents a BlocksVerificationDone event raised by the Lineaabi contract.
type LineaabiBlocksVerificationDone struct {
	LastBlockFinalized *big.Int
	StartingRootHash   [32]byte
	FinalRootHash      [32]byte
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterBlocksVerificationDone is a free log retrieval operation binding the contract event 0x5c885a794662ebe3b08ae0874fc2c88b5343b0223ba9cd2cad92b69c0d0c901f.
//
// Solidity: event BlocksVerificationDone(uint256 indexed lastBlockFinalized, bytes32 startingRootHash, bytes32 finalRootHash)
func (_Lineaabi *LineaabiFilterer) FilterBlocksVerificationDone(opts *bind.FilterOpts, lastBlockFinalized []*big.Int) (*LineaabiBlocksVerificationDoneIterator, error) {

	var lastBlockFinalizedRule []interface{}
	for _, lastBlockFinalizedItem := range lastBlockFinalized {
		lastBlockFinalizedRule = append(lastBlockFinalizedRule, lastBlockFinalizedItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "BlocksVerificationDone", lastBlockFinalizedRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiBlocksVerificationDoneIterator{contract: _Lineaabi.contract, event: "BlocksVerificationDone", logs: logs, sub: sub}, nil
}

// WatchBlocksVerificationDone is a free log subscription operation binding the contract event 0x5c885a794662ebe3b08ae0874fc2c88b5343b0223ba9cd2cad92b69c0d0c901f.
//
// Solidity: event BlocksVerificationDone(uint256 indexed lastBlockFinalized, bytes32 startingRootHash, bytes32 finalRootHash)
func (_Lineaabi *LineaabiFilterer) WatchBlocksVerificationDone(opts *bind.WatchOpts, sink chan<- *LineaabiBlocksVerificationDone, lastBlockFinalized []*big.Int) (event.Subscription, error) {

	var lastBlockFinalizedRule []interface{}
	for _, lastBlockFinalizedItem := range lastBlockFinalized {
		lastBlockFinalizedRule = append(lastBlockFinalizedRule, lastBlockFinalizedItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "BlocksVerificationDone", lastBlockFinalizedRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiBlocksVerificationDone)
				if err := _Lineaabi.contract.UnpackLog(event, "BlocksVerificationDone", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBlocksVerificationDone is a log parse operation binding the contract event 0x5c885a794662ebe3b08ae0874fc2c88b5343b0223ba9cd2cad92b69c0d0c901f.
//
// Solidity: event BlocksVerificationDone(uint256 indexed lastBlockFinalized, bytes32 startingRootHash, bytes32 finalRootHash)
func (_Lineaabi *LineaabiFilterer) ParseBlocksVerificationDone(log types.Log) (*LineaabiBlocksVerificationDone, error) {
	event := new(LineaabiBlocksVerificationDone)
	if err := _Lineaabi.contract.UnpackLog(event, "BlocksVerificationDone", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiDataFinalizedIterator is returned from FilterDataFinalized and is used to iterate over the raw logs and unpacked data for DataFinalized events raised by the Lineaabi contract.
type LineaabiDataFinalizedIterator struct {
	Event *LineaabiDataFinalized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiDataFinalizedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiDataFinalized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiDataFinalized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiDataFinalizedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiDataFinalizedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiDataFinalized represents a DataFinalized event raised by the Lineaabi contract.
type LineaabiDataFinalized struct {
	LastBlockFinalized *big.Int
	StartingRootHash   [32]byte
	FinalRootHash      [32]byte
	WithProof          bool
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterDataFinalized is a free log retrieval operation binding the contract event 0x1335f1a2b3ff25f07f5fef07dd35d8fb4312c3c73b138e2fad9347b3319ab53c.
//
// Solidity: event DataFinalized(uint256 indexed lastBlockFinalized, bytes32 indexed startingRootHash, bytes32 indexed finalRootHash, bool withProof)
func (_Lineaabi *LineaabiFilterer) FilterDataFinalized(opts *bind.FilterOpts, lastBlockFinalized []*big.Int, startingRootHash [][32]byte, finalRootHash [][32]byte) (*LineaabiDataFinalizedIterator, error) {

	var lastBlockFinalizedRule []interface{}
	for _, lastBlockFinalizedItem := range lastBlockFinalized {
		lastBlockFinalizedRule = append(lastBlockFinalizedRule, lastBlockFinalizedItem)
	}
	var startingRootHashRule []interface{}
	for _, startingRootHashItem := range startingRootHash {
		startingRootHashRule = append(startingRootHashRule, startingRootHashItem)
	}
	var finalRootHashRule []interface{}
	for _, finalRootHashItem := range finalRootHash {
		finalRootHashRule = append(finalRootHashRule, finalRootHashItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "DataFinalized", lastBlockFinalizedRule, startingRootHashRule, finalRootHashRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiDataFinalizedIterator{contract: _Lineaabi.contract, event: "DataFinalized", logs: logs, sub: sub}, nil
}

// WatchDataFinalized is a free log subscription operation binding the contract event 0x1335f1a2b3ff25f07f5fef07dd35d8fb4312c3c73b138e2fad9347b3319ab53c.
//
// Solidity: event DataFinalized(uint256 indexed lastBlockFinalized, bytes32 indexed startingRootHash, bytes32 indexed finalRootHash, bool withProof)
func (_Lineaabi *LineaabiFilterer) WatchDataFinalized(opts *bind.WatchOpts, sink chan<- *LineaabiDataFinalized, lastBlockFinalized []*big.Int, startingRootHash [][32]byte, finalRootHash [][32]byte) (event.Subscription, error) {

	var lastBlockFinalizedRule []interface{}
	for _, lastBlockFinalizedItem := range lastBlockFinalized {
		lastBlockFinalizedRule = append(lastBlockFinalizedRule, lastBlockFinalizedItem)
	}
	var startingRootHashRule []interface{}
	for _, startingRootHashItem := range startingRootHash {
		startingRootHashRule = append(startingRootHashRule, startingRootHashItem)
	}
	var finalRootHashRule []interface{}
	for _, finalRootHashItem := range finalRootHash {
		finalRootHashRule = append(finalRootHashRule, finalRootHashItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "DataFinalized", lastBlockFinalizedRule, startingRootHashRule, finalRootHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiDataFinalized)
				if err := _Lineaabi.contract.UnpackLog(event, "DataFinalized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDataFinalized is a log parse operation binding the contract event 0x1335f1a2b3ff25f07f5fef07dd35d8fb4312c3c73b138e2fad9347b3319ab53c.
//
// Solidity: event DataFinalized(uint256 indexed lastBlockFinalized, bytes32 indexed startingRootHash, bytes32 indexed finalRootHash, bool withProof)
func (_Lineaabi *LineaabiFilterer) ParseDataFinalized(log types.Log) (*LineaabiDataFinalized, error) {
	event := new(LineaabiDataFinalized)
	if err := _Lineaabi.contract.UnpackLog(event, "DataFinalized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiDataSubmittedIterator is returned from FilterDataSubmitted and is used to iterate over the raw logs and unpacked data for DataSubmitted events raised by the Lineaabi contract.
type LineaabiDataSubmittedIterator struct {
	Event *LineaabiDataSubmitted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiDataSubmittedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiDataSubmitted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiDataSubmitted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiDataSubmittedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiDataSubmittedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiDataSubmitted represents a DataSubmitted event raised by the Lineaabi contract.
type LineaabiDataSubmitted struct {
	DataHash   [32]byte
	StartBlock *big.Int
	EndBlock   *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterDataSubmitted is a free log retrieval operation binding the contract event 0x174b4a2e83ebebaf6824e559d2bab7b7e229c80d211e98298a1224970b719a42.
//
// Solidity: event DataSubmitted(bytes32 indexed dataHash, uint256 indexed startBlock, uint256 indexed endBlock)
func (_Lineaabi *LineaabiFilterer) FilterDataSubmitted(opts *bind.FilterOpts, dataHash [][32]byte, startBlock []*big.Int, endBlock []*big.Int) (*LineaabiDataSubmittedIterator, error) {

	var dataHashRule []interface{}
	for _, dataHashItem := range dataHash {
		dataHashRule = append(dataHashRule, dataHashItem)
	}
	var startBlockRule []interface{}
	for _, startBlockItem := range startBlock {
		startBlockRule = append(startBlockRule, startBlockItem)
	}
	var endBlockRule []interface{}
	for _, endBlockItem := range endBlock {
		endBlockRule = append(endBlockRule, endBlockItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "DataSubmitted", dataHashRule, startBlockRule, endBlockRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiDataSubmittedIterator{contract: _Lineaabi.contract, event: "DataSubmitted", logs: logs, sub: sub}, nil
}

// WatchDataSubmitted is a free log subscription operation binding the contract event 0x174b4a2e83ebebaf6824e559d2bab7b7e229c80d211e98298a1224970b719a42.
//
// Solidity: event DataSubmitted(bytes32 indexed dataHash, uint256 indexed startBlock, uint256 indexed endBlock)
func (_Lineaabi *LineaabiFilterer) WatchDataSubmitted(opts *bind.WatchOpts, sink chan<- *LineaabiDataSubmitted, dataHash [][32]byte, startBlock []*big.Int, endBlock []*big.Int) (event.Subscription, error) {

	var dataHashRule []interface{}
	for _, dataHashItem := range dataHash {
		dataHashRule = append(dataHashRule, dataHashItem)
	}
	var startBlockRule []interface{}
	for _, startBlockItem := range startBlock {
		startBlockRule = append(startBlockRule, startBlockItem)
	}
	var endBlockRule []interface{}
	for _, endBlockItem := range endBlock {
		endBlockRule = append(endBlockRule, endBlockItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "DataSubmitted", dataHashRule, startBlockRule, endBlockRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiDataSubmitted)
				if err := _Lineaabi.contract.UnpackLog(event, "DataSubmitted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDataSubmitted is a log parse operation binding the contract event 0x174b4a2e83ebebaf6824e559d2bab7b7e229c80d211e98298a1224970b719a42.
//
// Solidity: event DataSubmitted(bytes32 indexed dataHash, uint256 indexed startBlock, uint256 indexed endBlock)
func (_Lineaabi *LineaabiFilterer) ParseDataSubmitted(log types.Log) (*LineaabiDataSubmitted, error) {
	event := new(LineaabiDataSubmitted)
	if err := _Lineaabi.contract.UnpackLog(event, "DataSubmitted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Lineaabi contract.
type LineaabiInitializedIterator struct {
	Event *LineaabiInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiInitialized represents a Initialized event raised by the Lineaabi contract.
type LineaabiInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Lineaabi *LineaabiFilterer) FilterInitialized(opts *bind.FilterOpts) (*LineaabiInitializedIterator, error) {

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &LineaabiInitializedIterator{contract: _Lineaabi.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Lineaabi *LineaabiFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *LineaabiInitialized) (event.Subscription, error) {

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiInitialized)
				if err := _Lineaabi.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_Lineaabi *LineaabiFilterer) ParseInitialized(log types.Log) (*LineaabiInitialized, error) {
	event := new(LineaabiInitialized)
	if err := _Lineaabi.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiL1L2MessagesReceivedOnL2Iterator is returned from FilterL1L2MessagesReceivedOnL2 and is used to iterate over the raw logs and unpacked data for L1L2MessagesReceivedOnL2 events raised by the Lineaabi contract.
type LineaabiL1L2MessagesReceivedOnL2Iterator struct {
	Event *LineaabiL1L2MessagesReceivedOnL2 // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiL1L2MessagesReceivedOnL2Iterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiL1L2MessagesReceivedOnL2)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiL1L2MessagesReceivedOnL2)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiL1L2MessagesReceivedOnL2Iterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiL1L2MessagesReceivedOnL2Iterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiL1L2MessagesReceivedOnL2 represents a L1L2MessagesReceivedOnL2 event raised by the Lineaabi contract.
type LineaabiL1L2MessagesReceivedOnL2 struct {
	MessageHashes [][32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterL1L2MessagesReceivedOnL2 is a free log retrieval operation binding the contract event 0x95e84bb4317676921a29fd1d13f8f0153508473b899c12b3cd08314348801d64.
//
// Solidity: event L1L2MessagesReceivedOnL2(bytes32[] messageHashes)
func (_Lineaabi *LineaabiFilterer) FilterL1L2MessagesReceivedOnL2(opts *bind.FilterOpts) (*LineaabiL1L2MessagesReceivedOnL2Iterator, error) {

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "L1L2MessagesReceivedOnL2")
	if err != nil {
		return nil, err
	}
	return &LineaabiL1L2MessagesReceivedOnL2Iterator{contract: _Lineaabi.contract, event: "L1L2MessagesReceivedOnL2", logs: logs, sub: sub}, nil
}

// WatchL1L2MessagesReceivedOnL2 is a free log subscription operation binding the contract event 0x95e84bb4317676921a29fd1d13f8f0153508473b899c12b3cd08314348801d64.
//
// Solidity: event L1L2MessagesReceivedOnL2(bytes32[] messageHashes)
func (_Lineaabi *LineaabiFilterer) WatchL1L2MessagesReceivedOnL2(opts *bind.WatchOpts, sink chan<- *LineaabiL1L2MessagesReceivedOnL2) (event.Subscription, error) {

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "L1L2MessagesReceivedOnL2")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiL1L2MessagesReceivedOnL2)
				if err := _Lineaabi.contract.UnpackLog(event, "L1L2MessagesReceivedOnL2", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseL1L2MessagesReceivedOnL2 is a log parse operation binding the contract event 0x95e84bb4317676921a29fd1d13f8f0153508473b899c12b3cd08314348801d64.
//
// Solidity: event L1L2MessagesReceivedOnL2(bytes32[] messageHashes)
func (_Lineaabi *LineaabiFilterer) ParseL1L2MessagesReceivedOnL2(log types.Log) (*LineaabiL1L2MessagesReceivedOnL2, error) {
	event := new(LineaabiL1L2MessagesReceivedOnL2)
	if err := _Lineaabi.contract.UnpackLog(event, "L1L2MessagesReceivedOnL2", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiL2L1MessageHashAddedToInboxIterator is returned from FilterL2L1MessageHashAddedToInbox and is used to iterate over the raw logs and unpacked data for L2L1MessageHashAddedToInbox events raised by the Lineaabi contract.
type LineaabiL2L1MessageHashAddedToInboxIterator struct {
	Event *LineaabiL2L1MessageHashAddedToInbox // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiL2L1MessageHashAddedToInboxIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiL2L1MessageHashAddedToInbox)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiL2L1MessageHashAddedToInbox)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiL2L1MessageHashAddedToInboxIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiL2L1MessageHashAddedToInboxIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiL2L1MessageHashAddedToInbox represents a L2L1MessageHashAddedToInbox event raised by the Lineaabi contract.
type LineaabiL2L1MessageHashAddedToInbox struct {
	MessageHash [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterL2L1MessageHashAddedToInbox is a free log retrieval operation binding the contract event 0x810484e22f73d8f099aaee1edb851ec6be6d84d43045d0a7803e5f7b3612edce.
//
// Solidity: event L2L1MessageHashAddedToInbox(bytes32 indexed messageHash)
func (_Lineaabi *LineaabiFilterer) FilterL2L1MessageHashAddedToInbox(opts *bind.FilterOpts, messageHash [][32]byte) (*LineaabiL2L1MessageHashAddedToInboxIterator, error) {

	var messageHashRule []interface{}
	for _, messageHashItem := range messageHash {
		messageHashRule = append(messageHashRule, messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "L2L1MessageHashAddedToInbox", messageHashRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiL2L1MessageHashAddedToInboxIterator{contract: _Lineaabi.contract, event: "L2L1MessageHashAddedToInbox", logs: logs, sub: sub}, nil
}

// WatchL2L1MessageHashAddedToInbox is a free log subscription operation binding the contract event 0x810484e22f73d8f099aaee1edb851ec6be6d84d43045d0a7803e5f7b3612edce.
//
// Solidity: event L2L1MessageHashAddedToInbox(bytes32 indexed messageHash)
func (_Lineaabi *LineaabiFilterer) WatchL2L1MessageHashAddedToInbox(opts *bind.WatchOpts, sink chan<- *LineaabiL2L1MessageHashAddedToInbox, messageHash [][32]byte) (event.Subscription, error) {

	var messageHashRule []interface{}
	for _, messageHashItem := range messageHash {
		messageHashRule = append(messageHashRule, messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "L2L1MessageHashAddedToInbox", messageHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiL2L1MessageHashAddedToInbox)
				if err := _Lineaabi.contract.UnpackLog(event, "L2L1MessageHashAddedToInbox", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseL2L1MessageHashAddedToInbox is a log parse operation binding the contract event 0x810484e22f73d8f099aaee1edb851ec6be6d84d43045d0a7803e5f7b3612edce.
//
// Solidity: event L2L1MessageHashAddedToInbox(bytes32 indexed messageHash)
func (_Lineaabi *LineaabiFilterer) ParseL2L1MessageHashAddedToInbox(log types.Log) (*LineaabiL2L1MessageHashAddedToInbox, error) {
	event := new(LineaabiL2L1MessageHashAddedToInbox)
	if err := _Lineaabi.contract.UnpackLog(event, "L2L1MessageHashAddedToInbox", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiL2MerkleRootAddedIterator is returned from FilterL2MerkleRootAdded and is used to iterate over the raw logs and unpacked data for L2MerkleRootAdded events raised by the Lineaabi contract.
type LineaabiL2MerkleRootAddedIterator struct {
	Event *LineaabiL2MerkleRootAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiL2MerkleRootAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiL2MerkleRootAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiL2MerkleRootAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiL2MerkleRootAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiL2MerkleRootAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiL2MerkleRootAdded represents a L2MerkleRootAdded event raised by the Lineaabi contract.
type LineaabiL2MerkleRootAdded struct {
	L2MerkleRoot [32]byte
	TreeDepth    *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterL2MerkleRootAdded is a free log retrieval operation binding the contract event 0x300e6f978eee6a4b0bba78dd8400dc64fd5652dbfc868a2258e16d0977be222b.
//
// Solidity: event L2MerkleRootAdded(bytes32 indexed l2MerkleRoot, uint256 indexed treeDepth)
func (_Lineaabi *LineaabiFilterer) FilterL2MerkleRootAdded(opts *bind.FilterOpts, l2MerkleRoot [][32]byte, treeDepth []*big.Int) (*LineaabiL2MerkleRootAddedIterator, error) {

	var l2MerkleRootRule []interface{}
	for _, l2MerkleRootItem := range l2MerkleRoot {
		l2MerkleRootRule = append(l2MerkleRootRule, l2MerkleRootItem)
	}
	var treeDepthRule []interface{}
	for _, treeDepthItem := range treeDepth {
		treeDepthRule = append(treeDepthRule, treeDepthItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "L2MerkleRootAdded", l2MerkleRootRule, treeDepthRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiL2MerkleRootAddedIterator{contract: _Lineaabi.contract, event: "L2MerkleRootAdded", logs: logs, sub: sub}, nil
}

// WatchL2MerkleRootAdded is a free log subscription operation binding the contract event 0x300e6f978eee6a4b0bba78dd8400dc64fd5652dbfc868a2258e16d0977be222b.
//
// Solidity: event L2MerkleRootAdded(bytes32 indexed l2MerkleRoot, uint256 indexed treeDepth)
func (_Lineaabi *LineaabiFilterer) WatchL2MerkleRootAdded(opts *bind.WatchOpts, sink chan<- *LineaabiL2MerkleRootAdded, l2MerkleRoot [][32]byte, treeDepth []*big.Int) (event.Subscription, error) {

	var l2MerkleRootRule []interface{}
	for _, l2MerkleRootItem := range l2MerkleRoot {
		l2MerkleRootRule = append(l2MerkleRootRule, l2MerkleRootItem)
	}
	var treeDepthRule []interface{}
	for _, treeDepthItem := range treeDepth {
		treeDepthRule = append(treeDepthRule, treeDepthItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "L2MerkleRootAdded", l2MerkleRootRule, treeDepthRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiL2MerkleRootAdded)
				if err := _Lineaabi.contract.UnpackLog(event, "L2MerkleRootAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseL2MerkleRootAdded is a log parse operation binding the contract event 0x300e6f978eee6a4b0bba78dd8400dc64fd5652dbfc868a2258e16d0977be222b.
//
// Solidity: event L2MerkleRootAdded(bytes32 indexed l2MerkleRoot, uint256 indexed treeDepth)
func (_Lineaabi *LineaabiFilterer) ParseL2MerkleRootAdded(log types.Log) (*LineaabiL2MerkleRootAdded, error) {
	event := new(LineaabiL2MerkleRootAdded)
	if err := _Lineaabi.contract.UnpackLog(event, "L2MerkleRootAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiL2MessagingBlockAnchoredIterator is returned from FilterL2MessagingBlockAnchored and is used to iterate over the raw logs and unpacked data for L2MessagingBlockAnchored events raised by the Lineaabi contract.
type LineaabiL2MessagingBlockAnchoredIterator struct {
	Event *LineaabiL2MessagingBlockAnchored // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiL2MessagingBlockAnchoredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiL2MessagingBlockAnchored)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiL2MessagingBlockAnchored)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiL2MessagingBlockAnchoredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiL2MessagingBlockAnchoredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiL2MessagingBlockAnchored represents a L2MessagingBlockAnchored event raised by the Lineaabi contract.
type LineaabiL2MessagingBlockAnchored struct {
	L2Block *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterL2MessagingBlockAnchored is a free log retrieval operation binding the contract event 0x3c116827db9db3a30c1a25db8b0ee4bab9d2b223560209cfd839601b621c726d.
//
// Solidity: event L2MessagingBlockAnchored(uint256 indexed l2Block)
func (_Lineaabi *LineaabiFilterer) FilterL2MessagingBlockAnchored(opts *bind.FilterOpts, l2Block []*big.Int) (*LineaabiL2MessagingBlockAnchoredIterator, error) {

	var l2BlockRule []interface{}
	for _, l2BlockItem := range l2Block {
		l2BlockRule = append(l2BlockRule, l2BlockItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "L2MessagingBlockAnchored", l2BlockRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiL2MessagingBlockAnchoredIterator{contract: _Lineaabi.contract, event: "L2MessagingBlockAnchored", logs: logs, sub: sub}, nil
}

// WatchL2MessagingBlockAnchored is a free log subscription operation binding the contract event 0x3c116827db9db3a30c1a25db8b0ee4bab9d2b223560209cfd839601b621c726d.
//
// Solidity: event L2MessagingBlockAnchored(uint256 indexed l2Block)
func (_Lineaabi *LineaabiFilterer) WatchL2MessagingBlockAnchored(opts *bind.WatchOpts, sink chan<- *LineaabiL2MessagingBlockAnchored, l2Block []*big.Int) (event.Subscription, error) {

	var l2BlockRule []interface{}
	for _, l2BlockItem := range l2Block {
		l2BlockRule = append(l2BlockRule, l2BlockItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "L2MessagingBlockAnchored", l2BlockRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiL2MessagingBlockAnchored)
				if err := _Lineaabi.contract.UnpackLog(event, "L2MessagingBlockAnchored", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseL2MessagingBlockAnchored is a log parse operation binding the contract event 0x3c116827db9db3a30c1a25db8b0ee4bab9d2b223560209cfd839601b621c726d.
//
// Solidity: event L2MessagingBlockAnchored(uint256 indexed l2Block)
func (_Lineaabi *LineaabiFilterer) ParseL2MessagingBlockAnchored(log types.Log) (*LineaabiL2MessagingBlockAnchored, error) {
	event := new(LineaabiL2MessagingBlockAnchored)
	if err := _Lineaabi.contract.UnpackLog(event, "L2MessagingBlockAnchored", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiLimitAmountChangedIterator is returned from FilterLimitAmountChanged and is used to iterate over the raw logs and unpacked data for LimitAmountChanged events raised by the Lineaabi contract.
type LineaabiLimitAmountChangedIterator struct {
	Event *LineaabiLimitAmountChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiLimitAmountChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiLimitAmountChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiLimitAmountChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiLimitAmountChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiLimitAmountChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiLimitAmountChanged represents a LimitAmountChanged event raised by the Lineaabi contract.
type LineaabiLimitAmountChanged struct {
	AmountChangeBy           common.Address
	Amount                   *big.Int
	AmountUsedLoweredToLimit bool
	UsedAmountResetToZero    bool
	Raw                      types.Log // Blockchain specific contextual infos
}

// FilterLimitAmountChanged is a free log retrieval operation binding the contract event 0xbc3dc0cb5c15c51c81316450d44048838bb478b9809447d01c766a06f3e9f2c8.
//
// Solidity: event LimitAmountChanged(address indexed amountChangeBy, uint256 amount, bool amountUsedLoweredToLimit, bool usedAmountResetToZero)
func (_Lineaabi *LineaabiFilterer) FilterLimitAmountChanged(opts *bind.FilterOpts, amountChangeBy []common.Address) (*LineaabiLimitAmountChangedIterator, error) {

	var amountChangeByRule []interface{}
	for _, amountChangeByItem := range amountChangeBy {
		amountChangeByRule = append(amountChangeByRule, amountChangeByItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "LimitAmountChanged", amountChangeByRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiLimitAmountChangedIterator{contract: _Lineaabi.contract, event: "LimitAmountChanged", logs: logs, sub: sub}, nil
}

// WatchLimitAmountChanged is a free log subscription operation binding the contract event 0xbc3dc0cb5c15c51c81316450d44048838bb478b9809447d01c766a06f3e9f2c8.
//
// Solidity: event LimitAmountChanged(address indexed amountChangeBy, uint256 amount, bool amountUsedLoweredToLimit, bool usedAmountResetToZero)
func (_Lineaabi *LineaabiFilterer) WatchLimitAmountChanged(opts *bind.WatchOpts, sink chan<- *LineaabiLimitAmountChanged, amountChangeBy []common.Address) (event.Subscription, error) {

	var amountChangeByRule []interface{}
	for _, amountChangeByItem := range amountChangeBy {
		amountChangeByRule = append(amountChangeByRule, amountChangeByItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "LimitAmountChanged", amountChangeByRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiLimitAmountChanged)
				if err := _Lineaabi.contract.UnpackLog(event, "LimitAmountChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseLimitAmountChanged is a log parse operation binding the contract event 0xbc3dc0cb5c15c51c81316450d44048838bb478b9809447d01c766a06f3e9f2c8.
//
// Solidity: event LimitAmountChanged(address indexed amountChangeBy, uint256 amount, bool amountUsedLoweredToLimit, bool usedAmountResetToZero)
func (_Lineaabi *LineaabiFilterer) ParseLimitAmountChanged(log types.Log) (*LineaabiLimitAmountChanged, error) {
	event := new(LineaabiLimitAmountChanged)
	if err := _Lineaabi.contract.UnpackLog(event, "LimitAmountChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiMessageClaimedIterator is returned from FilterMessageClaimed and is used to iterate over the raw logs and unpacked data for MessageClaimed events raised by the Lineaabi contract.
type LineaabiMessageClaimedIterator struct {
	Event *LineaabiMessageClaimed // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiMessageClaimedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiMessageClaimed)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiMessageClaimed)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiMessageClaimedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiMessageClaimedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiMessageClaimed represents a MessageClaimed event raised by the Lineaabi contract.
type LineaabiMessageClaimed struct {
	MessageHash [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterMessageClaimed is a free log retrieval operation binding the contract event 0xa4c827e719e911e8f19393ccdb85b5102f08f0910604d340ba38390b7ff2ab0e.
//
// Solidity: event MessageClaimed(bytes32 indexed _messageHash)
func (_Lineaabi *LineaabiFilterer) FilterMessageClaimed(opts *bind.FilterOpts, _messageHash [][32]byte) (*LineaabiMessageClaimedIterator, error) {

	var _messageHashRule []interface{}
	for _, _messageHashItem := range _messageHash {
		_messageHashRule = append(_messageHashRule, _messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "MessageClaimed", _messageHashRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiMessageClaimedIterator{contract: _Lineaabi.contract, event: "MessageClaimed", logs: logs, sub: sub}, nil
}

// WatchMessageClaimed is a free log subscription operation binding the contract event 0xa4c827e719e911e8f19393ccdb85b5102f08f0910604d340ba38390b7ff2ab0e.
//
// Solidity: event MessageClaimed(bytes32 indexed _messageHash)
func (_Lineaabi *LineaabiFilterer) WatchMessageClaimed(opts *bind.WatchOpts, sink chan<- *LineaabiMessageClaimed, _messageHash [][32]byte) (event.Subscription, error) {

	var _messageHashRule []interface{}
	for _, _messageHashItem := range _messageHash {
		_messageHashRule = append(_messageHashRule, _messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "MessageClaimed", _messageHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiMessageClaimed)
				if err := _Lineaabi.contract.UnpackLog(event, "MessageClaimed", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMessageClaimed is a log parse operation binding the contract event 0xa4c827e719e911e8f19393ccdb85b5102f08f0910604d340ba38390b7ff2ab0e.
//
// Solidity: event MessageClaimed(bytes32 indexed _messageHash)
func (_Lineaabi *LineaabiFilterer) ParseMessageClaimed(log types.Log) (*LineaabiMessageClaimed, error) {
	event := new(LineaabiMessageClaimed)
	if err := _Lineaabi.contract.UnpackLog(event, "MessageClaimed", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiMessageSentIterator is returned from FilterMessageSent and is used to iterate over the raw logs and unpacked data for MessageSent events raised by the Lineaabi contract.
type LineaabiMessageSentIterator struct {
	Event *LineaabiMessageSent // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiMessageSentIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiMessageSent)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiMessageSent)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiMessageSentIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiMessageSentIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiMessageSent represents a MessageSent event raised by the Lineaabi contract.
type LineaabiMessageSent struct {
	From        common.Address
	To          common.Address
	Fee         *big.Int
	Value       *big.Int
	Nonce       *big.Int
	Calldata    []byte
	MessageHash [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterMessageSent is a free log retrieval operation binding the contract event 0xe856c2b8bd4eb0027ce32eeaf595c21b0b6b4644b326e5b7bd80a1cf8db72e6c.
//
// Solidity: event MessageSent(address indexed _from, address indexed _to, uint256 _fee, uint256 _value, uint256 _nonce, bytes _calldata, bytes32 indexed _messageHash)
func (_Lineaabi *LineaabiFilterer) FilterMessageSent(opts *bind.FilterOpts, _from []common.Address, _to []common.Address, _messageHash [][32]byte) (*LineaabiMessageSentIterator, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	var _messageHashRule []interface{}
	for _, _messageHashItem := range _messageHash {
		_messageHashRule = append(_messageHashRule, _messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "MessageSent", _fromRule, _toRule, _messageHashRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiMessageSentIterator{contract: _Lineaabi.contract, event: "MessageSent", logs: logs, sub: sub}, nil
}

// WatchMessageSent is a free log subscription operation binding the contract event 0xe856c2b8bd4eb0027ce32eeaf595c21b0b6b4644b326e5b7bd80a1cf8db72e6c.
//
// Solidity: event MessageSent(address indexed _from, address indexed _to, uint256 _fee, uint256 _value, uint256 _nonce, bytes _calldata, bytes32 indexed _messageHash)
func (_Lineaabi *LineaabiFilterer) WatchMessageSent(opts *bind.WatchOpts, sink chan<- *LineaabiMessageSent, _from []common.Address, _to []common.Address, _messageHash [][32]byte) (event.Subscription, error) {

	var _fromRule []interface{}
	for _, _fromItem := range _from {
		_fromRule = append(_fromRule, _fromItem)
	}
	var _toRule []interface{}
	for _, _toItem := range _to {
		_toRule = append(_toRule, _toItem)
	}

	var _messageHashRule []interface{}
	for _, _messageHashItem := range _messageHash {
		_messageHashRule = append(_messageHashRule, _messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "MessageSent", _fromRule, _toRule, _messageHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiMessageSent)
				if err := _Lineaabi.contract.UnpackLog(event, "MessageSent", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMessageSent is a log parse operation binding the contract event 0xe856c2b8bd4eb0027ce32eeaf595c21b0b6b4644b326e5b7bd80a1cf8db72e6c.
//
// Solidity: event MessageSent(address indexed _from, address indexed _to, uint256 _fee, uint256 _value, uint256 _nonce, bytes _calldata, bytes32 indexed _messageHash)
func (_Lineaabi *LineaabiFilterer) ParseMessageSent(log types.Log) (*LineaabiMessageSent, error) {
	event := new(LineaabiMessageSent)
	if err := _Lineaabi.contract.UnpackLog(event, "MessageSent", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the Lineaabi contract.
type LineaabiPausedIterator struct {
	Event *LineaabiPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiPaused represents a Paused event raised by the Lineaabi contract.
type LineaabiPaused struct {
	MessageSender common.Address
	PauseType     *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0xab40a374bc51de372200a8bc981af8c9ecdc08dfdaef0bb6e09f88f3c616ef3d.
//
// Solidity: event Paused(address messageSender, uint256 indexed pauseType)
func (_Lineaabi *LineaabiFilterer) FilterPaused(opts *bind.FilterOpts, pauseType []*big.Int) (*LineaabiPausedIterator, error) {

	var pauseTypeRule []interface{}
	for _, pauseTypeItem := range pauseType {
		pauseTypeRule = append(pauseTypeRule, pauseTypeItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "Paused", pauseTypeRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiPausedIterator{contract: _Lineaabi.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0xab40a374bc51de372200a8bc981af8c9ecdc08dfdaef0bb6e09f88f3c616ef3d.
//
// Solidity: event Paused(address messageSender, uint256 indexed pauseType)
func (_Lineaabi *LineaabiFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *LineaabiPaused, pauseType []*big.Int) (event.Subscription, error) {

	var pauseTypeRule []interface{}
	for _, pauseTypeItem := range pauseType {
		pauseTypeRule = append(pauseTypeRule, pauseTypeItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "Paused", pauseTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiPaused)
				if err := _Lineaabi.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0xab40a374bc51de372200a8bc981af8c9ecdc08dfdaef0bb6e09f88f3c616ef3d.
//
// Solidity: event Paused(address messageSender, uint256 indexed pauseType)
func (_Lineaabi *LineaabiFilterer) ParsePaused(log types.Log) (*LineaabiPaused, error) {
	event := new(LineaabiPaused)
	if err := _Lineaabi.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiRateLimitInitializedIterator is returned from FilterRateLimitInitialized and is used to iterate over the raw logs and unpacked data for RateLimitInitialized events raised by the Lineaabi contract.
type LineaabiRateLimitInitializedIterator struct {
	Event *LineaabiRateLimitInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiRateLimitInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiRateLimitInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiRateLimitInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiRateLimitInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiRateLimitInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiRateLimitInitialized represents a RateLimitInitialized event raised by the Lineaabi contract.
type LineaabiRateLimitInitialized struct {
	PeriodInSeconds  *big.Int
	LimitInWei       *big.Int
	CurrentPeriodEnd *big.Int
	Raw              types.Log // Blockchain specific contextual infos
}

// FilterRateLimitInitialized is a free log retrieval operation binding the contract event 0x8f805c372b66240792580418b7328c0c554ae235f0932475c51b026887fe26a9.
//
// Solidity: event RateLimitInitialized(uint256 periodInSeconds, uint256 limitInWei, uint256 currentPeriodEnd)
func (_Lineaabi *LineaabiFilterer) FilterRateLimitInitialized(opts *bind.FilterOpts) (*LineaabiRateLimitInitializedIterator, error) {

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "RateLimitInitialized")
	if err != nil {
		return nil, err
	}
	return &LineaabiRateLimitInitializedIterator{contract: _Lineaabi.contract, event: "RateLimitInitialized", logs: logs, sub: sub}, nil
}

// WatchRateLimitInitialized is a free log subscription operation binding the contract event 0x8f805c372b66240792580418b7328c0c554ae235f0932475c51b026887fe26a9.
//
// Solidity: event RateLimitInitialized(uint256 periodInSeconds, uint256 limitInWei, uint256 currentPeriodEnd)
func (_Lineaabi *LineaabiFilterer) WatchRateLimitInitialized(opts *bind.WatchOpts, sink chan<- *LineaabiRateLimitInitialized) (event.Subscription, error) {

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "RateLimitInitialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiRateLimitInitialized)
				if err := _Lineaabi.contract.UnpackLog(event, "RateLimitInitialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRateLimitInitialized is a log parse operation binding the contract event 0x8f805c372b66240792580418b7328c0c554ae235f0932475c51b026887fe26a9.
//
// Solidity: event RateLimitInitialized(uint256 periodInSeconds, uint256 limitInWei, uint256 currentPeriodEnd)
func (_Lineaabi *LineaabiFilterer) ParseRateLimitInitialized(log types.Log) (*LineaabiRateLimitInitialized, error) {
	event := new(LineaabiRateLimitInitialized)
	if err := _Lineaabi.contract.UnpackLog(event, "RateLimitInitialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiRoleAdminChangedIterator is returned from FilterRoleAdminChanged and is used to iterate over the raw logs and unpacked data for RoleAdminChanged events raised by the Lineaabi contract.
type LineaabiRoleAdminChangedIterator struct {
	Event *LineaabiRoleAdminChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiRoleAdminChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiRoleAdminChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiRoleAdminChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiRoleAdminChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiRoleAdminChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiRoleAdminChanged represents a RoleAdminChanged event raised by the Lineaabi contract.
type LineaabiRoleAdminChanged struct {
	Role              [32]byte
	PreviousAdminRole [32]byte
	NewAdminRole      [32]byte
	Raw               types.Log // Blockchain specific contextual infos
}

// FilterRoleAdminChanged is a free log retrieval operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Lineaabi *LineaabiFilterer) FilterRoleAdminChanged(opts *bind.FilterOpts, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (*LineaabiRoleAdminChangedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiRoleAdminChangedIterator{contract: _Lineaabi.contract, event: "RoleAdminChanged", logs: logs, sub: sub}, nil
}

// WatchRoleAdminChanged is a free log subscription operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Lineaabi *LineaabiFilterer) WatchRoleAdminChanged(opts *bind.WatchOpts, sink chan<- *LineaabiRoleAdminChanged, role [][32]byte, previousAdminRole [][32]byte, newAdminRole [][32]byte) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var previousAdminRoleRule []interface{}
	for _, previousAdminRoleItem := range previousAdminRole {
		previousAdminRoleRule = append(previousAdminRoleRule, previousAdminRoleItem)
	}
	var newAdminRoleRule []interface{}
	for _, newAdminRoleItem := range newAdminRole {
		newAdminRoleRule = append(newAdminRoleRule, newAdminRoleItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "RoleAdminChanged", roleRule, previousAdminRoleRule, newAdminRoleRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiRoleAdminChanged)
				if err := _Lineaabi.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleAdminChanged is a log parse operation binding the contract event 0xbd79b86ffe0ab8e8776151514217cd7cacd52c909f66475c3af44e129f0b00ff.
//
// Solidity: event RoleAdminChanged(bytes32 indexed role, bytes32 indexed previousAdminRole, bytes32 indexed newAdminRole)
func (_Lineaabi *LineaabiFilterer) ParseRoleAdminChanged(log types.Log) (*LineaabiRoleAdminChanged, error) {
	event := new(LineaabiRoleAdminChanged)
	if err := _Lineaabi.contract.UnpackLog(event, "RoleAdminChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiRoleGrantedIterator is returned from FilterRoleGranted and is used to iterate over the raw logs and unpacked data for RoleGranted events raised by the Lineaabi contract.
type LineaabiRoleGrantedIterator struct {
	Event *LineaabiRoleGranted // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiRoleGrantedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiRoleGranted)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiRoleGranted)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiRoleGrantedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiRoleGrantedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiRoleGranted represents a RoleGranted event raised by the Lineaabi contract.
type LineaabiRoleGranted struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleGranted is a free log retrieval operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Lineaabi *LineaabiFilterer) FilterRoleGranted(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*LineaabiRoleGrantedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiRoleGrantedIterator{contract: _Lineaabi.contract, event: "RoleGranted", logs: logs, sub: sub}, nil
}

// WatchRoleGranted is a free log subscription operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Lineaabi *LineaabiFilterer) WatchRoleGranted(opts *bind.WatchOpts, sink chan<- *LineaabiRoleGranted, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "RoleGranted", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiRoleGranted)
				if err := _Lineaabi.contract.UnpackLog(event, "RoleGranted", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleGranted is a log parse operation binding the contract event 0x2f8788117e7eff1d82e926ec794901d17c78024a50270940304540a733656f0d.
//
// Solidity: event RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)
func (_Lineaabi *LineaabiFilterer) ParseRoleGranted(log types.Log) (*LineaabiRoleGranted, error) {
	event := new(LineaabiRoleGranted)
	if err := _Lineaabi.contract.UnpackLog(event, "RoleGranted", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiRoleRevokedIterator is returned from FilterRoleRevoked and is used to iterate over the raw logs and unpacked data for RoleRevoked events raised by the Lineaabi contract.
type LineaabiRoleRevokedIterator struct {
	Event *LineaabiRoleRevoked // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiRoleRevokedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiRoleRevoked)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiRoleRevoked)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiRoleRevokedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiRoleRevokedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiRoleRevoked represents a RoleRevoked event raised by the Lineaabi contract.
type LineaabiRoleRevoked struct {
	Role    [32]byte
	Account common.Address
	Sender  common.Address
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterRoleRevoked is a free log retrieval operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Lineaabi *LineaabiFilterer) FilterRoleRevoked(opts *bind.FilterOpts, role [][32]byte, account []common.Address, sender []common.Address) (*LineaabiRoleRevokedIterator, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiRoleRevokedIterator{contract: _Lineaabi.contract, event: "RoleRevoked", logs: logs, sub: sub}, nil
}

// WatchRoleRevoked is a free log subscription operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Lineaabi *LineaabiFilterer) WatchRoleRevoked(opts *bind.WatchOpts, sink chan<- *LineaabiRoleRevoked, role [][32]byte, account []common.Address, sender []common.Address) (event.Subscription, error) {

	var roleRule []interface{}
	for _, roleItem := range role {
		roleRule = append(roleRule, roleItem)
	}
	var accountRule []interface{}
	for _, accountItem := range account {
		accountRule = append(accountRule, accountItem)
	}
	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "RoleRevoked", roleRule, accountRule, senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiRoleRevoked)
				if err := _Lineaabi.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRoleRevoked is a log parse operation binding the contract event 0xf6391f5c32d9c69d2a47ea670b442974b53935d1edc7fd64eb21e047a839171b.
//
// Solidity: event RoleRevoked(bytes32 indexed role, address indexed account, address indexed sender)
func (_Lineaabi *LineaabiFilterer) ParseRoleRevoked(log types.Log) (*LineaabiRoleRevoked, error) {
	event := new(LineaabiRoleRevoked)
	if err := _Lineaabi.contract.UnpackLog(event, "RoleRevoked", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiRollingHashUpdatedIterator is returned from FilterRollingHashUpdated and is used to iterate over the raw logs and unpacked data for RollingHashUpdated events raised by the Lineaabi contract.
type LineaabiRollingHashUpdatedIterator struct {
	Event *LineaabiRollingHashUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiRollingHashUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiRollingHashUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiRollingHashUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiRollingHashUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiRollingHashUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiRollingHashUpdated represents a RollingHashUpdated event raised by the Lineaabi contract.
type LineaabiRollingHashUpdated struct {
	MessageNumber *big.Int
	RollingHash   [32]byte
	MessageHash   [32]byte
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterRollingHashUpdated is a free log retrieval operation binding the contract event 0xea3b023b4c8680d4b4824f0143132c95476359a2bb70a81d6c5a36f6918f6339.
//
// Solidity: event RollingHashUpdated(uint256 indexed messageNumber, bytes32 indexed rollingHash, bytes32 indexed messageHash)
func (_Lineaabi *LineaabiFilterer) FilterRollingHashUpdated(opts *bind.FilterOpts, messageNumber []*big.Int, rollingHash [][32]byte, messageHash [][32]byte) (*LineaabiRollingHashUpdatedIterator, error) {

	var messageNumberRule []interface{}
	for _, messageNumberItem := range messageNumber {
		messageNumberRule = append(messageNumberRule, messageNumberItem)
	}
	var rollingHashRule []interface{}
	for _, rollingHashItem := range rollingHash {
		rollingHashRule = append(rollingHashRule, rollingHashItem)
	}
	var messageHashRule []interface{}
	for _, messageHashItem := range messageHash {
		messageHashRule = append(messageHashRule, messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "RollingHashUpdated", messageNumberRule, rollingHashRule, messageHashRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiRollingHashUpdatedIterator{contract: _Lineaabi.contract, event: "RollingHashUpdated", logs: logs, sub: sub}, nil
}

// WatchRollingHashUpdated is a free log subscription operation binding the contract event 0xea3b023b4c8680d4b4824f0143132c95476359a2bb70a81d6c5a36f6918f6339.
//
// Solidity: event RollingHashUpdated(uint256 indexed messageNumber, bytes32 indexed rollingHash, bytes32 indexed messageHash)
func (_Lineaabi *LineaabiFilterer) WatchRollingHashUpdated(opts *bind.WatchOpts, sink chan<- *LineaabiRollingHashUpdated, messageNumber []*big.Int, rollingHash [][32]byte, messageHash [][32]byte) (event.Subscription, error) {

	var messageNumberRule []interface{}
	for _, messageNumberItem := range messageNumber {
		messageNumberRule = append(messageNumberRule, messageNumberItem)
	}
	var rollingHashRule []interface{}
	for _, rollingHashItem := range rollingHash {
		rollingHashRule = append(rollingHashRule, rollingHashItem)
	}
	var messageHashRule []interface{}
	for _, messageHashItem := range messageHash {
		messageHashRule = append(messageHashRule, messageHashItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "RollingHashUpdated", messageNumberRule, rollingHashRule, messageHashRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiRollingHashUpdated)
				if err := _Lineaabi.contract.UnpackLog(event, "RollingHashUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRollingHashUpdated is a log parse operation binding the contract event 0xea3b023b4c8680d4b4824f0143132c95476359a2bb70a81d6c5a36f6918f6339.
//
// Solidity: event RollingHashUpdated(uint256 indexed messageNumber, bytes32 indexed rollingHash, bytes32 indexed messageHash)
func (_Lineaabi *LineaabiFilterer) ParseRollingHashUpdated(log types.Log) (*LineaabiRollingHashUpdated, error) {
	event := new(LineaabiRollingHashUpdated)
	if err := _Lineaabi.contract.UnpackLog(event, "RollingHashUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiSystemMigrationBlockInitializedIterator is returned from FilterSystemMigrationBlockInitialized and is used to iterate over the raw logs and unpacked data for SystemMigrationBlockInitialized events raised by the Lineaabi contract.
type LineaabiSystemMigrationBlockInitializedIterator struct {
	Event *LineaabiSystemMigrationBlockInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiSystemMigrationBlockInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiSystemMigrationBlockInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiSystemMigrationBlockInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiSystemMigrationBlockInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiSystemMigrationBlockInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiSystemMigrationBlockInitialized represents a SystemMigrationBlockInitialized event raised by the Lineaabi contract.
type LineaabiSystemMigrationBlockInitialized struct {
	SystemMigrationBlock *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterSystemMigrationBlockInitialized is a free log retrieval operation binding the contract event 0x405b3b16b9190c1e995514c13ab4e8e7d895d9103e91c3a8c8f12df6cd50aa2c.
//
// Solidity: event SystemMigrationBlockInitialized(uint256 systemMigrationBlock)
func (_Lineaabi *LineaabiFilterer) FilterSystemMigrationBlockInitialized(opts *bind.FilterOpts) (*LineaabiSystemMigrationBlockInitializedIterator, error) {

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "SystemMigrationBlockInitialized")
	if err != nil {
		return nil, err
	}
	return &LineaabiSystemMigrationBlockInitializedIterator{contract: _Lineaabi.contract, event: "SystemMigrationBlockInitialized", logs: logs, sub: sub}, nil
}

// WatchSystemMigrationBlockInitialized is a free log subscription operation binding the contract event 0x405b3b16b9190c1e995514c13ab4e8e7d895d9103e91c3a8c8f12df6cd50aa2c.
//
// Solidity: event SystemMigrationBlockInitialized(uint256 systemMigrationBlock)
func (_Lineaabi *LineaabiFilterer) WatchSystemMigrationBlockInitialized(opts *bind.WatchOpts, sink chan<- *LineaabiSystemMigrationBlockInitialized) (event.Subscription, error) {

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "SystemMigrationBlockInitialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiSystemMigrationBlockInitialized)
				if err := _Lineaabi.contract.UnpackLog(event, "SystemMigrationBlockInitialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseSystemMigrationBlockInitialized is a log parse operation binding the contract event 0x405b3b16b9190c1e995514c13ab4e8e7d895d9103e91c3a8c8f12df6cd50aa2c.
//
// Solidity: event SystemMigrationBlockInitialized(uint256 systemMigrationBlock)
func (_Lineaabi *LineaabiFilterer) ParseSystemMigrationBlockInitialized(log types.Log) (*LineaabiSystemMigrationBlockInitialized, error) {
	event := new(LineaabiSystemMigrationBlockInitialized)
	if err := _Lineaabi.contract.UnpackLog(event, "SystemMigrationBlockInitialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiUnPausedIterator is returned from FilterUnPaused and is used to iterate over the raw logs and unpacked data for UnPaused events raised by the Lineaabi contract.
type LineaabiUnPausedIterator struct {
	Event *LineaabiUnPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiUnPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiUnPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiUnPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiUnPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiUnPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiUnPaused represents a UnPaused event raised by the Lineaabi contract.
type LineaabiUnPaused struct {
	MessageSender common.Address
	PauseType     *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterUnPaused is a free log retrieval operation binding the contract event 0xef04ba2036ccaeab3a59717b51d2b9146b0b0904077177f1148a5418bf1eae23.
//
// Solidity: event UnPaused(address messageSender, uint256 indexed pauseType)
func (_Lineaabi *LineaabiFilterer) FilterUnPaused(opts *bind.FilterOpts, pauseType []*big.Int) (*LineaabiUnPausedIterator, error) {

	var pauseTypeRule []interface{}
	for _, pauseTypeItem := range pauseType {
		pauseTypeRule = append(pauseTypeRule, pauseTypeItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "UnPaused", pauseTypeRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiUnPausedIterator{contract: _Lineaabi.contract, event: "UnPaused", logs: logs, sub: sub}, nil
}

// WatchUnPaused is a free log subscription operation binding the contract event 0xef04ba2036ccaeab3a59717b51d2b9146b0b0904077177f1148a5418bf1eae23.
//
// Solidity: event UnPaused(address messageSender, uint256 indexed pauseType)
func (_Lineaabi *LineaabiFilterer) WatchUnPaused(opts *bind.WatchOpts, sink chan<- *LineaabiUnPaused, pauseType []*big.Int) (event.Subscription, error) {

	var pauseTypeRule []interface{}
	for _, pauseTypeItem := range pauseType {
		pauseTypeRule = append(pauseTypeRule, pauseTypeItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "UnPaused", pauseTypeRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiUnPaused)
				if err := _Lineaabi.contract.UnpackLog(event, "UnPaused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseUnPaused is a log parse operation binding the contract event 0xef04ba2036ccaeab3a59717b51d2b9146b0b0904077177f1148a5418bf1eae23.
//
// Solidity: event UnPaused(address messageSender, uint256 indexed pauseType)
func (_Lineaabi *LineaabiFilterer) ParseUnPaused(log types.Log) (*LineaabiUnPaused, error) {
	event := new(LineaabiUnPaused)
	if err := _Lineaabi.contract.UnpackLog(event, "UnPaused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// LineaabiVerifierAddressChangedIterator is returned from FilterVerifierAddressChanged and is used to iterate over the raw logs and unpacked data for VerifierAddressChanged events raised by the Lineaabi contract.
type LineaabiVerifierAddressChangedIterator struct {
	Event *LineaabiVerifierAddressChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LineaabiVerifierAddressChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LineaabiVerifierAddressChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LineaabiVerifierAddressChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LineaabiVerifierAddressChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LineaabiVerifierAddressChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LineaabiVerifierAddressChanged represents a VerifierAddressChanged event raised by the Lineaabi contract.
type LineaabiVerifierAddressChanged struct {
	VerifierAddress    common.Address
	ProofType          *big.Int
	VerifierSetBy      common.Address
	OldVerifierAddress common.Address
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterVerifierAddressChanged is a free log retrieval operation binding the contract event 0x4a29db3fc6b42bda201e4b4d69ce8d575eeeba5f153509c0d0a342af0f1bd021.
//
// Solidity: event VerifierAddressChanged(address indexed verifierAddress, uint256 indexed proofType, address indexed verifierSetBy, address oldVerifierAddress)
func (_Lineaabi *LineaabiFilterer) FilterVerifierAddressChanged(opts *bind.FilterOpts, verifierAddress []common.Address, proofType []*big.Int, verifierSetBy []common.Address) (*LineaabiVerifierAddressChangedIterator, error) {

	var verifierAddressRule []interface{}
	for _, verifierAddressItem := range verifierAddress {
		verifierAddressRule = append(verifierAddressRule, verifierAddressItem)
	}
	var proofTypeRule []interface{}
	for _, proofTypeItem := range proofType {
		proofTypeRule = append(proofTypeRule, proofTypeItem)
	}
	var verifierSetByRule []interface{}
	for _, verifierSetByItem := range verifierSetBy {
		verifierSetByRule = append(verifierSetByRule, verifierSetByItem)
	}

	logs, sub, err := _Lineaabi.contract.FilterLogs(opts, "VerifierAddressChanged", verifierAddressRule, proofTypeRule, verifierSetByRule)
	if err != nil {
		return nil, err
	}
	return &LineaabiVerifierAddressChangedIterator{contract: _Lineaabi.contract, event: "VerifierAddressChanged", logs: logs, sub: sub}, nil
}

// WatchVerifierAddressChanged is a free log subscription operation binding the contract event 0x4a29db3fc6b42bda201e4b4d69ce8d575eeeba5f153509c0d0a342af0f1bd021.
//
// Solidity: event VerifierAddressChanged(address indexed verifierAddress, uint256 indexed proofType, address indexed verifierSetBy, address oldVerifierAddress)
func (_Lineaabi *LineaabiFilterer) WatchVerifierAddressChanged(opts *bind.WatchOpts, sink chan<- *LineaabiVerifierAddressChanged, verifierAddress []common.Address, proofType []*big.Int, verifierSetBy []common.Address) (event.Subscription, error) {

	var verifierAddressRule []interface{}
	for _, verifierAddressItem := range verifierAddress {
		verifierAddressRule = append(verifierAddressRule, verifierAddressItem)
	}
	var proofTypeRule []interface{}
	for _, proofTypeItem := range proofType {
		proofTypeRule = append(proofTypeRule, proofTypeItem)
	}
	var verifierSetByRule []interface{}
	for _, verifierSetByItem := range verifierSetBy {
		verifierSetByRule = append(verifierSetByRule, verifierSetByItem)
	}

	logs, sub, err := _Lineaabi.contract.WatchLogs(opts, "VerifierAddressChanged", verifierAddressRule, proofTypeRule, verifierSetByRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LineaabiVerifierAddressChanged)
				if err := _Lineaabi.contract.UnpackLog(event, "VerifierAddressChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseVerifierAddressChanged is a log parse operation binding the contract event 0x4a29db3fc6b42bda201e4b4d69ce8d575eeeba5f153509c0d0a342af0f1bd021.
//
// Solidity: event VerifierAddressChanged(address indexed verifierAddress, uint256 indexed proofType, address indexed verifierSetBy, address oldVerifierAddress)
func (_Lineaabi *LineaabiFilterer) ParseVerifierAddressChanged(log types.Log) (*LineaabiVerifierAddressChanged, error) {
	event := new(LineaabiVerifierAddressChanged)
	if err := _Lineaabi.contract.UnpackLog(event, "VerifierAddressChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
