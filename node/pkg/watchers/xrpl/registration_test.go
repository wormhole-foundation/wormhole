package xrpl

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	addresscodec "github.com/Peersyst/xrpl-go/address-codec"
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

// managerAddress derives the classic r-address for a 20-byte account id — the
// XRPL sender that a genuine XREG publish for that manager is signed by.
func managerAddress(t *testing.T, m [20]byte) string {
	t.Helper()
	addr, err := addresscodec.EncodeAccountIDToClassicAddress(m[:])
	require.NoError(t, err)
	return addr
}

// newRegParser builds a Parser whose managed set contains `manager`, i.e. the
// XREG sender is trusted. This is the setup a genuine registration publish uses.
func newRegParser(t *testing.T, manager [20]byte) *Parser {
	t.Helper()
	return NewParser(testRegCoreAccount, []string{managerAddress(t, manager)}, nil)
}

// buildXregHubXRP builds an XREG Hub payload for an XRP custody (matching the
// sequencer's payloads.rs serialization): prefix + kind(0) + manager(20) +
// token_id(XRP=0x00) + decimals(1).
func buildXregHubXRP(manager [20]byte) []byte {
	out := append([]byte{}, xregPrefix[:]...)
	out = append(out, xregKindHub)
	out = append(out, manager[:]...)
	out = append(out, xregTokenXRP) // token_id wire
	out = append(out, xrpDecimals)  // XRP decimals are fixed at 6
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

// buildXregHubMPT builds an XREG Hub for an MPT custody:
// prefix + kind(0) + manager(20) + token_id(0x02 + mpt_issuance_id24) + decimals.
func buildXregHubMPT(manager [20]byte, mptID [24]byte, decimals uint8) []byte {
	out := append([]byte{}, xregPrefix[:]...)
	out = append(out, xregKindHub)
	out = append(out, manager[:]...)
	out = append(out, xregTokenMPT)
	out = append(out, mptID[:]...)
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

// regTxStream builds a registration publish signed by `sender` (the XRPL
// Account field) carrying the given XREG memo, targeting the core account.
func regTxStream(sender string, memoData []byte) *streamtypes.TransactionStream {
	return &streamtypes.TransactionStream{
		Hash:         "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-06-15T14:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         sender,
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
	manager := testRegManager20()
	p := newRegParser(t, manager)

	msg, err := p.ParseTransactionStream(regTxStream(managerAddress(t, manager), buildXregHubXRP(manager)))
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
	require.Equal(t, transceiverInfoLen, len(msg.Payload))
	assert.Equal(t, transceiverInfoPrefix[:], msg.Payload[0:4])
	assert.Equal(t, manager32From20(manager), [32]byte(msg.Payload[4:36]))
	assert.Equal(t, uint8(nttModeLocking), msg.Payload[36])
	assert.Equal(t, zeroToken, [32]byte(msg.Payload[37:69]))
	assert.Equal(t, uint8(xrpDecimals), msg.Payload[69])
}

func TestRegistration_HubIOU_EmitterMatchesTransferPath(t *testing.T) {
	manager := testRegManager20()
	p := newRegParser(t, manager)

	// Normalized currency (20) + issuer account id (20), as carried in XREG.
	var currency, issuer [20]byte
	copy(currency[12:15], []byte("FOO"))
	for i := range issuer {
		issuer[i] = byte(0xA0 + i)
	}

	msg, err := p.ParseTransactionStream(regTxStream(managerAddress(t, manager), buildXregHubIOU(manager, currency, issuer, 9)))
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Build the expected IOU sourceToken the SAME way the transfer path does:
	// 0x01 || keccak256(currency20 || issuer20)[1:].
	sourceToken, consumed, err := regSourceTokenFromWire(append([]byte{xregTokenIOU}, append(currency[:], issuer[:]...)...))
	require.NoError(t, err)
	require.Equal(t, xregTokenIOUWireLen, consumed)

	expected := p.calculateEmitterAddress(manager32From20(manager), sourceToken)
	assert.Equal(t, expected, msg.EmitterAddress)

	// issuer != manager, so this custody account is in Locking mode.
	assert.Equal(t, uint8(nttModeLocking), msg.Payload[36], "issuer != manager => Locking")

	// Canonical info payload carries the same sourceToken + decimals=9.
	require.Equal(t, transceiverInfoLen, len(msg.Payload))
	assert.Equal(t, sourceToken, [32]byte(msg.Payload[37:69]))
	assert.Equal(t, uint8(9), msg.Payload[69])
}

// TestRegistration_HubIOU_BurningWhenManagerIsIssuer verifies the manager mode
// is Burning when the custody account is itself the token issuer.
func TestRegistration_HubIOU_BurningWhenManagerIsIssuer(t *testing.T) {
	manager := testRegManager20()
	p := newRegParser(t, manager)

	var currency [20]byte
	copy(currency[12:15], []byte("FOO"))
	// issuer == manager => Burning mode.
	issuer := manager

	msg, err := p.ParseTransactionStream(regTxStream(managerAddress(t, manager), buildXregHubIOU(manager, currency, issuer, 9)))
	require.NoError(t, err)
	require.NotNil(t, msg)

	require.Equal(t, transceiverInfoLen, len(msg.Payload))
	assert.Equal(t, uint8(nttModeBurning), msg.Payload[36], "manager == issuer => Burning")
}

// TestRegistration_HubMPT_BurningWhenManagerIsIssuer verifies mode derivation for
// MPT, whose issuer is the first 20 bytes of the mpt_issuance_id.
func TestRegistration_HubMPT_BurningWhenManagerIsIssuer(t *testing.T) {
	manager := testRegManager20()
	p := newRegParser(t, manager)

	// mpt_issuance_id = Sequence(4) || Issuer(20) per XLS-0033; set issuer == manager.
	var mptID [24]byte
	mptID[3] = 0x07               // arbitrary sequence (first 4 bytes)
	copy(mptID[4:24], manager[:]) // issuer is the last 20 bytes

	msg, err := p.ParseTransactionStream(regTxStream(managerAddress(t, manager), buildXregHubMPT(manager, mptID, 6)))
	require.NoError(t, err)
	require.NotNil(t, msg)

	require.Equal(t, transceiverInfoLen, len(msg.Payload))
	assert.Equal(t, uint8(nttModeBurning), msg.Payload[36], "manager == MPT issuer => Burning")
}

func TestRegistration_PeerXRP(t *testing.T) {
	manager := testRegManager20()
	p := newRegParser(t, manager)

	var peerAddr [32]byte
	for i := range peerAddr {
		peerAddr[i] = byte(0x10 + i)
	}
	const peerChain = uint16(6) // Avalanche

	msg, err := p.ParseTransactionStream(regTxStream(managerAddress(t, manager), buildXregPeerXRP(manager, peerChain, peerAddr)))
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Emitter is keyed on the SAME (manager, XRP token) as the hub/transfers.
	var zeroToken [32]byte
	expected := p.calculateEmitterAddress(manager32From20(manager), zeroToken)
	assert.Equal(t, expected, msg.EmitterAddress)

	// Payload is a canonical WormholeTransceiverRegistration:
	// prefix(4) + chain_id(2) + transceiver_address(32) = 38 bytes.
	require.Equal(t, transceiverRegLen, len(msg.Payload))
	assert.Equal(t, transceiverRegPrefix[:], msg.Payload[0:4])
	assert.Equal(t, peerChain, binary.BigEndian.Uint16(msg.Payload[4:6]))
	assert.Equal(t, peerAddr, [32]byte(msg.Payload[6:38]))
}

func TestRegistration_WrongMemoFormat_NotRegistration(t *testing.T) {
	manager := testRegManager20()
	p := newRegParser(t, manager)
	tx := regTxStream(managerAddress(t, manager), buildXregHubXRP(manager))
	// A valid core publish (version 1 + nonce + payload) under coreMemoFormat must
	// NOT be parsed as a registration: parseRegistrationTransaction returns nil and
	// the generic core parser handles it (emitter = sender, not the keccak emitter).
	coreMemo := append([]byte{0x01, 0, 0, 0, 0}, []byte("hello")...)
	memos, ok := tx.Transaction["Memos"].([]any)
	require.True(t, ok, "Memos should be a slice")
	memoWrapper, ok := memos[0].(map[string]any)
	require.True(t, ok, "memo[0] should be a map")
	memo, ok := memoWrapper["Memo"].(map[string]any)
	require.True(t, ok, "Memo should be a map")
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

// TestRegistration_UntrustedSender_Declined ensures a registration publish from
// an account that is NOT a managed (guardian-controlled) account is ignored.
// Otherwise anyone could forge a hub/peer registration by paying the core
// account with a crafted XREG memo.
func TestRegistration_UntrustedSender_Declined(t *testing.T) {
	manager := testRegManager20()
	// Parser trusts `manager`, but the publish is signed by a DIFFERENT account.
	p := newRegParser(t, manager)

	var attacker [20]byte
	for i := range attacker {
		attacker[i] = byte(0xEE)
	}
	tx := regTxStream(managerAddress(t, attacker), buildXregHubXRP(manager))

	msg, err := p.ParseTransactionStream(tx)
	require.NoError(t, err)
	assert.Nil(t, msg, "registration from an untrusted sender must be declined")
}

// TestRegistration_ManagerMismatch_Declined ensures the XREG manager field must
// match the (trusted) sender. A trusted account must not be able to register a
// hub/peer on behalf of a different manager account.
func TestRegistration_ManagerMismatch_Declined(t *testing.T) {
	sender := testRegManager20()

	var otherManager [20]byte
	for i := range otherManager {
		otherManager[i] = byte(0x40 + i)
	}
	// Sender is trusted, but the memo names a different manager.
	p := newRegParser(t, sender)
	tx := regTxStream(managerAddress(t, sender), buildXregHubXRP(otherManager))

	msg, err := p.ParseTransactionStream(tx)
	require.NoError(t, err)
	assert.Nil(t, msg, "registration whose manager != sender must be declined")
}
