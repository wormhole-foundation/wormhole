package querystaking

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
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

	sf singleflight.Group
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

	if cached, ok := c.cache.Load(cidStr); ok {
		if table, ok := cached.(*ConversionTable); ok {
			ipfsCacheHitRate.WithLabelValues("hit").Inc()
			return table, nil
		}
		c.cache.Delete(cidStr)
	}

	res, err, shared := c.sf.Do(cidStr, func() (any, error) {
		// Check cache first (in case another goroutine cached it while we were waiting)
		if cached, ok := c.cache.Load(cidStr); ok {
			if table, ok := cached.(*ConversionTable); ok {
				ipfsCacheHitRate.WithLabelValues("singleflight_hit").Inc()
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
			return nil, err
		}

		// Store in cache
		c.cache.Store(cidStr, conversionTable)
		return conversionTable, nil
	})
	if err != nil {
		return nil, err
	}

	// Track when result was shared from another goroutine's fetch
	if shared {
		ipfsCacheHitRate.WithLabelValues("singleflight_shared").Inc()
	}

	// Safe type assertion
	table, ok := res.(*ConversionTable)
	if !ok {
		return nil, fmt.Errorf("singleflight returned unexpected type: %T", res)
	}
	return table, nil
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

	// check hash validity
	if err := verifyContentHash(body, cidStr); err != nil {
		ipfsFetchErrors.WithLabelValues("hash_mismatch").Inc()
		c.logger.Error("IPFS content hash mismatch - possible malicious gateway",
			zap.String("cid", cidStr),
			zap.Error(err))
		return nil, fmt.Errorf("IPFS content integrity check failed for CID %s: %w", cidStr, err)
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

// verifyContentHash checks that the content hashes to the digest specified in the CID
func verifyContentHash(content []byte, cidStr string) error {
	// Parse the CID
	c, err := cid.Decode(cidStr)
	if err != nil {
		return fmt.Errorf("failed to decode CID: %w", err)
	}

	// Extract the multihash from the CID
	mh := c.Hash()

	// Decode the multihash to get the hash algorithm and expected digest
	decoded, err := multihash.Decode(mh)
	if err != nil {
		return fmt.Errorf("failed to decode multihash: %w", err)
	}

	// Verify hash algorithm is SHA2-256
	if decoded.Code != multihash.SHA2_256 {
		return fmt.Errorf("unsupported hash algorithm: %d", decoded.Code)
	}

	// Hash the content
	contentHash := sha256.Sum256(content)

	// Compare
	if !bytes.Equal(contentHash[:], decoded.Digest) {
		return fmt.Errorf("hash mismatch: expected %x, got %x", decoded.Digest, contentHash)
	}

	return nil
}

// RateConfig represents the rate limits for a tranche
type RateConfig struct {
	QPS *uint64 `json:"qps,omitempty"` // Queries per second (optional)
	QPM *uint64 `json:"qpm,omitempty"` // Queries per minute (optional, defaults to QPS*60 if not set)
}

// ConversionTranche represents a single tranche in the conversion table
type ConversionTranche struct {
	RatePerSecond uint64 // Queries per second (0 if not set)
	RatePerMinute uint64 // Queries per minute
	Tranche       uint64 // Minimum stake amount required for this tranche
}

// ConversionTable represents the IPFS JSON structure for chain-specific conversion rates
type ConversionTable struct {
	EVM    map[string]RateConfig `json:"EVM"`
	Solana map[string]RateConfig `json:"Solana"`
}

// GetSupportedChains returns the list of chains that have rates defined in this conversion table
func (ct *ConversionTable) GetSupportedChains() []string {
	var chains []string
	if len(ct.EVM) > 0 {
		chains = append(chains, "EVM")
	}
	if len(ct.Solana) > 0 {
		chains = append(chains, "Solana")
	}
	return chains
}

// GetTranchesByChain extracts and parses tranches for a specific chain
// Returns sorted tranches (ascending by tranche amount)
func (ct *ConversionTable) GetTranchesByChain(chainName string) ([]ConversionTranche, error) {
	var chainRates map[string]RateConfig

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
	for trancheStr, rateConfig := range chainRates {
		// Parse tranche amount
		tranche, err := strconv.ParseUint(trancheStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid tranche amount '%s': %w", trancheStr, err)
		}

		// Validate tranche is not 0 (would cause division by zero in rate calculation)
		if tranche == 0 {
			return nil, fmt.Errorf("tranche amount cannot be 0 (would cause division by zero)")
		}

		// Extract QPS and QPM from the rate config
		var qps, qpm uint64

		if rateConfig.QPS != nil {
			qps = *rateConfig.QPS
		}

		if rateConfig.QPM != nil {
			qpm = *rateConfig.QPM
		} else if rateConfig.QPS != nil {
			// If QPM not specified, derive from QPS
			qpm = qps * 60
		}

		// Validate that at least one rate is specified
		if qps == 0 && qpm == 0 {
			return nil, fmt.Errorf("tranche %s has no rate specified (both qps and qpm are 0 or missing)", trancheStr)
		}

		tranches = append(tranches, ConversionTranche{
			RatePerSecond: qps,
			RatePerMinute: qpm,
			Tranche:       tranche,
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
