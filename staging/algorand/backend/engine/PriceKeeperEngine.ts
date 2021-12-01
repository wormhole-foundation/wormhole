/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { IEngine } from './IEngine'
import { PythPriceFetcher } from '../fetcher/PythPriceFetcher'
import { StdAlgoPublisher } from '../publisher/stdAlgoPublisher'
import { StrategyLastPrice } from '../strategy/strategyLastPrice'
import { IAppSettings } from '../common/settings'
import { IPriceFetcher } from '../fetcher/IPriceFetcher'
import { IPublisher, PublishInfo } from '../publisher/IPublisher'
import { PriceTicker } from '../common/priceTicker'
import { StatusCode } from '../common/statusCodes'
import { sleep } from '../common/sleep'
const algosdk = require('algosdk')
const charm = require('charm')()

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

export class PriceKeeperEngine implements IEngine {
  private settings: IAppSettings

  constructor (settings: IAppSettings) {
    this.settings = settings
  }

  async start () {
    charm.write('Setting up for')
      .foreground('yellow')
      .write(` ${this.settings.params.symbol} `)
      .foreground('white')
      .write('for PriceKeeper App')
      .foreground('yellow')
      .write(` ${this.settings.params.priceKeeperAppId} `)
      .foreground('white')
      .write(`interval ${this.settings.params.publishIntervalSecs} secs\n`)

    const publisher = new StdAlgoPublisher(this.settings.params.symbol,
      this.settings.params.priceKeeperAppId,
      this.settings.params.validator,
      (algosdk.mnemonicToSecretKey(this.settings.params.mnemo)).sk,
      this.settings.algo.token,
      this.settings.algo.api,
      this.settings.algo.port
    )

    const fetcher = new PythPriceFetcher(this.settings.params.symbol,
      new StrategyLastPrice(this.settings.params.bufferSize), this.settings.pyth?.solanaClusterName!)
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
      await sleep(this.settings.params.publishIntervalSecs * 1000)
    }
  }
}
