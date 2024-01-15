package promremotew

import (
	"bytes"
	"testing"

	prometheusv1 "github.com/certusone/wormhole/node/pkg/proto/prometheus/v1"
	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/require"
)

var writeRequestFixture = &prometheusv1.WriteRequest{
	Metadata: []*prometheusv1.MetricMetadata{
		&prometheusv1.MetricMetadata{
			MetricFamilyName: "http_request_duration_seconds",
			Type:             3,
			Help:             "A histogram of the request duration.",
		},
		{
			MetricFamilyName: "http_requests_total",
			Type:             1,
			Help:             "The total number of HTTP requests.",
		},
		{
			MetricFamilyName: "rpc_duration_seconds",
			Type:             5,
			Help:             "A summary of the RPC duration in seconds.",
		},
		{
			MetricFamilyName: "test_metric1",
			Type:             2,
			Help:             "This is a test metric.",
		},
	},
	Timeseries: []*prometheusv1.TimeSeries{
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_bucket"},
				{Name: "job", Value: "promtool"},
				{Name: "le", Value: "0.1"},
			},
			Samples: []*prometheusv1.Sample{{Value: 33444, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_bucket"},
				{Name: "job", Value: "promtool"},
				{Name: "le", Value: "0.5"},
			},
			Samples: []*prometheusv1.Sample{{Value: 129389, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_bucket"},
				{Name: "job", Value: "promtool"},
				{Name: "le", Value: "1"},
			},
			Samples: []*prometheusv1.Sample{{Value: 133988, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_bucket"},
				{Name: "job", Value: "promtool"},
				{Name: "le", Value: "+Inf"},
			},
			Samples: []*prometheusv1.Sample{{Value: 144320, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_sum"},
				{Name: "job", Value: "promtool"},
			},
			Samples: []*prometheusv1.Sample{{Value: 53423, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_request_duration_seconds_count"},
				{Name: "job", Value: "promtool"},
			},
			Samples: []*prometheusv1.Sample{{Value: 144320, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_requests_total"},
				{Name: "code", Value: "200"},
				{Name: "job", Value: "promtool"},
				{Name: "method", Value: "post"},
			},
			Samples: []*prometheusv1.Sample{{Value: 1027, Timestamp: 1395066363000}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "http_requests_total"},
				{Name: "code", Value: "400"},
				{Name: "job", Value: "promtool"},
				{Name: "method", Value: "post"},
			},
			Samples: []*prometheusv1.Sample{{Value: 3, Timestamp: 1395066363000}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "rpc_duration_seconds"},
				{Name: "job", Value: "promtool"},
				{Name: "quantile", Value: "0.01"},
			},
			Samples: []*prometheusv1.Sample{{Value: 3102, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "rpc_duration_seconds"},
				{Name: "job", Value: "promtool"},
				{Name: "quantile", Value: "0.5"},
			},
			Samples: []*prometheusv1.Sample{{Value: 4773, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "rpc_duration_seconds"},
				{Name: "job", Value: "promtool"},
				{Name: "quantile", Value: "0.99"},
			},
			Samples: []*prometheusv1.Sample{{Value: 76656, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "rpc_duration_seconds_sum"},
				{Name: "job", Value: "promtool"},
			},
			Samples: []*prometheusv1.Sample{{Value: 1.7560473e+07, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "rpc_duration_seconds_count"},
				{Name: "job", Value: "promtool"},
			},
			Samples: []*prometheusv1.Sample{{Value: 2693, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "test_metric1"},
				{Name: "b", Value: "c"},
				{Name: "baz", Value: "qux"},
				{Name: "d", Value: "e"},
				{Name: "foo", Value: "bar"},
				{Name: "job", Value: "promtool"},
			},
			Samples: []*prometheusv1.Sample{{Value: 1, Timestamp: 1}},
		},
		{
			Labels: []*prometheusv1.Label{
				{Name: "__name__", Value: "test_metric1"},
				{Name: "b", Value: "c"},
				{Name: "baz", Value: "qux"},
				{Name: "d", Value: "e"},
				{Name: "foo", Value: "bar"},
				{Name: "job", Value: "promtool"},
			},
			Samples: []*prometheusv1.Sample{{Value: 2, Timestamp: 1}},
		},
	},
}

func TestParseAndPushMetricsTextAndFormat(t *testing.T) {
	input := bytes.NewReader([]byte(`
	# HELP http_request_duration_seconds A histogram of the request duration.
	# TYPE http_request_duration_seconds histogram
	http_request_duration_seconds_bucket{le="0.1"} 33444 1
	http_request_duration_seconds_bucket{le="0.5"} 129389 1
	http_request_duration_seconds_bucket{le="1"} 133988 1
	http_request_duration_seconds_bucket{le="+Inf"} 144320 1
	http_request_duration_seconds_sum 53423 1
	http_request_duration_seconds_count 144320 1
	# HELP http_requests_total The total number of HTTP requests.
	# TYPE http_requests_total counter
	http_requests_total{method="post",code="200"} 1027 1395066363000
	http_requests_total{method="post",code="400"}    3 1395066363000
	# HELP rpc_duration_seconds A summary of the RPC duration in seconds.
	# TYPE rpc_duration_seconds summary
	rpc_duration_seconds{quantile="0.01"} 3102 1
	rpc_duration_seconds{quantile="0.5"} 4773 1
	rpc_duration_seconds{quantile="0.99"} 76656 1
	rpc_duration_seconds_sum 1.7560473e+07 1
	rpc_duration_seconds_count 2693 1
	# HELP test_metric1 This is a test metric.
	# TYPE test_metric1 gauge
	test_metric1{b="c",baz="qux",d="e",foo="bar"} 1 1
	test_metric1{b="c",baz="qux",d="e",foo="bar"} 2 1
	`))
	labels := map[string]string{"job": "promtool"}

	actual, err := MetricTextToWriteRequest(input, labels)
	require.NoError(t, err)

	require.Equal(t, writeRequestFixture, actual)
}

func TestMarshalUnmarshal(t *testing.T) {
	timeseries := []*prometheusv1.TimeSeries{}
	wr := prometheusv1.WriteRequest{Timeseries: timeseries}
	bytes, err := proto.Marshal(&wr)
	require.NoError(t, err)

	newWr := prometheusv1.WriteRequest{}
	err = proto.Unmarshal(bytes, &newWr)
	require.NoError(t, err)
}
