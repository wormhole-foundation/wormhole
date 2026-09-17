package adminrpc

import (
	guardianDB "codeql/canonicalvaaidparsing/node/pkg/db"
	"codeql/canonicalvaaidparsing/node/pkg/vaa"
	"strconv"
	"strings"
)

func VaaIDFromString(id string) (*guardianDB.VAAID, error) {
	parts := strings.Split(id, "/")
	seq, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return nil, err
	}
	return &guardianDB.VAAID{EmitterAddress: vaa.Address([]byte(parts[1])), Sequence: seq}, nil
}
