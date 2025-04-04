package altpub

import (
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

func TestDelayString(t *testing.T) {
	type test struct {
		inputUs int64
		output  string
	}
	tests := []test{
		{inputUs: 0, output: "immediate"},
		{inputUs: 250, output: "250µs"},
		{inputUs: 42000, output: "42ms"},
		{inputUs: 7000000, output: "7s"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc.inputUs), func(t *testing.T) {
			delay := time.Microsecond * time.Duration(tc.inputUs)
			require.Equal(t, tc.output, Delay(delay).String())
		})
	}
}

func TestEnabledChainsString(t *testing.T) {
	// NOTE: The output will be sorted in chainID order.
	type test struct {
		input     []vaa.ChainID
		exceptFor bool
		output    string
	}
	tests := []test{
		{input: []vaa.ChainID{}, output: "all-chains"},
		{input: []vaa.ChainID{vaa.ChainIDPythNet}, output: "pythnet"},
		{input: []vaa.ChainID{vaa.ChainIDEthereum, vaa.ChainIDSolana}, output: "solana,ethereum"},
		{input: []vaa.ChainID{vaa.ChainIDPythNet}, exceptFor: true, output: "all-except:pythnet"},
		{input: []vaa.ChainID{vaa.ChainIDEthereum, vaa.ChainIDSolana}, exceptFor: true, output: "all-except:solana,ethereum"},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d", tc.input), func(t *testing.T) {
			enabledChainsMap := make(map[vaa.ChainID]struct{})
			for _, chainID := range tc.input {
				enabledChainsMap[chainID] = struct{}{}
			}
			require.Equal(t, tc.output, EnabledChains{tc.exceptFor, enabledChainsMap}.String())
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	type test struct {
		input     string
		errText   string
		delayUs   int64
		chains    []vaa.ChainID
		exceptFor bool
		label     string
	}
	tests := []test{
		// Success cases
		{label: "Minimum_config", input: "test;http://localhost:3333"},
		{label: "With_delay_no_chains", input: "test;http://localhost:3333;1s", delayUs: 1000000},
		{label: "With_delay_and_one_chain_number", input: "test;http://localhost:3333;200us;1", delayUs: 200, chains: []vaa.ChainID{vaa.ChainIDSolana}},
		{label: "With_delay_and_one_chain_name", input: "test;http://localhost:3333;200us;solana", delayUs: 200, chains: []vaa.ChainID{vaa.ChainIDSolana}},
		{label: "With_delay_and_two_chain_numbers", input: "test;http://localhost:3333;500ms;1,2", delayUs: 500000, chains: []vaa.ChainID{vaa.ChainIDSolana, vaa.ChainIDEthereum}},
		{label: "With_delay_and_chain_number_and_name", input: "test;http://localhost:3333;500ms;1,ethereum", delayUs: 500000, chains: []vaa.ChainID{vaa.ChainIDSolana, vaa.ChainIDEthereum}},
		{label: "With_zero_delay_and_one_chain", input: "test;http://localhost:3333;0;1", chains: []vaa.ChainID{vaa.ChainIDSolana}},
		{label: "With_no_delay_and_one_chain", input: "test;http://localhost:3333;;1", chains: []vaa.ChainID{vaa.ChainIDSolana}},
		{label: "With_no_delay_and_except_one_chain", input: "test;http://localhost:3333;0;-pythnet", chains: []vaa.ChainID{vaa.ChainIDPythNet}, exceptFor: true},
		{label: "With_no_delay_and_except_two_chains", input: "test;http://localhost:3333;0;-pythnet,1", chains: []vaa.ChainID{vaa.ChainIDSolana, vaa.ChainIDPythNet}, exceptFor: true},

		// Error cases
		{label: "Empty", input: "", errText: "not enough fields"},
		{label: "Only_label", input: "test", errText: "not enough fields"},
		{label: "Too_many_fields", input: "1;2;3;4;5", errText: "too many fields"},
		{label: "Empty_label", input: ";http://localhost:3333", errText: "invalid label"},
		{label: "Bad_url", input: "test;ws://localhost:3333", errText: "invalid url"},
		{label: "Bad_delay", input: "test;https://localhost:3333;Hi_Mom", errText: "invalid delay duration"},
		{label: "Empty_chains", input: "test;https://localhost:3333;1h;", errText: "invalid chain ID"},
		{label: "Empty_except_chains", input: "test;https://localhost:3333;1h;-", errText: "invalid chain ID"},
		{label: "Invalid_chains", input: "test;https://localhost:3333;1h;Hi_Mom", errText: "invalid chain ID"},
		{label: "Invalid_chain_in_list", input: "test;https://localhost:3333;1h;1,Hi_Mom,3", errText: "invalid chain ID"},
		{label: "Too_big_chain_ID_in_list", input: "test;https://localhost:3333;1h;1,1000000,3", errText: "invalid chain ID"},
		{label: "Invalid_chain_ID_in_list", input: "test;https://localhost:3333;1h;1,65535,3", errText: "invalid chain ID"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			ep, err := parseEndpoint(tc.input)
			if tc.errText == "" {
				require.NoError(t, err)

				// Verify the delay.
				if tc.delayUs == 0 {
					require.Equal(t, Delay(0), ep.delay)
					require.Nil(t, ep.obsvBatchChan)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.delayUs, time.Duration(ep.delay).Microseconds())
					require.NotNil(t, ep.obsvBatchChan)
					assert.Equal(t, ObservationChanSize, cap(ep.obsvBatchChan))
				}

				require.True(t, slices.Equal(tc.chains, slices.Sorted(maps.Keys(ep.enabledChains.chains))))
				require.Equal(t, tc.exceptFor, ep.enabledChains.exceptFor)
				require.Equal(t, ep.baseUrl+"/SignedObservationBatch", ep.signedObservationUrl)
			} else {
				require.ErrorContains(t, err, tc.errText)
			}
		})
	}
}

