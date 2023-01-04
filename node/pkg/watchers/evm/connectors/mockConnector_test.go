package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"

	"go.uber.org/zap"
)

// mockConnector implements the connector interface for testing purposes.
type mockConnector struct {
	address         ethCommon.Address
	client          *ethClient.Client
	mutex           sync.Mutex
	results         []string
	err             error
	persistentError bool
	blockNumber     uint64
}

// setResults takes an array of json results strings. Each time a test makes an RPC call, it uses the first element in
// the array as the response, and the discards it. If the array is empty, the call will block until more results are stored.
func (m *mockConnector) setResults(results []string) {
	m.mutex.Lock()
	m.results = results
	m.mutex.Unlock()
}

// setError takes an error which will be returned on the next RPC call. The error will persist until cleared.
func (m *mockConnector) setError(err error) {
	m.mutex.Lock()
	m.err = err
	m.persistentError = true
	m.mutex.Unlock()
}

// setSingleError takes an error which will be returned on the next RPC call. After that, the error is reset to nil.
func (m *mockConnector) setSingleError(err error) {
	m.mutex.Lock()
	m.err = err
	m.persistentError = false
	m.mutex.Unlock()
}

func newMockConnector(ctx context.Context, networkName, rawUrl string, address ethCommon.Address, logger *zap.Logger) (*mockConnector, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnector) NetworkName() string {
	return "mockConnector"
}

func (e *mockConnector) ContractAddress() ethCommon.Address {
	return e.address
}

func (e *mockConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *mockConnector) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return ethAbi.StructsGuardianSet{}, fmt.Errorf("not implemented")
}

func (e *mockConnector) WatchLogMessagePublished(ctx context.Context, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	var s ethEvent.Subscription
	return s, fmt.Errorf("not implemented")
}

func (e *mockConnector) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *mockConnector) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnector) SubscribeForBlocks(ctx context.Context, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	var s ethEvent.Subscription
	return s, fmt.Errorf("not implemented")
}

func (e *mockConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) (err error) {
	if method == "eth_getBlockByNumber" {
		return e.getBlockByNumber(ctx, result, args...)
	}
	for {
		e.mutex.Lock()
		// If they set the error, return that immediately.
		if e.err != nil {
			err = e.err
			if !e.persistentError {
				e.err = nil
			}
			break
		}

		// If there are pending results, return the first one.
		if len(e.results) != 0 {
			str := e.results[0]
			e.results = e.results[1:]
			err = json.Unmarshal([]byte(str), &result)
			break
		}

		// If we don't have any results, sleep and try again.
		e.mutex.Unlock()
		time.Sleep(1 * time.Millisecond)
	}

	e.mutex.Unlock()
	return
}

func (e *mockConnector) setBlockNumber(blockNumber uint64) {
	e.mutex.Lock()
	e.blockNumber = blockNumber
	e.mutex.Unlock()
}

func (e *mockConnector) expectedHash() ethCommon.Hash {
	return ethCommon.HexToHash("0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf")
}

func (e *mockConnector) getBlockByNumber(ctx context.Context, result interface{}, args ...interface{}) (err error) {
	e.mutex.Lock()
	// If they set the error, return that immediately.
	if e.err != nil {
		err = e.err
		if !e.persistentError {
			e.err = nil
		}
	} else {
		str := fmt.Sprintf(`{"author":"0x24c275f0719fdaec6356c4eb9f39ecb9c4d37ce1","baseFeePerGas":"0x3b9aca00","difficulty":"0x0","extraData":"0x","gasLimit":"0xe4e1c0","gasUsed":"0x0","hash":"0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x24c275f0719fdaec6356c4eb9f39ecb9c4d37ce1","nonce":"0x0000000000000000","number":"0x%x","parentHash":"0x09d6d33a658b712f41db7fb9f775f94911ae0132123116aa4f8cf3da9f774e89","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x201","stateRoot":"0x0409ed10e03fd49424ae1489c6fbc6ff1897f45d0e214655ebdb8df94eedc3c0","timestamp":"0x6373ec24","totalDifficulty":"0x0","transactions":[],"transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","uncles":[]}`, e.blockNumber)
		err = json.Unmarshal([]byte(str), &result)
	}
	e.mutex.Unlock()
	return
}

func (e *mockConnector) Client() *ethClient.Client {
	return e.client
}

type mockFinalizer struct {
	mutex     sync.Mutex
	finalized bool
}

func newMockFinalizer(initialState bool) *mockFinalizer {
	return &mockFinalizer{finalized: initialState}
}

func (f *mockFinalizer) setFinalized(finalized bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.finalized = finalized
}

func (f *mockFinalizer) IsBlockFinalized(ctx context.Context, block *NewBlock) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.finalized, nil
}
