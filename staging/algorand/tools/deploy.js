/* eslint-disable linebreak-style */
const algosdk = require('algosdk')
const { exit } = require('process')
const readline = require('readline')
const PricecasterLib = require('../lib/pricecaster')
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
})

function ask (questionText) {
  return new Promise((resolve) => {
    rl.question(questionText, input => resolve(input))
  })
}

let globalMnemo = ''

function signCallback (sender, tx) {
  const txSigned = tx.signTxn(algosdk.mnemonicToSecretKey(globalMnemo).sk)
  return txSigned
}

async function startOp (algodClient, symbol, vaddr, fromAddress) {
  const pclib = new PricecasterLib.PricecasterLib(algodClient)
  console.log('Creating new app...')
  const txId = await pclib.createApp(fromAddress, vaddr, symbol, signCallback)
  console.log('txId: ' + txId)
  const txResponse = await pclib.waitForTransactionResponse(txId)
  const appId = pclib.appIdFromCreateAppResponse(txResponse)
  console.log('Deployment App Id: %d', appId)
}

(async () => {
  console.log('\nPricekeeper Deployment Tool')
  console.log('-----------------------------\n')

  if (process.argv.length !== 6) {
    console.log('Usage: deploy <symbol> <vaddr> <from> <network>\n')
    console.log('where:\n')
    console.log('symbol                 The supported symbol for this priceKeeper (e.g BTC/USD)')
    console.log('vaddr                  The validator address')
    console.log('from                   Deployer account')
    console.log('network                Testnet, betanet or mainnet')
    exit(0)
  }

  const symbol = process.argv[2]
  const vaddr = process.argv[3]
  const fromAddress = process.argv[4]
  const network = process.argv[5]

  const config = { server: '', apiToken: '', port: '' }
  if (network === 'betanet') {
    config.server = 'https://api.betanet.algoexplorer.io'
  } else if (network === 'mainnet') {
    config.server = 'https://api.algoexplorer.io'
  } else if (network === 'testnet') {
    config.server = 'https://api.testnet.algoexplorer.io'
  } else {
    console.error('Invalid network: ' + network)
    exit(1)
  }

  if (!algosdk.isValidAddress(vaddr)) {
    console.error('Invalid validator address: ' + vaddr)
    exit(1)
  }

  if (!algosdk.isValidAddress(fromAddress)) {
    console.error('Invalid deployer address: ' + fromAddress)
    exit(1)
  }

  const algodClient = new algosdk.Algodv2(config.apiToken, config.server, config.port)

  console.log('Parameters for deployment: ')
  console.log('symbol: ' + symbol)
  console.log('Validator addr: ' + vaddr)
  console.log('From: ' + fromAddress)
  console.log('Network: ' + network)
  const answer = await ask('\nEnter YES to confirm parameters, anything else to abort. ')
  if (answer !== 'YES') {
    console.warn('Aborted by user.')
    exit(1)
  }
  globalMnemo = await ask('\nEnter mnemonic for sender account.\nBE SURE TO DO THIS FROM A SECURED SYSTEM\n')
  await startOp(algodClient, symbol, vaddr, fromAddress)
  console.log('Bye.')
  exit(0)
})()
