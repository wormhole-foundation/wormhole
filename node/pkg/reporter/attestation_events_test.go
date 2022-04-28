package reporter

import (
	"sync"
	"testing"
)

func TestGetUniqueClientId(t *testing.T) {
	var almostFullMap = make(map[int]*lifecycleEventChannels)
	var mu sync.RWMutex = sync.RWMutex{}
	limit := 500000

	for i := 0; i < 500000; i++ {
		almostFullMap[i] = nil
	}

	re := AttestationEventReporter{mu, nil, almostFullMap}

	if re.getUniqueClientId() < limit {
		t.Error("getUniqueClientId() did not return a unique Id.")
	}
}
