/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * Copyright 2022 Wormhole Project Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { IEngine } from './IEngine'
import { IAppSettings } from '../common/settings'
import { IPriceFetcher } from '../fetcher/IPriceFetcher'
import { IPublisher, PublishInfo } from '../publisher/IPublisher'
import { StatusCode } from '../common/statusCodes'
import { WormholePythPriceFetcher } from '../fetcher/WormholePythPriceFetcher'
import { Pricekeeper2Publisher } from '../publisher/Pricekeeper2Publisher'
import * as Logger from '@randlabs/js-logger'
import { sleep } from '../common/sleep'
import { PythSymbolInfo } from './SymbolInfo'
import { PythData } from 'backend/common/basetypes'
const fs = require('fs')
const algosdk = require('algosdk')

type WorkerRoutineStatus = {
  status: StatusCode,
  reason?: string,
  data?: PythData,
  pub?: PublishInfo
}

async function workerRoutine (fetcher: IPriceFetcher, publisher: IPublisher): Promise<WorkerRoutineStatus> {
  const data: PythData = fetcher.queryData()
  if (data === undefined) {
    return { status: StatusCode.NO_TICKER }
  }
  const pub = await publisher.publish(data)
  return { status: pub.status, reason: pub.reason, data, pub }
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
      throw new Error('❌ Cannot read account and/or verify program source: ' + e)
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

    Logger.info(`Gathering prices from Pyth network ${this.settings.symbols.sourceNetwork}...`)
    const symbolInfo = new PythSymbolInfo(this.settings.symbols.sourceNetwork)
    await symbolInfo.load()
    Logger.info(`Loaded ${symbolInfo.getSymbolCount()} product(s)`)

    const fetcher = new WormholePythPriceFetcher(this.settings.wormhole.spyServiceHost,
      this.settings.pyth.chainId,
      this.settings.pyth.emitterAddress,
      symbolInfo)

    Logger.info('Waiting for fetcher to boot...')
    await fetcher.start()

    Logger.info('Waiting for publisher to boot...')
    await publisher.start()

    Logger.info(`Starting worker routine, interval ${this.settings.pollInterval}s`)
    setInterval(this.callWorkerRoutine, this.settings.pollInterval * 1000, fetcher, publisher)

    while (!this.shouldQuit) {
      await sleep(1000)
    }
  }

  async callWorkerRoutine (fetcher: IPriceFetcher, publisher: IPublisher) {
    const wrs = await workerRoutine(fetcher, publisher)
    switch (wrs.status) {
      case StatusCode.OK: {
        Logger.info(`    TxID ${wrs.pub?.txid}`)
        const pendingInfo = await wrs.pub?.confirmation
        if (pendingInfo!['pool-error'] === '') {
          if (pendingInfo!['confirmed-round']) {
            Logger.info(` ✔ Confirmed at round ${pendingInfo!['confirmed-round']}`)
          } else {
            Logger.info('⚠ No confirmation information')
          }
        } else {
          Logger.error(`❌ Rejected: ${pendingInfo!['pool-error']}`)
        }

        if (wrs.data?.attestations === undefined) {
          Logger.warn(`No attestation data available. Txid= ${wrs.pub?.txid}`)
        } else {
          for (let i = 0; i < wrs!.data!.attestations!.length; ++i) {
            const att = wrs.data.attestations[i]
            Logger.info(`     ${att.symbol}     ${att.price} ± ${att.confidence} exp: ${att.exponent} twap:${att.twap}`)
          }
        }
        break
      }
      case StatusCode.NO_TICKER:
        // Logger.warn('⚠ Poll: No new data available from fetcher data source')
        break
      default:
        Logger.error('❌ ' + wrs.reason)
    }
  }
}
