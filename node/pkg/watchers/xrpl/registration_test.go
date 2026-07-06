package xrpl

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// testRegCoreAccount is the core (GMP) account that registration publishes target.
const testRegCoreAccount = "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"

// testRegManager20 is a sample XRPL custody account id (20 bytes).
func testRegManager20() [20]byte {
	var m [20]byte
	for i := range m {
		m[i] = byte(i + 1)
	}
	return m
}

// buildXregHubXRP builds an XREG Hub payload for an XRP custody (matching the
// sequencer's payloads.rs serialization): prefix + kind(0) + manager(20) +
// token_id(XRP=0x00) + decimals(1).
func buildXregHubXRP(manager [20]byte, decimals uint8) []byte {
	out := append([]byte{}, xregPrefix[:]...)
	out = append(out, xregKindHub)
	out = append(out, manager[:]...)
	out = append(out, xregTokenXRP) // token_id wire
	out = append(out, decimals)
	return out
}

// buildXregHubIOU builds an XREG Hub for an IOU custody:
// prefix + kind(0) + manager(20) + token_id(0x01 + currency20 + issuer20) + decimals.
func buildXregHubIOU(manager [20]byte, currency, issuer [20]byte, decimals uint8) []byte {
	out := append([]byte{}, xregPrefix[:]...)
	out = append(out, xregKindHub)
	out = append(out, manager[:]...)
	out = append(out, xregTokenIOU)
	out = append(out, currency[:]...)
	out = append(out, issuer[:]...)
	out = append(out, decimals)
	return out
}

// buildXregPeerXRP builds an XREG Peer for an XRP custody:
// prefix + kind(1) + manager(20) + token_id(XRP) + peer_chain(2) + peer_address(32).
func buildXregPeerXRP(manager [20]byte, peerChain uint16, peerAddr [32]byte) []byte {
	out := append([]byte{}, xregPrefix[:]...)
	out = append(out, xregKindPeer)
	out = append(out, manager[:]...)
	out = append(out, xregTokenXRP)
	out = binary.BigEndian.AppendUint16(out, peerChain)
	out = append(out, peerAddr[:]...)
	return out
}

func regTxStream(memoData []byte) *streamtypes.TransactionStream {
	return &streamtypes.TransactionStream{
		Hash:         "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-06-15T14:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     testRegCoreAccount,
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": regMemoFormat,
						"MemoData":   hex.EncodeToString(memoData),
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  7,
			TransactionResult: "tesSUCCESS",
		},
	}
}

// managerToEmitter mirrors the transfer-path manager32 derivation (20-byte
// account id left-padded into 32 bytes).
func manager32From20(m [20]byte) [32]byte {
	var out [32]byte
	copy(out[12:], m[:])
	return out
}

func TestRegistration_HubXRP_EmitterMatchesTransferPath(t *testing.T) {
	p := NewParser(testRegCoreAccount, nil, nil)
	manager := testRegManager20()

	msg, err := p.ParseTransactionStream(regTxStream(buildXregHubXRP(manager, 6)))
	require.NoError(t, err)
	require.NotNil(t, msg, "registration publish should produce a message")

	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)

	// The synthesized emitter MUST equal the transfer-path emitter for the same
	// (manager, XRP sourceToken=zeros) — this is the core correctness guarantee.
	var zeroToken [32]byte
	expected := p.calculateEmitterAddress(manager32From20(manager), zeroToken)
	assert.Equal(t, expected, msg.EmitterAddress, "hub emitter must match transfer emitter")

	// Payload is a canonical WormholeTransceiverInfo: prefix(4) + manager(32) +
	// mode(1) + token(32) + decimals(1) = 70 bytes.
	require.Equal(t, 70, len(msg.Payload))
	assert.Equal(t, transceiverInfoPrefix[:], msg.Payload[0:4])
	assert.Equal(t, manager32From20(manager), [32]byte(msg.Payload[4:36]))
	assert.Equal(t, uint8(nttModeLocking), msg.Payload[36])
	assert.Equal(t, zeroToken, [32]byte(msg.Payload[37:69]))
	assert.Equal(t, uint8(6), msg.Payload[69])
}

