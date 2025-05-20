package txverifier

// TODO
// Change constant naming convention to PascalCase (maybe goimports can do this automatically)
// Can the actual ethCalls be factored into their own function?

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"math/big"
	"time"

	connectors "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Event Signatures
const (
	// LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);
	EVENTHASH_WORMHOLE_LOG_MESSAGE_PUBLISHED = "0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2"
	// Transfer(address,address,uint256)
	EVENTHASH_ERC20_TRANSFER = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	// Deposit(address,uint256)
	EVENTHASH_WETH_DEPOSIT = "0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c"
)

// Function signatures
var (
	// wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) => 0x1ff1e286
	TOKEN_BRIDGE_WRAPPED_ASSET_SIGNATURE = []byte("\x1f\xf1\xe2\x86")
	// isWrappedAsset(address token) => 0x1a2be4da
	TOKEN_BRIDGE_IS_WRAPPED_ASSET_SIGNATURE = []byte("\x1a\x2b\xe4\xda")
	// decimals() => 0x313ce567
	ERC20_DECIMALS_SIGNATURE = []byte("\x31\x3c\xe5\x67")
	// chainId() => 0x9a8a0592
	WRAPPED_ERC20_CHAIN_ID_SIGNATURE = []byte("\x9a\x8a\x05\x92")
)

// Fixed addresses
var (
	// https://etherscan.io/token/0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2
	ZERO_ADDRESS     = common.BytesToAddress([]byte{0x00})
	ZERO_ADDRESS_VAA = VAAAddrFrom(ZERO_ADDRESS)
)

// EVM chain constants
const (
	// EVM uses 32 bytes for words. Note that vaa.Address is an alias for a slice of 32 bytes
	EVM_WORD_LENGTH = 32
	// The expected total number of indexed topics for an ERC20 Transfer event
	TOPICS_COUNT_TRANSFER = 3
	// The expected total number of indexed topics for a WETH Deposit event
	TOPICS_COUNT_DEPOSIT = 2
)

const (
	RPC_TIMEOUT = 10 * time.Second
)

// Important addresses for Transfer Verification.
type TVAddresses struct {
	CoreBridgeAddr common.Address
	// Address of the Wormhole token bridge contract for this chain
	TokenBridgeAddr common.Address
	// Wrapped version of the native asset, e.g. WETH for Ethereum
	WrappedNativeAddr common.Address
}

// Stores the EVM chain ID and corresponding Wormhole chain ID for the current chain being monitored by the connector.
type chainIds struct {
	evmChainId      uint64
	wormholeChainId vaa.ChainID
}

type TransferVerifierInterface interface {
	ProcessEvent(ctx context.Context, txHash common.Hash, receipt *types.Receipt) bool
	Addrs() *TVAddresses
}

func (tv *TransferVerifier[ethClient, Connector]) Addrs() *TVAddresses {
	return tv.Addresses
}

// TransferVerifier contains configuration values for verifying transfers.
type TransferVerifier[E evmClient, C connector] struct {
	Addresses *TVAddresses
	// The chainId being monitored as reported by the client connector.
	chainIds *chainIds
	// Wormhole connector for wrapping contract-specific interactions
	logger zap.Logger
	// Corresponds to the connector interface for EVM chains
	evmConnector C
	// Corresponds to an ethClient from go-ethereum
	client E
	// Mapping to track the transactions that have been processed. Indexed by a log's txHash.
	processedTransactions map[common.Hash]*types.Receipt
	// The latest transaction block number, used to determine the size of historic receipts to keep in memory.
	lastBlockNumber uint64
	// The block height difference between the latest block and the oldest block to keep in memory.
	pruneHeightDelta uint64

	// Holds previously-recorded decimals (uint8) for token addresses
	// (common.Address) that have been observed.
	decimalsCache map[common.Address]uint8

	// Records whether an asset is wrapped but does not store the native data
	isWrappedCache map[string]bool

	// Maps the 32-byte token addresses received via LogMessagePublished
	// events to their unwrapped 20-byte addresses. This mapping is also
	// used for non-wrapped token addresses.
	wrappedCache map[string]common.Address

	// Native chain cache for wrapped assets.
	nativeChainCache map[string]vaa.ChainID
}

