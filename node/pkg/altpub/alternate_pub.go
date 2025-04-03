package altpub

// Please see `node/pkg/altpub/README.md` for an overview of this feature.

// TODO: Think about transport tuning parameters.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	// PubChanSize is the size of the channel used to communicate with the pool of HTTP workers.
	PubChanSize = 1000

	// ObservationChanSize is the size of the channel used to post observations to the batching worker for a given endpoint.
	ObservationChanSize = 1000

	// NumWorkersPerEndpoint is how many HTTP workers will be created per endpoint (so we create NumWorkersPerEndpoint * len(endpoints)).
	NumWorkersPerEndpoint = 10

	// These are used to create the http.Transport used by the http.Client. See https://www.loginradius.com/blog/engineering/tune-the-go-http-client-for-high-performance/ for tuning details.
	// TODO: What values make sense here? Maybe use NumWorkersPerEndpoint for last two and NumWorkersPerEndpoint * len(endpoints) for first one?
	MaxIdleConns        = 100
	MaxConnsPerHost     = 100
	MaxIdleConnsPerHost = 100

	// HttpClientTimeout is the timeout used on the HTTP client connection.
	HttpClientTimeout = 10 * time.Second
)

type (
	// AlternatePublisher is used to manage alternate publishing. There is a single instance for a guardian if alternate publishing is enabled.
	AlternatePublisher struct {
		// logger uses the "altpub" component to identify our log messages.
		logger *zap.Logger

		// guardianAddr is used in gossipv1.SignedObservationBatch.
		guardianAddr []byte

		// endpoints is the list of enabled endpoints. Must not be empty.
		endpoints Endpoints

		// httpWorkerChan is the channel used to post requests to the HTTP worker pool.
		httpWorkerChan chan *HttpRequest

		// status is the static string returned by GetFeatures. It gets published in the p2p heartbeats.
		status string
	}

	// Endpoint defines a single endpoint to which we should publish.
	Endpoint struct {
		// label is used to identify the endpoint in the logs and Prometheus metrics.
		label string

		// baseUrl is the URL string before adding the topic.
		baseUrl string

		// delay is how long to delay for batching, zero for publish immediately.
		delay Delay

		// enabledChains is the set of chains for which we should publish events on this endpoint, empty means all.
		enabledChains EnabledChains

		// obsvBatchChan is used to post individual observations to the batch worker.
		obsvBatchChan chan *gossipv1.Observation

		// signedObservationUrl is an optimization so we don't have to format the URL on each post.
		signedObservationUrl string
	}

	// Endpoints defines the list of enabled endpoints.
	Endpoints []*Endpoint

	// Delay extends time.Duration to allow us to override `String`.
	Delay time.Duration

	// EnabledChains tracks the chains for which publishing is enabled.
	// - If the map is empty, all chains are enabled.
	// - If `exceptFor` is false, then the chains in the map are enabled.
	// - If `exceptFor is true, then all but the chains in the map are enabled.
	EnabledChains struct {
		exceptFor bool
		chains    map[vaa.ChainID]struct{}
	}

	// HttpRequest is the object sent to the worker for publishing.
	HttpRequest struct {
		// start is used for computing wormhole_alt_pub_channel_delay_in_us.
		start time.Time

		// ep is the endpoint this request is for (used for logging and metrics)
		ep *Endpoint

		// url is the URL used in the POST request.
		url string

		// Data is the body of the POST request.
		data []byte
	}

	// HttpWorker is the data passed to an individual HTTP worker on startup.
	HttpWorker struct {
		// ap provides access to the application publisher in the worker.
		ap *AlternatePublisher

		// client is the global HTTP client used by the application publisher.
		client *http.Client

		// workerId is used for logging.
		workerId int
	}

	// BatchWorker is the data passed to the batch worker for an endpoint that is doing batching (delay not zero).
	BatchWorker struct {
		// ap provides access to the application publisher in the worker.
		ap *AlternatePublisher

		// ep provides access to the endpoint in the worker.
		ep *Endpoint
	}
)

