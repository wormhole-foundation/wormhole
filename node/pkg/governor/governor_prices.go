// This file contains the code to query for and update token prices for the chain governor.
//
// The initial prices are read from the static config (tokens.go). After that, prices are
// queried from CoinGecko. The chain governor then uses the maximum of the static price and
// the latest CoinGecko price. The CoinGecko poll interval is specified by coinGeckoQueryIntervalInMins.

package governor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// The CoinGecko API is documented here: https://www.coingecko.com/en/api/documentation
// An example of the query to be generated: https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd

// coinGeckoQueryIntervalInMins specifies how often we query CoinGecko for prices.
const coinGeckoQueryIntervalInMins = 15

// CoinGecko API response structures for /coins/{platform}/contract/{address} endpoint

// CoinGeckoContractResponse represents the response from CoinGecko's contract address lookup endpoint
type CoinGeckoContractResponse struct {
	ID              string                    `json:"id"`
	Symbol          string                    `json:"symbol"`
	Name            string                    `json:"name"`
	DetailPlatforms map[string]PlatformDetail `json:"detail_platforms"`
	MarketData      MarketData                `json:"market_data"`
}

// PlatformDetail contains platform-specific token information
type PlatformDetail struct {
	DecimalPlace    int    `json:"decimal_place"`
	ContractAddress string `json:"contract_address"`
}

// MarketData contains current market information for a token
type MarketData struct {
	CurrentPrice map[string]float64 `json:"current_price"`
}

// tokensPerCoinGeckoQuery specifies how many tokens will be in each CoinGecko query. The token list will be broken up into chunks of this size.
const tokensPerCoinGeckoQuery = 200

// initCoinGecko builds the set of CoinGecko queries that will be used to update prices. It also starts a go routine to periodically do the queries.
func (gov *ChainGovernor) initCoinGecko(ctx context.Context, run bool) error {
	// Create a slice of all the CoinGecko IDs so we can create the corresponding queries.
	ids := make([]string, 0, len(gov.tokensByCoinGeckoId))
	for id := range gov.tokensByCoinGeckoId {
		ids = append(ids, id)
	}

	// Create the set of queries, breaking the IDs into the appropriate size chunks.
	gov.coinGeckoQueries = createCoinGeckoQueries(ids, tokensPerCoinGeckoQuery, gov.coinGeckoApiKey)
	for queryIdx, query := range gov.coinGeckoQueries {
		gov.logger.Info("coingecko query: ", zap.Int("queryIdx", queryIdx), zap.String("query", query))
	}

	if len(gov.coinGeckoQueries) == 0 {
		gov.logger.Info("did not find any tokens, nothing to do!")
		return nil
	}

	if run {
		if err := supervisor.Run(ctx, "govpricer", gov.priceQuery); err != nil {
			return err
		}
	}

	return nil
}

// createCoinGeckoQueries creates the set of CoinGecko queries, breaking the set of IDs into the appropriate size chunks.
func createCoinGeckoQueries(idList []string, tokensPerQuery int, coinGeckoApiKey string) []string {
	var queries []string
	queryIdx := 0
	tokenIdx := 0
	ids := ""
	first := true
	for _, coinGeckoId := range idList {
		if tokenIdx%tokensPerQuery == 0 && tokenIdx != 0 {
			queries = append(queries, createCoinGeckoQuery(ids, coinGeckoApiKey))
			ids = ""
			first = true
			queryIdx += 1
		}
		if first {
			first = false
		} else {
			ids += ","
		}

		ids += coinGeckoId
		tokenIdx += 1
	}

	if ids != "" {
		queries = append(queries, createCoinGeckoQuery(ids, coinGeckoApiKey))
	}

	return queries
}

// createCoinGeckoQuery creates a CoinGecko query for the specified set of IDs.
func createCoinGeckoQuery(ids string, coinGeckoApiKey string) string {
	params := url.Values{}
	params.Add("ids", ids)
	params.Add("vs_currencies", "usd")

	// If modifying this code, ensure that the test 'TestCoinGeckoPriceChecks' passes when adding a pro API key to it.
	// Since the code requires an API key (which we don't want to publish to git), this
	// part of the test is normally skipped but mods to sensitive places should still be checked
	query := ""
	if coinGeckoApiKey == "" {
		query = "https://api.coingecko.com/api/v3/simple/price?" + params.Encode()
	} else { // Pro version API key path
		params.Add("x_cg_pro_api_key", coinGeckoApiKey)
		query = "https://pro-api.coingecko.com/api/v3/simple/price?" + params.Encode()
	}

	return query
}

