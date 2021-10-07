import algosdk from 'algosdk'
import { IPublisher, PublishInfo } from '../IPublisher'
import { PriceTicker } from '../PriceTicker'
const PricecasterLib = require('../../lib/pricecaster')
const settings = require('../../settings')

export class StdAlgoPublisher implements IPublisher {
  private pclib: any
  private symbol: string
  private signKey: Uint8Array
  private validator: string
  constructor (symbol: string, appId: BigInt, validator: string, signKey: Uint8Array) {
    this.symbol = symbol
    this.signKey = signKey
    this.validator = validator
    const algodClient = new algosdk.Algodv2(settings.algo.token, settings.algo.api, settings.algo.port)
    this.pclib = new PricecasterLib.PricecasterLib(algodClient)
    this.pclib.setAppId(appId)
  }

  signCallback (sender: string, tx: algosdk.Transaction) {
    const txSigned = tx.signTxn(this.signKey)
    return txSigned
  }

  async publish (tick: PriceTicker): Promise<PublishInfo> {
    const publishInfo = new PublishInfo()
    const msg = this.pclib.createMessage(
      this.symbol,
      tick.price,
      tick.exponent,
      tick.confidence,
      tick.networkTime,
      this.signKey)

    const txId = await this.pclib.submitMessage(
      this.validator,
      msg,
      this.signCallback
    )

    publishInfo.txid = txId
    return publishInfo
  }
}
