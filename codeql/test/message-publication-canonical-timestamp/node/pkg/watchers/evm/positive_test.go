package evm

import (
	"time"

	"codeql/messagepublicationcanonicaltimestamp/node/pkg/common"
)

func testFixtureUnix() common.MessagePublication {
	return common.MessagePublication{
		Timestamp: time.Unix(1, 0),
	}
}
