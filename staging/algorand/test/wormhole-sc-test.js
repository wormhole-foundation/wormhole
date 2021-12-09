/* eslint-disable no-unused-expressions */
const PricecasterLib = require('../lib/pricecaster')
const tools = require('../tools/app-tools')
const algosdk = require('algosdk')
const { expect } = require('chai')
const chai = require('chai')
chai.use(require('chai-as-promised'))
const spawnSync = require('child_process').spawnSync
const fs = require('fs')
const TestLib = require('../test/testlib.js')
const { makePaymentTxnWithSuggestedParams } = require('algosdk')
const testLib = new TestLib.TestLib()
let pclib
let algodClient
let verifyProgramHash
let compiledVerifyProgram
const OWNER_ADDR = 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU'
const OWNER_MNEMO = 'assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master'
const OTHER_ADDR = 'DMTBK62XZ6KNI7L5E6TRBTPB4B3YNVB4WYGSWR42SEV4XKV4LYHGBW4O34'
const OTHER_MNEMO = 'old agree harbor cost pink fog chunk hope vital used rural soccer model acquire clown host friend bring marriage surge dirt surge slab absent punch'
const SIGNATURES = {}
SIGNATURES[OWNER_ADDR] = algosdk.mnemonicToSecretKey(OWNER_MNEMO)
SIGNATURES[OTHER_ADDR] = algosdk.mnemonicToSecretKey(OTHER_MNEMO)

const guardianKeys = [
  '52A26Ce40F8CAa8D36155d37ef0D5D783fc614d2',
  '389A74E8FFa224aeAD0778c786163a7A2150768C',
  'B4459EA6482D4aE574305B239B4f2264239e7599',
  '072491bd66F63356090C11Aae8114F5372aBf12B',
  '51280eA1fd2B0A1c76Ae29a7d54dda68860A2bfF',
  'fa9Aa60CfF05e20E2CcAA784eE89A0A16C2057CB',
  'e42d59F8FCd86a1c5c4bA351bD251A5c5B05DF6A',
  '4B07fF9D5cE1A6ed58b6e9e7d6974d1baBEc087e',
  'c8306B84235D7b0478c61783C50F990bfC44cFc0',
  'C8C1035110a13fe788259A4148F871b52bAbcb1B',
  '58A2508A20A7198E131503ce26bBE119aA8c62b2',
  '8390820f04ddA22AFe03be1c3bb10f4ba6CF94A0',
  '1FD6e97387C34a1F36DE0f8341E9D409E06ec45b',
  '255a41fC2792209CB998A8287204D40996df9E54',
  'bA663B12DD23fbF4FbAC618Be140727986B3BBd0',
  '79040E577aC50486d0F6930e160A5C75FD1203C6',
  '3580D2F00309A9A85efFAf02564Fc183C0183A96',
  '3869795913D3B6dBF3B24a1C7654672c69A23c35',
  '1c0Cc52D7673c52DE99785741344662F5b2308a0'

]
const guardianPrivKeys = [
  '563d8d2fd4e701901d3846dee7ae7a92c18f1975195264d676f8407ac5976757',
  '8d97f25916a755df1d9ef74eb4dbebc5f868cb07830527731e94478cdc2b9d5f',
  '9bd728ad7617c05c31382053b57658d4a8125684c0098f740a054d87ddc0e93b',
  '5a02c4cd110d20a83a7ce8d1a2b2ae5df252b4e5f6781c7855db5cc28ed2d1b4',
  '93d4e3b443bf11f99a00901222c032bd5f63cf73fc1bcfa40829824d121be9b2',
  'ea40e40c63c6ff155230da64a2c44fcd1f1c9e50cacb752c230f77771ce1d856',
  '87eaabe9c27a82198e618bca20f48f9679c0f239948dbd094005e262da33fe6a',
  '61ffed2bff38648a6d36d6ed560b741b1ca53d45391441124f27e1e48ca04770',
  'bd12a242c6da318fef8f98002efb98efbf434218a78730a197d981bebaee826e',
  '20d3597bb16525b6d09e5fb56feb91b053d961ab156f4807e37d980f50e71aff',
  '344b313ffbc0199ff6ca08cacdaf5dc1d85221e2f2dc156a84245bd49b981673',
  '848b93264edd3f1a521274ca4da4632989eb5303fd15b14e5ec6bcaa91172b05',
  'c6f2046c1e6c172497fc23bd362104e2f4460d0f61984938fa16ef43f27d93f6',
  '693b256b1ee6b6fb353ba23274280e7166ab3be8c23c203cc76d716ba4bc32bf',
  '13c41508c0da03018d61427910b9922345ced25e2bbce50652e939ee6e5ea56d',
  '460ee0ee403be7a4f1eb1c63dd1edaa815fbaa6cf0cf2344dcba4a8acf9aca74',
  'b25148579b99b18c8994b0b86e4dd586975a78fa6e7ad6ec89478d7fbafd2683',
  '90d7ac6a82166c908b8cf1b352f3c9340a8d1f2907d7146fb7cd6354a5436cca',
  'b71d23908e4cf5d6cd973394f3a4b6b164eb1065785feee612efdfd8d30005ed'
]

