package logs

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var lines = `Program F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF invoke [1]
Program log: Unstake NFT Call
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]
Program log: Instruction: Transfer
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 3121 of 157104 compute units
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]
Program log: Instruction: CloseAccount
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 2297 of 150388 compute units
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success
Program F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF consumed 55122 of 200000 compute units
Program F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF success
Program F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF invoke [1]
Program log: Unstake NFT Call
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]
Program log: Instruction: Transfer
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 3121 of 157104 compute units
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA invoke [2]
Program log: Instruction: CloseAccount
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA consumed 2297 of 150388 compute units
Program TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA success
Program F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF consumed 55122 of 200000 compute units
Program F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF success`

func TestParseLogs(t *testing.T) {
	val, err := ParseLogs(strings.Split(lines, "\n"))
	if err != nil {
		panic(err)
	}

	b, _ := json.Marshal(val)
	require.Equal(t, "[{\"Program\":\"F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF\",\"Success\":true,\"Error\":\"\",\"Logs\":[{\"Type\":1,\"Data\":null,\"String\":\"Unstake NFT Call\"}],\"ComputeConsumed\":55122,\"ComputeAvailable\":200000,\"Subcalls\":[{\"Program\":\"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA\",\"Success\":true,\"Error\":\"\",\"Logs\":[{\"Type\":1,\"Data\":null,\"String\":\"Instruction: Transfer\"}],\"ComputeConsumed\":3121,\"ComputeAvailable\":157104,\"Subcalls\":null},{\"Program\":\"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA\",\"Success\":true,\"Error\":\"\",\"Logs\":[{\"Type\":1,\"Data\":null,\"String\":\"Instruction: CloseAccount\"}],\"ComputeConsumed\":2297,\"ComputeAvailable\":150388,\"Subcalls\":null}]},{\"Program\":\"F11YDwLVireDZ7zFgnjo3psyiSCW3oumYsWWaXbqR5bF\",\"Success\":true,\"Error\":\"\",\"Logs\":[{\"Type\":1,\"Data\":null,\"String\":\"Unstake NFT Call\"}],\"ComputeConsumed\":55122,\"ComputeAvailable\":200000,\"Subcalls\":[{\"Program\":\"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA\",\"Success\":true,\"Error\":\"\",\"Logs\":[{\"Type\":1,\"Data\":null,\"String\":\"Instruction: Transfer\"}],\"ComputeConsumed\":3121,\"ComputeAvailable\":157104,\"Subcalls\":null},{\"Program\":\"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA\",\"Success\":true,\"Error\":\"\",\"Logs\":[{\"Type\":1,\"Data\":null,\"String\":\"Instruction: CloseAccount\"}],\"ComputeConsumed\":2297,\"ComputeAvailable\":150388,\"Subcalls\":null}]}]", string(b))
}
