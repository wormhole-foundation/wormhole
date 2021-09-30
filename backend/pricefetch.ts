import { getPythProgramKeyForCluster, PriceData, Product, PythConnection } from '@pythnetwork/client'
import {Cluster, clusterApiUrl, Connection, PublicKey} from '@solana/web3.js'

const SOLANA_CLUSTER_NAME: Cluster = 'devnet'
const connection = new Connection(clusterApiUrl(SOLANA_CLUSTER_NAME))
const pythPublicKey = getPythProgramKeyForCluster(SOLANA_CLUSTER_NAME)

const algosdk = require('algosdk')



const pythConnection = new PythConnection(connection, pythPublicKey)
pythConnection.onPriceChange((product: Product, price: PriceData) => {
    if (product.symbol == 'BTC/USD') {
        // sample output:
        // SRM/USD: $8.68725 ±$0.0131
        // tslint:disable-next-line:no-console
        console.log(`${product.symbol}: $${price.price} \xB1$${price.confidence}`)
    }
})

// tslint:disable-next-line:no-console
console.log("Reading from Pyth price feed...")
pythConnection.start()