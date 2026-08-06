package xrpl

type ChainID int

type chainIDs struct {
	ChainIDXRPL   ChainID
	ChainIDSolana ChainID
}

var vaa chainIDs

type Address [20]byte
type Hash [32]byte

type MessagePublication struct {
	EmitterChain   ChainID
	EmitterAddress Hash
	Payload        []byte
}

var xtcfPrefix = []byte("XTCF")
var xackPrefix = []byte("XACK")
var generatedEmitterPrefix = []byte("XRPL")

func addressToEmitter(account Address) Hash {
	var out Hash
	copy(out[12:], account[:])
	return out
}

func Keccak256(data []byte) Hash {
	var out Hash
	copy(out[:], data)
	return out
}

func calculateGeneratedEmitterAddress(account Address) Hash {
	out := addressToEmitter(account)
	copy(out[:4], generatedEmitterPrefix)
	return out
}

func calculateEmitterAddress(sourceNTTManager Hash, sourceToken Hash) Hash {
	buf := make([]byte, 67)
	copy(buf[:3], "ntt")
	copy(buf[3:35], sourceNTTManager[:])
	copy(buf[35:67], sourceToken[:])
	return Keccak256(buf)
}

func buildNTTPayload(sourceToken Hash) []byte {
	return sourceToken[:]
}

func sha256Hash(data []byte) Hash {
	var out Hash
	copy(out[:], data)
	return out
}
