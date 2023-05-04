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
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/supervisor"
)

// The CoinGecko API is documented here: https://www.coingecko.com/en/api/documentation
// An example of the query to be generated: https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd

// coinGeckoQueryIntervalInMins specifies how often we query CoinGecko for prices.
const coinGeckoQueryIntervalInMins = 15

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
	gov.coinGeckoQueries = createCoinGeckoQueries(ids, tokensPerCoinGeckoQuery)
	for queryIdx, query := range gov.coinGeckoQueries {
		gov.logger.Info("coingecko query: ", zap.Int("queryIdx", queryIdx), zap.String("query", query))
	}

	if len(gov.coinGeckoQueries) == 0 {
		gov.logger.Info("did not find any tokens, nothing to do!")
		return nil
	}

	if run {
		if err := supervisor.Run(ctx, "govpricer", gov.PriceQuery); err != nil {
			return err
		}
	}

	return nil
}

// createCoinGeckoQueries creates the set of CoinGecko queries, breaking the set of IDs into the appropriate size chunks.
func createCoinGeckoQueries(idList []string, tokensPerQuery int) []string {
	var queries []string
	queryIdx := 0
	tokenIdx := 0
	ids := ""
	first := true
	for _, coinGeckoId := range idList {
		if tokenIdx%tokensPerQuery == 0 && tokenIdx != 0 {
			queries = append(queries, createCoinGeckoQuery(ids))
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
		queries = append(queries, createCoinGeckoQuery(ids))
	}

	return queries
}

// createCoinGeckoQuery creates a CoinGecko query for the specified set of IDs.
func createCoinGeckoQuery(ids string) string {
	params := url.Values{}
	params.Add("ids", ids)
	params.Add("vs_currencies", "usd")

	query := "https://api.coingecko.com/api/v3/simple/price?" + params.Encode()
	return query
}

// PriceQuery is the entry point for the routine that periodically queries CoinGecko for prices.
func (gov *ChainGovernor) PriceQuery(ctx context.Context) error {
	// Do a query immediately, then once each interval.
	// We ignore the error because an error would already have been logged, and we don't want to bring down the
	// guardian due to a CoinGecko error. The prices would already have been reverted to the config values.
	_ = gov.queryCoinGecko()

	ticker := time.NewTicker(time.Duration(coinGeckoQueryIntervalInMins) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_ = gov.queryCoinGecko()
		}
	}
}

// queryCoinGecko sends a series of of one or more queries to the CoinGecko server to get the latest prices. It can
// return an error, but that is only used by the tool that validates the query. In the actual governor,
// it just logs the error and we will try again next interval. If an error happens, any tokens that have
// not been updated will be assigned their pre-configured price.
func (gov *ChainGovernor) queryCoinGecko() error {
	result := make(map[string]interface{})
	for queryIdx, query := range gov.coinGeckoQueries {
		thisResult, err := gov.queryCoinGeckoChunk(query)
		if err != nil {
			gov.logger.Error("CoinGecko query failed", zap.Int("queryIdx", queryIdx), zap.String("query", query), zap.Error(err))
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

	for coinGeckoId, data := range result {
		cge, exists := gov.tokensByCoinGeckoId[coinGeckoId]
		if exists {
			// If a price is not set in CoinGecko, they return an empty entry. Treat that as a zero price.
			var price float64

			m := data.(map[string]interface{})
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
	response, err := http.Get(query) //nolint:gosec,noctx
	if err != nil {
		return result, fmt.Errorf("failed to query CoinGecko: %w", err)
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			gov.logger.Error("failed to close CoinGecko query: %w", zap.Error(err))
		}
	}()

	responseData, err := io.ReadAll(response.Body)
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

// CheckQuery is a free function used to test that the CoinGecko query still works after the mainnet token list has been updated.
func CheckQuery(logger *zap.Logger) error {
	logger.Info("Instantiating governor.")
	ctx := context.Background()
	var db db.MockGovernorDB
	gov := NewChainGovernor(logger, &db, MainNetMode)

	if err := gov.initConfig(); err != nil {
		return err
	}

	logger.Info("Building CoinGecko query.")
	if err := gov.initCoinGecko(ctx, false); err != nil {
		return err
	}

	logger.Info("Initiating CoinGecko query.")
	if err := gov.queryCoinGecko(); err != nil {
		return err
	}

	logger.Info("CoinGecko query complete.")
	return nil
}
