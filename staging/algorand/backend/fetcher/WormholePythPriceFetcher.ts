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
import { PythAttestation, PythData, VAA } from 'backend/common/basetypes'
import { IStrategy } from '../strategy/strategy'
import { IPriceFetcher } from './IPriceFetcher'
import * as Logger from '@randlabs/js-logger'
import { PythSymbolInfo } from 'backend/engine/SymbolInfo'
const { extract3 } = require('../../tools/app-tools')

export class WormholePythPriceFetcher implements IPriceFetcher {
  private client: SpyRPCServiceClient
  private pythEmitterAddress: { s: string, data: number[] }
  private pythChainId: number
  private stream: any
  private _hasData: boolean
  private coreWasm: any
  private data: PythData | undefined
  private symbolInfo: PythSymbolInfo
  constructor (spyRpcServiceHost: string, pythChainId: number, pythEmitterAddress: string, symbolInfo: PythSymbolInfo) {
    setDefaultWasm('node')
    this._hasData = false
    this.client = createSpyRPCServiceClient(spyRpcServiceHost)
    this.pythChainId = pythChainId
    this.symbolInfo = symbolInfo
    this.pythEmitterAddress = {
      data: Buffer.from(pythEmitterAddress, 'hex').toJSON().data,
      s: pythEmitterAddress
    }
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
  }

  hasData (): boolean {
    // Return when any price is ready
    return this._hasData
  }

  queryData (id?: string): any | undefined {
    const data = this.data
    this.data = undefined
    return data
  }

  private async onPythData (vaaBytes: Buffer) {
    // console.log(vaaBytes.toString('hex'))
    const v: VAA = this.coreWasm.parse_vaa(new Uint8Array(vaaBytes))
    const payload = Buffer.from(v.payload)
    const header = payload.readInt32BE(0)
    const version = payload.readInt16BE(4)

    if (header === 0x50325748) {
      if (version === 2) {
        const payloadId = payload.readUInt8(6)
        if (payloadId === 2) {
          const numAttest = payload.readInt16BE(7)
          const sizeAttest = payload.readInt16BE(9)

          //
          // Extract attestations for VAA body
          //
          const attestations: PythAttestation[] = []
          for (let i = 0; i < numAttest; ++i) {
            const attestation = extract3(payload, i * sizeAttest, sizeAttest)
            const productId = extract3(attestation, 7, 32)
            const priceId = extract3(attestation, 7 + 32, 32)

            // console.log(base58.encode(productId))
            // console.log(base58.encode(priceId))

            const pythAttest: PythAttestation = {
              symbol: this.symbolInfo.getSymbol(productId, priceId),
              productId,
              priceId,
              price_type: attestation.readInt8(71),
              price: attestation.readBigUInt64BE(72),
              exponent: attestation.readInt32BE(80),
              twap: attestation.readBigUInt64BE(84),
              twap_num_upd: attestation.readBigUInt64BE(92),
              twap_denom_upd: attestation.readBigUInt64BE(100),
              twac: attestation.readBigUInt64BE(108),
              twac_num_upd: attestation.readBigUInt64BE(116),
              twac_denom_upd: attestation.readBigUInt64BE(124),
              confidence: attestation.readBigUInt64BE(132),
              status: attestation.readInt8(140),
              corporate_act: attestation.readInt8(141),
              timestamp: attestation.readBigUInt64BE(142)
            }

            attestations.push(pythAttest)
          }
          this.data = {
            vaaBody: vaaBytes.slice(6 + v.signatures.length * 66),
            signatures: vaaBytes.slice(6, 6 + v.signatures.length * 66),
            attestations
          }

          Logger.info(`VAA gs=${v.guardian_set_index} #sig=${v.signatures.length} ts=${v.timestamp} nonce=${v.nonce} seq=${v.sequence} clev=${v.consistency_level} payload_size=${payload.length} #attestations=${numAttest}`)
          this._hasData = true
        } else {
          Logger.error(`Bad Pyth VAA payload Id (${payloadId}). Expected 2`)
        }
      } else {
        Logger.error(`Bad Pyth VAA version (${version}). Expected 2`)
      }
    } else {
      Logger.error(`Bad VAA header (0x${header.toString(16)}). Expected 'P2WH'`)
    }
  }
}
