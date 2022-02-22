import algosdk from 'algosdk'
import { IPublisher, PublishInfo } from './IPublisher'
import { StatusCode } from '../common/statusCodes'
import { PythData } from 'backend/common/basetypes'
const PricecasterLib = require('../../lib/pricecaster')
const tools = require('../../tools/app-tools')
const { arrayChunks } = require('../../tools/app-tools')

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

      if (guardianKeys.length === 0) {
        throw new Error('No guardian keys in global state.')
      }

      const keyChunks = arrayChunks(guardianKeys, this.stepSize)
      const sigChunks = arrayChunks(data.signatures, this.stepSize * 132)

      const gid = this.pclib.beginTxGroup()
      for (let i = 0; i < this.numOfVerifySteps; i++) {
        this.pclib.addVerifyTx(gid, this.compiledVerifyProgram.hash, txParams, data.vaaBody, keyChunks[i], this.guardianCount)
      }
      this.pclib.addPriceStoreTx(gid, this.vaaProcessorOwner, txParams, data.vaaBody.slice(51))
      const txId = await this.pclib.commitVerifyTxGroup(gid, this.compiledVerifyProgram.bytes, data.signatures.length, sigChunks, this.vaaProcessorOwner, this.signCallback.bind(this))
      publishInfo.txid = txId
      publishInfo.confirmation = algosdk.waitForConfirmation(this.algodClient, txId, 10)
    } catch (e: any) {
      publishInfo.status = StatusCode.ERROR_SUBMIT_MESSAGE
      if (e.response) {
        publishInfo.reason = e.response.text ? e.response.text : e.toString()
      } else {
        publishInfo.reason = e.toString()
      }
      return publishInfo
    }

    return publishInfo
  }
}
