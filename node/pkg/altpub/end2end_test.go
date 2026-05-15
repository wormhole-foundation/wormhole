package altpub

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	NumObservations = 100

	// These are just meaningless values used to generate observation message IDs.
	pythEmitterAddr   = "00000000000000000000000022427d90B7dA3fA4642F7025A854c7254E4e45BF"
	solanaEmitterAddr = "3b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98"
)

// TestEndToEnd creates an ApplicationPublisher with two endpoints. Each endpoint is handled by a localhost HTTP server running on a unique port.
// - The first simulates what Pyth might use, an immediate publisher for the PythNet chain only.
// - The second simulates what Wormholescan might use, a delayed publisher for all chains.
// The test then blasts a bunch of observations, interleaving PythNet and Solana.
// It then verifies that the results received on the two endpoint servers match what was sent.
func TestEndToEnd(t *testing.T) {
	logger := zap.NewNop()
	guardianAddr, err := hex.DecodeString("13947Bd48b18E53fdAeEe77F3473391aC727C638")
	require.NoError(t, err)
	require.Equal(t, ethCommon.AddressLength, len(guardianAddr))

	logger.Info("Starting two endpoint servers")
	pythEP := newServer(t, logger, guardianAddr)
	wormscanEP := newServer(t, logger, guardianAddr)

	// Create an alternate publisher with two endpoints. Note that the labels start with "e2e_" so our metrics don't clash with other tests.
	ap, err := NewAlternatePublisher(logger, guardianAddr, []string{"e2e_pyth;" + pythEP.url + ";0;pythnet", "e2e_wormholescan;" + wormscanEP.url + ";10ms"})
	require.NoError(t, err)
	require.NotNil(t, ap)
	require.Equal(t, 2, len(ap.endpoints))

	logger.Info("Starting the alternate publisher")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		err := ap.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			require.NoError(t, err)
		}
	}()

	// Give the alternate publisher some time to start.
	time.Sleep(10 * time.Millisecond) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep

	logger.Info("Publishing some observations across pythnet and solana")
	pythSeqNum := 0
	solanaSeqNum := 0
	expectedPythObsv := []*gossipv1.Observation{}
	expectedWormscanObsv := []*gossipv1.Observation{}
	for count := range NumObservations {
		pythObs := createObservation(vaa.ChainIDPythNet, pythEmitterAddr, pythSeqNum)
		ap.PublishObservation(vaa.ChainIDPythNet, pythObs)
		expectedPythObsv = append(expectedPythObsv, pythObs)
		expectedWormscanObsv = append(expectedWormscanObsv, pythObs)
		pythSeqNum++

		if count%5 == 0 {
			solanaObs := createObservation(vaa.ChainIDSolana, solanaEmitterAddr, solanaSeqNum)
			ap.PublishObservation(vaa.ChainIDSolana, solanaObs)
			expectedWormscanObsv = append(expectedWormscanObsv, solanaObs)
			solanaSeqNum++
		}

		// Put in some delay so batching can do something.
		time.Sleep(time.Millisecond) //nolint:forbidigo // TODO: This code should be refactored to not use time.Sleep
	}

	logger.Info("Waiting for all observations to be received before shutting down")
	require.Eventually(t, func() bool {
		return len(pythEP.getStatus().observations) == len(expectedPythObsv) &&
			len(wormscanEP.getStatus().observations) == len(expectedWormscanObsv)
	}, 5*time.Second, 5*time.Millisecond)

	logger.Info("Canceling context")
	cancel()

	// Make sure we didn't drop anything.
	require.Equal(t, 0.0, getCounterValue(obsvDropped, "e2e_pyth"))
	require.Equal(t, 0.0, getCounterValue(obsvDropped, "e2e_wormholescan"))

	// Get the results from the endpoint servers.
	pythStats := pythEP.getStatus()
	wormscanStats := wormscanEP.getStatus()

	// Since the workers may publish the observations out of order, we need to sort the observations.
	sortObservations(expectedPythObsv)
	sortObservations(expectedWormscanObsv)
	sortObservations(pythStats.observations)
	sortObservations(wormscanStats.observations)

	// Since the pyth endpoint is immediate, the number of batches should equal the number of pythnet observations
	assert.Equal(t, len(expectedPythObsv), pythStats.numBatches)
	compareObservations(t, expectedPythObsv, pythStats.observations, "pyth")

	// The wormholescan endpoint is listening to everything, so it should see the total of both pythnet and solana observations.
	// Since it is using batching, it should see fewer batches than the number of observations. I don't think we can predict the exact number.
	assert.Greater(t, len(expectedWormscanObsv), wormscanStats.numBatches)
	compareObservations(t, expectedWormscanObsv, wormscanStats.observations, "wormholescan")

	logger.Info("Exiting")
}

