// This file contains the code to query for and update token prices for the chain governor.
//
// The initial prices are read from the static config (tokens.go). After that, prices are
// queried from CoinGecko. The chain governor then uses the maximum of the static price and
// the latest CoinGecko price. The CoinGecko poll interval is specified by coinGeckoQueryIntervalInMins.

package governor

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/supervisor"
)

// The CoinGecko API is documented here: https://www.coingecko.com/en/api/documentation
// An example of the query to be generated: https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd

const coinGeckoQueryIntervalInMins = 5

func (gov *ChainGovernor) initCoinGecko(ctx context.Context) error {
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

	if err := supervisor.Run(ctx, "govpricer", gov.PriceQuery); err != nil {
		return err
	}

	return nil
}

func (gov *ChainGovernor) PriceQuery(ctx context.Context) error {
	// Do a query immediately, then once each interval.
	gov.queryCoinGecko()

	ticker := time.NewTicker(time.Duration(coinGeckoQueryIntervalInMins) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			gov.queryCoinGecko()
		}
	}
}

// This does not return an error. Instead, it just logs the error and we will try again five minutes later.
func (gov *ChainGovernor) queryCoinGecko() {
	response, err := http.Get(gov.coinGeckoQuery)
	if err != nil {
		gov.logger.Error("cgov: failed to query coin gecko, reverting to configured prices", zap.String("query", gov.coinGeckoQuery), zap.Error(err))
		gov.revertAllPrices()
		return
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
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseData, &result); err != nil {
		gov.logger.Error("cgov: failed to unmarshal coin gecko json, reverting to configured prices", zap.Error(err))
		gov.revertAllPrices()
		return
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
			price, ok := data.(map[string]interface{})["usd"].(float64)
			if !ok {
				gov.logger.Error("cgov: failed to parse coin gecko response, reverting to configured price for this token", zap.String("coinGeckoId", coinGeckoId))
				// By continuing, we leave this one in the local map so the price will get reverted below.
				continue
			}

			for _, te := range cge {
				te.coinGeckoPrice = big.NewFloat(price)
				te.updatePrice()
				te.priceTime = now

				gov.logger.Info("cgov: updated price",
					zap.String("symbol", te.symbol),
					zap.String("coinGeckoId",
						te.coinGeckoId),
					zap.Stringer("price", te.price),
					zap.Stringer("cfgPrice", te.cfgPrice),
					zap.Stringer("coinGeckoPrice", te.coinGeckoPrice),
				)
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
}

func (gov *ChainGovernor) revertAllPrices() {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for _, cge := range gov.tokensByCoinGeckoId {
		for _, te := range cge {
			gov.logger.Error("cgov: reverting to configured price",
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
