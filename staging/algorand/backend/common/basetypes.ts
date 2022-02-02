/* eslint-disable no-unused-vars */
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

export type Symbol = {
    name: string,
    productId: string,
    priceId: string,
    publishIntervalSecs: number,
    pubCount: number
}

export type VAA = {
    version: number,
    guardian_set_index: number,
    signatures: [],
    timestamp: number,
    nonce: number,
    emitter_chain: number,
    emitter_address: [],
    sequence: number,
    consistency_level: number,
    payload: []
  }

export type PythData = {
  vaaBody: Buffer,
  signatures: Buffer,

  // Informational fields.

  symbol?: string,
  price_type?: number,
  price?: BigInt,
  exponent?: number,
  twap?: BigInt,
  twap_num_upd?: BigInt,
  twap_denom_upd?: BigInt,
  twac?: BigInt,
  twac_num_upd?: BigInt,
  twac_denom_upd?: BigInt,
  confidence?: BigInt,
  status?: number,
  corporate_act?: number,
  timestamp?: BigInt
}
