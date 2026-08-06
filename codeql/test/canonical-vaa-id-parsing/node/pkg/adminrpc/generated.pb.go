package adminrpc

import (
	guardianDB "codeql/canonicalvaaidparsing/node/pkg/db"
	"codeql/canonicalvaaidparsing/node/pkg/vaa"
	"strings"
)

func generatedManualParse(id string) guardianDB.VAAID {
	parts := strings.Split(id, "/")
	return guardianDB.VAAID{EmitterAddress: vaa.Address([]byte(parts[1]))}
}
