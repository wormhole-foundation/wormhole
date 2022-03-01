/**
 * Pricecaster JS Client SDK
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
const { parseProductData } = require('@pythnetwork/client')
const { PublicKey, clusterApiUrl, Connection } = require('@solana/web3.js')
const { base58 } = require('ethers/lib/utils')
const { Uint64BE } = require('int64-buffer')

const PRICE_DATA_BYTES_LEN = 46
const algosdk = require('algosdk')
const clusterToPythProgramKey = {}
clusterToPythProgramKey['mainnet-beta'] = 'FsJ3A3u2vn5cTVofAjvy6y5kwABJAqYWpe4975bi2epH'
clusterToPythProgramKey.devnet = 'gSbePebfvPy7tRqimPoVecS2UsBvYv46ynrzWocc92s'
clusterToPythProgramKey.testnet = '8tfDNiaEyrV6Q1U4DEXrEigs9DoDtkugzFbybENEbCDz'

/**
 * The Pricecaster SDK class.
 */
class PricecasterSdk {
  /**
   * Constructs a new PricecasterSDK Object.
   * @param {*} algoToken  The token for connecting to the desired Algorand indexer.
   * @param {*} algoApi  The host API URL for connecting to the desired Algorand indexer.
   * @param {*} algoPort The port number for connecting to the desired Algorand indexer.
   * @param {*} contractAppId The application Id containing the on-chain Pricekeeper contract to interact with.
   * @param {*} sourceCluster The Solana cluster (devnet, mainnet-beta, testnet) where symbol name information
   *                          is fetched. If unspecified or invalid, no symbol name will be retrieved.
   */
  constructor (token, api, port, appId, sourceCluster) {
    this.indexer = new algosdk.Indexer(token, api, port)
    this.appId = appId
    this.sourceCluster = sourceCluster
    this.symbolInfo = new Map()
  }

  getPythProgramKeyForCluster (cluster) {
    if (clusterToPythProgramKey[cluster] !== undefined) {
      return new PublicKey(clusterToPythProgramKey[cluster])
    } else {
      throw new Error(
        `Invalid Solana cluster name: ${cluster}. Valid options are: ${JSON.stringify(
          Object.keys(clusterToPythProgramKey)
        )}`
      )
    }
  }

  /**
   * Connect to Solana cluster to download symbol name information.
   */
  async connect () {
    this.symbolInfo = new Map()
    if (this.sourceCluster === 'mainnet-beta' ||
      this.sourceCluster === 'devnet' ||
      this.sourceCluster === 'testnet') {
      const connection = new Connection(clusterApiUrl(this.sourceCluster))
      const pythPublicKey = this.getPythProgramKeyForCluster(this.sourceCluster)
      const accounts = await connection.getProgramAccounts(pythPublicKey, 'finalized')
      for (const acc of accounts) {
        const productData = parseProductData(acc.account.data)
        if (productData.type === 2) {
          this.symbolInfo.set(acc.pubkey.toBase58() + productData.priceAccountKey.toBase58(), productData.product.symbol)
        }
      }
    }
  }

  /**
   * Retrive the current price information.  If symbol information is available,
   * price and product identifiers will be mapped to readable name in format:
   * market.name.pair  for example  Crypto.ALGO/USD.
   * @returns Array with objects.
   * @remarks Read Pyth documentation for meaning of fields: https://docs.pyth.network/how-pyth-works/account-structure
   */
  async queryData () {
    const priceData = []
    const app = await this.indexer.lookupApplications(this.appId).do()
    const gstate = app.application.params['global-state']
    for (const entry of gstate) {
      const key = Buffer.from(entry.key, 'base64')
      const productId = base58.encode(key.slice(0, 32))
      const priceId = base58.encode(key.slice(32, 64))
      const v = Buffer.from(entry.value.bytes, 'base64')
      if (v.length === PRICE_DATA_BYTES_LEN) {
        const sym = this.symbolInfo.get(productId + priceId)
        const price = new Uint64BE(v, 0)
        priceData.push({
          price,
          exp: v.readInt32BE(8),
          twap: new Uint64BE(v, 12),
          twac: new Uint64BE(v, 20),
          conf: new Uint64BE(v, 20 + 8),
          time: new Uint64BE(v, 20 + 8 + 1 + 1 + 8),
          symbol: sym === undefined ? productId.toString() + priceId.toString() : sym
        })
      }
    }
    return priceData
  }
}

module.exports = PricecasterSdk
