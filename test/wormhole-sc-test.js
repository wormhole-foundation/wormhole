/* eslint-disable no-unused-expressions */
const PricecasterLib = require('../lib/pricecaster')
const tools = require('../tools/app-tools')
const algosdk = require('algosdk')
const { expect } = require('chai')
const chai = require('chai')
chai.use(require('chai-as-promised'))
const spawnSync = require('child_process').spawnSync
const fs = require('fs')
let pclib
let algodClient

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

function makeVAA () {

}

async function createApp (gsexptime, gkeys) {
  const txId = await pclib.createVaaProcessorApp(OWNER_ADDR, gsexptime, gkeys.join(''), signCallback)
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
    await expect(createApp(gsexptime, ['BADADDRESS'])).to.be.rejectedWith('Bad Request')
  })
  it('Must create app with initial guardians and proper initial state', async function () {
    const gsexptime = 2524618800
    appId = await createApp(gsexptime, gkeys)
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
    const compiledProgram = await pclib.compileProgram(program)
    const txid = await pclib.setVAAVerifyProgramHash(OWNER_ADDR, compiledProgram.hash, signCallback)
    await pclib.waitForTransactionResponse(txid)
    const vphstate = await tools.readAppGlobalStateByKey(algodClient, appId, OWNER_ADDR, 'vphash')
    expect(vphstate).to.equal(compiledProgram.hash)
  })
  it('Must disallow setting stateless logic hash from non-owner', async function () {
    const teal = 'test/temp/vaa-verify.teal'
    spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py', appId, teal])
    const program = fs.readFileSync(teal, 'utf8')
    const compiledProgram = await pclib.compileProgram(program)
    await expect(pclib.setVAAVerifyProgramHash(OTHER_ADDR, compiledProgram.hash, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject setting stateless logic hash from group transaction', async function () {
    const teal = 'test/temp/vaa-verify.teal'
    spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py', appId, teal])
    const program = fs.readFileSync(teal, 'utf8')
    const compiledProgram = await pclib.compileProgram(program)
    const hash = algosdk.decodeAddress(compiledProgram.hash).publicKey
    const appArgs = [new Uint8Array(Buffer.from('setvphash')), hash.subarray(0, 10)]

    const params = await algodClient.getTransactionParams().do()
    params.fee = 1000
    params.flatFee = true

    pclib.beginTxGroup()
    const appTx = algosdk.makeApplicationNoOpTxn(OWNER_ADDR, params, this.appId, appArgs)
    const dummyTx = algosdk.makeApplicationNoOpTxn(OWNER_ADDR, params, this.appId, appArgs)
    pclib.addTxToGroup(appTx)
    pclib.addTxToGroup(dummyTx)
    await expect(pclib.commitTxGroup(OWNER_ADDR, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject setting stateless logic hash with invalid address length', async function () {
    const teal = 'test/temp/vaa-verify.teal'
    spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py', appId, teal])
    const program = fs.readFileSync(teal, 'utf8')
    const compiledProgram = await pclib.compileProgram(program)
    const hash = algosdk.decodeAddress(compiledProgram.hash).publicKey
    const appArgs = [new Uint8Array(Buffer.from('setvphash')), hash.subarray(0, 10)]
    await expect(pclib.callApp(OWNER_ADDR, appArgs, [], signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must verify and handle Pyth VAA', async function () {

  })
  it('Must verify and handle governance VAA', async function () {

  })
  it('Must reject unknown VAA', async function () {

  })
  it('Must reject incorrect transaction group size', async function () {

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
