/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { getPythProgramKeyForCluster, PriceData, Product, PythConnection } from '@pythnetwork/client'
import { Cluster, clusterApiUrl, Connection } from '@solana/web3.js'
import algosdk from 'algosdk'
const PricecasterLib = require('../lib/pricecaster')
const settings = require('../settings')

/**
 * The main Price fetcher service class.
 */
export class PriceFetcher {
  private pclib: any
  pythConnection: PythConnection

  constructor () {
    const SOLANA_CLUSTER_NAME: Cluster = settings.pyth.solanaClusterName as Cluster
    const connection = new Connection(clusterApiUrl(SOLANA_CLUSTER_NAME))
    const pythPublicKey = getPythProgramKeyForCluster(SOLANA_CLUSTER_NAME)
    const algodClient = new algosdk.Algodv2(settings.token, settings.api, settings.port)
    this.pythConnection = new PythConnection(connection, pythPublicKey)
    this.pclib = new PricecasterLib.PricecasterLib(algodClient)
  }

  /**
   * Starts the service.
   */
  run () {
    console.log('Pricecaster Service Fetcher  -- (c) 2021 Randlabs.io\n')
    console.log('AlgoClient Configuration: ')
    console.log(`API: '${settings.api}' PORT:'${settings.port}'`)

    if (this.preflightCheck()) {
      console.log('Preflight check passed, starting Pyth listener...')
      this.pythConnection.start()
      this.pythConnection.onPriceChange((product: Product, price: PriceData) => {
        this.onPriceChange(product, price)
      })
    }

    console.log('Booting done.')
  }

  /**
   * Executes a preflight check of configuration parameters.
   * @returns True if parameters are ok, false otherwise.
   */
  preflightCheck () {
    return true
  }

  /**
   * Price reception handler.
   * @param product The reported symbol/pair
   * @param price The reported price data
   */
  onPriceChange (product: Product, price: PriceData) {
    // eslint-disable-next-line no-prototype-builtins
    if (settings.symbols.hasOwnProperty(product.symbol)) {
      console.log(`${product.symbol}: $${price.price} \xB1$${price.confidence}`)
    }
  }
}
