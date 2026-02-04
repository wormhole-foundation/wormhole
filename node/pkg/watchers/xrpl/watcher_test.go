package xrpl

import (
	"testing"

	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// =============================================================================
// processTransaction tests (Watcher integration)
// =============================================================================

func TestProcessTransaction_SkipsUnvalidated(t *testing.T) {
	msgChan := make(chan *common.MessagePublication, 1)
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
		msgChan:  msgChan,
		parser:   NewParser(nil),
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    false, // Not validated
		Transaction:  createFlatTransactionWithMemos(testNTTMemoFormat, sampleNTTMemoData),
	}

	err := w.processTransaction(zap.NewNop(), tx)

	require.NoError(t, err)
	assert.Empty(t, msgChan, "No message should be sent for unvalidated transaction")
}

func TestProcessTransaction_SendsValidatedMessage(t *testing.T) {
	contract := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"
	msgChan := make(chan *common.MessagePublication, 1)
	w := &Watcher{
		contract: contract,
		msgChan:  msgChan,
		parser:   NewParser(nil),
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true, // Validated
		Transaction:  createValidNTTTransaction(),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  3,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000", // XRP drops as string
		},
	}

	err := w.processTransaction(zap.NewNop(), tx)

	require.NoError(t, err)
	require.Len(t, msgChan, 1, "Message should be sent for validated transaction")

	// Verify the message contents
	msg := <-msgChan
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)
	// Sequence = (ledgerIndex << 32) | txIndex = (12345 << 32) | 3
	assert.Equal(t, (uint64(12345)<<32)|3, msg.Sequence)
	assert.False(t, msg.IsReobservation)
}