func NewTransferVerifier(ctx context.Context, connector connectors.Connector, tvAddrs *TVAddresses, pruneHeightDelta uint64, logger *zap.Logger) (*TransferVerifier[*ethClient.Client, connectors.Connector], error) {
	// Retrieve the chainId from the connector.
	chainIdFromClient, err := connector.Client().ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Fetch EVM chain ID from the connector and attempt to convert it to a Wormhole chain ID.
	evmChainId, parseErr := strconv.ParseUint(chainIdFromClient.String(), 10, 16)
	if parseErr != nil {
		return nil, fmt.Errorf("Failed to parse chainId from string returned by connector client: %w", parseErr)
	}

	wormholeChainId, unregisteredErr := TryWormholeChainIdFromNative(evmChainId)
	if unregisteredErr != nil {
		return nil, fmt.Errorf("Could not get Wormhole chain ID from EVM chain ID: %w", unregisteredErr)
	}

	return &TransferVerifier[*ethClient.Client, connectors.Connector]{
		Addresses: tvAddrs,
		chainIds: &chainIds{
			evmChainId:      evmChainId,
			wormholeChainId: wormholeChainId,
		},
		logger:                *logger,
		evmConnector:          connector,
		client:                connector.Client(),
		processedTransactions: make(map[common.Hash]*types.Receipt),
		lastBlockNumber:       0,
		pruneHeightDelta:      pruneHeightDelta,
		decimalsCache:         make(map[common.Address]uint8),
		isWrappedCache:        make(map[string]bool),
		wrappedCache:          make(map[string]common.Address),
		nativeChainCache:      make(map[string]vaa.ChainID),
	}, nil
}

type connector interface {
	ParseLogMessagePublished(log types.Log) (*ethabi.AbiLogMessagePublished, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
}

type evmClient interface {
	// getDecimals()
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

type Subscription struct {
	// TODO make generic or use an interface
	client    *ethClient.Client
	connector connectors.Connector
	logC      chan *ethabi.AbiLogMessagePublished
	errC      chan error
	quit      chan struct{}
}

func NewSubscription(client *ethClient.Client, connector connectors.Connector) *Subscription {
	return &Subscription{
		client:    client,
		connector: connector,
		logC:      make(chan *ethabi.AbiLogMessagePublished),
		errC:      make(chan error),
		quit:      make(chan struct{}),
	}
}

// Subscribe creates a subscription to WatchLogMessagePublished events and will
// attempt to reconnect when errors occur, such as Websocket connection
// problems.
func (s *Subscription) Subscribe(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.quit:
				return
			default:
				subscription, err := s.connector.WatchLogMessagePublished(
					ctx,
					s.errC,
					s.logC,
				)

				if err != nil {
					s.errC <- fmt.Errorf("failed to subscribe to logs: %w", err)
					time.Sleep(RECONNECT_DELAY) // Wait before retrying
					continue
				}

				// Handle subscription until error occurs
				// TODO: This section of code should have a limit on the number of times it will retry
				// and fail if it can't connect a certain number of times
				err = s.handleSubscription(ctx, subscription)

				if err != nil {
					s.errC <- err
					time.Sleep(RECONNECT_DELAY) // Wait before retrying
				}
			}
		}
	}()
}

func (s *Subscription) handleSubscription(ctx context.Context, subscription event.Subscription) error {
	for {
		select {
		case <-ctx.Done():
			subscription.Unsubscribe()
			return nil
		case <-s.quit:
			subscription.Unsubscribe()
			return nil
		case err := <-subscription.Err():
			subscription.Unsubscribe()
			return fmt.Errorf("subscription error: %w", err)
		}
	}
}

func (s *Subscription) Events() <-chan *ethabi.AbiLogMessagePublished {
	return s.logC
}

func (s *Subscription) Errors() <-chan error {
	return s.errC
}

func (s *Subscription) Close() {
	close(s.quit)
}

// Abstraction over the fields that are expected to be present for Transfer
// types encoded in receipt logs: Deposits, Transfers, and LogMessagePublished
// events.
type TransferLog interface {
	// Amount after (de)normalization
	TransferAmount() *big.Int
	// The Transferror: EOA or contract that initiated the transfer. Not to be confused with msg.sender.
	Sender() vaa.Address
	// The Transferee. Ultimate recipient of funds.
	Destination() vaa.Address
	// Event emitter
	Emitter() common.Address // Emitter will always be an Ethereum address
	// Chain where the token was minted
	OriginChain() vaa.ChainID
	// Address that minted the token
	OriginAddress() vaa.Address
}

// Abstraction over a Deposit event for a wrapped native asset, e.g. WETH for Ethereum.
type NativeDeposit struct {
	// The address of the token.
	TokenAddress common.Address
	// The native chain of the token (where it was minted)
	TokenChain vaa.ChainID
	Receiver   common.Address
	Amount     *big.Int
}

func (d *NativeDeposit) TransferAmount() *big.Int {
	return d.Amount
}

func (d *NativeDeposit) Destination() vaa.Address {
	return VAAAddrFrom(d.Receiver)
}

// Deposit does not actually have a sender but this is required to implement the interface
func (d *NativeDeposit) Sender() vaa.Address {
	// Sender is not present in the Logs emitted for a Deposit
	return ZERO_ADDRESS_VAA
}

func (d *NativeDeposit) Emitter() common.Address {
	// Event emitter of the Deposit should be equal to TokenAddress.
	return d.TokenAddress
}

func (d *NativeDeposit) OriginChain() vaa.ChainID {
	return d.TokenChain
}

func (d *NativeDeposit) OriginAddress() vaa.Address {
	return VAAAddrFrom(d.TokenAddress)
}

