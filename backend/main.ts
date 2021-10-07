/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PythPriceFetcher } from './PythPriceFetcher'
import { StdAlgoPublisher } from './publisher/StdAlgoPublisher'
import { StrategyLastPrice } from './strategy/strategyLastPrice'
const settings = require('../settings')
const algosdk = require('algosdk')

console.log('Pricecaster Service Fetcher  -- (c) 2021 Randlabs.io\n')

const fetchers: { [key: string]: PythPriceFetcher } = {}
const publishers: { [key: string]: StdAlgoPublisher } = {}

for (const sym in settings.symbols) {
  console.log(`Setting up fetcher/publisher for ${sym}`)
  publishers[sym] = new StdAlgoPublisher(sym,
    settings.symbols[sym].priceKeeperAppId,
    settings.symbols[sym].validator,
    algosdk.mnemonicToSecretKey(settings.symbols[sym].mnemo)
  )
  fetchers[sym] = new PythPriceFetcher(sym, new StrategyLastPrice())
}

// const pricefetcher = new PythPriceFetcher('BTC/USD', new StrategyLastPrice(10))
// const publisher = new StdAlgoPublisher('BTC/USD', 38888888, )

// async function processTick() {
//   const tick = pricefetcher.queryTicker()
//   const publishInfo = await publisher.publish(tick)
//   setTimeout(processTick, 1000)
// })
