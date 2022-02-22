import React from 'react'
import { parseProductData } from '@pythnetwork/client'
import { PublicKey, clusterApiUrl, Connection } from '@solana/web3.js'
import { base58 } from 'ethers/lib/utils'
import { Buffer } from 'buffer/'
const { Uint64BE , Int64BE } = require("int64-buffer");

const algosdk = require('algosdk')
const clusterToPythProgramKey = {}
clusterToPythProgramKey['mainnet-beta'] = 'FsJ3A3u2vn5cTVofAjvy6y5kwABJAqYWpe4975bi2epH'
clusterToPythProgramKey['devnet'] = 'gSbePebfvPy7tRqimPoVecS2UsBvYv46ynrzWocc92s'
clusterToPythProgramKey['testnet'] = '8tfDNiaEyrV6Q1U4DEXrEigs9DoDtkugzFbybENEbCDz'

class PriceView extends React.Component {
  constructor(props) {
    super(props)
    this.timer = null
    this.client = new algosdk.Algodv2('', 'https://api.testnet.algoexplorer.io', '');
    this.state = {
      appId: 73652776,
      symbolInfo: undefined,
      priceData: []
    }
  }

  async readAppGlobalState(appId, accountAddr) {
    const accountInfoResponse = await this.client.accountInformation(accountAddr).do()
    for (let i = 0; i < accountInfoResponse['created-apps'].length; i++) {
      if (accountInfoResponse['created-apps'][i].id === appId) {
        const globalState = accountInfoResponse['created-apps'][i].params['global-state']
        return globalState
      }
    }
  }

  getPythProgramKeyForCluster(cluster) {
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
  async componentDidMount() {
    const symbolInfo = new Map()
    const SOLANA_CLUSTER_NAME = 'devnet'
    const connection = new Connection(clusterApiUrl(SOLANA_CLUSTER_NAME))
    const pythPublicKey = this.getPythProgramKeyForCluster(SOLANA_CLUSTER_NAME)
    const accounts = await connection.getProgramAccounts(pythPublicKey, 'finalized')
      for (const acc of accounts) {
        const productData = parseProductData(acc.account.data)
        if (productData.type === 2) {
          //console.log(`prod: 0x${Buffer.from(acc.pubkey.toBytes()).toString('hex')} price: 0x${Buffer.from(productData.priceAccountKey.toBytes()).toString('hex')} ${productData.product.symbol}`)
          symbolInfo.set(acc.pubkey.toBase58() + productData.priceAccountKey.toBase58(), productData.product.symbol)
        }
        this.setState({
          symbolInfo
        })

        // console.log(symbolInfo)
      }

    this.timer = setInterval( () => this.fetchGlobalState(), 1000 )

    await this.fetchGlobalState()
  }
  componentWillUnmount() {
    clearInterval(this.timer)
    this.timer = null
  }
  async fetchGlobalState() {
    const priceData = []
    const gstate = await this.readAppGlobalState(73652776, 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU')
    for (const entry of gstate) {
      const key = Buffer.from(entry.key, 'base64')
      const productId = base58.encode(key.slice(0, 32))
      const priceId = base58.encode(key.slice(32, 64))
      const v = Buffer.from(entry.value.bytes, 'base64');
      const sym = this.state.symbolInfo.get(productId + priceId);

      if (sym !== undefined) {
        priceData.push({
          price: new Uint64BE(v, 0),
          exp: v.readInt32BE(8),
          twap: new Uint64BE(v, 12),
          conf: new Uint64BE(v, 12 + 8),
          symbol: this.state.symbolInfo.get(productId + priceId),
        })
      }
    }
    priceData.sort( (a, b) => { 
      return (a.symbol > b.symbol) ? 1 : -1
    })
    this.setState({ priceData })
  }

  render() {
    return (
      <div>
        <h1>Price Explorer</h1>
        <p>
          Algorand Application <strong>{this.state.appId}</strong>
        </p>
        <p>
          Loaded {this.state.symbolInfo?.size} product(s) from Pyth <strong>devnet</strong>
        </p>
        <hr />
        <table>
          <tbody>
          <tr>
            <th>Symbol</th>
            <th>Price</th>
            <th>Confidence</th>
          </tr>
          {this.state.priceData.map((k, i) => {
            const exp = parseFloat(k.exp.toString())
            return (<tr key={i}>
              <td>{k.symbol.toString()}</td>
              <td>{parseFloat(k.price.toString()) / (10 ** -exp) }</td>
              <td>{parseFloat(k.conf.toString()) / (10 ** -exp) }</td>
            </tr>)
          })}
          </tbody>
        </table>
      </div>
    )

  }
}



export default PriceView