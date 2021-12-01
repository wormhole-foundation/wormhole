/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { IPriceFetcher } from './IPriceFetcher'
import { IStrategy } from '../strategy/strategy'
import { getPythProgramKeyForCluster, PriceData, Product, PythConnection } from '@pythnetwork/client'
import { Cluster, clusterApiUrl, Connection } from '@solana/web3.js'
import { PriceTicker } from '../common/priceTicker'
import { getEmitterAddressEth, getSignedVAA } from '@certusone/wormhole-sdk'

export class WormholePythPriceFetcher implements IPriceFetcher {
   private strategy: IStrategy
   private symbol: string
   private pythConnection: PythConnection

   constructor (symbol: string, strategy: IStrategy, solanaClusterName: string) {
     const SOLANA_CLUSTER_NAME: Cluster = solanaClusterName as Cluster
     const connection = new Connection(clusterApiUrl(SOLANA_CLUSTER_NAME))
     const pythPublicKey = getPythProgramKeyForCluster(SOLANA_CLUSTER_NAME)
     this.pythConnection = new PythConnection(connection, pythPublicKey)
     this.strategy = strategy
     this.symbol = symbol
   }

   async start () {
     await this.pythConnection.start()
     this.pythConnection.onPriceChange((product: Product, price: PriceData) => {
       if (product.symbol === this.symbol) {
         this.onPriceChange(price)
       }
     })
   }

   stop (): void {
     this.pythConnection.stop()
   }

   setStrategy (s: IStrategy) {
     this.strategy = s
   }

   hasData (): boolean {
     return this.strategy.bufferCount() > 0
   }

   queryTicker (): PriceTicker | undefined {
     getEmitterAddressEth()
    
     await getSignedVAA("https://wormhole-v2-testnet-api.certus.one", )
     //return this.strategy.getPrice()
   }

   private onPriceChange (price: PriceData) {
    GrpcWebImpl
    PublicRPCServiceClientImpl
     getSignedVAA()
     const pt: PriceTicker = new PriceTicker(price.priceComponent,
       price.confidenceComponent, price.exponent, price.publishSlot)
     this.strategy.put(pt)
   }
}