func (d *NativeDeposit) String() string {
	return fmt.Sprintf(
		"Deposit: {TokenAddress=%s TokenChain=%d Receiver=%s Amount=%s}",
		d.TokenAddress.String(),
		d.TokenChain,
		d.Receiver.String(),
		d.Amount.String(),
	)
}

// DepositFromLog() creates a NativeDeposit struct given a log and Wormhole chain ID.
func DepositFromLog(
	log *types.Log,
	// This chain ID should correspond to the Wormhole chain ID, not the EVM chain ID. In this context it's
	// important to track the transfer as Wormhole sees it, not as the EVM network itself sees it.
	chainId vaa.ChainID,
) (deposit *NativeDeposit, err error) {
	dest, amount := parseWNativeDepositEvent(log.Topics, log.Data)

	if amount == nil {
		return deposit, errors.New("could not parse Deposit from log")
	}

	deposit = &NativeDeposit{
		TokenAddress: log.Address,
		TokenChain:   chainId,
		Receiver:     dest,
		Amount:       amount,
	}
	return
}

// parseWNativeDepositEvent parses an event for a deposit of a wrapped version of the chain's native asset, i.e. WETH for Ethereum.
func parseWNativeDepositEvent(logTopics []common.Hash, logData []byte) (destination common.Address, amount *big.Int) {

	// https://etherscan.io/token/0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2#code#L29
	// event  Deposit(address indexed dst, uint wad);
	if len(logData) != EVM_WORD_LENGTH || len(logTopics) != TOPICS_COUNT_DEPOSIT {
		return common.Address{}, nil
	}

	destination = common.BytesToAddress(logTopics[1][:])
	amount = new(big.Int).SetBytes(logData[:])

	return destination, amount
}

// Abstraction over an ERC20 Transfer event.
type ERC20Transfer struct {
	// The address of the token. Also equivalent to the Emitter of the event.
	TokenAddress common.Address
	// The native chain of the token (where it was minted)
	TokenChain vaa.ChainID
	From       common.Address
	To         common.Address
	Amount     *big.Int
}

func (t *ERC20Transfer) TransferAmount() *big.Int {
	return t.Amount
}

func (t *ERC20Transfer) Sender() vaa.Address {
	// Note that this value may return zero for receipt logs that are in
	// fact Transfers emitted from e.g. UniswapV2 which have the same event
	// signature as ERC20 Transfers.
	return VAAAddrFrom(t.From)
}

func (t *ERC20Transfer) Destination() vaa.Address {
	// Note that this value may return zero when tokens are being burned.
	return VAAAddrFrom(t.To)
}

func (t *ERC20Transfer) Emitter() common.Address {
	// The TokenAddress is equal to the Emitter for ERC20 Transfers
	return t.TokenAddress
}

func (t *ERC20Transfer) OriginChain() vaa.ChainID {
	return t.TokenChain
}

func (t *ERC20Transfer) OriginAddress() vaa.Address {
	return VAAAddrFrom(t.TokenAddress)
}

func (t *ERC20Transfer) String() string {
	return fmt.Sprintf(
		"ERC20Transfer: {TokenAddress=%s TokenChain=%d From=%s To=%s Amount=%s}",
		t.TokenAddress.String(),
		t.TokenChain,
		t.From.String(),
		t.To.String(),
		t.Amount.String(),
	)
}

// ERC20TransferFromLog() creates an ERC20Transfer struct given a log and Wormhole chain ID.
func ERC20TransferFromLog(
	log *types.Log,
	// This chain ID should correspond to the Wormhole chain ID, not the EVM chain ID. In this context it's
	// important to track the transfer as Wormhole sees it, not as the EVM network itself sees it.
	chainId vaa.ChainID,
) (transfer *ERC20Transfer, err error) {
	from, to, amount := parseERC20TransferEvent(log.Topics, log.Data)

	// NOTE: When minting tokens, some ERC20 implementations will emit a
	// Transfer event that has the From field set to the zero address.
	// Similarly, burn operations may set the To field to the zero address.
	// However, there shouldn't be a case where both fields are equal to
	// the zero address.
	if to == ZERO_ADDRESS && from == ZERO_ADDRESS {
		return nil, errors.New("could not parse ERC20 Transfer from log: transfer's To and From fields are both zero")
	}

	if amount == nil {
		return nil, errors.New("could not parse ERC20 Transfer from log: nil Amount")
	}

	transfer = &ERC20Transfer{
		TokenAddress: log.Address,
		// Initially, set Token's chain to the chain being monitored. This will be updated by making an RPC call later.
		TokenChain: chainId,
		From:       from,
		To:         to,
		Amount:     amount,
	}
	return
}

