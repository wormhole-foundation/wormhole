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

const gkeys = [
  '13947Bd48b18E53fdAeEe77F3473391aC727C638',
  'F18AbBac073741DD0F002147B735Ff642f3D113F',
  '9925A94DC043D0803f8ef502D2dB15cAc9e02D76',
  '9e4EC2D92af8602bCE74a27F99A836f93C4a31E4',
  '9C40c4052A3092AfB8C99B985fcDfB586Ed19c98',
  'B86020cF1262AA4dd5572Af76923E271169a2CA7',
  '1937617fE1eD801fBa14Bd8BB9EDEcBA7A942FFe',
  '9475b8D45DdE53614d92c779787C27fE2ef68752',
  '15A53B22c28AbC7B108612146B6aAa4a537bA305',
  '63842657C7aC7e37B04FBE76b8c54EFe014D04E1',
  '948ca1bBF4B858DF1A505b4C69c5c61bD95A12Bd',
  'A6923e2259F8B5541eD18e410b8DdEE618337ff0',
  'F678Daf4b7f2789AA88A081618Aa966D6a39e064',
  '8cF31021838A8B3fFA43a71a50609877846f9E6d',
  'eB15bCF2ae4f957012330B4741ecE3242De96184',
  'cc3766a03e4faec44Bda7a46D9Ea2A9D124e9Bf8',
  '841f499Ba89a6a8E9dD273BAd82Beb175094E5d7',
  'f5F2b82576e6CA17965dee853d08bbB471FA2433',
  '2bC2B1204599D4cA0d4Dde4a658a42c4dD13103a'
]
const sigkeys = [
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
const PYTH_PAYLOAD = '50325748000101230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd010000002f17254388fffffff70000002eed73d9000000000070d3b43f0000000037faa03d000000000e9e555100000000894af11c0000000037faa03d000000000dda6eb801000000000061a5ff9a'

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

describe('VAA Processor Smart-contract Tests', function () {
  let appId

  before(async function () {
    algodClient = new algosdk.Algodv2('', 'https://api.testnet.algoexplorer.io', '')
    pclib = new PricecasterLib.PricecasterLib(algodClient)

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
  }
  )
  it('Must fail to create app with incorrect guardian keys length', async function () {
    const gsexptime = 2524618800
    await expect(createApp(gsexptime, 0, ['BADADDRESS'])).to.be.rejectedWith('Bad Request')
  })
  it('Must create app with initial guardians and proper initial state', async function () {
    const gsexptime = 2524618800
    appId = await createApp(gsexptime, 0, gkeys)
    console.log('       [Created appId: %d]', appId)

    const gscount = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gscount')
    const gsexp = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'gsexp')
    expect(gscount.toString()).to.equal((gkeys.length).toString())
    expect(gsexp.toString()).to.equal(gsexptime.toString())

    let i = 0
    const buf = Buffer.alloc(8)
    for (const gk of gkeys) {
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
    const txid = await pclib.setVAAVerifyProgramHash(OWNER_ADDR, verifyProgramHash, signCallback)
    await pclib.waitForTransactionResponse(txid)
    const vphstate = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vphash')
    expect(vphstate).to.equal(verifyProgramHash)
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
    const params = await getTxParams()
    const vaa = testLib.createSignedVAA(0, sigkeys, 1, 1, 1, PYTH_EMITTER, 0, 0, PYTH_PAYLOAD)
    pclib.beginTxGroup()
    const vaaBody = Buffer.from(vaa.substr(12 + sigkeys.length * 132), 'hex')

    for (let i = 0; i < badSize; i++) {
      const sigSubset = sigkeys.slice(vssize * i, i < badSize - 1 ? ((vssize * i) + vssize) : undefined)
      const keySubset = gkeys.slice(vssize * i, i < badSize - 1 ? ((vssize * i) + vssize) : undefined)
      const lsig = new algosdk.LogicSigAccount(compiledVerifyProgram.compiledBytes, [new Uint8Array(Buffer.from(sigSubset.join(''), 'hex'))])
      pclib.addVerifyTx(verifyProgramHash, params, vaaBody, keySubset, gscount, lsig)
    }
    await expect(pclib.commitTxGroupSignedByLogic()).to.be.rejectedWith('Bad Request')
  })
  it('Must reject incorrect argument count for verify call', async function () {

  })
  it('Must reject unknown sender for verify call', async function () {

  })
  it('Must reject guardian set count argument not matching global state', async function () {

  })
  it('Must reject guardian key list argument not matching global state', async function () {

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
    const expectedSize = Math.ceil(gscount / vssize)
    const params = await getTxParams()
    const vaa = testLib.createSignedVAA(0, sigkeys, 1, 1, 1, PYTH_EMITTER, 0, 0, PYTH_PAYLOAD)
    const vaaBody = Buffer.from(vaa.substr(12 + sigkeys.length * 132), 'hex')

    pclib.beginTxGroup()
    for (let i = 0; i < expectedSize; i++) {
      const sigSubset = sigkeys.slice(vssize * i, Math.min(gscount, (vssize * i) + (vssize - 1)))
      const keySubset = gkeys.slice(vssize * i, Math.min(gscount, (vssize * i) + (vssize - 1)))
      const lsig = new algosdk.LogicSigAccount(compiledVerifyProgram, new Uint8Array([Buffer.from(sigSubset)]))
      pclib.addVerifyTx(verifyProgramHash, params, vaaBody, keySubset, gscount, lsig)
    }
    await pclib.commitTxGroupSignedByLogic()
  })
  it('Must verify and handle governance VAA', async function () {

  })
  it('Must reject unknown VAA', async function () {

  })
  it('Stateless: Must reject transaction with excess fee', async function () {

  })
  it('Stateless: Must reject incorrect number of logic program arguments', async function () {

  })
  it('Stateless: Must reject transaction with mismatching number of signatures', async function () {

  })
  it('Stateless: Must reject transaction with non-zero rekey', async function () {

  })
  it('Stateless: Must reject transaction call from bad app-id', async function () {

  })
  it('Stateless: Must reject non-app call tx type', async function () {

  })
  it('Stateless: Must reject invalid group size', async function () {

  })
  it('Stateless: Must reject signature verification failure', async function () {

  })
})
async function getTxParams () {
  const params = await algodClient.getTransactionParams().do()
  params.fee = 1000
  params.flatFee = true
  return params
}