// priceQuery is the entry point for the routine that periodically queries CoinGecko for prices.
func (gov *ChainGovernor) priceQuery(ctx context.Context) error {
	// Do a query immediately, then once each interval.
	// We ignore the error because an error would already have been logged, and we don't want to bring down the
	// guardian due to a CoinGecko error. The prices would already have been reverted to the config values.
	_ = gov.queryCoinGecko(ctx)

	ticker := time.NewTicker(time.Duration(coinGeckoQueryIntervalInMins) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Process pending token discoveries before regular price updates
			_ = gov.processPendingTokenDiscovery(ctx)
			_ = gov.queryCoinGecko(ctx)
		}
	}
}

// queryCoinGecko sends a series of one or more queries to the CoinGecko server to get the latest prices. It can
// return an error, but that is only used by the tool that validates the query. In the actual governor,
// it just logs the error and we will try again next interval. If an error happens, any tokens that have
// not been updated will be assigned their pre-configured price.
func (gov *ChainGovernor) queryCoinGecko(ctx context.Context) error {
	result := make(map[string]interface{})

	// Cache buster of Unix timestamp concatenated with random number
	params := url.Values{}
	params.Add("bust", strconv.Itoa(int(time.Now().Unix()))+strconv.Itoa(rand.Int())) // #nosec G404

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Throttle the queries to the CoinGecko API. We query for 200 tokens at a time, so this throttling would
	// allow us to query up to 12,000 tokens in a 15 minute window (the query interval). Currently there are
	// between 1000 and 2000 tokens.
	throttle := make(chan int, 1)
	go func() {
		ticker := time.NewTicker(time.Duration(15) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				throttle <- 1 //nolint:channelcheck // We want this to block for throttling
			case <-ctx.Done():
				return
			}
		}
	}()

	for queryIdx, query := range gov.coinGeckoQueries {
		<-throttle
		query := query + "&" + params.Encode()
		thisResult, err := gov.queryCoinGeckoChunk(query)
		if err != nil {
			gov.logger.Error("CoinGecko query failed", zap.Error(err), zap.Int("queryIdx", queryIdx), zap.String("query", query))
			gov.revertAllPrices()
			return err
		}

		for key, value := range thisResult {
			result[key] = value
		}

		time.Sleep(1 * time.Second)
	}

	now := time.Now()
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	localTokenMap := make(map[string][]*tokenEntry)
	for coinGeckoId, cge := range gov.tokensByCoinGeckoId {
		localTokenMap[coinGeckoId] = cge
	}
	// Also include dynamic tokens in the update
	for coinGeckoId, cge := range gov.dynamicTokensByCoinGeckoId {
		localTokenMap[coinGeckoId] = cge
	}

	for coinGeckoId, data := range result {
		// Check both static and dynamic token maps
		cge, exists := gov.tokensByCoinGeckoId[coinGeckoId]
		if !exists {
			cge, exists = gov.dynamicTokensByCoinGeckoId[coinGeckoId]
		}

		if exists {
			// If a price is not set in CoinGecko, they return an empty entry. Treat that as a zero price.
			var price float64
			m, ok := data.(map[string]interface{})
			if !ok {
				gov.logger.Error("failed to parse CoinGecko response, reverting to configured price for this token", zap.String("coinGeckoId", coinGeckoId))
				// By continuing, we leave this one in the local map so the price will get reverted below.
				continue
			}
			if len(m) != 0 {
				var ok bool
				price_, ok := m["usd"]
				if !ok {
					gov.logger.Error("failed to parse CoinGecko response, reverting to configured price for this token", zap.String("coinGeckoId", coinGeckoId))
					// By continuing, we leave this one in the local map so the price will get reverted below.
					continue
				}

				price, ok = price_.(float64)
				if !ok {
					gov.logger.Error("failed to parse CoinGecko response, reverting to configured price for this token", zap.String("coinGeckoId", coinGeckoId))
					// By continuing, we leave this one in the local map so the price will get reverted below.
					continue
				}
			}

			for _, te := range cge {
				te.coinGeckoPrice = big.NewFloat(price)
				te.updatePrice()
				te.priceTime = now
			}

			delete(localTokenMap, coinGeckoId)
		} else {
			gov.logger.Error("received a CoinGecko response for an unexpected symbol", zap.String("coinGeckoId", coinGeckoId))
		}
	}

	if len(localTokenMap) != 0 {
		for _, lcge := range localTokenMap {
			for _, te := range lcge {
				gov.logger.Error("did not receive a CoinGecko response for symbol, reverting to configured price",
					zap.String("symbol", te.symbol),
					zap.String("coinGeckoId",
						te.coinGeckoId),
					zap.Stringer("cfgPrice", te.cfgPrice),
				)

				te.price = te.cfgPrice
				// Don't update the timestamp so we'll know when we last received an update from CoinGecko.
			}
		}

		return fmt.Errorf("failed to update prices for some tokens")
	}

	return nil
}