// This function parses an ERC20 transfer event from a log topic and data.
// It verifies the input lengths, extracts the 'from', 'to' and amount fields from the log data,
// and returns these values as common.Address and big.Int types.
// - Error handling: The function checks if the log data and topic lengths are correct before attempting to parse them.
// - Input validation: The function verifies that the input lengths match expected values, preventing potential attacks or errors.
func parseERC20TransferEvent(logTopics []common.Hash, logData []byte) (from common.Address, to common.Address, amount *big.Int) {

	// https://github.com/OpenZeppelin/openzeppelin-contracts/blob/6e224307b44bc4bd0cb60d408844e028cfa3e485/contracts/token/ERC20/IERC20.sol#L16
	// event Transfer(address indexed from, address indexed to, uint256 value)
	if len(logData) != EVM_WORD_LENGTH || len(logTopics) != TOPICS_COUNT_TRANSFER {
		return common.Address{}, common.Address{}, nil
	}

	from = common.BytesToAddress(logTopics[1][:])
	to = common.BytesToAddress(logTopics[2][:])
	amount = new(big.Int).SetBytes(logData[:])

	return
}

// Abstraction over a LogMessagePublished event emitted by the core bridge.
// TODO add String() method
type LogMessagePublished struct {
	// Which contract emitted the event.
	EventEmitter common.Address
	// Which address sent the transaction that triggered the message publication.
	MsgSender common.Address
	// Abstraction over fields encoded in the event's Data field which in turn contains the transfer's payload.
	TransferDetails *TransferDetails
	// Note: these fields are non-exhaustive. Data not needed for Transfer Verification is not encoded here.
}

func (l *LogMessagePublished) String() string {
	return fmt.Sprintf("LogMessagePublished: {emitter=%s sender=%s transferDetails=%s}",
		l.EventEmitter,
		l.MsgSender,
		l.TransferDetails,
	)
}

func (l *LogMessagePublished) Destination() (destination vaa.Address) {
	if l.TransferDetails != nil {
		destination = l.TransferDetails.TargetAddress
	}
	return
}

func (l *LogMessagePublished) Emitter() common.Address {
	return l.EventEmitter
}

func (l *LogMessagePublished) Sender() vaa.Address {
	return VAAAddrFrom(l.MsgSender)
}

func (l *LogMessagePublished) TransferAmount() (amount *big.Int) {
	if l.TransferDetails != nil {
		return l.TransferDetails.Amount
	}
	return
}

func (l *LogMessagePublished) OriginAddress() (origin vaa.Address) {
	if l.TransferDetails != nil {
		origin = VAAAddrFrom(l.TransferDetails.OriginAddress)
	}
	return
}

func (l *LogMessagePublished) OriginChain() (chainID vaa.ChainID) {
	if l.TransferDetails != nil {
		chainID = l.TransferDetails.TokenChain
	}
	return
}

// TransferReceipt is an abstraction over an EVM transaction receipt for a
// Token Bridge transfer. It represents Deposit, Transfer, and
// LogMessagePublished events that can appear in a Receipt logs. Other event
// types are not represented by this program because they are not relevant for
// checking the invariants on transfers sent from the token bridge.
type TransferReceipt struct {
	Deposits  *[]*NativeDeposit
	Transfers *[]*ERC20Transfer
	// There must be at least one LogMessagePublished for a valid receipt.
	MessagePublications *[]*LogMessagePublished
}

// Validate ensures that a parsed TransferReceipt struct is well-formed (i.e.
// structurally valid, even if the semantic contents of the TransferReceipt
// would be evaluated as "bad" from a security perspective.
//
// Well-formed means:
// - The structs fields must not be nil.
// - The MessagePublications fields must have at least one element.
// As as result, this function should only be used near the end of parsing and processing
// when all the logs have been parsed and used to populate the TransferReceipt instance.
func (r *TransferReceipt) Validate() (err error) {
	if r.Deposits == nil {
		return errors.Join(err, errors.New("parsed receipt's Deposits field is nil"))
	}
	if r.Transfers == nil {
		return errors.Join(err, errors.New("parsed receipt's Transfers field is nil"))
	}
	if r.MessagePublications == nil {
		return errors.Join(err, errors.New("parsed receipt's MessagePublications field is nil"))
	}
	if len(*r.MessagePublications) == 0 {
		return errors.Join(err, errors.New("parsed receipt has no Message Publications"))
	}

	return
}

func (r *TransferReceipt) String() string {
	dStr := ""
	if r.Deposits != nil {
		for _, d := range *r.Deposits {
			if d != nil {
				dStr += d.String()
			}
		}
	}

	tStr := ""
	if r.Transfers != nil {
		for _, t := range *r.Transfers {
			if t != nil {
				tStr += t.String()
			}
		}
	}

	mStr := ""
	if r.MessagePublications != nil {
		for _, m := range *r.MessagePublications {
			if m != nil {
				mStr += m.String()
			}
		}
	}

	return fmt.Sprintf(
		"receipt: {deposits=%s transfers=%s messages=%s}",
		dStr,
		tStr,
		mStr,
	)
}