// createObservation creates a completely bogus unique observation so we can compare sent to received.
func createObservation(emitterChain vaa.ChainID, emitterAddress string, seqNum int) *gossipv1.Observation {
	messageId := fmt.Sprintf("%d/%s/%d", emitterChain, emitterAddress, seqNum)
	txHash := crypto.Keccak256Hash([]byte(messageId)).Bytes()
	digest := crypto.Keccak256Hash(txHash).Bytes()
	sig := append(txHash, digest...)
	return &gossipv1.Observation{
		Hash:      digest,
		Signature: sig,
		TxHash:    txHash,
		MessageId: messageId,
	}
}

// sortObservations sorts a slice of observations into ascending order.
func sortObservations(observations []*gossipv1.Observation) {
	slices.SortFunc(observations, func(a, b *gossipv1.Observation) int {
		if a.MessageId < b.MessageId {
			return -1
		}
		if a.MessageId > b.MessageId {
			return 1
		}
		return 0
	})
}

// compareObservations compares two slices of observations to make sure they are equal.
func compareObservations(t *testing.T, expected []*gossipv1.Observation, actual []*gossipv1.Observation, tag string) {
	t.Helper()
	require.Equal(t, len(expected), len(actual))
	for idx := range expected {
		t.Run(fmt.Sprintf("%s-%d", tag, idx), func(t *testing.T) {
			require.True(t, bytes.Equal(expected[idx].Hash, actual[idx].Hash))
			require.True(t, bytes.Equal(expected[idx].Signature, actual[idx].Signature))
			require.True(t, bytes.Equal(expected[idx].TxHash, actual[idx].TxHash))
			require.Equal(t, expected[idx].MessageId, actual[idx].MessageId)
		})
	}
}

/////// Below here is the implementation of our test HTTP server. It just counts and stores observations and batches.

type (
	// Server represents a single endpoint.
	Server struct {
		logger       *zap.Logger
		guardianAddr []byte
		url          string
		statsLock    sync.Mutex
		stats        ServerStats
	}

	// ServerStats is the data protected by the lock.
	ServerStats struct {
		numBatches   int
		observations []*gossipv1.Observation
	}
)

// newServer creates a new localhost HTTP test server on a free port and registers cleanup with t.
func newServer(t *testing.T, logger *zap.Logger, guardianAddr []byte) *Server {
	t.Helper()
	s := &Server{
		logger:       logger.With(zap.String("component", "server")),
		guardianAddr: guardianAddr,
		stats:        newServerStats(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/SignedObservationBatch", s.handleSignedObservationBatch)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	s.url = ts.URL
	return s
}

// newServerStats initializes the server stats object.
func newServerStats() ServerStats {
	return ServerStats{observations: make([]*gossipv1.Observation, 0, 1000)}
}

// handleSignedObservationBatch is the handler for a signed observation.
func (s *Server) handleSignedObservationBatch(w http.ResponseWriter, r *http.Request) {
	body, err := common.SafeRead(r.Body)
	r.Body.Close()
	if err != nil {
		s.logger.Fatal("error extracting body", zap.Error(err))
	}

	var batch gossipv1.SignedObservationBatch
	err = proto.Unmarshal(body, &batch)
	if err != nil {
		s.logger.Fatal("failed to unmarshal batch", zap.Error(err))
	}

	if !slices.Equal(s.guardianAddr, batch.Addr) {
		s.logger.Fatal("invalid guardian address", zap.String("expected", hex.EncodeToString(s.guardianAddr)), zap.String("actual", hex.EncodeToString(batch.Addr)))
	}

	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	s.stats.numBatches++
	s.stats.observations = append(s.stats.observations, batch.Observations...)
}

// getStatus returns the stats for a server.
func (s *Server) getStatus() ServerStats {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	return s.stats
}
