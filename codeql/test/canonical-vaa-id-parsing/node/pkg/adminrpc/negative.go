package adminrpc

import (
	guardianDB "codeql/canonicalvaaidparsing/node/pkg/db"
	"codeql/canonicalvaaidparsing/node/pkg/vaa"
	"fmt"
	"strconv"
	"strings"
)

type typedRequest struct {
	EmitterChain   uint16
	EmitterAddress vaa.Address
	Sequence       uint64
}

func canonicalPinnedParser(id string, s *server) (bool, error) {
	vaaID, err := guardianDB.VaaIDFromString(id)
	if err != nil {
		return false, err
	}
	return s.db.HasVAA(*vaaID)
}

func canonicalFutureParser(id string) (*vaa.VAAID, error) {
	parsed, err := vaa.VAAIDFromString(id)
	if err != nil {
		return nil, err
	}
	if parsed.EmitterChain == 0 {
		return nil, fmt.Errorf("unsupported emitter chain")
	}
	return parsed, nil
}

func typedFieldConstruction(req typedRequest) guardianDB.VAAID {
	return guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(req.EmitterChain),
		EmitterAddress: req.EmitterAddress,
		Sequence:       req.Sequence,
	}
}

func stringProducerOnly(req typedRequest) string {
	return fmt.Sprintf("%d/%s/%d", req.EmitterChain, req.EmitterAddress, req.Sequence)
}

type msgID struct {
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	Sequence       uint64
}

func txverifierNearMiss(id string) (msgID, error) {
	parts := strings.Split(id, "/")
	seq, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return msgID{}, err
	}
	return msgID{EmitterAddress: vaa.Address([]byte(parts[1])), Sequence: seq}, nil
}

func cliNearMiss(id string) (string, string, string) {
	parts := strings.Split(id, "/")
	return parts[0], parts[1], parts[2]
}
