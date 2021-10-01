const PricecasterLib = require('../lib/pricecaster')
const tools = require('../tools/app-tools')
const algosdk = require('algosdk')
// Test general configuration for Betanet

const validatorAddr = 'OPDM7ACAW64Q4VBWAL77Z5SHSJVZZ44V3BAN7W44U43SUXEOUENZMZYOQU'
const validatorMnemo = 'assault approve result rare float sugar power float soul kind galaxy edit unusual pretty tone tilt net range pelican avoid unhappy amused recycle abstract master'
const otherAddr = 'DMTBK62XZ6KNI7L5E6TRBTPB4B3YNVB4WYGSWR42SEV4XKV4LYHGBW4O34'
const otherMnemo = 'old agree harbor cost pink fog chunk hope vital used rural soccer model acquire clown host friend bring marriage surge dirt surge slab absent punch'
const symbol = 'BTC/USD         '
const signatures = {}
signatures[validatorAddr] = algosdk.mnemonicToSecretKey(validatorMnemo)
signatures[otherAddr] = algosdk.mnemonicToSecretKey(otherMnemo)

function signCallback (sender, tx) {
  const txSigned = tx.signTxn(signatures[sender].sk)
  return txSigned
}

describe('Price-Keeper contract tests', function () {
  let pclib
  let algodClient

  before(async function () {
    algodClient = new algosdk.Algodv2('', 'https://api.betanet.algoexplorer.io', '')
    pclib = new PricecasterLib.PricecasterLib(algodClient)

    console.log('Clearing accounts of all previous apps...')
    const appsTo = await tools.readCreatedApps(algodClient, validatorAddr)
    for (let i = 0; i < appsTo.length; i++) {
      console.log('Clearing ' + appsTo[i].id)
      try {
        const txId = await pclib.deleteApp(validatorAddr, signCallback, appsTo[i].id)
        await pclib.waitForConfirmation(txId)
      } catch (e) {
        console.error('Could not delete application! Reason: ' + e)
      }
    }

    console.log('Creating new app...')
    const txId = await pclib.createApp(validatorAddr, validatorAddr, symbol, signCallback)
    const txResponse = await pclib.waitForTransactionResponse(txId)
    const appId = pclib.appIdFromCreateAppResponse(txResponse)
    pclib.setAppId(appId);
    console.log('App Id: %d', appId)
  })
  it('x', function () {

  })
})
