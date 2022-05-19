package reporter

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUniqueClientId(t *testing.T) {
	/*
		Rationale:
		Pro: This test does not have false positives. It is guaranteed to fail if the magic value for the maximum client ID changes.
		Con: It takes ca. 0.463s to run this test
	*/

	var almostFullMap = make(map[int]*lifecycleEventChannels, maxClientId)

	firstExpectedValue := 0
	secondExpectedValue := 1

	// build a full map
	for i := 0; i < maxClientId; i++ {
		almostFullMap[i] = nil
	}

	// Test that we can find the empty slot in the map
	delete(almostFullMap, firstExpectedValue)
	re := AttestationEventReporter{sync.RWMutex{}, nil, almostFullMap}
	assert.Equal(t, re.getUniqueClientId(), firstExpectedValue)

	// Test that we can find a different empty slot in the map
	almostFullMap[firstExpectedValue] = nil
	delete(almostFullMap, secondExpectedValue)
	assert.Equal(t, re.getUniqueClientId(), secondExpectedValue)
}
