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
	"io/ioutil"
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

const coinGeckoQueryIntervalInMins = 15

func (gov *ChainGovernor) initCoinGecko(ctx context.Context, run bool) error {
	ids := ""
	first := true
	for coinGeckoId := range gov.tokensByCoinGeckoId {
		if first {
			first = false
		} else {
			ids += ","
		}

		ids += coinGeckoId
	}

	params := url.Values{}
	params.Add("ids", ids)
	params.Add("vs_currencies", "usd")

	if first {
		gov.logger.Info("cgov: did not find any tokens, nothing to do!")
		return nil
	}

	gov.coinGeckoQuery = "https://api.coingecko.com/api/v3/simple/price?" + params.Encode()
	gov.logger.Info("cgov: coingecko query: ", zap.String("query", gov.coinGeckoQuery))

	if run {
		if err := supervisor.Run(ctx, "govpricer", gov.PriceQuery); err != nil {
			return err
		}
	}

	return nil
}

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

// queryCoinGecko sends a query to the CoinGecko server to get the latest prices. It can
// return an error, but that is only used by the tool that validates the query. In the actual governor,
// it just logs the error and we will try again next interval. If an error happens, any tokens that have
// not been updated will be assigned their pre-configured price.
func (gov *ChainGovernor) queryCoinGecko() error {
	response, err := http.Get(gov.coinGeckoQuery)
	if err != nil {
		gov.logger.Error("cgov: failed to query coin gecko, reverting to configured prices", zap.String("query", gov.coinGeckoQuery), zap.Error(err))
		gov.revertAllPrices()
		return fmt.Errorf("failed to query CoinGecko")
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			gov.logger.Error("cgov: failed to close coin gecko query")
			// We can't safely call revertAllPrices() here because we don't know if we hold the lock or not.
			// Also, we don't need to because the prices have already been updated / reverted by this point.
		}
	}()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		gov.logger.Error("cgov: failed to parse coin gecko response, reverting to configured prices", zap.Error(err))
		gov.revertAllPrices()
		return fmt.Errorf("failed to parse CoinGecko response")
	}

	resp := string(responseData)
	if strings.Contains(resp, "error_code") {
		gov.logger.Error("cgov: coin gecko query failed, reverting to configured prices",
			zap.String("response", resp),
			zap.String("query", gov.coinGeckoQuery),
		)

		gov.revertAllPrices()
		return fmt.Errorf("coin gecko query failed: %s", resp)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseData, &result); err != nil {
		gov.logger.Error("cgov: failed to unmarshal coin gecko json, reverting to configured prices", zap.Error(err))
		gov.revertAllPrices()
		return fmt.Errorf("failed to unmarshal json")
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
			price := float64(0)
			m := data.(map[string]interface{})
			if len(m) != 0 {
				var ok bool
				price, ok = m["usd"].(float64)
				if !ok {
					gov.logger.Error("cgov: failed to parse coin gecko response, reverting to configured price for this token", zap.String("coinGeckoId", coinGeckoId))
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
			gov.logger.Error("cgov: received a CoinGecko response for an unexpected symbol", zap.String("coinGeckoId", coinGeckoId))
		}
	}

	if len(localTokenMap) != 0 {
		for _, lcge := range localTokenMap {
			for _, te := range lcge {
				gov.logger.Error("cgov: did not receive a CoinGecko response for symbol, reverting to configured price",
					zap.String("symbol", te.symbol),
					zap.String("coinGeckoId",
						te.coinGeckoId),
					zap.Stringer("cfgPrice", te.cfgPrice),
				)

				te.price = te.cfgPrice
				// Don't update the timestamp so we'll know when we last received an update from CoinGecko.
			}
		}
	}

	return nil
}

func (gov *ChainGovernor) revertAllPrices() {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, cge := range gov.tokensByCoinGeckoId {
		for _, te := range cge {
			gov.logger.Info("cgov: reverting to configured price",
				zap.String("symbol", te.symbol),
				zap.String("coinGeckoId",
					te.coinGeckoId),
				zap.Stringer("cfgPrice", te.cfgPrice),
			)

			te.price = te.cfgPrice
			// Don't update the timestamp so we'll know when we last received an update from CoinGecko.
		}
	}
}

// We should use the max(coinGeckoPrice, configuredPrice) as our price for computing notional value.
func (te tokenEntry) updatePrice() {
	if (te.coinGeckoPrice == nil) || (te.coinGeckoPrice.Cmp(te.cfgPrice) < 0) {
		te.price.Set(te.cfgPrice)
	} else {
		te.price.Set(te.coinGeckoPrice)
	}
}

func CheckQuery(logger *zap.Logger) error {
	logger.Info("Instantiating governor.")
	ctx := context.Background()
	var db db.MockGovernorDB
	gov := NewChainGovernor(logger, &db, MainNetMode)

	if err := gov.initConfig(); err != nil {
		return err
	}

	logger.Info("Building Coin Gecko query.")
	if err := gov.initCoinGecko(ctx, false); err != nil {
		return err
	}

	logger.Info("Initiating Coin Gecko query.")
	if err := gov.queryCoinGecko(); err != nil {
		return err
	}

	logger.Info("Coin Gecko query complete.")
	return nil
}