// Summary of a processed TransferReceipt. Contains information about relevant
// transfers requested in and out of the bridge.
type ReceiptSummary struct {
	// Number of LogMessagePublished events in the receipt
	logsProcessed int
	// The sum of tokens transferred into the Token Bridge contract.
	in map[string]*big.Int
	// The sum of tokens parsed from the core bridge's LogMessagePublished payload.
	out map[string]*big.Int
}

func NewReceiptSummary() *ReceiptSummary {
	return &ReceiptSummary{
		logsProcessed: 0,
		// The sum of tokens transferred into the Token Bridge contract.
		in: make(map[string]*big.Int),
		// The sum of tokens parsed from the core bridge's LogMessagePublished payload.
		out: make(map[string]*big.Int),
	}
}

func (s *ReceiptSummary) String() (outStr string) {

	ins := ""

	for key, amountIn := range s.in {
		ins += fmt.Sprintf("%s=%s", key, amountIn.String())
	}

	outs := ""
	for key, amountOut := range s.out {
		outs += fmt.Sprintf("%s=%s ", key, amountOut.String())
	}

	outStr = fmt.Sprintf(
		"receipt summary: logsProcessed=%d requestedIn={%s} requestedOut={%s}",
		s.logsProcessed,
		ins,
		outs,
	)
	return outStr
}

// https://wormhole.com/docs/learn/infrastructure/vaas/#payload-types
type VAAPayloadType uint8

const (
	TransferTokens            VAAPayloadType = 1
	TransferTokensWithPayload VAAPayloadType = 3
)

// Abstraction of a Token Bridge transfer payload encoded in the Data field of a LogMessagePublished event.
// It is meant to correspond to the API for Token Transfer messages as described in the Token Bridge whitepaper:
// https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0003_token_bridge.md#api--database-schema
type TransferDetails struct {
	PayloadType VAAPayloadType
	// Denormalized amount, accounting for decimal differences between contracts and chains
	Amount *big.Int
	// Amount as sent in the raw payload
	AmountRaw *big.Int
	// Original wormhole chain ID where the token was minted.
	TokenChain vaa.ChainID
	// Original address of the token when minted natively. Corresponds to the "unwrapped" address in the token bridge.
	OriginAddress common.Address
	// Raw token address parsed from the payload. May be wrapped.
	OriginAddressRaw []byte
	// Not necessarily an EVM address, so vaa.Address is used instead
	TargetAddress vaa.Address
}

func (td *TransferDetails) String() string {
	return fmt.Sprintf(
		"PayloadType: %d OriginAddressRaw(hex-encoded): %s TokenChain: %d OriginAddress: %s TargetAddress: %s AmountRaw: %s Amount: %s",
		td.PayloadType,
		fmt.Sprintf("%x", td.OriginAddressRaw),
		td.TokenChain,
		td.OriginAddress.String(),
		td.TargetAddress.String(),
		td.AmountRaw.String(),
		td.Amount.String(),
	)
}

// unwrapIfWrapped returns the "unwrapped" address for a token a.k.a. the OriginAddress
// of the token's original minting contract.
func (tv *TransferVerifier[ethClient, connector]) unwrapIfWrapped(
	tokenAddress []byte,
	tokenChain vaa.ChainID,
) (unwrappedTokenAddress common.Address, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), RPC_TIMEOUT)
	defer cancel()

	tokenAddressAsKey := hex.EncodeToString(tokenAddress)

	// If the token address already exists in the wrappedCache mapping the
	// cached value can be returned.
	if addr, exists := tv.wrappedCache[tokenAddressAsKey]; exists {
		tv.logger.Debug("wrapped asset found in cache, returning")
		return addr, nil
	}

	// prepare eth_call data, 4-byte signature + 2x 32 byte arguments
	calldata := make([]byte, 4+EVM_WORD_LENGTH+EVM_WORD_LENGTH)

	copy(calldata, TOKEN_BRIDGE_WRAPPED_ASSET_SIGNATURE)
	// Add the uint16 tokenChain as the last two bytes in the first argument
	binary.BigEndian.PutUint16(calldata[4+30:], uint16(tokenChain))
	copy(calldata[4+EVM_WORD_LENGTH:], common.LeftPadBytes(tokenAddress, EVM_WORD_LENGTH))

	ethCallMsg := ethereum.CallMsg{
		To:   &tv.Addresses.TokenBridgeAddr,
		Data: calldata,
	}
	tv.logger.Debug("calling wrappedAsset",
		zap.Uint16("tokenChain", uint16(tokenChain)),
		zap.String("tokenChainString", tokenChain.String()),
		zap.String("tokenAddress", fmt.Sprintf("%x", tokenAddress)),
		zap.String("callData", fmt.Sprintf("%x", calldata)))

	result, err := tv.client.CallContract(ctx, ethCallMsg, nil)
	if err != nil {
		// This strictly handles the error case. The contract call will
		// return the zero address for assets not in its map.
		return common.Address{}, fmt.Errorf("failed to get mapping for token %s", tokenAddressAsKey)
	}

	tokenAddressNative := common.BytesToAddress(result)
	tv.wrappedCache[tokenAddressAsKey] = tokenAddressNative

	tv.logger.Debug("got wrappedAsset result",
		zap.String("tokenAddressNative", fmt.Sprintf("%x", tokenAddressNative)))

	if Cmp(tokenAddressNative, ZERO_ADDRESS) == 0 {
		tv.logger.Info("got zero address for wrappedAsset result. this asset is probably not registered correctly",
			zap.String("queried tokenAddress", fmt.Sprintf("%x", tokenAddress)),
			zap.Uint16("queried tokenChain", uint16(tokenChain)),
			zap.String("tokenChain name", tokenChain.String()),
		)
	}

	return tokenAddressNative, nil
}

