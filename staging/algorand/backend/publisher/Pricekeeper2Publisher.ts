import algosdk from 'algosdk'
import { IPublisher, PublishInfo } from './IPublisher'
import { StatusCode } from '../common/statusCodes'
import { PythData } from 'backend/common/basetypes'
const PricecasterLib = require('../../lib/pricecaster')
const tools = require('../../tools/app-tools')

export class Pricekeeper2Publisher implements IPublisher {
  private algodClient: algosdk.Algodv2
  private pclib: any
  private account: algosdk.Account
  private vaaProcessorAppId: number
  private vaaProcessorOwner: string
  private numOfVerifySteps: number = 0
  private guardianCount: number = 0
  private stepSize: number = 0
  private dumpFailedTx: boolean
  private dumpFailedTxDirectory: string | undefined
  private compiledVerifyProgram: { bytes: Uint8Array, hash: string } = { bytes: new Uint8Array(), hash: '' }
  constructor (vaaProcessorAppId: number,
    priceKeeperAppId: number,
    vaaProcessorOwner: string,
    verifyProgramBinary: Uint8Array,
    verifyProgramHash: string,
    signKey: algosdk.Account,
    algoClientToken: string,
    algoClientServer: string,
    algoClientPort: string,
    dumpFailedTx: boolean = false,
    dumpFailedTxDirectory: string = './') {
    this.account = signKey
    this.compiledVerifyProgram.bytes = verifyProgramBinary
    this.compiledVerifyProgram.hash = verifyProgramHash
    this.vaaProcessorAppId = vaaProcessorAppId
    this.vaaProcessorOwner = vaaProcessorOwner
    this.dumpFailedTx = dumpFailedTx
    this.dumpFailedTxDirectory = dumpFailedTxDirectory
    this.algodClient = new algosdk.Algodv2(algoClientToken, algoClientServer, algoClientPort)
    this.pclib = new PricecasterLib.PricecasterLib(this.algodClient)
    this.pclib.setAppId('vaaProcessor', vaaProcessorAppId)
    this.pclib.setAppId('pricekeeper', priceKeeperAppId)
    this.pclib.enableDumpFailedTx(this.dumpFailedTx)
    this.pclib.setDumpFailedTxDirectory(this.dumpFailedTxDirectory)
  }

  async start () {
  }

  stop () {
  }

  signCallback (sender: string, tx: algosdk.Transaction) {
    const txSigned = tx.signTxn(this.account.sk)
    return txSigned
  }

  async publish (data: PythData): Promise<PublishInfo> {
    const publishInfo: PublishInfo = { status: StatusCode.OK }

    const txParams = await this.algodClient.getTransactionParams().do()
    txParams.fee = 1000
    txParams.flatFee = true

    this.guardianCount = await tools.readAppGlobalStateByKey(this.algodClient, this.vaaProcessorAppId, this.vaaProcessorOwner, 'gscount')
    this.stepSize = await tools.readAppGlobalStateByKey(this.algodClient, this.vaaProcessorAppId, this.vaaProcessorOwner, 'vssize')
    this.numOfVerifySteps = Math.ceil(this.guardianCount / this.stepSize)
    if (this.guardianCount === 0 || this.stepSize === 0) {
      throw new Error('cannot get guardian count and/or step-size from global state')
    }
    //
    // (!)
    // Stateless programs cannot access state nor stack from stateful programs, so
    // for the VAA Verify program to use the guardian set, we pass the global state as TX argument,
    // (and check it against the current global list to be sure it's ok). This way it can be read by
    // VAA verifier as a stateless program CAN DO READS of call transaction arguments in a group.
    // The same technique is used for the note field, where the payload is set.
    //

    try {
      const guardianKeys = []
      const buf = Buffer.alloc(8)
      for (let i = 0; i < this.guardianCount; i++) {
        buf.writeBigUInt64BE(BigInt(i++))
        const gk = await tools.readAppGlobalStateByKey(this.algodClient, this.vaaProcessorAppId, this.vaaProcessorOwner, buf.toString())
        guardianKeys.push(Buffer.from(gk, 'base64').toString('hex'))
      }

      const strSig = data.signatures.toString('hex')

      const gid = this.pclib.beginTxGroup()
      const sigSubsets = []
      for (let i = 0; i < this.numOfVerifySteps; i++) {
        const st = this.stepSize * i
        const sigSetLen = 132 * this.stepSize

        const keySubset = guardianKeys.slice(st, i < this.numOfVerifySteps - 1 ? st + this.stepSize : undefined)

        sigSubsets.push(strSig.slice(i * sigSetLen, i < this.numOfVerifySteps - 1 ? ((i * sigSetLen) + sigSetLen) : undefined))
        this.pclib.addVerifyTx(gid, this.compiledVerifyProgram.hash, txParams, data.vaaBody, keySubset, this.guardianCount)
      }
      this.pclib.addPriceStoreTx(gid, this.vaaProcessorOwner, txParams, data.symbol, data.vaaBody.slice(51))
      const txId = await this.pclib.commitVerifyTxGroup(gid, this.compiledVerifyProgram.bytes, sigSubsets, this.vaaProcessorOwner, this.signCallback.bind(this))
      publishInfo.txid = txId
    } catch (e: any) {
      publishInfo.status = StatusCode.ERROR_SUBMIT_MESSAGE
      publishInfo.reason = e.response.text ? e.response.text : e.toString()
      return publishInfo
    }

    return publishInfo
  }
}
