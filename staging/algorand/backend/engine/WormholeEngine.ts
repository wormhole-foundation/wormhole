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
import { WormholePythPriceFetcher } from '../fetcher/WormholePythPriceFetcher'
import { Symbol } from 'backend/common/basetypes'
import { Pricekeeper2Publisher } from '../publisher/Pricekeeper2Publisher'
import * as Logger from '@randlabs/js-logger'
import { sleep } from '../common/sleep'
const fs = require('fs')
const algosdk = require('algosdk')

type WorkerRoutineStatus = {
  status: StatusCode,
  reason?: string,
  tick?: PriceTicker,
  pub?: PublishInfo
}

async function workerRoutine (sym: Symbol, fetcher: IPriceFetcher, publisher: IPublisher): Promise<WorkerRoutineStatus> {
  const tick = fetcher.queryData(sym.productId + sym.priceId)
  if (tick === undefined) {
    return { status: StatusCode.NO_TICKER }
  }
  const pub = await publisher.publish(tick)
  return { status: pub.status, reason: pub.reason, tick, pub }
}

export class WormholeClientEngine implements IEngine {
  private settings: IAppSettings
  private shouldQuit: boolean
  constructor (settings: IAppSettings) {
    this.settings = settings
    this.shouldQuit = false
  }

  async start () {
    process.on('SIGINT', () => {
      console.log('Received SIGINT')
      Logger.finalize()
      this.shouldQuit = true
    })

    let mnemo, verifyProgramBinary
    try {
      mnemo = fs.readFileSync(this.settings.apps.ownerKeyFile)
      verifyProgramBinary = Uint8Array.from(fs.readFileSync(this.settings.apps.vaaVerifyProgramBinFile))
    } catch (e) {
      throw new Error('Cannot read account and/or verify program source: ' + e)
    }

    const publisher = new Pricekeeper2Publisher(this.settings.apps.vaaProcessorAppId,
      this.settings.apps.priceKeeperV2AppId,
      this.settings.apps.ownerAddress,
      verifyProgramBinary,
      this.settings.apps.vaaVerifyProgramHash,
      algosdk.mnemonicToSecretKey(mnemo.toString()),
      this.settings.algo.token,
      this.settings.algo.api,
      this.settings.algo.port,
      this.settings.algo.dumpFailedTx,
      this.settings.algo.dumpFailedTxDirectory
    )
    const fetcher = new WormholePythPriceFetcher(this.settings.wormhole.spyServiceHost,
      this.settings.pyth.chainId,
      this.settings.pyth.emitterAddress,
      this.settings.symbols,
      new StrategyLastPrice(this.settings.strategy.bufferSize))

    Logger.info('Waiting for fetcher to boot...')
    await fetcher.start()

    Logger.info('Waiting for publisher to boot...')
    await publisher.start()

    for (const sym of this.settings.symbols) {
      sym.pubCount = 0
      Logger.info(`Starting worker for symbol ${sym.name}, interval ${sym.publishIntervalSecs}s`)
      setInterval(this.callWorkerRoutine, sym.publishIntervalSecs * 1000, sym, fetcher, publisher)
    }

    while (!this.shouldQuit) {
      await sleep(1000)
    }
  }

  async callWorkerRoutine (sym: Symbol, fetcher: IPriceFetcher, publisher: IPublisher) {
    const wrs = await workerRoutine(sym, fetcher, publisher)
    switch (wrs.status) {
      case StatusCode.OK: {
        Logger.info(`${sym.name} [#${sym.pubCount++}] price: ${wrs.tick!.price} ± ${wrs.tick!.confidence}    exp: ${wrs.tick!.exponent}    t: ${wrs.tick!.timestamp}    TxID: ${wrs.pub!.txid}`)
        break
      }
      case StatusCode.NO_TICKER:
        Logger.warn(`${sym.name}: No ticker available from fetcher data source`)
        break
      default:
        Logger.error(`${sym.name}: Error. Reason: ` + wrs.reason)
    }
  }
}
