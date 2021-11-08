/* eslint-disable linebreak-style */
const algosdk = require('algosdk')
const { exit } = require('process')
const readline = require('readline')
const PricecasterLib = require('../lib/pricecaster')
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
})
const spawnSync = require('child_process').spawnSync
const fs = require('fs')

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

async function startOp (algodClient, fromAddress, gexpTime, gkeys) {
  console.log('Compiling VAA Processor program code...')
  const out = spawnSync('python', ['teal/wormhole/pyteal/vaa-processor.py'])
  console.log(out.output.toString())

  // console.log('Compiling VAA Verify stateless program code...')
  // out = spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py'])
  // console.log(out.output.toString())

  const pclib = new PricecasterLib.PricecasterLib(algodClient)
  console.log('Creating new app...')
  const txId = await pclib.createVaaProcessorApp(fromAddress, gexpTime, gkeys.join(''), signCallback)
  console.log('txId: ' + txId)
  const txResponse = await pclib.waitForTransactionResponse(txId)
  const appId = pclib.appIdFromCreateAppResponse(txResponse)
  console.log('Deployment App Id: %d', appId)
}

(async () => {
  console.log('\nVAA Processor for Wormhole Deployment Tool -- (c)2021-22 Randlabs, Inc.')
  console.log('-----------------------------------------------------------------------\n')

  if (process.argv.length !== 6) {
    console.log('Usage: deploy <glistfile> <from> <network>\n')
    console.log('where:\n')
    console.log('glistfile              File containing the initial list of guardians')
    console.log('gexptime               Guardian set expiration time')
    console.log('from                   Deployer account')
    console.log('network                Testnet, betanet or mainnet')
    console.log('\nFile must contain one guardian key per line, formatted in hex, without hex prefix.')
    exit(0)
  }

  const listfile = process.argv[2]
  const gexpTime = process.argv[3]
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

  const fileDataStr = fs.readFileSync(listfile).toString()
  const gkeys = fileDataStr.match(/[^\r\n]+/g)
  if (!algosdk.isValidAddress(fromAddress)) {
    console.error('Invalid deployer address: ' + fromAddress)
    exit(1)
  }

  const algodClient = new algosdk.Algodv2(config.apiToken, config.server, config.port)

  console.log('Parameters for deployment: ')
  console.log('From: ' + fromAddress)
  console.log('Network: ' + network)
  console.log('Guardian expiration time: ' + gexpTime)
  console.log(`Guardian Keys: (${gkeys.length}) ` + gkeys)
  const answer = await ask('\nEnter YES to confirm parameters, anything else to abort. ')
  if (answer !== 'YES') {
    console.warn('Aborted by user.')
    exit(1)
  }
  globalMnemo = await ask('\nEnter mnemonic for sender account.\nBE SURE TO DO THIS FROM A SECURED SYSTEM\n')
  try {
    await startOp(algodClient, fromAddress, gexpTime, gkeys)
  } catch (e) {
    console.error('(!) Deployment Failed: ' + e.toString())
  }
  console.log('Bye.')
  exit(0)
})()
