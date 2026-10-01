package evm

import (
	"time"

	"codeql/messagepublicationcanonicaltimestamp/node/pkg/common"
)

func unsupportedFieldAssignment(blockTime uint64) common.MessagePublication {
	publication := common.MessagePublication{}
	publication.Timestamp = time.Unix(int64(blockTime), 0)
	return publication
}

func unsupportedFunctionValue(blockTime uint64) common.MessagePublication {
	converter := time.Unix
	return common.MessagePublication{
		Timestamp: converter(int64(blockTime), 0),
	}
}
