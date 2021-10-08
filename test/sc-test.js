const PricecasterLib = require('../lib/pricecaster')
const tools = require('../tools/app-tools')
const algosdk = require('algosdk')
const { expect } = require('chai')
const chai = require('chai')
chai.use(require('chai-as-promised'))
// Test general configuration for Betanet

const VALIDATOR_ADDR = 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU'
const VALIDATOR_MNEMO = 'assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master'
const OTHER_ADDR = 'DMTBK62XZ6KNI7L5E6TRBTPB4B3YNVB4WYGSWR42SEV4XKV4LYHGBW4O34'
const OTHER_MNEMO = 'old agree harbor cost pink fog chunk hope vital used rural soccer model acquire clown host friend bring marriage surge dirt surge slab absent punch'
const SIGNATURES = {}
SIGNATURES[VALIDATOR_ADDR] = algosdk.mnemonicToSecretKey(VALIDATOR_MNEMO)
SIGNATURES[OTHER_ADDR] = algosdk.mnemonicToSecretKey(OTHER_MNEMO)

const VALID_SYMBOL = 'BTC/USD         '
const VALID_PRICE = BigInt(485265555)
const VALID_EXPONENT = BigInt(4)
const VALID_CONF = BigInt(1400000)
const VALID_SLOT = BigInt(104566700)

function signCallback (sender, tx) {
  const txSigned = tx.signTxn(SIGNATURES[sender].sk)
  return txSigned
}

describe('Price-Keeper contract tests', function () {
  let pclib
  let algodClient
  let appId
  let lastTs

  before(async function () {
    algodClient = new algosdk.Algodv2('', 'https://api.betanet.algoexplorer.io', '')
    pclib = new PricecasterLib.PricecasterLib(algodClient)

    console.log('Clearing accounts of all previous apps...')
    const appsTo = await tools.readCreatedApps(algodClient, VALIDATOR_ADDR)
    for (let i = 0; i < appsTo.length; i++) {
      console.log('Clearing ' + appsTo[i].id)
      try {
        const txId = await pclib.deleteApp(VALIDATOR_ADDR, signCallback, appsTo[i].id)
        await pclib.waitForConfirmation(txId)
      } catch (e) {
        console.error('Could not delete application! Reason: ' + e)
      }
    }

    console.log('Creating new app...')
    const txId = await pclib.createApp(VALIDATOR_ADDR, VALIDATOR_ADDR, VALID_SYMBOL, signCallback)
    const txResponse = await pclib.waitForTransactionResponse(txId)
    appId = pclib.appIdFromCreateAppResponse(txResponse)
    pclib.setAppId(appId)
    console.log('App Id: %d', appId)
  })
  it('Must create app with proper initial global state', async function () {
    const stVAddr = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'vaddr')
    const stSym = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'sym')
    expect(Buffer.from(stSym, 'base64').toString()).to.equal(VALID_SYMBOL)
    expect(stVAddr.toString()).to.equal(VALIDATOR_ADDR)
  })
  it('Must accept valid message and store data', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, VALID_SLOT, SIGNATURES[VALIDATOR_ADDR].sk)
    const txid = await pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)
    expect(txid).to.have.length(52)
    await pclib.waitForTransactionResponse(txid)
    // console.log(await tools.printAppGlobalState(algodClient, appId, VALIDATOR_ADDR))
    const stPrice = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'price')
    const stExp = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'exp')
    const stConf = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'conf')
    const stSlot = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'slot')
    expect(stPrice.toString()).to.equal(VALID_PRICE.toString())
    expect((Buffer.from(stExp, 'base64')).readBigInt64BE()).to.equal(VALID_EXPONENT)
    expect(stSlot.toString()).to.equal(VALID_SLOT.toString())
    expect(stConf.toString()).to.equal(VALID_CONF.toString())
  })
  it('Must accept second message with different price', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE + BigInt(400), VALID_EXPONENT + BigInt(3), VALID_CONF + BigInt(2), VALID_SLOT + BigInt(100), SIGNATURES[VALIDATOR_ADDR].sk)
    const txid = await pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)
    expect(txid).to.have.length(52)
    await pclib.waitForTransactionResponse(txid)
    const stPrice = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'price')
    const stExp = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'exp')
    const stConf = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'conf')
    const stSlot = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'slot')
    expect(stPrice.toString()).to.equal((VALID_PRICE + BigInt(400)).toString())
    expect((Buffer.from(stExp, 'base64')).readBigInt64BE()).to.equal(VALID_EXPONENT + BigInt(3))
    expect(stSlot.toString()).to.equal((VALID_SLOT + BigInt(100)).toString())
    expect(stConf.toString()).to.equal((VALID_CONF + BigInt(2)).toString())
    lastTs = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'ts')
  })
  it('Must accept negative exponent, stored as 2-complement 64bit', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, BigInt(-9), VALID_CONF, VALID_SLOT, SIGNATURES[VALIDATOR_ADDR].sk)
    const txid = await pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)
    expect(txid).to.have.length(52)
    await pclib.waitForTransactionResponse(txid)
    const stExp = await tools.readAppGlobalStateByKey(algodClient, appId, VALIDATOR_ADDR, 'exp')
    const bufExp = Buffer.from(stExp, 'base64')
    const val = bufExp.readBigInt64BE()
    expect(val.toString()).to.equal('-9')
  })
  it('Must reject non-validator as signer', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, VALID_SLOT, SIGNATURES[OTHER_ADDR].sk)
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject non-validator as sender', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, VALID_SLOT, SIGNATURES[VALIDATOR_ADDR].sk)
    await expect(pclib.submitMessage(OTHER_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject future timestamp', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, VALID_SLOT, SIGNATURES[VALIDATOR_ADDR].sk, undefined, undefined, undefined, BigInt(lastTs + 200))
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject old timestamp', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, VALID_SLOT, SIGNATURES[VALIDATOR_ADDR].sk, undefined, undefined, undefined, BigInt(lastTs - 999999))
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject zero-priced message', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, BigInt(0), VALID_EXPONENT, VALID_CONF, VALID_SLOT, SIGNATURES[VALIDATOR_ADDR].sk)
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject zero slot', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, BigInt(0), SIGNATURES[VALIDATOR_ADDR].sk)
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject bad header', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, BigInt(0), SIGNATURES[VALIDATOR_ADDR].sk, 'BADHEADER')
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject bad destination appId', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, BigInt(0), SIGNATURES[VALIDATOR_ADDR].sk, undefined, BigInt(100))
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
  it('Must reject bad message version', async function () {
    const msgBuffer = pclib.createMessage(VALID_SYMBOL, VALID_PRICE, VALID_EXPONENT, VALID_CONF, BigInt(0), SIGNATURES[VALIDATOR_ADDR].sk, undefined, undefined, 0)
    await expect(pclib.submitMessage(VALIDATOR_ADDR, msgBuffer, signCallback)).to.be.rejectedWith('Bad Request')
  })
})