// queryCoinGeckoChunk sends a single CoinGecko query and returns the result.
func (gov *ChainGovernor) queryCoinGeckoChunk(query string) (map[string]interface{}, error) {
	var result map[string]interface{}

	gov.logger.Debug("executing CoinGecko query", zap.String("query", query))
	// #nosec G107 // the URL is hard-coded to the CoinGecko API. See [createCoinGeckoQuery].
	response, err := http.Get(query) //nolint:noctx // TODO: a context should be added here.
	if err != nil {
		return result, fmt.Errorf("failed to query CoinGecko: %w", err)
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			gov.logger.Error("failed to close CoinGecko query: %w", zap.Error(err))
		}
	}()

	responseData, err := common.SafeRead(response.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read CoinGecko response: %w", err)
	}

	resp := string(responseData)
	if strings.Contains(resp, "error_code") {
		return result, fmt.Errorf("CoinGecko query failed: %s", resp)
	}

	if err := json.Unmarshal(responseData, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal CoinGecko json: %w", err)
	}

	return result, nil
}

// revertAllPrices reverts the price of all tokens to the configured prices. It is used when a CoinGecko query fails.
func (gov *ChainGovernor) revertAllPrices() {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, cge := range gov.tokensByCoinGeckoId {
		for _, te := range cge {
			gov.logger.Info("reverting to configured price",
				zap.String("symbol", te.symbol),
				zap.String("coinGeckoId", te.coinGeckoId),
				zap.Stringer("cfgPrice", te.cfgPrice),
			)

			te.price = te.cfgPrice
			// Don't update the timestamp so we'll know when we last received an update from CoinGecko.
		}
	}
}

// updatePrice updates the price of a single token. We should use the max(coinGeckoPrice, configuredPrice) as our price for computing notional value.
func (te tokenEntry) updatePrice() {
	if (te.coinGeckoPrice == nil) || (te.coinGeckoPrice.Cmp(te.cfgPrice) < 0) {
		te.price.Set(te.cfgPrice)
	} else {
		te.price.Set(te.coinGeckoPrice)
	}
}

// buildCoinGeckoIdFromToken attempts to construct a CoinGecko ID from token metadata.
// For now, this is a simple mapping based on chain ID and token address.
// In a production system, this would likely query a token metadata service or use
// a more sophisticated mapping algorithm.
func buildCoinGeckoIdFromToken(chain vaa.ChainID, addr vaa.Address) (string, error) {
	// This is a simplified implementation. In production, you would:
	// 1. Query on-chain token metadata (symbol, name)
	// 2. Use a token metadata service or database
	// 3. Apply heuristics to map to CoinGecko IDs
	//
	// For now, we construct a placeholder that could be enhanced
	// based on the actual CoinGecko API's /coins/list endpoint

	// Return empty string to signal that we cannot construct a CoinGecko ID yet
	// This will be populated when we actually query CoinGecko
	return "", fmt.Errorf("cannot automatically determine CoinGecko ID for chain %d, addr %s", chain, addr)
}

