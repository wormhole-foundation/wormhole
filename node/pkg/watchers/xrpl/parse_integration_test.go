//go:build integration

package xrpl

import (
	"testing"

	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/websocket"
	"github.com/stretchr/testify/require"
)

const xrplTestnetWSS = "wss://s.altnet.rippletest.net:51233"

// TestValidateTransactionResult_Integration fetches a real transaction from XRPL testnet
// and validates that validateTransactionResult correctly handles it.
//
// Transaction: 751DAA107AE317E79DD45C50AFC1F72762EF013D2584628F09CC8423EAC01040
// This is a successful Payment transaction on testnet.
func TestValidateTransactionResult_Integration(t *testing.T) {
	txHash := "751DAA107AE317E79DD45C50AFC1F72762EF013D2584628F09CC8423EAC01040"

	// Connect to XRPL testnet
	cfg := websocket.NewClientConfig().WithHost(xrplTestnetWSS)
	client := websocket.NewClient(cfg)
	err := client.Connect()
	require.NoError(t, err, "failed to connect to XRPL testnet")
	defer func() {
		_ = client.Disconnect()
	}()

	// Fetch the transaction
	txReq := &transactions.TxRequest{
		Transaction: txHash,
	}

	resp, err := client.Request(txReq)
	require.NoError(t, err, "failed to fetch transaction")

	var txResp transactions.TxResponse
	err = resp.GetResult(&txResp)
	require.NoError(t, err, "failed to decode transaction response")

	// Verify the transaction is validated
	require.True(t, txResp.Validated, "transaction should be validated")

	// Create a parser and test validateTransactionResult
	parser := NewParser("", nil, nil)

	// Create a GenericTx with the transaction result from the response
	tx := GenericTx{
		Transaction:           txResp.TxJSON,
		MetaTransactionResult: txResp.Meta.TransactionResult,
	}

	// Test validateTransactionResult - should succeed for a successful transaction
	err = parser.validateTransactionResult(tx)
	require.NoError(t, err, "validateTransactionResult should succeed for tesSUCCESS transaction")

	// Log some details about the fetched transaction
	t.Logf("Transaction hash: %s", txHash)
	t.Logf("Transaction type: %v", txResp.TxJSON["TransactionType"])
	t.Logf("Transaction result: %s", txResp.Meta.TransactionResult)
	t.Logf("Ledger index: %d", txResp.LedgerIndex)
	t.Logf("Validated: %t", txResp.Validated)
}

// TestValidateTransactionResult_Integration_FailedTransaction tests that
// validateTransactionResult correctly rejects a failed transaction.
func TestValidateTransactionResult_Integration_FailedTransaction(t *testing.T) {
	// Create a parser
	parser := NewParser("", nil, nil)

	// Create a GenericTx with a non-success result
	tx := GenericTx{
		MetaTransactionResult: "tecPATH_DRY",
	}

	// Test validateTransactionResult - should fail for non-success transaction
	err := parser.validateTransactionResult(tx)
	require.Error(t, err, "validateTransactionResult should fail for non-tesSUCCESS transaction")
	require.Contains(t, err.Error(), "tecPATH_DRY")
	require.Contains(t, err.Error(), "not tesSUCCESS")
}

// TestValidateTransactionResult_Integration_EmptyResult tests that
// validateTransactionResult correctly handles a transaction with empty result.
func TestValidateTransactionResult_Integration_EmptyResult(t *testing.T) {
	parser := NewParser("", nil, nil)

	tx := GenericTx{
		MetaTransactionResult: "",
	}

	err := parser.validateTransactionResult(tx)
	require.Error(t, err, "validateTransactionResult should fail when result is empty")
	require.Contains(t, err.Error(), "not tesSUCCESS")
}