const PYTH_EMITTER = '0x3afda841c1f43dd7d546c8a581ba1f92a139f4133f9f6ab095558f6a359df5d4'
const OTHER_EMITTER = '0x1111111111111111111111111111111111111111111111111111111111111111'
const PYTH_PAYLOAD = '50325748000101230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd010000002f17254388fffffff70000002eed73d9000000000070d3b43f0000000037faa03d000000000e9e555100000000894af11c0000000037faa03d000000000dda6eb801000000000061a5ff9a'
const OTHER_PAYLOAD = 'f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0f0'

let pythVaa
let pythVaaBody
let pythVaaSignatures
let otherVaa
let otherVaaBody
let otherVaaSignatures

// --------------------------------------------------------------------------
// Utility functions
// --------------------------------------------------------------------------

async function createApp (gsexptime, gsindex, gkeys) {
  const txId = await pclib.createVaaProcessorApp(OWNER_ADDR, gsexptime, gsindex, gkeys.join(''), signCallback)
  const txResponse = await pclib.waitForTransactionResponse(txId)
  const appId = pclib.appIdFromCreateAppResponse(txResponse)
  pclib.setAppId(appId)
  return appId
}

function signCallback (sender, tx) {
  const txSigned = tx.signTxn(SIGNATURES[sender].sk)
  return txSigned
}

async function getTxParams () {
  const params = await algodClient.getTransactionParams().do()
  params.fee = 1000
  params.flatFee = true
  return params
}

async function execVerify (groupSize, vsSize, gkeys, signatures, vaaBody, gscount, fee, sender, verifyCallback) {
  const params = await getTxParams()
  if (fee !== undefined) {
    params.fee = fee
  }
  const senderAddress = sender !== undefined ? sender : verifyProgramHash
  const verifyCallbackFn = verifyCallback !== undefined ? verifyCallback : pclib.addVerifyTx.bind(pclib)
  pclib.beginTxGroup()
  const sigSubsets = []
  for (let i = 0; i < groupSize; i++) {
    const st = vsSize * i
    const keySubset = gkeys.slice(st, i < groupSize - 1 ? st + vsSize : undefined)
    sigSubsets.push(signatures.slice(i * 132 * vsSize, i < groupSize - 1 ? ((i * 132 * vsSize) + 132 * vsSize) : undefined))
    verifyCallbackFn(senderAddress, params, vaaBody, keySubset, gscount)
  }
  const tx = await pclib.commitVerifyTxGroup(compiledVerifyProgram.compiledBytes, sigSubsets)
  return tx
}

// ===============================================================================================================
//
// Test suite starts here
//
// ===============================================================================================================