func TestNewAlternatePublisher(t *testing.T) {
	logger := zap.NewNop()
	guardianAddr, err := hex.DecodeString("13947Bd48b18E53fdAeEe77F3473391aC727C638")
	require.NoError(t, err)
	require.Equal(t, ethCommon.AddressLength, len(guardianAddr))

	// Empty config (feature disabled) should return nil, nil
	ap, err := NewAlternatePublisher(logger, []byte{}, []string{})
	require.NoError(t, err)
	require.Nil(t, ap)

	// Wrong length guardian key should return an error.
	ap, err = NewAlternatePublisher(logger, guardianAddr[:ethCommon.AddressLength-1], []string{"test;http://localhost:3333"})
	require.ErrorContains(t, err, "unexpected guardian key length")
	require.Nil(t, ap)

	// Bad config entry should return an error.
	ap, err = NewAlternatePublisher(logger, guardianAddr, []string{"test;http://localhost:3333", "test"})
	require.ErrorContains(t, err, "not enough fields")
	require.Nil(t, ap)

	// Duplicate label should return an error.
	ap, err = NewAlternatePublisher(logger, guardianAddr, []string{"test;http://localhost:3333", "test;http://localhost:3334"})
	require.ErrorContains(t, err, "duplicate label")
	require.Nil(t, ap)

	// Success case.
	ap, err = NewAlternatePublisher(logger, guardianAddr, []string{"test1;http://localhost:3333", "test2;http://localhost:3333;500ms;1,2"})
	require.NoError(t, err)
	require.NotNil(t, ap)
	assert.NotNil(t, ap.logger)
	assert.True(t, slices.Equal(guardianAddr, ap.guardianAddr))
	assert.Equal(t, 2, len(ap.endpoints))
	// TestParseEndpoint actually verifies that the endpoints are created properly.
	require.NotNil(t, ap.httpWorkerChan)
	assert.Equal(t, PubChanSize, cap(ap.httpWorkerChan))
}

func TestGetFeatures(t *testing.T) {
	logger := zap.NewNop()
	guardianAddr, err := hex.DecodeString("13947Bd48b18E53fdAeEe77F3473391aC727C638")
	require.NoError(t, err)
	require.Equal(t, ethCommon.AddressLength, len(guardianAddr))

	// One endpoint.
	ap, err := NewAlternatePublisher(logger, guardianAddr, []string{"pyth;http://localhost:3333"})
	require.NoError(t, err)
	require.NotNil(t, ap)
	assert.Equal(t, "altpub:pyth", ap.GetFeatures())

	// Two endpoints.
	ap, err = NewAlternatePublisher(logger, guardianAddr, []string{"pyth;http://localhost:3333", "wormholescan;http://localhost:3334"})
	require.NoError(t, err)
	require.NotNil(t, ap)
	assert.Equal(t, "altpub:pyth|wormholescan", ap.GetFeatures())

	// Three endpoints.
	ap, err = NewAlternatePublisher(logger, guardianAddr, []string{"pyth;http://localhost:3333", "wormholescan;http://localhost:3334", "joe_integrator;http://localhost:3335"})
	require.NoError(t, err)
	require.NotNil(t, ap)
	assert.Equal(t, "altpub:pyth|wormholescan|joe_integrator", ap.GetFeatures())
}