// chainId() calls the chainId() function on the contract at the supplied address. To get the chain ID being monitored
// by the Transfer Verifier, use the field TransferVerifier.chain.
func (tv *TransferVerifier[ethClient, Connector]) chainId(
	addr common.Address,
) (vaa.ChainID, error) {

	if Cmp(addr, ZERO_ADDRESS) == 0 {
		return 0, errors.New("got zero address as parameter for chainId() call")
	}
	ctx, cancel := context.WithTimeout(context.Background(), RPC_TIMEOUT)
	defer cancel()

	tokenAddressAsKey := addr.Hex()

	// If the token address already exists in the wrappedCache mapping the
	// cached value can be returned.
	if chainId, exists := tv.nativeChainCache[tokenAddressAsKey]; exists {
		tv.logger.Debug("wrapped asset found in native chain cache, returning")
		return chainId, nil
	}

	// prepare eth_call data, 4-byte signature
	calldata := make([]byte, 4)

	copy(calldata, WRAPPED_ERC20_CHAIN_ID_SIGNATURE)

	ethCallMsg := ethereum.CallMsg{
		To:   &addr,
		Data: calldata,
	}

	tv.logger.Debug("calling chainId()", zap.String("tokenAddress", addr.String()))

	result, err := tv.client.CallContract(ctx, ethCallMsg, nil)

	if err != nil {
		// TODO add more checks here
		return 0, err
	}
	if len(result) < EVM_WORD_LENGTH {
		tv.logger.Warn("result for chainId has insufficient length",
			zap.Int("length", len(result)),
			zap.String("result", fmt.Sprintf("%x", result)))
		return 0, errors.New("result for chainId has insufficient length")
	}

	chainID := vaa.ChainID(binary.BigEndian.Uint16(result))

	tv.nativeChainCache[tokenAddressAsKey] = chainID

	return chainID, nil
}

func (tv *TransferVerifier[ethClient, Connector]) isWrappedAsset(
	addr common.Address,
	// chainID common.Address,
) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), RPC_TIMEOUT)
	defer cancel()

	tokenAddressAsKey := addr.Hex()

	// If the token address already exists in the isWrappedCache mapping the
	// cached value can be returned.
	if wrapped, exists := tv.isWrappedCache[tokenAddressAsKey]; exists {
		tv.logger.Debug("asset found in isWrapped cache, returning")
		return wrapped, nil
	}

	// Prepare eth_call data: 4-byte signature + 32 byte address
	calldata := make([]byte, 4+EVM_WORD_LENGTH)
	copy(calldata, TOKEN_BRIDGE_IS_WRAPPED_ASSET_SIGNATURE)
	copy(calldata[4:], common.LeftPadBytes(addr.Bytes(), EVM_WORD_LENGTH))

	evmCallMsg := ethereum.CallMsg{
		To:   &tv.Addresses.TokenBridgeAddr,
		Data: calldata,
	}

	tv.logger.Debug("calling isWrappedAsset()", zap.String("tokenAddress", addr.String()))

	result, err := tv.client.CallContract(ctx, evmCallMsg, nil)

	if err != nil {
		// TODO add more info here
		tv.logger.Warn("isWrappedAsset() call error", zap.Error(err))
		return false, err
	}
	if len(result) < EVM_WORD_LENGTH {
		tv.logger.Warn("isWrappedAsset() result length is too small", zap.String("result", fmt.Sprintf("%x", result)))
		return false, err
	}
	tv.logger.Debug("isWrappedAsset result", zap.String("result", fmt.Sprintf("%x", result)))

	// The boolean result will be returned as a byte string with length
	// equal to EVM_WORD_LENGTH. Grab the last byte.
	wrapped := result[EVM_WORD_LENGTH-1] == 1

	tv.isWrappedCache[tokenAddressAsKey] = wrapped

	return wrapped, nil
}

