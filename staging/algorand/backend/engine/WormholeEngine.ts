/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { IEngine } from './IEngine'
import { StrategyLastPrice } from '../strategy/strategyLastPrice'
import { IAppSettings } from '../common/settings'
import { IPriceFetcher } from '../fetcher/IPriceFetcher'
import { IPublisher, PublishInfo } from '../publisher/IPublisher'
import { PriceTicker } from '../common/priceTicker'
import { StatusCode } from '../common/statusCodes'
import { sleep } from '../common/sleep'
import { NullPublisher } from '../publisher/NullPublisher'
import { WormholePythPriceFetcher } from '../fetcher/WormholePythPriceFetcher'

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

export class WormholeClientEngine implements IEngine {
  private settings: IAppSettings

  constructor (settings: IAppSettings) {
    this.settings = settings
  }

  async start () {
    charm.write('Supported symbols:')
    this.settings.symbols.forEach(sym => {
      charm.foreground('yellow')
        .write(` ${sym.name} `)
        .foreground('white')
        .write(' AppId: ')
        .foreground('yellow')
        .write(` ${sym.priceKeeperV2AppId} `)
        .foreground('white')
        .write(`interval ${sym.publishIntervalSecs} secs\n`)
    })

    const publisher = new NullPublisher()
    const fetcher = new WormholePythPriceFetcher(this.settings.wormhole.spyServiceHost,
      this.settings.pyth.chainId,
      this.settings.pyth.emitterAddress,
      this.settings.symbols,
      new StrategyLastPrice(this.settings.strategy.bufferSize))

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
          console.log(`[PUB ${pubCount++}] ${wrs.tick!.price}±${wrs.tick!.confidence} exp:${wrs.tick!.exponent}  t:${wrs.tick!.timestamp} TXID:${wrs.pub!.txid}`)
          break
        }
        case StatusCode.NO_TICKER:
          console.log('No ticker available from fetcher data source')
          break
        default:
          console.log('Error. Reason: ' + wrs.reason)
      }
      //await sleep(this.settings.params.publishIntervalSecs * 1000)
      await sleep(1000)
    }
  }
}