func TestRegistration_HubIOU_EmitterMatchesTransferPath(t *testing.T) {
	p := NewParser(testRegCoreAccount, nil, nil)
	manager := testRegManager20()

	// Normalized currency (20) + issuer account id (20), as carried in XREG.
	var currency, issuer [20]byte
	copy(currency[12:15], []byte("FOO"))
	for i := range issuer {
		issuer[i] = byte(0xA0 + i)
	}

	msg, err := p.ParseTransactionStream(regTxStream(buildXregHubIOU(manager, currency, issuer, 9)))
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Build the expected IOU sourceToken the SAME way the transfer path does:
	// 0x01 || keccak256(currency20 || issuer20)[1:].
	sourceToken, consumed, err := regSourceTokenFromWire(append([]byte{xregTokenIOU}, append(currency[:], issuer[:]...)...))
	require.NoError(t, err)
	require.Equal(t, 41, consumed)

	expected := p.calculateEmitterAddress(manager32From20(manager), sourceToken)
	assert.Equal(t, expected, msg.EmitterAddress)

	// Canonical info payload carries the same sourceToken + decimals=9.
	require.Equal(t, 70, len(msg.Payload))
	assert.Equal(t, sourceToken, [32]byte(msg.Payload[37:69]))
	assert.Equal(t, uint8(9), msg.Payload[69])
}

func TestRegistration_PeerXRP(t *testing.T) {
	p := NewParser(testRegCoreAccount, nil, nil)
	manager := testRegManager20()

	var peerAddr [32]byte
	for i := range peerAddr {
		peerAddr[i] = byte(0x10 + i)
	}
	const peerChain = uint16(6) // Avalanche

	msg, err := p.ParseTransactionStream(regTxStream(buildXregPeerXRP(manager, peerChain, peerAddr)))
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Emitter is keyed on the SAME (manager, XRP token) as the hub/transfers.
	var zeroToken [32]byte
	expected := p.calculateEmitterAddress(manager32From20(manager), zeroToken)
	assert.Equal(t, expected, msg.EmitterAddress)

	// Payload is a canonical WormholeTransceiverRegistration:
	// prefix(4) + chain_id(2) + transceiver_address(32) = 38 bytes.
	require.Equal(t, 38, len(msg.Payload))
	assert.Equal(t, transceiverRegPrefix[:], msg.Payload[0:4])
	assert.Equal(t, peerChain, binary.BigEndian.Uint16(msg.Payload[4:6]))
	assert.Equal(t, peerAddr, [32]byte(msg.Payload[6:38]))
}

func TestRegistration_WrongMemoFormat_NotRegistration(t *testing.T) {
	p := NewParser(testRegCoreAccount, nil, nil)
	tx := regTxStream(buildXregHubXRP(testRegManager20(), 6))
	// A valid core publish (version 1 + nonce + payload) under coreMemoFormat must
	// NOT be parsed as a registration: parseRegistrationTransaction returns nil and
	// the generic core parser handles it (emitter = sender, not the keccak emitter).
	coreMemo := append([]byte{0x01, 0, 0, 0, 0}, []byte("hello")...)
	memos := tx.Transaction["Memos"].([]any)
	memo := memos[0].(map[string]any)["Memo"].(map[string]any)
	memo["MemoFormat"] = coreMemoFormat
	memo["MemoData"] = hex.EncodeToString(coreMemo)

	// A valid core publish parses (as a generic core message), but must NOT be a
	// transceiver-info payload — proving the registration parser declined it.
	msg, err := p.ParseTransactionStream(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.NotEqual(t, transceiverInfoPrefix[:], msg.Payload[0:4],
		"non-XREG memo must not synthesize a transceiver-info payload")
}