// Determine whether a log is relevant for the addresses passed into TVAddresses. Returns a string of the form "address-chain" for relevant entries.
func relevant[L TransferLog](tLog TransferLog, tv *TVAddresses) (key string, relevant bool) {

	switch log := tLog.(type) {
	case *NativeDeposit:
		// Skip native deposit events emitted by contracts other than the configured wrapped native address.
		if Cmp(log.Emitter(), tv.WrappedNativeAddr) != 0 {
			return
		}

		// We only care about deposits into the token bridge.
		if Cmp(log.Destination(), tv.TokenBridgeAddr) != 0 {
			return
		}

	case *ERC20Transfer:
		// We only care about transfers sent to the token bridge.
		if Cmp(log.Destination(), tv.TokenBridgeAddr) != 0 {
			return
		}

	case *LogMessagePublished:
		// This check is already done elsewhere but it's important.
		if Cmp(log.Emitter(), tv.CoreBridgeAddr) != 0 {
			return
		}

		// Only consider LogMessagePublished events with msg.sender equal to the Token Bridge
		if Cmp(log.Sender(), tv.TokenBridgeAddr) != 0 {
			return
		}

		// The following values are not exposed by the interface, so check them directly here.
		if log.TransferDetails.PayloadType != TransferTokens && log.TransferDetails.PayloadType != TransferTokensWithPayload {
			return
		}

	}
	return fmt.Sprintf(KEY_FORMAT, tLog.OriginAddress(), tLog.OriginChain()), true
}

// Custom error type indicating an issue in issue in a type that implements the
// TransferLog interface. Used to ensure that a TransferLog is well-formed.
// Typically indicates a bug in the code.
type InvalidLogError struct {
	Msg string
}

func (i InvalidLogError) Error() string {
	return fmt.Sprintf("invalid log: %s", i.Msg)
}

// validate() ensures a TransferLog is well-formed. This means that its fields
// are not nil and in most cases are not equal to the zero-value for the
// field's type.
func validate[L TransferLog](tLog TransferLog) error {

	// Generic validation for all TransferLogs
	if Cmp(tLog.Emitter(), ZERO_ADDRESS) == 0 {
		return &InvalidLogError{Msg: "emitter is the zero address"}
	}

	if tLog.OriginChain() == 0 {
		return &InvalidLogError{Msg: "originChain is zero"}
	}

	if tLog.TransferAmount() == nil {
		return &InvalidLogError{Msg: "transfer amount is nil"}
	}

	if tLog.TransferAmount().Sign() == -1 {
		return &InvalidLogError{Msg: "transfer amount is negative"}
	}

	switch log := tLog.(type) {
	case *NativeDeposit:
		// Deposit does not actually have a sender, so it should always be equal to the zero address.
		if Cmp(log.Sender(), ZERO_ADDRESS_VAA) != 0 {
			return &InvalidLogError{Msg: "sender address for Deposit must be 0"}
		}
		if Cmp(log.Emitter(), log.TokenAddress) != 0 {
			return &InvalidLogError{Msg: "deposit emitter is not equal to its token address"}
		}
		if Cmp(log.Destination(), ZERO_ADDRESS_VAA) == 0 {
			return &InvalidLogError{Msg: "destination is not set"}
		}
		if Cmp(log.OriginAddress(), ZERO_ADDRESS_VAA) == 0 {
			return &InvalidLogError{Msg: "originAddress is the zero address"}
		}
	case *ERC20Transfer:
		// Note: The token bridge transfers to the zero address in
		// order to burn tokens for some kinds of transfers. For this
		// reason, there is no validation here to check if Destination
		// is the zero address.

		// Sender must not be checked to be non-zero here. The event
		// hash for Transfer also shows up in other popular contracts
		// (e.g. UniswapV2) and may have a valid reason to set this
		// field to zero.

		// TODO ensure that, if the Token is wrapped, that its tokenchain is not equal to NATIVE_CHAIN_ID.
		// at this point, this should've been updated

		if Cmp(log.Emitter(), log.TokenAddress) != 0 {
			return &InvalidLogError{Msg: "transfer emitter is not equal to its token address"}
		}
		if Cmp(log.OriginAddress(), ZERO_ADDRESS_VAA) == 0 {
			return &InvalidLogError{Msg: "originAddress is the zero address"}
		}
	case *LogMessagePublished:
		// LogMessagePublished cannot have a sender with a 0 address
		if Cmp(log.Sender(), ZERO_ADDRESS_VAA) == 0 {
			return &InvalidLogError{Msg: "sender cannot be zero"}
		}
		if Cmp(log.Destination(), ZERO_ADDRESS_VAA) == 0 {
			return &InvalidLogError{Msg: "destination is not set"}
		}

		// TODO is this valid for assets that return the zero address from unwrap?
		// if Cmp(log.OriginAddress(), ZERO_ADDRESS_VAA) == 0 {
		// 	return errors.New("origin cannot be zero")
		// }

		// The following values are not exposed by the interface, so check them directly here.
		if log.TransferDetails == nil {
			return &InvalidLogError{Msg: "TransferDetails cannot be nil"}
		}
		if Cmp(log.TransferDetails.TargetAddress, ZERO_ADDRESS_VAA) == 0 {
			return &InvalidLogError{Msg: "target address cannot be zero"}
		}

		if len(log.TransferDetails.OriginAddressRaw) == 0 {
			return &InvalidLogError{Msg: "origin address raw cannot be empty"}
		}

		// if bytes.Compare(log.TransferDetails.OriginAddressRaw, ZERO_ADDRESS_VAA.Bytes()) == 0 {
		// 	return &InvalidLogError{Msg: "origin address raw cannot be zero"}
		// }

		if log.TransferDetails.AmountRaw == nil {
			return &InvalidLogError{Msg: "amountRaw cannot be nil"}
		}
		if log.TransferDetails.AmountRaw.Sign() == -1 {
			return &InvalidLogError{Msg: "amountRaw cannot be negative"}
		}
		if log.TransferDetails.PayloadType != TransferTokens && log.TransferDetails.PayloadType != TransferTokensWithPayload {
			return &InvalidLogError{Msg: "payload type is not a transfer type"}
		}
	default:
		return &InvalidLogError{Msg: "invalid transfer log type: unknown"}
	}

	return nil
}

