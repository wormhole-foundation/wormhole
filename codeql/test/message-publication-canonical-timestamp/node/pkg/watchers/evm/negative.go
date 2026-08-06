package evm

import (
	"time"

	"codeql/messagepublicationcanonicaltimestamp/node/pkg/common"
	fakecommon "codeql/messagepublicationcanonicaltimestamp/node/pkg/watchers/evm/fakecommon"
	"codeql/messagepublicationcanonicaltimestamp/sdk/vaa"
)

func checkedTimeFromUnix(blockTime uint64) (common.MessagePublication, error) {
	timestamp, err := vaa.TimeFromUnix(blockTime)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{
		Timestamp: timestamp,
	}, nil
}

func checkedTimeFromUnixNilGuard(blockTime uint64) (common.MessagePublication, error) {
	timestamp, err := vaa.TimeFromUnix(blockTime)
	if err == nil {
		return common.MessagePublication{
			Timestamp: timestamp,
		}, nil
	}
	return common.MessagePublication{}, err
}

func localWallClock() common.MessagePublication {
	return common.MessagePublication{
		Timestamp: time.Unix(time.Now().Unix(), 0),
	}
}

func typedTimeFromParser(timestamp time.Time) common.MessagePublication {
	return common.MessagePublication{
		Timestamp: timestamp,
	}
}

func checkedWrapper(blockTime uint64) (common.MessagePublication, error) {
	timestamp, err := canonicalPublicationTime(blockTime)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{
		Timestamp: timestamp,
	}, nil
}

func canonicalPublicationTime(blockTime uint64) (time.Time, error) {
	return vaa.TimeFromUnix(blockTime)
}

func unrelatedMessagePublicationType(blockTime uint64) fakecommon.MessagePublication {
	return fakecommon.MessagePublication{
		Timestamp: time.Unix(int64(blockTime), 0),
	}
}