// processPendingTokenDiscovery attempts to discover prices for tokens in the pending queue.
// It builds CoinGecko queries for pending tokens and attempts to fetch their prices.
// If a price can be retrieved, the token is added to the dynamic tokens map.
func (gov *ChainGovernor) processPendingTokenDiscovery(ctx context.Context) error {
	gov.mutex.Lock()

	// Get all pending tokens
	pendingTokens := make([]tokenKey, 0, len(gov.pendingTokenDiscovery))
	for tk := range gov.pendingTokenDiscovery {
		pendingTokens = append(pendingTokens, tk)
	}

	gov.mutex.Unlock()

	if len(pendingTokens) == 0 {
		return nil
	}

	gov.logger.Info("processing pending token discoveries", zap.Int("count", len(pendingTokens)))

	// Process each pending token
	for _, tk := range pendingTokens {
		// Attempt to build CoinGecko ID from token metadata
		// In a real implementation, this would query token metadata or use a lookup service
		coinGeckoId, err := buildCoinGeckoIdFromToken(tk.chain, tk.addr)
		if err != nil {
			gov.logger.Debug("cannot determine CoinGecko ID for token",
				zap.Stringer("chain", tk.chain),
				zap.Stringer("addr", tk.addr),
				zap.Error(err))

			// For now, we'll use a simple query by contract address
			// CoinGecko supports querying by contract address on some chains
			priceFound := gov.queryTokenByContractAddress(ctx, tk)
			if !priceFound {
				// Keep in pending queue for next attempt
				continue
			}
		} else {
			// We have a CoinGecko ID, query for the price
			priceFound := gov.queryTokenByCoinGeckoId(ctx, tk, coinGeckoId)
			if !priceFound {
				// Keep in pending queue for next attempt
				continue
			}
		}

		// Successfully discovered price, remove from pending queue
		gov.mutex.Lock()
		delete(gov.pendingTokenDiscovery, tk)
		gov.mutex.Unlock()
	}

	return nil
}

