/* eslint-disable no-unused-vars */
/**
 * Pricecaster Service.
 *
 * Main program file.
 *
 * (c) 2021 Randlabs, Inc.
 */

import * as Config from '@randlabs/js-config-reader'
import { PythPriceFetcher } from './fetcher/PythPriceFetcher'
import { StdAlgoPublisher } from './publisher/stdAlgoPublisher'
import { StrategyLastPrice } from './strategy/strategyLastPrice'
import { IPriceFetcher } from './fetcher/IPriceFetcher'
import { IPublisher, PublishInfo } from './publisher/IPublisher'
import { PriceTicker } from './common/priceTicker'
import { StatusCode } from './common/statusCodes'
import { IAppSettings } from './common/settings'
import { exit } from 'process'
const algosdk = require('algosdk')
const charm = require('charm')()

function sleep (ms: number) {
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
  charm.foreground('cyan').display('bright')
  console.log('Pricecaster Service Fetcher  -- (c) 2021 Randlabs.io\n')
  charm.foreground('white')

  let settings: IAppSettings
  try {
    await Config.initialize<IAppSettings>({ envVar: 'PRICECASTER_SETTINGS' })
    settings = Config.get<IAppSettings>()
  } catch (e: any) {
    console.error('Cannot initialize configuration: ' + e.toString())
    exit(1)
  }

  charm.write('Setting up for')
    .foreground('yellow')
    .write(` ${settings.params.symbol} `)
    .foreground('white')
    .write('for PriceKeeper App')
    .foreground('yellow')
    .write(` ${settings.params.priceKeeperAppId} `)
    .foreground('white')
    .write(`interval ${settings.params.publishIntervalSecs} secs\n`)

  const publisher = new StdAlgoPublisher(settings.params.symbol,
    settings.params.priceKeeperAppId,
    settings.params.validator,
    (algosdk.mnemonicToSecretKey(settings.params.mnemo)).sk,
    settings.algo.token,
    settings.algo.api,
    settings.algo.port
  )
  const fetcher = new PythPriceFetcher(settings.params.symbol, new StrategyLastPrice(settings.params.bufferSize), settings.pyth?.solanaClusterName!)
  await fetcher.start()

  console.log('Waiting for fetcher to boot...')
  while (!fetcher.hasData()) {
    await sleep(250)
  }
  console.log('Waiting for publisher to boot...')
  await publisher.start()
  console.log('Starting worker.')

  let active = true
  charm.removeAllListeners('^C')
  charm.on('^C', () => {
    console.log('CTRL+C: Aborted by user.')
    active = false
  })
  let pubCount = 0
  // eslint-disable-next-line no-unmodified-loop-condition
  while (active) {
    const wrs = await workerRoutine(fetcher, publisher)
    switch (wrs.status) {
      case StatusCode.OK: {
        console.log(`[PUB ${pubCount++}] ${wrs.tick!.price}±${wrs.tick!.confidence} exp:${wrs.tick!.exponent}  t:${wrs.tick!.networkTime} TXID:${wrs.pub!.txid}`)
        break
      }
      case StatusCode.NO_TICKER:
        console.log('No ticker available from fetcher data source')
        break
      default:
        console.log('Error. Reason: ' + wrs.reason)
    }
    await sleep(settings.params.publishIntervalSecs * 1000)
  }
})()