// All of the Prometheus metrics are indexed by the endpoint label.
var (
	obsvDropped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_alt_pub_requests_dropped",
			Help: "Total number of alternate publication requests dropped due to channel overflow",
		}, []string{"endpoint"})

	requestSuccess = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_alt_pub_requests_success",
			Help: "Total number of alternate publication requests that succeeded",
		}, []string{"endpoint"})

	requestFailed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_alt_pub_requests_failed",
			Help: "Total number of alternate publication requests that failed by failure reason",
		}, []string{"endpoint", "reason"})

	channelDelay = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wormhole_alt_pub_channel_delay_in_us",
			Help:    "Latency histogram for the time it took the request to reach the publisher in microseconds",
			Buckets: []float64{10.0, 100.0, 500.0, 1000.0, 2000.0, 5000.0, 10000.0, 100000.0},
		}, []string{"endpoint"})

	postTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wormhole_alt_pub_post_time_in_ms",
			Help:    "Latency histogram for the time it took to post a request in milliseconds",
			Buckets: []float64{10.0, 100.0, 500.0, 1000.0, 2000.0, 5000.0, 10000.0, 100000.0},
		}, []string{"endpoint"})
)

// String implementation of our duration.
func (d Delay) String() string {
	if d == 0 {
		return "immediate"
	}

	return time.Duration(d).String()
}

// String implementation of our enabled chains map. It prints the enabled chains in chainID order.
func (e EnabledChains) String() string {
	if len(e.chains) == 0 {
		return "all-chains"
	}

	str := ""
	chainIds := slices.Sorted(maps.Keys(e.chains))
	for _, chainId := range chainIds {
		if str != "" {
			str += ","
		}
		str += chainId.String()
	}
	if e.exceptFor {
		return "all-except:" + str
	}
	return str
}

// shouldPublish checks to see if the chain passed in is in is enabled.
func (e EnabledChains) shouldPublish(chainId vaa.ChainID) bool {
	if len(e.chains) == 0 {
		return true
	}

	_, exists := e.chains[chainId]
	if e.exceptFor {
		return !exists
	}
	return exists
}

// NewAlternatePublisher creates an alternate publisher object, validating the endpoint parameters. Returns nil,nil if the feature is not enabled.
func NewAlternatePublisher(logger *zap.Logger, guardianAddr []byte, configs []string) (*AlternatePublisher, error) {
	if len(configs) == 0 {
		return nil, nil
	}

	if len(guardianAddr) != ethCommon.AddressLength {
		return nil, fmt.Errorf("unexpected guardian key length, should be %d, is %d", ethCommon.AddressLength, len(guardianAddr))
	}

	// Validate the endpoint parameters and create endpoint objects.
	endpoints := make([]*Endpoint, len(configs))
	labels := map[string]struct{}{}
	status := ""
	for idx, config := range configs {
		ep, err := parseEndpoint(config)
		if err != nil {
			return nil, err
		}

		if _, exists := labels[ep.label]; exists {
			return nil, fmt.Errorf("duplicate label in --additionalPublishEndpoint '%s'", config)
		}

		labels[ep.label] = struct{}{}
		endpoints[idx] = ep

		if status != "" {
			status += "|"
		}
		status += ep.label
	}

	if len(endpoints) == 0 {
		// Not sure this can happen, but let's be safe!
		return nil, errors.New("there are no enabled endpoints")
	}

	return &AlternatePublisher{
		logger:         logger.With(zap.String("component", "altpub")),
		guardianAddr:   guardianAddr,
		endpoints:      endpoints,
		httpWorkerChan: make(chan *HttpRequest, PubChanSize),
		status:         "altpub:" + status,
	}, nil
}

// parseEndpoint parses an `--additionalPublishEndpoint` parameter into an Endpoint object. It returns an error if the string is invalid.
func parseEndpoint(config string) (*Endpoint, error) {
	fields := strings.Split(config, ";")
	if len(fields) < 2 {
		return nil, fmt.Errorf("not enough fields in --additionalPublishEndpoint '%s': should be at least 2, there are %d", config, len(fields))
	}

	if len(fields) > 4 {
		return nil, fmt.Errorf("too many fields in --additionalPublishEndpoint '%s': may not be more than 4, there are %d", config, len(fields))
	}

	label := fields[0]
	if len(label) == 0 {
		return nil, fmt.Errorf("invalid label in --additionalPublishEndpoint '%s': may not be zero length", config)
	}

	baseUrl := fields[1]
	if valid := common.ValidateURL(baseUrl, []string{"http", "https"}); !valid {
		return nil, fmt.Errorf("invalid url in --additionalPublishEndpoint '%s': must be `http` or `https`", config)
	}

	delay := time.Duration(0)
	if len(fields) > 2 && len(fields[2]) != 0 {
		var err error
		delay, err = time.ParseDuration(fields[2])
		if err != nil {
			return nil, fmt.Errorf("invalid delay duration in --additionalPublishEndpoint '%s', delay %s: %w", config, fields[2], err)
		}
	}

	enabledChainsMap := make(map[vaa.ChainID]struct{})
	exceptFor := false
	if len(fields) > 3 {
		str := fields[3]
		if strings.HasPrefix(str, "-") {
			exceptFor = true
			str = str[1:]
		}
		chainIds := strings.Split(str, ",")
		for _, str := range chainIds {
			chainId, err := vaa.StringToKnownChainID(str)
			if err != nil {
				return nil, fmt.Errorf("invalid chain ID --additionalPublishEndpoint '%s' ('%s'): %w", config, str, err)
			}

			enabledChainsMap[chainId] = struct{}{}
		}
	}

	enabledChains := EnabledChains{exceptFor, enabledChainsMap}

	// Only create an observation batch channel if batching is enabled.
	var obsvBatchChan chan *gossipv1.Observation
	if delay != 0 {
		obsvBatchChan = make(chan *gossipv1.Observation, ObservationChanSize)
	}

	ep := &Endpoint{
		label:         label,
		baseUrl:       baseUrl,
		delay:         Delay(delay),
		enabledChains: enabledChains,
		obsvBatchChan: obsvBatchChan,
	}

	// Format our topic URLs and add the endpoint to the list.
	ep.createUrls()
	return ep, nil
}

