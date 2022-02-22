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
import { parseProductData } from '@pythnetwork/client'
import { Cluster, clusterApiUrl, Connection, PublicKey } from '@solana/web3.js'
import { stringToHex } from 'web3-utils'

const CLUSTER_TO_PYTH_PROGRAM_KEY: Record<Cluster, string> = {
  'mainnet-beta': 'FsJ3A3u2vn5cTVofAjvy6y5kwABJAqYWpe4975bi2epH',
  devnet: 'gSbePebfvPy7tRqimPoVecS2UsBvYv46ynrzWocc92s',
  testnet: '8tfDNiaEyrV6Q1U4DEXrEigs9DoDtkugzFbybENEbCDz'
}

export class PythSymbolInfo {
  private network: Cluster;
  private symbolMap: Map<{ productId: string, priceId: string }, string>
  constructor (network: Cluster) {
    this.symbolMap = new Map()
    this.network = network
  }

  /**
   * Gets the public key of the Pyth program running on the given cluster.
   */
  private getPythProgramKeyForCluster (cluster: Cluster): PublicKey {
    if (CLUSTER_TO_PYTH_PROGRAM_KEY[cluster] !== undefined) {
      return new PublicKey(CLUSTER_TO_PYTH_PROGRAM_KEY[cluster])
    } else {
      throw new Error(
        `Invalid Solana cluster name: ${cluster}. Valid options are: ${JSON.stringify(
          Object.keys(CLUSTER_TO_PYTH_PROGRAM_KEY)
        )}`
      )
    }
  }

  async load () {
    const connection = new Connection(clusterApiUrl(this.network))
    const pythPublicKey = this.getPythProgramKeyForCluster(this.network)
    const accounts = await connection.getProgramAccounts(pythPublicKey, 'finalized')

    for (const acc of accounts) {
      const productData = parseProductData(acc.account.data)
      if (productData.type === 2) {
        // console.log(`prod: ${acc.pubkey.toBase58()} price: ${productData.priceAccountKey.toBase58()} ${productData.product.symbol}`)
        this.symbolMap.set({
          productId: acc.pubkey.toBase58(),
          priceId: productData.priceAccountKey.toBase58()
        },
        productData.product.symbol)
      }
    }
  }

  getSymbol (productId: string, priceId: string) {
    return this.symbolMap.get({ productId, priceId })
  }

  getSymbolCount () {
    return this.symbolMap.size
  }
}
