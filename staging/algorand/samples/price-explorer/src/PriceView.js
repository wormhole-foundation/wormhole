/**
 * Price Explorer Sample Application.
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
import React from 'react'
const PricecasterSdk = require('pricecaster-client-sdk')
const humanizeDuration = require('humanize-duration')

const APP_ID = 75517911
const SOLANA_CLUSTER = 'devnet'

class PriceView extends React.Component {
  constructor(props) {
    super(props)
    this.timer = null
    this.sdk = new PricecasterSdk('', 'https://algoindexer.testnet.algoexplorerapi.io/', '', APP_ID, SOLANA_CLUSTER);
    this.state = {
      symbolInfo: new Map(),
      priceData: []
    }
  }

  
  async componentDidMount() {
    await this.sdk.connect()
    this.timer = setInterval(() => this.fetchGlobalState(), 2000)
  }
  componentWillUnmount() {
    clearInterval(this.timer)
    this.timer = null
  }
  async fetchGlobalState() {
    const priceData = await this.sdk.queryData()
    for (const priceItem of priceData) {
      const prev = this.state.priceData.find((item) => (item.symbol === priceItem.symbol))
      priceItem['change'] = (prev !== undefined) ? (prev.price.toNumber() - priceItem.price.toNumber()) : 0
    }
    priceData.sort((a, b) => {
      return (a.symbol > b.symbol) ? 1 : -1
    })
    this.setState({ priceData })
  }

  render() {
    return (
      <div>
        <h1>Price Explorer</h1>
        <h2>
          Algorand Application <a href={"https://testnet.algoexplorer.io/application/" + APP_ID}
            target="_blank" rel="noreferrer">{APP_ID}</a>
        </h2>
        <hr />
        <table>
          <tbody>
            <tr>
              <th>Symbol</th>
              <th>Price</th>
              <th>Avg Price</th>
              <th>Confidence</th>
              <th>Avg Confidence</th>
              <th>Last update</th>
            </tr>
            {this.state.priceData.map((k, i) => {
              const exp = parseFloat(k.exp.toString())
              return (<tr key={i} className={k.change < 0 ? "valueup" : (k.change > 0 ? "valuedown" : "valueequal")}>
                <td>{k.symbol.toString()}</td>
                <td>{parseFloat(k.price.toString()) / (10 ** -exp)}</td>
                <td>{parseFloat(k.twap.toString()) / (10 ** -exp)}</td>
                <td>{parseFloat(k.conf.toString()) / (10 ** -exp)}</td>
                <td>{parseFloat(k.twac.toString()) / (10 ** -exp)}</td>
                <td>{humanizeDuration(Date.now() - parseInt(k.time) * 1000, { round: true })}</td>
                
              </tr>)
            })}
          </tbody>
        </table>
      </div>
    )
  }
}



export default PriceView