package querystaking

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// IPFS-related metrics
var (
	ipfsFetchErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_ipfs_fetch_errors_total",
			Help: "Total number of IPFS fetch errors by error type",
		}, []string{"error_type"})

	ipfsCacheHitRate = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ccq_ipfs_cache_total",
			Help: "Total number of IPFS cache lookups by result",
		}, []string{"result"})
)

// IPFSClient handles fetching and caching conversion tables from IPFS
type IPFSClient struct {
	httpClient *http.Client
	gateway    string
	cache      *sync.Map // CID string -> *ConversionTable
	logger     *zap.Logger
}

// NewIPFSClient creates a new IPFS client
func NewIPFSClient(gateway string, timeout time.Duration, logger *zap.Logger) *IPFSClient {
	return &IPFSClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		gateway: gateway,
		cache:   &sync.Map{},
		logger:  logger.With(zap.String("component", "ipfs-client")),
	}
}

// FetchConversionTable fetches and parses a conversion table from IPFS if not previously retrieved
func (c *IPFSClient) FetchConversionTable(ctx context.Context, cidBytes [32]byte) (*ConversionTable, error) {
	// Convert bytes32 to CID string
	cidStr, err := bytes32ToCIDString(cidBytes)
	if err != nil {
		ipfsFetchErrors.WithLabelValues("cid_parse").Inc()
		return nil, fmt.Errorf("failed to parse CID from bytes32: %w", err)
	}

	// Check cache first
	if cached, ok := c.cache.Load(cidStr); ok {
		if table, ok := cached.(*ConversionTable); ok {
			ipfsCacheHitRate.WithLabelValues("hit").Inc()
			c.logger.Debug("cache hit for conversion table", zap.String("cid", cidStr))
			return table, nil
		}
		// Invalid cache entry, remove it and fall through to fetch
		c.cache.Delete(cidStr)
	}
	ipfsCacheHitRate.WithLabelValues("miss").Inc()

	// Fetch from IPFS
	c.logger.Debug("fetching conversion table from IPFS", zap.String("cid", cidStr))
	conversionTable, err := c.fetchFromIPFS(ctx, cidStr)
	if err != nil {
		// Check cache again in case of network error (stale cache is better than no data)
		if cached, ok := c.cache.Load(cidStr); ok {
			if table, ok := cached.(*ConversionTable); ok {
				c.logger.Warn("IPFS fetch failed, using stale cache", zap.String("cid", cidStr), zap.Error(err))
				return table, nil
			}
		}
		return nil, err
	}

	// Store in cache
	c.cache.Store(cidStr, conversionTable)

	return conversionTable, nil
}

// fetchFromIPFS performs the HTTP GET request to the IPFS gateway
func (c *IPFSClient) fetchFromIPFS(ctx context.Context, cidStr string) (*ConversionTable, error) {
	url := c.gateway + cidStr

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		ipfsFetchErrors.WithLabelValues("request_creation").Inc()
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		ipfsFetchErrors.WithLabelValues("network").Inc()
		c.logger.Error("IPFS HTTP request failed",
			zap.String("url", url),
			zap.Error(err))
		return nil, fmt.Errorf("failed to fetch from IPFS gateway: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ipfsFetchErrors.WithLabelValues("http_status").Inc()
		c.logger.Error("IPFS gateway returned non-200 status",
			zap.String("url", url),
			zap.Int("status", resp.StatusCode))
		return nil, fmt.Errorf("IPFS gateway returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := common.SafeRead(resp.Body)
	if err != nil {
		ipfsFetchErrors.WithLabelValues("read_body").Inc()
		return nil, fmt.Errorf("failed to read IPFS response body: %w", err)
	}

	// Parse JSON
	var conversionTable ConversionTable
	if err := json.Unmarshal(body, &conversionTable); err != nil {
		ipfsFetchErrors.WithLabelValues("json_parse").Inc()
		c.logger.Error("failed to parse conversion table JSON",
			zap.String("cid", cidStr),
			zap.Error(err),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("failed to parse conversion table JSON: %w", err)
	}

	c.logger.Info("successfully fetched conversion table from IPFS",
		zap.String("cid", cidStr),
		zap.String("url", url))

	return &conversionTable, nil
}

// ConversionTranche represents a single tranche in the conversion table
type ConversionTranche struct {
	Rate    uint64 // Queries per minute for this tranche
	Tranche uint64 // Minimum stake amount required for this tranche
}

// ConversionTable represents the IPFS JSON structure for chain-specific conversion rates
type ConversionTable struct {
	EVM    map[string]string `json:"EVM"`
	Solana map[string]string `json:"Solana"`
}

// GetTranchesByChain extracts and parses tranches for a specific chain
// Returns sorted tranches (ascending by tranche amount)
func (ct *ConversionTable) GetTranchesByChain(chainName string) ([]ConversionTranche, error) {
	var chainRates map[string]string

	// Select the appropriate chain data
	switch chainName {
	case "EVM":
		chainRates = ct.EVM
	case "Solana":
		chainRates = ct.Solana
	default:
		return nil, fmt.Errorf("unknown chain: %s", chainName)
	}

	if chainRates == nil {
		return nil, fmt.Errorf("no rates found for chain: %s", chainName)
	}

	// Parse the map into tranches
	tranches := make([]ConversionTranche, 0, len(chainRates))
	for trancheStr, rateStr := range chainRates {
		// Parse tranche amount
		tranche, err := strconv.ParseUint(trancheStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid tranche amount '%s': %w", trancheStr, err)
		}

		// Parse rate string (e.g., "1 QPS" or "5 QPM")
		rate, err := parseRateString(rateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid rate string '%s': %w", rateStr, err)
		}

		tranches = append(tranches, ConversionTranche{
			Rate:    rate,
			Tranche: tranche,
		})
	}

	if len(tranches) == 0 {
		return nil, fmt.Errorf("no valid tranches found for chain: %s", chainName)
	}

	// Sort by tranche amount (ascending)
	sort.Slice(tranches, func(i, j int) bool {
		return tranches[i].Tranche < tranches[j].Tranche
	})

	return tranches, nil
}

// parseRateString parses rate strings like "1 QPS" or "5 QPM" into queries per minute
// QPS (queries per second) is converted to QPM by multiplying by 60
// QPM (queries per minute) is used directly
func parseRateString(rateStr string) (uint64, error) {
	parts := strings.Fields(rateStr)
	if len(parts) != 2 {
		return 0, fmt.Errorf("rate string must have format '<number> <unit>', got: %s", rateStr)
	}

	// Parse the numeric value
	value, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid rate value '%s': %w", parts[0], err)
	}

	// Parse the unit and convert to QPM
	unit := strings.ToUpper(parts[1])
	switch unit {
	case "QPS":
		// Convert QPS to QPM
		return value * 60, nil
	case "QPM":
		// Already in QPM
		return value, nil
	default:
		return 0, fmt.Errorf("unknown rate unit '%s', expected QPS or QPM", parts[1])
	}
}

// bytes32ToCIDString converts a 32-byte hash digest to a CIDv1 base32 string
// The bytes32 contains the raw sha256 hash digest
func bytes32ToCIDString(hashBytes [32]byte) (string, error) {
	// Create a multihash from the sha256 digest
	mh, err := multihash.Encode(hashBytes[:], multihash.SHA2_256)
	if err != nil {
		return "", fmt.Errorf("failed to encode multihash: %w", err)
	}

	// Create CIDv1 with raw codec
	c := cid.NewCidV1(cid.Raw, mh)

	// Encode as base32 (default for CIDv1)
	return c.String(), nil
}