describe('VAA Processor Smart-contract Tests', function () {
  let appId

  before(async function () {
    // algodClient = new algosdk.Algodv2('', 'https://api.betanet.algoexplorer.io', '')
    algodClient = new algosdk.Algodv2('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa', 'http://localhost', '4001')
    pclib = new PricecasterLib.PricecasterLib(algodClient)
    const ownerAcc = algosdk.mnemonicToSecretKey(OWNER_MNEMO)

    const ownerAccInfo = await algodClient.accountInformation(ownerAcc.addr).do()
    expect(ownerAccInfo.amount).to.be.at.least(algosdk.algosToMicroalgos(10), 'Owner must have enough funding (10 ALGO) to run tests')

    console.log('Clearing accounts of all previous apps...')
    const appsTo = await tools.readCreatedApps(algodClient, OWNER_ADDR)
    for (let i = 0; i < appsTo.length; i++) {
      console.log('Clearing ' + appsTo[i].id)
      try {
        const txId = await pclib.deleteApp(OWNER_ADDR, signCallback, appsTo[i].id)
        await pclib.waitForConfirmation(txId)
      } catch (e) {
        console.error('Could not delete application! Reason: ' + e)
      }
    }

    const vaaProcessorClearState = 'test/temp/vaa-clear-state.teal'
    const vaaProcessorApproval = 'test/temp/vaa-processor.teal'

    pclib.setVaaProcessorApprovalFile(vaaProcessorApproval)
    pclib.setVaaProcessorClearStateFile(vaaProcessorClearState)
    console.log(spawnSync('python', ['teal/wormhole/pyteal/vaa-processor.py', vaaProcessorApproval, vaaProcessorClearState]).output.toString())

    pythVaa = testLib.createSignedVAA(0, guardianPrivKeys, 1, 1, 1, PYTH_EMITTER, 0, 0, PYTH_PAYLOAD)
    pythVaaBody = Buffer.from(pythVaa.substr(12 + guardianPrivKeys.length * 132), 'hex')
    pythVaaSignatures = pythVaa.substr(12, guardianPrivKeys.length * 132)

    otherVaa = testLib.createSignedVAA(0, guardianPrivKeys, 1, 1, 1, OTHER_EMITTER, 0, 0, OTHER_PAYLOAD)
    otherVaaBody = Buffer.from(otherVaa.substr(12 + guardianPrivKeys.length * 132), 'hex')
    otherVaaSignatures = otherVaa.substr(12, guardianPrivKeys.length * 132)
  }
  )

  it('Must fail to create app with incorrect guardian keys length', async function () {
    const gsexptime = 2524618800
    await expect(createApp(gsexptime, 0, ['BADADDRESS'])).to.be.rejectedWith('Bad Request')
  })

  it('Must create app with initial guardians and proper initial state', async function () {
    const gsexptime = 2524618800
    appId = await createApp(gsexptime, 0, guardianKeys)
    console.log('    - [Created appId: %d]', appId)

    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const gsexp = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gsexp')
    expect(gscount.toString()).to.equal((guardianKeys.length).toString())
    expect(gsexp.toString()).to.equal(gsexptime.toString())

    let i = 0
    const buf = Buffer.alloc(8)
    for (const gk of guardianKeys) {
      buf.writeBigUint64BE(BigInt(i++))
      const gkstate = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, buf.toString())
      expect(Buffer.from(gkstate, 'base64').toString('hex')).to.equal(gk.toLowerCase())
    }
  })

  it('Must set stateless logic hash from owner', async function () {
    const teal = 'test/temp/vaa-verify.teal'
    spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py', appId, teal])
    const program = fs.readFileSync(teal, 'utf8')
    compiledVerifyProgram = await pclib.compileProgram(program)
    verifyProgramHash = compiledVerifyProgram.hash
    console.log('    - Stateless program: ', verifyProgramHash)

    const txid = await pclib.setVAAVerifyProgramHash(OWNER_ADDR, verifyProgramHash, signCallback)
    await pclib.waitForTransactionResponse(txid)
    const vphstate = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vphash')
    expect(vphstate).to.equal(verifyProgramHash)

    // Feed this account for verification fees.
    const parms = await getTxParams()
    const tx = makePaymentTxnWithSuggestedParams(OWNER_ADDR, verifyProgramHash, 200000, undefined, undefined, parms)
    const signedTx = signCallback(OWNER_ADDR, tx)
    await algodClient.sendRawTransaction(signedTx).do()
    await pclib.waitForTransactionResponse(tx.txID().toString())
  })

  it('Must disallow setting stateless logic hash from non-owner', async function () {
    await expect(pclib.setVAAVerifyProgramHash(OTHER_ADDR, verifyProgramHash, signCallback)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject setting stateless logic hash from group transaction', async function () {
    const appArgs = [new Uint8Array(Buffer.from('setvphash')), new Uint8Array(verifyProgramHash)]
    const params = await getTxParams()

    pclib.beginTxGroup()
    const appTx = algosdk.makeApplicationNoOpTxn(OWNER_ADDR, params, this.appId, appArgs)
    const dummyTx = algosdk.makeApplicationNoOpTxn(OWNER_ADDR, params, this.appId, appArgs)
    pclib.addTxToGroup(appTx)
    pclib.addTxToGroup(dummyTx)
    await expect(pclib.commitTxGroup(OWNER_ADDR, signCallback)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject setting stateless logic hash with invalid address length', async function () {
    const appArgs = [new Uint8Array(Buffer.from('setvphash')), new Uint8Array(verifyProgramHash).subarray(0, 10)]
    await expect(pclib.callApp(OWNER_ADDR, appArgs, [], signCallback)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject incorrect transaction group size', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const badSize = 1 + Math.ceil(gscount / vssize)
    await expect(execVerify(badSize, vssize, guardianKeys, pythVaaSignatures, pythVaaBody, gscount)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject incorrect argument count for verify call', async function () {
    const verifyFunc = function (sender, params, payload, gksubset, totalguardians) {
      const appArgs = []
      appArgs.push(new Uint8Array(Buffer.from('verify')))
      const tx = algosdk.makeApplicationNoOpTxn(sender,
        params,
        appId,
        appArgs, undefined, undefined, undefined,
        new Uint8Array(payload))
      pclib.groupTx.push(tx)

      return tx.txID()
    }
    pclib.beginTxGroup()
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    await expect(execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures, pythVaaBody, gscount, undefined, undefined, verifyFunc)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject unknown sender for verify call', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    await expect(execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures, pythVaaBody, gscount, undefined, OTHER_ADDR)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject guardian set count argument not matching global state', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    await expect(execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures, pythVaaBody, 2)).to.be.rejectedWith('Bad Request')
  })

  it('Must reject guardian key list argument not matching global state', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    const gkBad = guardianKeys.slice(0, guardianKeys.length - 3)
    await expect(execVerify(groupSize, vssize, gkBad, pythVaaSignatures, pythVaaBody, 2)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject non-app call transaction in group', async function () {

  })
  it('Must reject app-call with mismatched AppId in group', async function () {

  })
  it('Must reject transaction with not verified bit set in group', async function () {

  })

  it('Must verify and handle Pyth VAA', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    const tx = await execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures, pythVaaBody, gscount)
    await pclib.waitForConfirmation(tx)
  })
  it('Must verify and handle governance VAA', async function () {
    // TBD
  })

  it('Must reject unknown emitter VAA', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    await expect(execVerify(groupSize, vssize, guardianKeys, otherVaaSignatures, otherVaaBody, gscount)).to.be.rejectedWith('Bad Request')
  })

  it('Stateless: Must reject transaction with excess fee', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    await expect(execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures, pythVaaBody, gscount, 800000)).to.be.rejectedWith('Bad Request')
  })

  it('Stateless: Must reject incorrect number of logic program arguments', async function () {

  })

  it('Stateless: Must reject transaction with mismatching number of signatures', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    const pythVaaSignatures2 = pythVaaSignatures.substr(0, pythVaaSignatures.length - 132 - 1)
    await expect(execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures2, pythVaaBody, gscount)).to.be.rejectedWith('Bad Request')
  })

  it('Stateless: Must reject transaction with non-zero rekey', async function () {

  })

  it('Stateless: Must reject transaction call from bad app-id', async function () {

  })

  it('Stateless: Must reject signature verification failure', async function () {
    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const vssize = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vssize')
    const groupSize = Math.ceil(gscount / vssize)
    let pythVaaSignatures2 = pythVaaSignatures.substr(0, pythVaaSignatures.length - 132 - 1)
    pythVaaSignatures2 += '0d525ac1524ec9d9ee623ef535a867e8f86d9b3f8e4c7b4234dbe7bb40dc8494327af2fa37c3db50064d6114f2e1441c4eee444b83636f11ce1f730f7b38490e2800'
    await expect(execVerify(groupSize, vssize, guardianKeys, pythVaaSignatures2, pythVaaBody, gscount)).to.be.rejectedWith('Bad Request')
  })
})
