package xrpl

// XRPL registration (XREG) wire/domain vocabulary used by the parser.

// regMemoFormat is the hex-encoded MemoFormat for XREG registration publishes: "application/x-ntt-registration"
const regMemoFormat = "6170706C69636174696F6E2F782D6E74742D726567697374726174696F6E"

// xregPrefix is the 4-byte prefix for XRPL registration payloads
var xregPrefix = [4]byte{'X', 'R', 'E', 'G'}

// Prefixes for canonical NTT transceiver messages (from the NTT spec / ntt-messages)
var transceiverInfoPrefix = [4]byte{0x9c, 0x23, 0xbd, 0x3b} // WormholeTransceiverInfo
var transceiverRegPrefix = [4]byte{0x18, 0xfc, 0x67, 0xc2}  // WormholeTransceiverRegistration

// XREG constants
const (
	xregKindHub  = 0x00 // hub sub-type (byte after prefix), mirrors sequencer XREG_KIND_HUB
	xregKindPeer = 0x01 // peer sub-type (byte after prefix), mirrors sequencer XREG_KIND_PEER
	xregTokenXRP = 0x00 // wire token_id opening byte for XRP, mirrors XrplTokenId
	xregTokenIOU = 0x01 // wire token_id opening byte for issued currencies
	xregTokenMPT = 0x02 // wire token_id opening byte for MPT

	// NTT manager modes (ntt-messages Mode enum). An XRPL custody account is in
	// Burning mode when it is itself the token issuer (it can mint/burn), and
	// Locking mode otherwise (it holds a token issued by another account). XRP
	// has no issuer, so it is always Locking.
	nttModeLocking = 0x00
	nttModeBurning = 0x01

	xregManagerOffset = 5  // offset of the manager field: after prefix(4) + kind(1)
	xregHeaderLen     = 25 // prefix(4) + kind(1) + manager(20); token_id + tail follow
	xregHubTailLen    = 1  // hub tail: token_decimals(1)
	xregPeerTailLen   = 34 // peer tail: peer_chain(2) + peer_address(32)

	xregTokenXRPWireLen = 1  // XRP wire token_id: type(1)
	xregTokenIOUWireLen = 41 // IOU wire token_id: type(1) + currency(20) + issuer(20)
	xregTokenMPTWireLen = 25 // MPT wire token_id: type(1) + mpt_issuance_id(24)

	// Issuer offsets within an XREG wire token_id (used to derive manager mode).
	xregIouIssuerOffset = 21 // IOU issuer starts after type(1) + currency(20)
	xregMptIssuerOffset = 5  // MPT issuer = last 20 bytes of the 24-byte id: after type(1) + sequence(4) (MPTokenIssuanceID = Sequence(4) || Issuer(20))

	transceiverInfoLen = 70 // WormholeTransceiverInfo: prefix(4) + manager(32) + mode(1) + token(32) + decimals(1)
	transceiverRegLen  = 38 // WormholeTransceiverRegistration: prefix(4) + chain_id(2) + transceiver(32)
)
