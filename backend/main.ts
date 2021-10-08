/* eslint-disable no-unused-vars */
/**
 * Pricecaster Service.
 *
 * Main program file.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PythPriceFetcher } from './PythPriceFetcher'
import { StdAlgoPublisher } from './publisher/StdAlgoPublisher'
import { StrategyLastPrice } from './strategy/strategyLastPrice'
import { IPriceFetcher } from './IPriceFetcher'
import { IPublisher, PublishInfo } from './IPublisher'
import { PriceTicker } from './PriceTicker'
import { StatusCode } from './statusCodes'
import Status from 'algosdk/dist/types/src/client/v2/algod/status'
const settings = require('../settings')
const algosdk = require('algosdk')
const charm = require('charm')()

export function sleep (ms: number) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

type WorkerRoutineStatus = {
  status: StatusCode,
  reason?: string,
  tick?: PriceTicker,
  pub?: PublishInfo
}

async function workerRoutine (fetcher: IPriceFetcher, publisher: IPublisher): Promise<WorkerRoutineStatus> {
  const tick = fetcher.queryTicker()
  if (tick === undefined) {
    return { status: StatusCode.NO_TICKER }
  }
  const pub = await publisher.publish(tick)
  return { status: pub.status, reason: pub.reason, tick, pub }
}

(async () => {
  charm.pipe(process.stdout)
  charm.reset()
  console.log('Pricecaster Service Fetcher  -- (c) 2021 Randlabs.io\n')
  const params = settings.params
  console.log(`Setting up fetcher/publisher for ${params.symbol} for PriceKeeper App ${params.priceKeeperAppId}, interval ${params.publishIntervalSecs} secs`)

  const publisher = new StdAlgoPublisher(params.symbol,
    params.priceKeeperAppId,
    params.validator,
    (algosdk.mnemonicToSecretKey(params.mnemo)).sk
  )
  const fetcher = new PythPriceFetcher(params.symbol, new StrategyLastPrice(params.bufferSize))
  await fetcher.start()

  console.log('Waiting for fetcher to boot...')
  while (!fetcher.hasData()) {
    await sleep(250)
  }
  console.log('Waiting for publisher to boot...')
  await publisher.start()
  console.log('Starting worker.')

  let active = true
  charm.on('^C', () => {
    console.log('CTRL+C: Aborted by user.')
    active = false
  })
  // eslint-disable-next-line no-unmodified-loop-condition
  let pubCount = 0
  while (active) {
    const wrs = await workerRoutine(fetcher, publisher)
    switch (wrs.status) {
      case StatusCode.OK:
        console.log(`[PUB ${pubCount++}] ${wrs.tick!.price}±${wrs.tick!.confidence} t:${wrs.tick!.networkTime} TXID:${wrs.pub!.txid})`)
        break
      case StatusCode.NO_TICKER:
        console.log('No ticker available from fetcher data source')
        break
      default:
        console.log('Error. Reason: ' + wrs.reason)
    }
    await sleep(params.publishIntervalSecs * 1000)
  }
})()
