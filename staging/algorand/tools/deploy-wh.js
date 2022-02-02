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
  console.log('Compiling programs ...\n')
  let out = spawnSync('python', ['teal/wormhole/pyteal/vaa-processor.py'])
  console.log(out.output.toString())
  out = spawnSync('python', ['teal/wormhole/pyteal/pricekeeper-v2.py'])
  console.log(out.output.toString())

  const pclib = new PricecasterLib.PricecasterLib(algodClient)
  console.log('Creating VAA Processor...')
  let txId = await pclib.createVaaProcessorApp(fromAddress, gexpTime, 0, gkeys.join(''), signCallback)
  console.log('txId: ' + txId)
  let txResponse = await pclib.waitForTransactionResponse(txId)
  const appId = pclib.appIdFromCreateAppResponse(txResponse)
  console.log('Deployment App Id: %d', appId)
  pclib.setAppId('vaaProcessor', appId)

  console.log('Creating Pricekeeper V2...')
  txId = await pclib.createPricekeeperApp(fromAddress, appId, signCallback)
  console.log('txId: ' + txId)
  txResponse = await pclib.waitForTransactionResponse(txId)
  const pkAppId = pclib.appIdFromCreateAppResponse(txResponse)
  console.log('Deployment App Id: %d', pkAppId)
  pclib.setAppId('pricekeeper', pkAppId)

  console.log('Setting VAA Processor authid parameter...')
  txId = await pclib.setAuthorizedAppId(fromAddress, pkAppId, signCallback)
  console.log('txId: ' + txId)
  txResponse = await pclib.waitForTransactionResponse(txId)

  console.log('Compiling verify VAA stateless code...')
  out = spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py'])
  console.log(out.output.toString())

  spawnSync('python', ['teal/wormhole/pyteal/vaa-verify.py', appId])
  const program = fs.readFileSync('teal/wormhole/build/vaa-verify.teal', 'utf8')
  const compiledVerifyProgram = await pclib.compileProgram(program)
  console.log('Stateless program address: ', compiledVerifyProgram.hash)

  console.log('Setting VAA Processor stateless code...')
  const txid = await pclib.setVAAVerifyProgramHash(fromAddress, compiledVerifyProgram.hash, signCallback)
  console.log('txId: ' + txId)
  await pclib.waitForTransactionResponse(txid)

  const dt = Date.now().toString()
  const resultsFileName = 'DEPLOY-' + dt
  const binaryFileName = 'VAA-VERIFY-' + dt + '.BIN'

  console.log(`Writing deployment results file ${resultsFileName}...`)
  fs.writeFileSync(resultsFileName, `vaaProcessorAppId: ${appId}\npriceKeeperV2AppId: ${pkAppId}\nvaaVerifyProgramHash: '${compiledVerifyProgram.hash}'`)

  console.log(`Writing stateless code binary file ${binaryFileName}...`)
  fs.writeFileSync(binaryFileName, compiledVerifyProgram.bytes)
}

(async () => {
  console.log('\nPricecaster v2 Apps Deployment Tool')

  if (process.argv.length !== 7) {
    console.log('Usage: deploy <glistfile> <from> <network>\n')
    console.log('where:\n')
    console.log('glistfile              File containing the initial list of guardians')
    console.log('gexptime               Guardian set expiration time')
    console.log('from                   Deployer account')
    console.log('network                Testnet, betanet or mainnet')
    console.log('keyfile                Secret file containing signing key mnemonic')
    console.log('\n- File must contain one guardian key per line, formatted in hex, without hex prefix.')
    console.log('\n- Deployment process will generate one DEPLOY-xxxx file with application Ids, stateless hash, and')
    console.log('  a VAA-VERIFY-XXXX.bin with stateless compiled bytes, to use with backend configuration')
    exit(0)
  }

  const listfile = process.argv[2]
  const gexpTime = process.argv[3]
  const fromAddress = process.argv[4]
  const network = process.argv[5]
  const keyfile = process.argv[6]

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
  globalMnemo = fs.readFileSync(keyfile).toString()
  try {
    await startOp(algodClient, fromAddress, gexpTime, gkeys)
  } catch (e) {
    console.error('(!) Deployment Failed: ' + e.toString())
  }
  console.log('Bye.')
  exit(0)
})()
