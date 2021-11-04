const PricecasterLib = require('../lib/pricecaster')
const tools = require('../tools/app-tools')
const algosdk = require('algosdk')
const { expect } = require('chai')
const chai = require('chai')
chai.use(require('chai-as-promised'))

const OWNER_ADDR = 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU'
const OWNER_MNEMO = 'assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master'
const OTHER_ADDR = 'DMTBK62XZ6KNI7L5E6TRBTPB4B3YNVB4WYGSWR42SEV4XKV4LYHGBW4O34'
const OTHER_MNEMO = 'old agree harbor cost pink fog chunk hope vital used rural soccer model acquire clown host friend bring marriage surge dirt surge slab absent punch'
const SIGNATURES = {}
SIGNATURES[OWNER_ADDR] = algosdk.mnemonicToSecretKey(OWNER_MNEMO)
SIGNATURES[OTHER_ADDR] = algosdk.mnemonicToSecretKey(OTHER_MNEMO)

function makeVAA() {
  
}

function signCallback (sender, tx) {
  const txSigned = tx.signTxn(SIGNATURES[sender].sk)
  return txSigned
}

describe('VAA Processor Smart-contract Tests', function () {
  let pclib
  let algodClient
  let appId
  let lastTs

  before(async function () {
    algodClient = new algosdk.Algodv2('', 'https://api.betanet.algoexplorer.io', '')
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

    console.log('Creating new app...')
    const txId = await pclib.createApp(OWNER_ADDR, OWNER_ADDR, VALID_SYMBOL, signCallback)
    const txResponse = await pclib.waitForTransactionResponse(txId)
    appId = pclib.appIdFromCreateAppResponse(txResponse)
    pclib.setAppId(appId)
    console.log('App Id: %d', appId)
  })
})