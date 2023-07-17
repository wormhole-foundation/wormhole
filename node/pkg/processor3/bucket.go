package processor3

import "encoding/binary"

var numPriorityBuckets = 8
var numProcessingBuckets = 10 // must be greater than numPriorityBuckets

func calculateLeaderSetSize(guardianSetSize int) int {
	return guardianSetSize/3 + 1
}

func calculateBucket(hash []byte, myGuardianSetIndex int, numGuardians int) (inLeaderSet bool, processingBucket int) {
	if !EnableLeaderSets {
		inLeaderSet = true
	}
	// determine if this guardian is responsible for this observation
	hashId := binary.BigEndian.Uint64(hash)
	targetIdx := int(hashId % uint64(numGuardians)) // TODO support variable size guardian set
	r := (targetIdx + myGuardianSetIndex) % numGuardians
	if r < calculateLeaderSetSize(numGuardians) {
		inLeaderSet = true
	}

	if r == 0 {
		// we are the "primary leader" -- assign this to one of the priority buckets
		processingBucket = int(targetIdx % numPriorityBuckets)
	} else {
		processingBucket = int(targetIdx%(numProcessingBuckets-numPriorityBuckets)) + numPriorityBuckets
	}

	return
}