// GetFeatures returns the status string to be published in P2P heartbeats. For now, it just returns a static string
// listing the enabled endpoints, but in the future, it might return the actual status of each endpoint or something.
func (ap *AlternatePublisher) GetFeatures() string {
	return ap.status
}

// Run is the runnable for the alternate publisher. It creates the various workers and then waits for shutdown.
func (ap *AlternatePublisher) Run(ctx context.Context) error {
	errC := make(chan error)

	ap.logger.Info("Starting alternate publisher", zap.Int("numEndpoints", len(ap.endpoints)))

	client, err := ap.createClient()
	if err != nil {
		return fmt.Errorf("failed to create http client: %w", err)
	}

	ap.startHttpWorkers(ctx, client, errC)

	for _, ep := range ap.endpoints {
		ap.logger.Info("Enabling endpoint", zap.String("endpoint", ep.label), zap.String("url", ep.baseUrl), zap.Stringer("delay", ep.delay), zap.Stringer("enabledChains", ep.enabledChains))
		if ep.delay != 0 {
			worker := &BatchWorker{ap, ep}
			common.RunWithScissors(ctx, errC, fmt.Sprintf("alt_pub_batcher_%s", ep.label), worker.batchWorker)
		}
	}

	// Wait until shutdown or an error occurs.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

// PublishObservation publishes a signed observation to all the endpoints that care about it. It handles both immediate and delayed publishers.
func (ap *AlternatePublisher) PublishObservation(emitterChain vaa.ChainID, obs *gossipv1.Observation) {
	var data []byte
	for _, ep := range ap.endpoints {
		if !ep.shouldPublish(emitterChain) {
			continue
		}

		if ep.delay == 0 {
			// We are publishing immediately, build the HTTP request and post it directly to the HTTP worker pool.
			if data == nil {
				batch := gossipv1.SignedObservationBatch{
					Addr:         ap.guardianAddr,
					Observations: []*gossipv1.Observation{obs},
				}

				var err error
				data, err = proto.Marshal((&batch))
				if err != nil {
					panic("failed to marshal batch")
				}
			}

			req := &HttpRequest{start: time.Now(), ep: ep, url: ep.signedObservationUrl, data: data}
			select {
			case ap.httpWorkerChan <- req:
			default:
				obsvDropped.WithLabelValues(ep.label).Inc()
			}
		} else {
			// We are batching, post it to the endpoint batching worker.
			select {
			case ep.obsvBatchChan <- obs:
			default:
				obsvDropped.WithLabelValues(ep.label).Inc()
			}
		}
	}
}

// createClient creates the HTTP client using a custom configured transport.
func (ap *AlternatePublisher) createClient() (*http.Client, error) {
	defTrans, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("failed to cast default transport for alternate publisher")
	}

	t := defTrans.Clone()
	t.MaxIdleConns = MaxIdleConns
	t.MaxConnsPerHost = MaxConnsPerHost
	t.MaxIdleConnsPerHost = MaxIdleConnsPerHost

	return &http.Client{
		Timeout:   HttpClientTimeout,
		Transport: t,
	}, nil
}

// startHttpWorkers starts all of the workers in the HTTP worker pool.
func (ap *AlternatePublisher) startHttpWorkers(ctx context.Context, client *http.Client, errC chan error) {
	numWorkers := NumWorkersPerEndpoint * len(ap.endpoints)
	for count := range numWorkers {
		worker := &HttpWorker{ap, client, count}
		common.RunWithScissors(ctx, errC, "alt_pub_worker", worker.httpWorker)
	}
}

