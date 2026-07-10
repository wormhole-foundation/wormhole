package solana

// This file provides a test-only mock for the subset of the solana-go RPC
// client API used by the Solana watcher. The mocked methods intentionally
// correspond to methods on github.com/gagliardetto/solana-go/rpc.Client,
// which this repository replaces with the Solana Foundation fork in go.mod.
//
// Library reference:
// https://github.com/solana-foundation/solana-go/tree/main/rpc
//
// Solana JSON-RPC API reference:
// https://solana.com/docs/rpc/http
//
// Keep this mock narrow. Add methods here only when watcher code calls the
// matching solana-go RPC method through solanaRPCClient.

import (
	"context"
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

var _ solanaRPCClient = (*mockSolanaRPCClient)(nil)

type mockSolanaRPCClient struct {
	slot    uint64
	slotErr error

	version    *rpc.GetVersionResult
	versionErr error

	blocks   map[uint64]*rpc.GetBlockResult
	blockErr map[uint64]error

	accounts   map[solana.PublicKey]*rpc.GetAccountInfoResult
	accountErr map[solana.PublicKey]error

	accountsWithOpts   map[solana.PublicKey]*rpc.GetAccountInfoResult
	accountWithOptsErr map[solana.PublicKey]error

	transactions   map[solana.Signature]*rpc.GetTransactionResult
	transactionErr map[solana.Signature]error

	signatures   map[solana.PublicKey][]*rpc.TransactionSignature
	signatureErr map[solana.PublicKey]error

	rpcCallForInto func(ctx context.Context, out interface{}, method string, params []interface{}) error
}

func newMockSolanaRPCClient() *mockSolanaRPCClient {
	return &mockSolanaRPCClient{
		blocks:             map[uint64]*rpc.GetBlockResult{},
		blockErr:           map[uint64]error{},
		accounts:           map[solana.PublicKey]*rpc.GetAccountInfoResult{},
		accountErr:         map[solana.PublicKey]error{},
		accountsWithOpts:   map[solana.PublicKey]*rpc.GetAccountInfoResult{},
		accountWithOptsErr: map[solana.PublicKey]error{},
		transactions:       map[solana.Signature]*rpc.GetTransactionResult{},
		transactionErr:     map[solana.Signature]error{},
		signatures:         map[solana.PublicKey][]*rpc.TransactionSignature{},
		signatureErr:       map[solana.PublicKey]error{},
	}
}

func (m *mockSolanaRPCClient) SetAccount(key solana.PublicKey, owner string, data []byte) {
	ownerKey := solana.MustPublicKeyFromBase58(owner)
	info := makeMockAccountInfoResult(ownerKey, data)
	m.accounts[key] = info
	m.accountsWithOpts[key] = info
}

func (m *mockSolanaRPCClient) SetAccountError(key solana.PublicKey, msg string) {
	err := errors.New(msg)
	m.accountErr[key] = err
	m.accountWithOptsErr[key] = err
}

func (m *mockSolanaRPCClient) GetSlot(_ context.Context, _ rpc.CommitmentType) (uint64, error) {
	return m.slot, m.slotErr
}

func (m *mockSolanaRPCClient) GetBlockWithOpts(_ context.Context, slot uint64, _ *rpc.GetBlockOpts) (*rpc.GetBlockResult, error) {
	if err := m.blockErr[slot]; err != nil {
		return nil, err
	}
	if block := m.blocks[slot]; block != nil {
		return block, nil
	}
	return nil, rpc.ErrNotFound
}

func (m *mockSolanaRPCClient) GetTransaction(_ context.Context, sig solana.Signature, _ *rpc.GetTransactionOpts) (*rpc.GetTransactionResult, error) {
	if err := m.transactionErr[sig]; err != nil {
		return nil, err
	}
	if tx := m.transactions[sig]; tx != nil {
		return tx, nil
	}
	return nil, rpc.ErrNotFound
}

func (m *mockSolanaRPCClient) GetAccountInfo(_ context.Context, account solana.PublicKey) (*rpc.GetAccountInfoResult, error) {
	if err := m.accountErr[account]; err != nil {
		return nil, err
	}
	if info := m.accounts[account]; info != nil {
		return info, nil
	}
	return nil, rpc.ErrNotFound
}

func (m *mockSolanaRPCClient) GetAccountInfoWithOpts(_ context.Context, account solana.PublicKey, _ *rpc.GetAccountInfoOpts) (*rpc.GetAccountInfoResult, error) {
	if err := m.accountWithOptsErr[account]; err != nil {
		return nil, err
	}
	if info := m.accountsWithOpts[account]; info != nil {
		return info, nil
	}
	return nil, rpc.ErrNotFound
}

func (m *mockSolanaRPCClient) GetSignaturesForAddressWithOpts(_ context.Context, account solana.PublicKey, _ *rpc.GetSignaturesForAddressOpts) ([]*rpc.TransactionSignature, error) {
	if err := m.signatureErr[account]; err != nil {
		return nil, err
	}
	return m.signatures[account], nil
}

func (m *mockSolanaRPCClient) GetVersion(_ context.Context) (*rpc.GetVersionResult, error) {
	return m.version, m.versionErr
}

func (m *mockSolanaRPCClient) RPCCallForInto(ctx context.Context, out interface{}, method string, params []interface{}) error {
	if m.rpcCallForInto == nil {
		return errors.New("unexpected RPCCallForInto")
	}
	return m.rpcCallForInto(ctx, out, method, params)
}

func makeMockAccountInfoResult(owner solana.PublicKey, data []byte) *rpc.GetAccountInfoResult {
	return &rpc.GetAccountInfoResult{
		Value: &rpc.Account{
			Owner:      owner,
			Data:       rpc.DataBytesOrJSONFromBytes(data),
			Lamports:   1,
			Executable: false,
			RentEpoch:  0,
		},
	}
}

/*
Example table-driven usage for future tests:

func TestFetchMessageAccountWithMockRPC(t *testing.T) {
	contract := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xAA}, solana.PublicKeyLength))
	messageAccount := solana.PublicKeyFromBytes(bytes.Repeat([]byte{0xBB}, solana.PublicKeyLength))
	accountData := encodeMessagePublicationAccount(t, accountPrefixReliable, testMessagePublicationAccount([]byte("hello"), 32))

	tests := []struct {
		name             string
		mock             *mockSolanaRPCClient
		wantObservations uint32
		wantRetryable    bool
	}{
		{
			name: "valid message account",
			mock: &mockSolanaRPCClient{
				accountsWithOpts: map[solana.PublicKey]*rpc.GetAccountInfoResult{
					messageAccount: makeMockAccountInfoResult(contract, accountData),
				},
			},
			wantObservations: 1,
			wantRetryable:    false,
		},
		{
			name: "rpc error is retryable",
			mock: &mockSolanaRPCClient{
				accountWithOptsErr: map[solana.PublicKey]error{
					messageAccount: errors.New("rpc down"),
				},
			},
			wantObservations: 0,
			wantRetryable:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgC := make(chan *common.MessagePublication, 1)
			s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
			s.contract = contract

			num, retryable := s.fetchMessageAccount(context.Background(), tc.mock, messageAccount, 1, false, solana.Signature{})

			require.Equal(t, tc.wantObservations, num)
			require.Equal(t, tc.wantRetryable, retryable)
		})
	}
}
*/