// queryTokenByContractAddress attempts to query CoinGecko for a token price using its contract address.
// Returns true if a price was successfully found and the token was added to dynamicTokens.
func (gov *ChainGovernor) queryTokenByContractAddress(ctx context.Context, tk tokenKey) bool {
	// Map Wormhole chain IDs to CoinGecko platform IDs
	platformId := mapChainToCoinGeckoPlatform(tk.chain)
	if platformId == "" {
		gov.logger.Debug("chain not supported for CoinGecko contract address lookup",
			zap.Stringer("chain", tk.chain))
		return false
	}

	// Build query URL
	// Example: https://api.coingecko.com/api/v3/coins/ethereum/contract/0x...
	contractAddr := tk.addr.String()
	var query string
	if gov.coinGeckoApiKey == "" {
		query = fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/contract/%s", platformId, contractAddr)
	} else {
		query = fmt.Sprintf("https://pro-api.coingecko.com/api/v3/coins/%s/contract/%s?x_cg_pro_api_key=%s", platformId, contractAddr, gov.coinGeckoApiKey)
	}

	gov.logger.Debug("querying CoinGecko by contract address", zap.String("query", query))

	// #nosec G107 // the URL is constructed from validated inputs
	response, err := http.Get(query) //nolint:noctx
	if err != nil {
		gov.logger.Debug("failed to query CoinGecko by contract address",
			zap.Stringer("chain", tk.chain),
			zap.Stringer("addr", tk.addr),
			zap.Error(err))
		return false
	}

	defer func() {
		_ = response.Body.Close()
	}()

	responseData, err := common.SafeRead(response.Body)
	if err != nil {
		gov.logger.Debug("failed to read CoinGecko response",
			zap.Stringer("chain", tk.chain),
			zap.Stringer("addr", tk.addr),
			zap.Error(err))
		return false
	}

	// Parse response using struct
	var result CoinGeckoContractResponse
	if err := json.Unmarshal(responseData, &result); err != nil {
		gov.logger.Debug("failed to unmarshal CoinGecko response",
			zap.Stringer("chain", tk.chain),
			zap.Stringer("addr", tk.addr),
			zap.Error(err))
		return false
	}

	// Validate required fields
	if result.ID == "" {
		gov.logger.Debug("no id in CoinGecko response",
			zap.Stringer("chain", tk.chain),
			zap.Stringer("addr", tk.addr))
		return false
	}

	// Extract USD price
	priceUsd, ok := result.MarketData.CurrentPrice["usd"]
	if !ok {
		gov.logger.Debug("no USD price in CoinGecko response",
			zap.Stringer("chain", tk.chain),
			zap.Stringer("addr", tk.addr))
		return false
	}

	// Use CoinGecko ID and symbol from response
	coinGeckoId := result.ID
	symbol := result.Symbol
	if symbol == "" {
		symbol = fmt.Sprintf("%d:%s", tk.chain, tk.addr.String())
	}

	// Extract decimals from detail_platforms if available
	decimals := int64(8) // Default to 8 if not found
	if platformData, ok := result.DetailPlatforms[platformId]; ok {
		if platformData.DecimalPlace > 0 {
			decimals = int64(platformData.DecimalPlace)
			gov.logger.Debug("extracted decimals from CoinGecko",
				zap.Stringer("chain", tk.chain),
				zap.Stringer("addr", tk.addr),
				zap.Int64("decimals", decimals))
		}
	}

	// Transfers have a maximum of eight decimal places (same logic as static tokens).
	if decimals > 8 {
		decimals = 8
	}

	decimalsFloat := big.NewFloat(math.Pow(10.0, float64(decimals)))
	decimalsBigInt, _ := decimalsFloat.Int(nil)

	cfgPrice := big.NewFloat(priceUsd)
	initialPrice := new(big.Float)
	initialPrice.Set(cfgPrice)

	te := &tokenEntry{
		cfgPrice:       cfgPrice,
		price:          initialPrice,
		decimals:       decimalsBigInt,
		symbol:         symbol,
		coinGeckoId:    coinGeckoId,
		token:          tk,
		coinGeckoPrice: big.NewFloat(priceUsd),
		priceTime:      time.Now(),
	}
	te.updatePrice()

	// Add to dynamic tokens
	gov.mutex.Lock()
	gov.dynamicTokens[tk] = te

	// Add to CoinGecko ID map for future updates
	cge, exists := gov.dynamicTokensByCoinGeckoId[coinGeckoId]
	if !exists {
		gov.dynamicTokensByCoinGeckoId[coinGeckoId] = []*tokenEntry{te}
	} else {
		cge = append(cge, te)
		gov.dynamicTokensByCoinGeckoId[coinGeckoId] = cge
	}
	gov.mutex.Unlock()

	gov.logger.Info("discovered new token dynamically",
		zap.Stringer("chain", tk.chain),
		zap.Stringer("addr", tk.addr),
		zap.String("symbol", symbol),
		zap.String("coinGeckoId", coinGeckoId),
		zap.Float64("price", priceUsd))

	return true
}

// queryTokenByCoinGeckoId queries CoinGecko for a token price using a known CoinGecko ID.
func (gov *ChainGovernor) queryTokenByCoinGeckoId(ctx context.Context, tk tokenKey, coinGeckoId string) bool {
	// Build query URL
	params := url.Values{}
	params.Add("ids", coinGeckoId)
	params.Add("vs_currencies", "usd")

	var query string
	if gov.coinGeckoApiKey == "" {
		query = "https://api.coingecko.com/api/v3/simple/price?" + params.Encode()
	} else {
		params.Add("x_cg_pro_api_key", gov.coinGeckoApiKey)
		query = "https://pro-api.coingecko.com/api/v3/simple/price?" + params.Encode()
	}

	result, err := gov.queryCoinGeckoChunk(query)
	if err != nil {
		return false
	}

	data, ok := result[coinGeckoId]
	if !ok {
		return false
	}

	m, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	priceFloat, ok := m["usd"].(float64)
	if !ok {
		return false
	}

	// Create token entry
	decimals := int64(8)
	decimalsFloat := big.NewFloat(math.Pow(10.0, float64(decimals)))
	decimalsBigInt, _ := decimalsFloat.Int(nil)

	cfgPrice := big.NewFloat(priceFloat)
	initialPrice := new(big.Float)
	initialPrice.Set(cfgPrice)

	te := &tokenEntry{
		cfgPrice:       cfgPrice,
		price:          initialPrice,
		decimals:       decimalsBigInt,
		symbol:         fmt.Sprintf("%d:%s", tk.chain, tk.addr.String()),
		coinGeckoId:    coinGeckoId,
		token:          tk,
		coinGeckoPrice: big.NewFloat(priceFloat),
		priceTime:      time.Now(),
	}
	te.updatePrice()

	gov.mutex.Lock()
	gov.dynamicTokens[tk] = te

	cge, exists := gov.dynamicTokensByCoinGeckoId[coinGeckoId]
	if !exists {
		gov.dynamicTokensByCoinGeckoId[coinGeckoId] = []*tokenEntry{te}
	} else {
		cge = append(cge, te)
		gov.dynamicTokensByCoinGeckoId[coinGeckoId] = cge
	}
	gov.mutex.Unlock()

	return true
}