// httpWorker is the entrypoint for an HTTP worker. Each httpWorker is responsible for posting an HTTP request and waiting for the response.
func (w *HttpWorker) httpWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case req, ok := <-w.ap.httpWorkerChan:
			if !ok {
				return fmt.Errorf("httpWorker failed to read request because the channel has been closed")
			}
			if err := w.ap.httpPost(ctx, w.client, req); err != nil {
				// These errors are not fatal, so just log it an continue.
				w.ap.logger.Error("failed to post http request", zap.String("endpoint", req.ep.label), zap.Error(err))
			}
		}
	}
}

// httpPost actually posts an HTTP request and waits for the response. It pegs metrics based on the results.
func (ap *AlternatePublisher) httpPost(ctx context.Context, client *http.Client, req *HttpRequest) error {
	channelDelay.WithLabelValues(req.ep.label).Observe(float64(time.Since(req.start).Microseconds()))

	// Create the HTTP POST request using our context so it can be interrupted.
	// Note that we are not using a timeout context because the HTTP client already has a timeout.
	// Note that we make a copy of the payload because `bytes.NewBuffer` takes ownership of the data, and the same data might be in multiple requests.
	r, err := http.NewRequestWithContext(ctx, "POST", req.url, bytes.NewBuffer(bytes.Clone(req.data)))
	if err != nil {
		requestFailed.WithLabelValues(req.ep.label, "create_failed").Inc()
		return fmt.Errorf("create failed: %w", err)
	}
	r.Header.Add("Content-Type", "application/octet-stream")

	start := time.Now()
	resp, err := client.Do(r)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			requestFailed.WithLabelValues(req.ep.label, "connection_refused").Inc()
		} else {
			requestFailed.WithLabelValues(req.ep.label, "post_failed").Inc()
		}
		return fmt.Errorf("post failed: %w", err)
	}
	stop := time.Now()
	resp.Body.Close()

	// Peg the appropriate metric, based on the request result.
	if resp.StatusCode < 300 {
		requestSuccess.WithLabelValues(req.ep.label).Inc()
		postTime.WithLabelValues(req.ep.label).Observe(float64(stop.Sub(start).Milliseconds()))
	} else {
		reason := http.StatusText(resp.StatusCode)
		if reason == "" {
			reason = fmt.Sprintf("status_code_%d", resp.StatusCode)
		}
		requestFailed.WithLabelValues(req.ep.label, reason).Inc()
		return fmt.Errorf("unexpected status code: %d (%s)", resp.StatusCode, reason)
	}

	return nil
}

// createUrls is a helper for the endpoint that formats the URLs for each request type.
// NOTE: Initially there is only one request type, but that may change later.
func (ep *Endpoint) createUrls() {
	ep.signedObservationUrl = ep.baseUrl + "/SignedObservationBatch"
}

// shouldPublish is a helper for the endpoint that checks to see if the chain passed in is in the set of chains enabled for the endpoint.
func (ep *Endpoint) shouldPublish(chainId vaa.ChainID) bool {
	return ep.enabledChains.shouldPublish(chainId)
}

// batchWorker is the entrypoint for a per-endpoint worker that batches observations based on the batching interval.
// This is only created if batching is enabled for the endpoint.
func (w *BatchWorker) batchWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := w.handleBatch(ctx); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}

				return fmt.Errorf("handleBatch failed: %w", err)
			}
		}
	}
}

// handleBatch waits up to the delay interval to batch requests. It then formats a request and posts it to the httpWorkerChan.
func (w *BatchWorker) handleBatch(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(w.ep.delay))
	defer cancel()

	observations, err := common.ReadFromChannelWithTimeout(ctx, w.ep.obsvBatchChan, p2p.MaxObservationBatchSize)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("failed to read observations from the internal observation batch channel: %w", err)
	}

	if len(observations) != 0 {
		batch := gossipv1.SignedObservationBatch{
			Addr:         w.ap.guardianAddr,
			Observations: observations,
		}

		data, err := proto.Marshal((&batch))
		if err != nil {
			panic("failed to marshal batch")
		}

		req := &HttpRequest{start: time.Now(), ep: w.ep, url: w.ep.signedObservationUrl, data: data}
		select {
		case w.ap.httpWorkerChan <- req:
		default:
			obsvDropped.WithLabelValues(w.ep.label).Add(float64(len(observations)))
		}
	}

	return nil
}
