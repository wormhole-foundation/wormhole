package evm

import (
	t "time"
	"time"

	"codeql/messagepublicationcanonicaltimestamp/node/pkg/common"
	"codeql/messagepublicationcanonicaltimestamp/sdk/vaa"
)

func directUnix(blockTime uint64) common.MessagePublication {
	return common.MessagePublication{
		Timestamp: time.Unix(int64(blockTime), 0),
	}
}

func aliasedUnix(blockTime uint64) common.MessagePublication {
	return common.MessagePublication{
		Timestamp: t.Unix(int64(blockTime), 0),
	}
}

func localUnix(blockTime uint64) common.MessagePublication {
	timestamp := time.Unix(int64(blockTime), 0)
	return common.MessagePublication{
		Timestamp: timestamp,
	}
}

func ignoredTimeFromUnixError(blockTime uint64) common.MessagePublication {
	timestamp, _ := vaa.TimeFromUnix(blockTime)
	return common.MessagePublication{
		Timestamp: timestamp,
	}
}

func fallbackAfterTimeFromUnixError(blockTime uint64) common.MessagePublication {
	timestamp, err := vaa.TimeFromUnix(blockTime)
	if err != nil {
		timestamp = time.Time{}
	}
	return common.MessagePublication{
		Timestamp: timestamp,
	}
}

func publishInErrorBranch(blockTime uint64) (common.MessagePublication, error) {
	timestamp, err := vaa.TimeFromUnix(blockTime)
	if err != nil {
		return common.MessagePublication{
			Timestamp: timestamp,
		}, nil
	}
	return common.MessagePublication{}, err
}

func unixWrapper(blockTime uint64) common.MessagePublication {
	return common.MessagePublication{
		Timestamp: publicationTime(blockTime),
	}
}

func publicationTime(blockTime uint64) time.Time {
	return time.Unix(int64(blockTime), 0)
}
