package badlayout

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

var xackPrefix = []byte("XACK")
var generatedEmitterPrefix = []byte("XRPL")

func addressToEmitter(account Address) Hash {
	var out Hash
	copy(out[12:], account[:])
	return out
}

func calculateGeneratedEmitterAddress(account Address) Hash {
	out := addressToEmitter(account)
	copy(out[:4], generatedEmitterPrefix)
	out[4] = 1
	return out
}

func positiveXACKBadGeneratedLayout(account Address) MessagePublication {
	payload := make([]byte, 8)
	copy(payload, xackPrefix)
	emitterAddress := calculateGeneratedEmitterAddress(account)
	return MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: emitterAddress,
		Payload:        payload,
	}
}
