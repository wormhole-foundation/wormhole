package adminrpc

import (
	guardianDB "codeql/canonicalvaaidparsing/node/pkg/db"
	"codeql/canonicalvaaidparsing/node/pkg/vaa"
	"strconv"
	"strings"
)

type MissingVAA struct {
	VaaKey string
}

type server struct {
	db database
}

type database interface {
	HasVAA(id guardianDB.VAAID) (bool, error)
}

func (s *server) observeMissingASCIIRegression(missingVAA MissingVAA) (bool, error) {
	splits := strings.Split(missingVAA.VaaKey, "/")
	chain, err := strconv.ParseUint(splits[0], 10, 16)
	if err != nil {
		return false, err
	}
	sequence, err := strconv.ParseUint(splits[2], 10, 64)
	if err != nil {
		return false, err
	}
	vaaID := guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(chain),
		EmitterAddress: vaa.Address([]byte(splits[1])),
		Sequence:       sequence,
	}
	return s.db.HasVAA(vaaID)
}

func renamedAliasesStillBypassCanonicalParser(input MissingVAA) guardianDB.VAAID {
	messageID := input.VaaKey
	fields := strings.Split(messageID, "/")
	seq, _ := strconv.ParseUint(fields[2], 10, 64)
	return guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(2),
		EmitterAddress: vaa.Address([]byte(fields[1])),
		Sequence:       seq,
	}
}

func helperBypassWithComponentAddressParser(vaaKey string) (guardianDB.VAAID, error) {
	parts := strings.Split(vaaKey, "/")
	chain, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return guardianDB.VAAID{}, err
	}
	addr, err := vaa.StringToAddress(parts[1])
	if err != nil {
		return guardianDB.VAAID{}, err
	}
	seq, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return guardianDB.VAAID{}, err
	}
	return guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(chain),
		EmitterAddress: addr,
		Sequence:       seq,
	}, nil
}

func helperBypassWithShortHex(vaaKey string) guardianDB.VAAID {
	segments := strings.Split(vaaKey, "/")
	return guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(2),
		EmitterAddress: vaa.Address([]byte(segments[1])),
		Sequence:       7,
	}
}

func ignoredCanonicalResultThenManualParse(vaaKey string) guardianDB.VAAID {
	_, _ = guardianDB.VaaIDFromString(vaaKey)
	parts := strings.Split(vaaKey, "/")
	return guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(2),
		EmitterAddress: vaa.Address([]byte(parts[1])),
		Sequence:       9,
	}
}
