package nttwrong

type ChainID int

type chainIDs struct{ ChainIDXRPL ChainID }

var vaa chainIDs

type Address [20]byte
type Hash [32]byte

type MessagePublication struct {
	EmitterChain   ChainID
	EmitterAddress Hash
	Payload        []byte
}

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

func buildNTTPayload(sourceToken Hash) []byte {
	return sourceToken[:]
}

func calculateEmitterAddress(sourceNTTManager Hash, sourceToken Hash) Hash {
	buf := make([]byte, 67)
	copy(buf[:32], sourceToken[:])
	copy(buf[32:64], sourceNTTManager[:])
	copy(buf[64:67], "ntt")
	return Keccak256(buf)
}

func parseNttTransaction(destination Address, sourceToken Hash) MessagePublication {
	sourceNTTManager := addressToEmitter(destination)
	payload := buildNTTPayload(sourceToken)
	emitterAddress := calculateEmitterAddress(sourceNTTManager, sourceToken)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}
