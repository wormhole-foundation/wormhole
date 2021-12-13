/* eslint-disable camelcase */
/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import {
  importCoreWasm,
  setDefaultWasm
} from '@certusone/wormhole-sdk/lib/cjs/solana/wasm'
import {
  createSpyRPCServiceClient, subscribeSignedVAA
} from '@certusone/wormhole-spydk'
import { SpyRPCServiceClient } from '@certusone/wormhole-spydk/lib/cjs/proto/spy/v1/spy'
import { PythPayload, Symbol, VAA } from 'backend/common/basetypes'
import { PriceTicker } from '../common/priceTicker'
import { IStrategy } from '../strategy/strategy'
import { IPriceFetcher } from './IPriceFetcher'

export class WormholePythPriceFetcher implements IPriceFetcher {
  private symbolMap: Map<string, {
    name: string,
    publishIntervalSecs: number,
    priceKeeperV2AppId: number
  }>

  private client: SpyRPCServiceClient
  private pythEmitterAddress: { s: string, data: number[] }
  private pythChainId: number
  private strategy: IStrategy
  private stream: any

  constructor (spyRpcServiceHost: string, pythChainId: number, pythEmitterAddress: string, symbols: Symbol[], strategy: IStrategy) {
    setDefaultWasm('node')
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
        priceKeeperV2AppId: sym.priceKeeperV2AppId
      })
    })
  }

  async start () {
    // eslint-disable-next-line camelcase
    const { parse_vaa } = await importCoreWasm()
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
        const parsedVAA: VAA = parse_vaa(new Uint8Array(data.vaaBytes))
        this.onPythData(parsedVAA)
      } catch (e) {
        console.error(`Failed to parse VAA data. \nReason: ${e}\nData: ${data}`)
      }
    })

    this.stream.on('error', (e: Error) => {
      console.log('Stream error: ' + e)
    })
  }

  stop (): void {
  }

  setStrategy (s: IStrategy) {
    this.strategy = s
  }

  hasData (): boolean {
    return this.strategy.bufferCount() > 0
  }

  queryTicker (): PriceTicker | undefined {
    return this.strategy.getPrice()
  }

  private onPythData (v: VAA) {
    // unpack payload

    const payload = Buffer.from(v.payload)
    const productId = payload.slice(7, 7 + 32)
    const priceId = payload.slice(7 + 32, 7 + 32 + 32)
    console.log(productId.toString('hex'), priceId.toString('hex'))

    const k = productId.toString('hex') + priceId.toString('hex')
    if (this.symbolMap.has(k)) {
      const pythPayload: PythPayload = {
        price_type: payload.readInt8(71),
        price: payload.readBigUInt64BE(72),
        exponent: payload.readUInt32BE(80),
        twap: payload.readBigUInt64BE(84),
        twap_num_upd: payload.readBigUInt64BE(92),
        twap_denom_upd: payload.readBigUInt64BE(100),
        twac: payload.readBigUInt64BE(108),
        twac_num_upd: payload.readBigUInt64BE(116),
        twac_denom_upd: payload.readBigUInt64BE(124),
        confidence: payload.readBigUInt64BE(132),
        status: payload.readInt8(133),
        corporate_act: payload.readInt8(134),
        timestamp: payload.readBigUInt64BE(135)
      }

      if (pythPayload.status === 0) {
        console.log('WARNING: Symbol trading status currently halted (0). Publication will be skipped.')
      }

      console.log('got price for ' + this.symbolMap.get(k)?.name)

      console.log(pythPayload)
    }
  }
}
