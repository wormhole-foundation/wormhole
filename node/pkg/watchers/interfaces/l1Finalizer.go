package interfaces

type L1Finalizer interface {
	GetLatestFinalizedBlockNumber() uint64
}

// MockL1Finalizer implements the L1Finalizer interface for testing purposes.
type MockL1Finalizer struct {
	LatestFinalizedBlockNumber uint64
}

func (m *MockL1Finalizer) SetLatestFinalizedBlockNumber(latestFinalizedBlockNumber uint64) {
	m.LatestFinalizedBlockNumber = latestFinalizedBlockNumber
}

func (m *MockL1Finalizer) GetLatestFinalizedBlockNumber() uint64 {
	return m.LatestFinalizedBlockNumber
}