// getDecimals() is equivalent to calling decimals() on a contract that follows the ERC20 standard.
func (tv *TransferVerifier[evmClient, connector]) getDecimals(
	tokenAddress common.Address,
) (decimals uint8, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), RPC_TIMEOUT)
	defer cancel()

	// First check if this token's decimals is stored in cache
	if _, exists := tv.decimalsCache[tokenAddress]; exists {
		tv.logger.Debug("asset decimals found in cache, returning")
		return tv.decimalsCache[tokenAddress], nil
	}

	// If the decimals aren't cached, perform an eth_call lookup for the decimals
	// This RPC call should only be made once per token, until the guardian is restarted
	evmCallMsg := ethereum.CallMsg{
		To:   &tokenAddress,
		Data: ERC20_DECIMALS_SIGNATURE,
	}

	result, err := tv.client.CallContract(ctx, evmCallMsg, nil)
	if err != nil {
		tv.logger.Warn("error from getDecimals() for token",
			zap.String("tokenAddress", tokenAddress.String()),
			zap.ByteString("result", result),
			zap.Error(err))
		return 0, err
	}

	if len(result) < EVM_WORD_LENGTH {
		tv.logger.Warn("failed to get decimals for token: decimals() result has insufficient length",
			zap.String("tokenAddress", tokenAddress.String()),
			zap.ByteString("result", result))
		return 0, err
	}

	// An ERC20 token's decimals should fit in a single byte. A call to `decimals()`
	// returns a uint8 value encoded in string with 32-bytes. To get the decimals,
	// we grab the last byte, expecting all the preceding bytes to be equal to 0.
	decimals = result[EVM_WORD_LENGTH-1]

	// Add the decimal value to the cache
	tv.decimalsCache[tokenAddress] = decimals
	tv.logger.Debug("adding new token's decimals to cache",
		zap.String("tokenAddress", tokenAddress.String()),
		zap.Uint8("tokenDecimals", decimals))

	return decimals, nil
}

// Yields the registered Wormhole chain ID corresponding to an EVM chain ID.
func TryWormholeChainIdFromNative(evmChainId uint64) (wormholeChainID vaa.ChainID, err error) {
	wormholeChainID = vaa.ChainIDUnset
	// Add additional cases below to support more EVM chains.
	// Note: it might be better for this function to be moved into the SDK in case other codebases need similar functionality.
	switch evmChainId {
	// Special carve out for anvil-based testing. This chain ID  1337 anvil's default.
	// In this case, report the native chain ID as the mainnet chain ID for the purposes of testing.
	case 1, 1337:
		wormholeChainID = vaa.ChainIDEthereum
	case 11155111:
		wormholeChainID = vaa.ChainIDSepolia
	default:
		err = fmt.Errorf(
			"Transfer Verifier does not have a registered mapping from EVM chain ID %d to a Wormhole chain ID",
			evmChainId,
		)
	}
	return
}

// Gives the representation of a geth address in vaa.Address
func VAAAddrFrom(gethAddr common.Address) (vaaAddr vaa.Address) {
	// Geth uses 20 bytes to represent an address. A VAA address is equivalent if it has the same
	// final 20 bytes. The leading bytes are expected to be zero for both types.
	vaaAddr = vaa.Address(common.LeftPadBytes(gethAddr[:], EVM_WORD_LENGTH))
	return
}

// Interface useful for comparing vaa.Address and common.Address
type Bytes interface {
	Bytes() []byte
}

// Utility method for comparing common.Address and vaa.Address at the byte level.
func Cmp[some Bytes, other Bytes](a some, b other) int {

	// Compare bytes, prepending 0s to ensure that both values are of EVM_WORD_LENGTH.
	return bytes.Compare(
		common.LeftPadBytes(a.Bytes(), EVM_WORD_LENGTH),
		common.LeftPadBytes(b.Bytes(), EVM_WORD_LENGTH),
	)
}
