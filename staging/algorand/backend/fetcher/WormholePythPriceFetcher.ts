/* eslint-disable camelcase */
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

import {
  importCoreWasm,
  setDefaultWasm
} from '@certusone/wormhole-sdk/lib/cjs/solana/wasm'
import {
  createSpyRPCServiceClient, subscribeSignedVAA
} from '@certusone/wormhole-spydk'
import { SpyRPCServiceClient } from '@certusone/wormhole-spydk/lib/cjs/proto/spy/v1/spy'
import { PythData, Symbol, VAA } from 'backend/common/basetypes'
import { IStrategy } from '../strategy/strategy'
import { IPriceFetcher } from './IPriceFetcher'
import * as Logger from '@randlabs/js-logger'

export class WormholePythPriceFetcher implements IPriceFetcher {
  private symbolMap: Map<string, {
    name: string,
    publishIntervalSecs: number,
    pythData: PythData | undefined
  }>

  private client: SpyRPCServiceClient
  private pythEmitterAddress: { s: string, data: number[] }
  private pythChainId: number
  private strategy: IStrategy
  private stream: any
  private _hasData: boolean
  private coreWasm: any

  constructor (spyRpcServiceHost: string, pythChainId: number, pythEmitterAddress: string, symbols: Symbol[], strategy: IStrategy) {
    setDefaultWasm('node')
    this._hasData = false
    this.client = createSpyRPCServiceClient(spyRpcServiceHost)
    this.pythChainId = pythChainId
    this.pythEmitterAddress = {
      data: Buffer.from(pythEmitterAddress, 'hex').toJSON().data,
      s: pythEmitterAddress
    }
    this.strategy = strategy
    this.symbolMap = new Map()

    symbols.forEach((sym) => {
      this.symbolMap.set(sym.productId + sym.priceId, {
        name: sym.name,
        publishIntervalSecs: sym.publishIntervalSecs,
        pythData: undefined
      })
    })
  }

  async start () {
    this.coreWasm = await importCoreWasm()
    // eslint-disable-next-line camelcase
    this.stream = await subscribeSignedVAA(this.client,
      {
        filters:
          [{
            emitterFilter: {
              chainId: this.pythChainId,
              emitterAddress: this.pythEmitterAddress.s
            }
          }]
      })

    this.stream.on('data', (data: { vaaBytes: Buffer }) => {
      try {
        this._hasData = true
        this.onPythData(data.vaaBytes)
      } catch (e) {
        Logger.error(`Failed to parse VAA data. \nReason: ${e}\nData: ${data}`)
      }
    })

    this.stream.on('error', (e: Error) => {
      Logger.error('Stream error: ' + e)
    })
  }

  stop (): void {
    this._hasData = false
  }

  setStrategy (s: IStrategy) {
    this.strategy = s
  }

  hasData (): boolean {
    // Return when any price is ready
    return this._hasData
  }

  queryData (id: string): any | undefined {
    const v = this.symbolMap.get(id)
    if (v === undefined) {
      Logger.error(`Unsupported symbol with identifier ${id}`)
    } else {
      return v.pythData
    }
  }

  private async onPythData (vaaBytes: Buffer) {
    // console.log(vaaBytes.toString('hex'))
    const v: VAA = this.coreWasm.parse_vaa(new Uint8Array(vaaBytes))
    const payload = Buffer.from(v.payload)
    const productId = payload.slice(7, 7 + 32)
    const priceId = payload.slice(7 + 32, 7 + 32 + 32)
    // console.log(productId.toString('hex'), priceId.toString('hex'))

    const k = productId.toString('hex') + priceId.toString('hex')
    const sym = this.symbolMap.get(k)

    if (sym !== undefined) {
      sym.pythData = {
        symbol: sym.name,
        vaaBody: vaaBytes.slice(6 + v.signatures.length * 66),
        signatures: vaaBytes.slice(6, 6 + v.signatures.length * 66),
        price_type: payload.readInt8(71),
        price: payload.readBigUInt64BE(72),
        exponent: payload.readInt32BE(80),
        confidence: payload.readBigUInt64BE(132),
        status: payload.readInt8(140),
        corporate_act: payload.readInt8(141),
        timestamp: payload.readBigUInt64BE(142)
      }
    }

    // if (pythPayload.status === 0) {
    //  console.log('WARNING: Symbol trading status currently halted (0). Publication will be skipped.')
    // } else
    // eslint-disable-next-line no-lone-blocks
  }
}