// mapChainToCoinGeckoPlatform maps Wormhole chain IDs to CoinGecko platform identifiers.
// Uses ChainID.String() where possible, with exceptions for chains that need different naming.
func mapChainToCoinGeckoPlatform(chain vaa.ChainID) string {
	// Special cases where CoinGecko platform name differs from ChainID.String()
	switch chain {
	case vaa.ChainIDBSC:
		return "binance-smart-chain"
	case vaa.ChainIDPolygon:
		return "polygon-pos"
	case vaa.ChainIDArbitrum:
		return "arbitrum-one"
	case vaa.ChainIDOptimism:
		return "optimistic-ethereum"
	case vaa.ChainIDGnosis:
		return "xdai"
	case vaa.ChainIDNear:
		return "near-protocol"
	case vaa.ChainIDSonic:
		return "sonic"
	case vaa.ChainIDWorldchain:
		return "world-chain"
	case vaa.ChainIDXLayer:
		return "x-layer"
	case vaa.ChainIDHyperEVM:
		return "hyperevm"
	case vaa.ChainIDSeiEVM:
		return "sei-v2"
	}

	// For most chains, the ChainID.String() matches CoinGecko's platform identifier
	chainStr := chain.String()

	// List of chains we know are supported by CoinGecko using their direct string representation
	// This includes: ethereum, solana, avalanche, fantom, base, celo, moonbeam, scroll, linea,
	// unichain, ink, monad, aptos, sui, algorand, etc.
	switch chain {
	case vaa.ChainIDEthereum, vaa.ChainIDSolana, vaa.ChainIDAvalanche, vaa.ChainIDFantom,
		vaa.ChainIDBase, vaa.ChainIDCelo, vaa.ChainIDMoonbeam, vaa.ChainIDScroll,
		vaa.ChainIDLinea, vaa.ChainIDUnichain, vaa.ChainIDInk, vaa.ChainIDMonad,
		vaa.ChainIDAptos, vaa.ChainIDSui, vaa.ChainIDAlgorand, vaa.ChainIDBerachain,
		vaa.ChainIDBOB, vaa.ChainIDMantle, vaa.ChainIDRootstock, vaa.ChainIDFileCoin,
		vaa.ChainIDKlaytn:
		return chainStr
	}

	// Return empty string for unsupported chains
	return ""
}

// CheckQuery is a free function used to test that the CoinGecko query still works after the mainnet token list has been updated.
func CheckQuery(logger *zap.Logger) error {
	logger.Info("Instantiating governor.")
	ctx := context.Background()
	var db guardianDB.MockGovernorDB
	gov := NewChainGovernor(logger, &db, common.MainNet, true, "")

	if err := gov.initConfig(); err != nil {
		return err
	}

	logger.Info("Building CoinGecko query.")
	if err := gov.initCoinGecko(ctx, false); err != nil {
		return err
	}

	logger.Info("Initiating CoinGecko query.")
	if err := gov.queryCoinGecko(ctx); err != nil {
		return err
	}

	logger.Info("CoinGecko query complete.")
	return nil
}

