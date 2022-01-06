/* eslint-disable no-unused-vars */
/* eslint-disable camelcase */
/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
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
