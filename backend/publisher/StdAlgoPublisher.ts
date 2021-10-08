import algosdk from 'algosdk'
import { IPublisher, PublishInfo } from '../IPublisher'
import { PriceTicker } from '../PriceTicker'
import { StatusCode } from '../statusCodes'
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

  async start () {
    await this.pclib.compileApprovalProgram()
  }

  stop () {

  }

  signCallback (sender: string, tx: algosdk.Transaction) {
    const txSigned = tx.signTxn(this.signKey)
    return txSigned
  }

  async publish (tick: PriceTicker): Promise<PublishInfo> {
    const publishInfo: PublishInfo = { status: StatusCode.OK }
    let msg, txId
    try {
      msg = this.pclib.createMessage(
        this.symbol,
        tick.price,
        BigInt(tick.exponent),
        tick.confidence,
        tick.networkTime,
        this.signKey)
      publishInfo.msgb64 = msg.toString('base64')
    } catch (e: any) {
      publishInfo.status = StatusCode.ERROR_CREATE_MESSAGE
      publishInfo.reason = e.toString()
      return publishInfo
    }

    try {
      txId = await this.pclib.submitMessage(
        this.validator,
        msg,
        this.signCallback.bind(this)
      )
      publishInfo.txid = txId
    } catch (e: any) {
      publishInfo.status = StatusCode.ERROR_SUBMIT_MESSAGE
      publishInfo.reason = e.response.text ? e.response.text : e.toString()
      return publishInfo
    }

    return publishInfo
  }
}