func TestShouldPublish(t *testing.T) {
	ep := &Endpoint{
		label:   "test",
		baseUrl: "some_url",
		delay:   Delay(0),
		enabledChains: EnabledChains{chains: map[vaa.ChainID]struct{}{
			vaa.ChainIDSolana: {},
		}},
	}

	require.True(t, ep.shouldPublish(vaa.ChainIDSolana))
	require.False(t, ep.shouldPublish(vaa.ChainIDEthereum))

	ep = &Endpoint{
		label:   "test",
		baseUrl: "some_url",
		delay:   Delay(0),
		enabledChains: EnabledChains{chains: map[vaa.ChainID]struct{}{
			vaa.ChainIDSolana:   {},
			vaa.ChainIDEthereum: {},
		}},
	}

	require.True(t, ep.shouldPublish(vaa.ChainIDSolana))
	require.True(t, ep.shouldPublish(vaa.ChainIDEthereum))
	require.False(t, ep.shouldPublish(vaa.ChainIDPythNet))

	ep = &Endpoint{
		label:         "test",
		baseUrl:       "some_url",
		delay:         Delay(0),
		enabledChains: EnabledChains{chains: map[vaa.ChainID]struct{}{}},
	}

	require.True(t, ep.shouldPublish(vaa.ChainIDSolana))
	require.True(t, ep.shouldPublish(vaa.ChainIDEthereum))
	require.True(t, ep.shouldPublish(vaa.ChainIDPythNet))
}

func TestShouldPublishExceptFor(t *testing.T) {
	ep := &Endpoint{
		label:   "test",
		baseUrl: "some_url",
		delay:   Delay(0),
		enabledChains: EnabledChains{true, map[vaa.ChainID]struct{}{
			vaa.ChainIDSolana: {},
		}},
	}

	require.False(t, ep.shouldPublish(vaa.ChainIDSolana))
	require.True(t, ep.shouldPublish(vaa.ChainIDEthereum))

	ep = &Endpoint{
		label:   "test",
		baseUrl: "some_url",
		delay:   Delay(0),
		enabledChains: EnabledChains{true, map[vaa.ChainID]struct{}{
			vaa.ChainIDSolana:   {},
			vaa.ChainIDEthereum: {},
		}},
	}

	require.False(t, ep.shouldPublish(vaa.ChainIDSolana))
	require.False(t, ep.shouldPublish(vaa.ChainIDEthereum))
	require.True(t, ep.shouldPublish(vaa.ChainIDPythNet))
}

func TestPublishObservation(t *testing.T) {
	logger := zap.NewNop()
	guardianAddr, err := hex.DecodeString("13947Bd48b18E53fdAeEe77F3473391aC727C638")
	require.NoError(t, err)
	require.Equal(t, ethCommon.AddressLength, len(guardianAddr))

	ap, err := NewAlternatePublisher(logger, guardianAddr, []string{"pyth;http://localhost:3333;0;26", "wormholescan;http://localhost:3333;500ms"})
	require.NoError(t, err)
	require.NotNil(t, ap)

	// When we post something to PythNet, it should go directly to the httpWorkerChan (tagged with the first endpoint).
	// When we post something to any chain, it should go to the obsvBatchChan on the second endpoint.

	ep2 := ap.endpoints[1]

	assert.Equal(t, 0.0, getCounterValue(obsvDropped, "pyth"))
	assert.Equal(t, 0.0, getCounterValue(obsvDropped, "wormholescan"))

	// A Solana observation should go to the second endpoint but not the first.
	ap.PublishObservation(vaa.ChainIDSolana, &gossipv1.Observation{})
	require.Equal(t, 0, len(ap.httpWorkerChan))
	require.Equal(t, 1, len(ep2.obsvBatchChan))

	// A PythNet observation should go to both.
	ap.PublishObservation(vaa.ChainIDPythNet, &gossipv1.Observation{})
	require.Equal(t, 1, len(ap.httpWorkerChan))
	require.Equal(t, 2, len(ep2.obsvBatchChan))

	// Fill up both channels and make sure we don't block. (If this test doesn't complete, then we blocked!)
	for range PubChanSize + 10 {
		ap.PublishObservation(vaa.ChainIDPythNet, &gossipv1.Observation{})
	}

	// Make sure we pegged the drop metric.
	assert.Equal(t, 11.0, getCounterValue(obsvDropped, "pyth"))         // One initial plus ten extra.
	assert.Equal(t, 12.0, getCounterValue(obsvDropped, "wormholescan")) // Two initial plus ten extra.
}

func getCounterValue(metric *prometheus.CounterVec, runnableName string) float64 {
	var m = &dto.Metric{}
	if err := metric.WithLabelValues(runnableName).Write(m); err != nil {
		return 0
	}
	return m.Counter.GetValue()
}
