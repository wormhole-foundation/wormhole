package sui

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
	"github.com/test-go/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type consoleEncoder struct {
	zapcore.Encoder
}

func TestSuiResultParsing(tst *testing.T) {
	msgChan := make(chan *common.MessagePublication, 10)
	defer close(msgChan)

	obsvChan := make(chan *gossipv1.ObservationRequest)
	defer close(obsvChan)

	watcher := NewWatcher("suiRPC", "suiWS", "suiMoveEventType", false, msgChan, obsvChan)
	watcher.logger = zap.New(zapcore.NewCore(
		consoleEncoder{zapcore.NewConsoleEncoder(
			zap.NewDevelopmentEncoderConfig())},
		zapcore.AddSync(zapcore.Lock(os.Stderr)),
		zap.NewAtomicLevelAt(zapcore.Level(zapcore.DebugLevel))))

	// Setup the Fields
	var cl uint8 = 0
	var n uint64 = 64
	pl := []byte("cGF5bG9hZA==") // payload
	var sndr string = "65"
	var seq string = "66"
	var fieldTS string = "1681935313"
	fields := FieldsData{
		ConsistencyLevel: &cl,
		Nonce:            &n,
		Payload:          pl,
		Sender:           &sndr,
		Sequence:         &seq,
		Timestamp:        &fieldTS,
	}
	var evSeq string = "12"

	// Setup the SuiResult
	var pid string = "packageID"
	var tm string = "transactionModule"
	var sd string = "sender"
	var t string = "incorrectType"
	var bcs string = "bcs"
	var ts string = "12346"
	res := SuiResult{
		ID: struct {
			TxDigest *string "json:\"txDigest\""
			EventSeq *string "json:\"eventSeq\""
		}{nil, &evSeq},
		Timestamp:         &ts,
		PackageID:         &pid,
		TransactionModule: &tm,
		Sender:            &sd,
		Type:              nil,
		Fields:            &fields,
		Bcs:               &bcs,
	}

	// Test missing TxDigest field
	err := watcher.inspectBody(res)
	require.Error(tst, err)
	fmt.Println("inspectBody had an error", err)
	assert.Equal(tst, err.Error(), "Missing TxDigest field")

	var txd string = "LUuNLy6iumu" // txDigest
	res.ID.TxDigest = &txd
	err = watcher.inspectBody(res)
	require.Error(tst, err)
	fmt.Println("inspectBody had an error", err)
	assert.Equal(tst, err.Error(), "Missing Type field")

	res.Type = &t
	err = watcher.inspectBody(res)
	require.Error(tst, err)
	fmt.Println("inspectBody had an error", err)
	assert.Equal(tst, err.Error(), "type mismatch")

	t = "suiMoveEventType"
	err = watcher.inspectBody(res)
	require.Error(tst, err)
	fmt.Println("inspectBody had an error", err)
	assert.Contains(tst, err.Error(), "Transaction hash is not 32 bytes")

	hashBase := "This needs to be a 32 byte hash."
	txd = base58.Encode([]byte(hashBase))
	err = watcher.inspectBody(res)
	require.NoError(tst, err)
	require.Equal(tst, 1, len(msgChan))
	obsv := <-msgChan
	fmt.Println("Received msgChan msg...")
	fmt.Println("Checking txDigest...")
	require.Equal(tst, eth_common.BytesToHash([]byte(hashBase)), obsv.TxHash)
	fmt.Println("Checking time...")
	decodedTime, _ := strconv.ParseInt(fieldTS, 10, 64)
	require.Equal(tst, time.Unix(decodedTime, 0), obsv.Timestamp)
	fmt.Println("Checking nonce...")
	assert.Equal(tst, uint32(64), obsv.Nonce)
	fmt.Println("Checking sequence...")
	assert.Equal(tst, uint64(66), obsv.Sequence)
	fmt.Println("Checking EmitterChain...")
	assert.Equal(tst, obsv.EmitterChain, vaa.ChainIDSui)
	fmt.Println("Checking EmitterAddress...")
	emitter, _ := vaa.StringToAddress(sndr)
	assert.Equal(tst, emitter, obsv.EmitterAddress)
}