// CheckContractQuery is a free function used to test the CoinGecko contract address lookup endpoint
// and validate the struct-based JSON parsing for dynamic token discovery.
// This function queries real tokens on different chains to ensure the API integration works correctly.
func CheckContractQuery(logger *zap.Logger, coinGeckoApiKey string) error {
	logger.Info("Testing CoinGecko contract address lookup endpoint")

	// Test cases: well-known tokens on different chains
	testCases := []struct {
		name     string
		chain    vaa.ChainID
		address  string
		expected struct {
			symbol   string
			decimals int
		}
	}{
		{
			name:    "USDC on Ethereum",
			chain:   vaa.ChainIDEthereum,
			address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			expected: struct {
				symbol   string
				decimals int
			}{symbol: "usdc", decimals: 6},
		},
		{
			name:    "USDC on Solana",
			chain:   vaa.ChainIDSolana,
			address: "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
			expected: struct {
				symbol   string
				decimals int
			}{symbol: "usdc", decimals: 6},
		},
		{
			name:    "WETH on Ethereum",
			chain:   vaa.ChainIDEthereum,
			address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			expected: struct {
				symbol   string
				decimals int
			}{symbol: "weth", decimals: 18},
		},
		{
			name:    "USDC on Polygon",
			chain:   vaa.ChainIDPolygon,
			address: "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359",
			expected: struct {
				symbol   string
				decimals int
			}{symbol: "usdc", decimals: 6},
		},
	}

	for _, tc := range testCases {
		logger.Info("Testing token",
			zap.String("name", tc.name),
			zap.Stringer("chain", tc.chain),
			zap.String("address", tc.address))

		// Map chain to CoinGecko platform
		platformId := mapChainToCoinGeckoPlatform(tc.chain)
		if platformId == "" {
			return fmt.Errorf("chain %s not supported for CoinGecko lookup", tc.chain)
		}
		logger.Info("Mapped chain to platform", zap.String("platform", platformId))

		// Build query URL
		var query string
		if coinGeckoApiKey == "" {
			query = fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/contract/%s", platformId, tc.address)
		} else {
			query = fmt.Sprintf("https://pro-api.coingecko.com/api/v3/coins/%s/contract/%s?x_cg_pro_api_key=%s",
				platformId, tc.address, coinGeckoApiKey)
		}
		logger.Info("Querying CoinGecko", zap.String("url", query))

		// Query CoinGecko
		// #nosec G107 // the URL is constructed from validated inputs
		response, err := http.Get(query) //nolint:noctx
		if err != nil {
			return fmt.Errorf("failed to query CoinGecko for %s: %w", tc.name, err)
		}

		responseData, err := common.SafeRead(response.Body)
		_ = response.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response for %s: %w", tc.name, err)
		}

		// Parse response using struct
		var result CoinGeckoContractResponse
		if err := json.Unmarshal(responseData, &result); err != nil {
			return fmt.Errorf("failed to unmarshal response for %s: %w", tc.name, err)
		}

		// Validate response
		if result.ID == "" {
			return fmt.Errorf("missing CoinGecko ID for %s", tc.name)
		}
		if result.Symbol == "" {
			return fmt.Errorf("missing symbol for %s", tc.name)
		}

		// Check price exists
		priceUsd, ok := result.MarketData.CurrentPrice["usd"]
		if !ok {
			return fmt.Errorf("missing USD price for %s", tc.name)
		}
		if priceUsd <= 0 {
			return fmt.Errorf("invalid USD price for %s: %f", tc.name, priceUsd)
		}

		// Check decimals
		platformData, ok := result.DetailPlatforms[platformId]
		if !ok {
			return fmt.Errorf("missing platform data for %s on %s", tc.name, platformId)
		}
		if platformData.DecimalPlace == 0 {
			return fmt.Errorf("missing decimal place for %s", tc.name)
		}

		// Log results
		logger.Info("Successfully retrieved token data",
			zap.String("name", tc.name),
			zap.String("id", result.ID),
			zap.String("symbol", result.Symbol),
			zap.Float64("price", priceUsd),
			zap.Int("decimals", platformData.DecimalPlace),
			zap.String("contract", platformData.ContractAddress))

		// Validate against expected values
		if result.Symbol != tc.expected.symbol {
			logger.Warn("Symbol mismatch",
				zap.String("expected", tc.expected.symbol),
				zap.String("got", result.Symbol))
		}
		if platformData.DecimalPlace != tc.expected.decimals {
			logger.Warn("Decimals mismatch",
				zap.Int("expected", tc.expected.decimals),
				zap.Int("got", platformData.DecimalPlace))
		}

		logger.Info("✓ Test passed for " + tc.name)

		// Be nice to CoinGecko API - rate limit ourselves
		time.Sleep(2 * time.Second)
	}

	logger.Info("✓ All contract address lookups succeeded")
	return nil
}
