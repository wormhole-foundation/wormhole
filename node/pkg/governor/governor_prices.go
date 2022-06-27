// This file contains the code to query for and update token prices for the chain governor.
//
// The initial prices are read from the static config (tokens.go). After that, prices are
// queried from CoinGecko. The chain governor then uses the maximum of the static price and
// the latest CoinGecko price.

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
)

// An example of the query to be generated: https://api.coingecko.com/api/v3/simple/price?ids=gemma-extending-tech,bitcoin,weth&vs_currencies=usd

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
		if gov.logger != nil {
			gov.logger.Info("cgov: did not find any securities, nothing to do!")
		}

		return nil
	}

	gov.coinGeckoQuery = "https://api.coingecko.com/api/v3/simple/price?" + params.Encode()

	if gov.logger != nil {
		gov.logger.Info("cgov: coingecko query: ", zap.String("query", gov.coinGeckoQuery))
	}

	timer := time.NewTimer(time.Millisecond) // Start immediately.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				gov.queryCoinGecko()
				timer = time.NewTimer(time.Duration(5) * time.Minute)
			}
		}
	}()

	return nil
}

// This does not return an error. Instead, it just logs the error and we will try again five minutes later.
func (gov *ChainGovernor) queryCoinGecko() {
	response, err := http.Get(gov.coinGeckoQuery)
	if err != nil {
		gov.logger.Error("cgov: failed to query coin gecko", zap.String("query", gov.coinGeckoQuery), zap.Error(err))
		return
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			gov.logger.Error("cgov: failed to close coin gecko query")
		}
	}()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		gov.logger.Error("cgov: failed to parse coin gecko response", zap.Error(err))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseData, &result); err != nil {
		gov.logger.Error("cgov: failed to unmarshal coin gecko json", zap.Error(err))
		return
	}

	now := time.Now()
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	for coinGeckoId, data := range result {
		te, exists := gov.tokensByCoinGeckoId[coinGeckoId]
		if exists {
			price, ok := data.(map[string]interface{})["usd"].(float64)
			if !ok {
				gov.logger.Error("cgov: failed to parse coin gecko response", zap.String("coinGeckoId", coinGeckoId))
				continue
			}
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
