#!/usr/bin/env node
// Deploy Wormhole Core (Implementation, Setup, Wormhole proxy) to Tron.
// Talks to Tron's native HTTP API directly. Requires Node 18+ and `make build`
// to have produced artifacts in build-forge/.

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const { ethers } = require('ethers');

const FULL_HOST = (process.env.TRON_FULL_HOST || 'https://nile.trongrid.io').replace(/\/+$/, '');
const API_KEY = process.env.TRON_API_KEY || '';
const PRIVATE_KEY = (process.env.TRON_PRIVATE_KEY || process.env.MNEMONIC || '').replace(/^0x/, '');
const FEE_LIMIT_SUN = Number(process.env.TRON_FEE_LIMIT_SUN || 5_000_000_000); // 5000 TRX
const ORIGIN_ENERGY_LIMIT = Number(process.env.TRON_ORIGIN_ENERGY_LIMIT || 10_000_000);
const USER_FEE_PERCENT = Number(process.env.TRON_USER_FEE_PERCENT || 100);

// .env is shell-sourced, which strips the double quotes around array elements.
// Re-quote any bare 0x-hex strings so JSON.parse can handle either form.
const INIT_SIGNERS = JSON.parse(
  (process.env.INIT_SIGNERS || '[]')
    .replace(/"/g, '')
    .replace(/0x[a-fA-F0-9]+/g, (m) => `"${m}"`),
);
const INIT_CHAIN_ID = Number(process.env.INIT_CHAIN_ID);
const INIT_GOV_CHAIN_ID = Number(process.env.INIT_GOV_CHAIN_ID);
const INIT_GOV_CONTRACT = process.env.INIT_GOV_CONTRACT;
const INIT_EVM_CHAIN_ID = process.env.INIT_EVM_CHAIN_ID;

function die(msg) { console.error('error:', msg); process.exit(1); }
if (!PRIVATE_KEY) die('TRON_PRIVATE_KEY (or MNEMONIC) is required');
if (!INIT_SIGNERS.length) die('INIT_SIGNERS is required');
if (!INIT_CHAIN_ID) die('INIT_CHAIN_ID is required');
if (!INIT_GOV_CHAIN_ID) die('INIT_GOV_CHAIN_ID is required');
if (!INIT_GOV_CONTRACT) die('INIT_GOV_CONTRACT is required');
if (!INIT_EVM_CHAIN_ID) die('INIT_EVM_CHAIN_ID is required');

const BASE58 = '123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz';

function base58Encode(buf) {
  let n = buf.length === 0 ? 0n : BigInt('0x' + buf.toString('hex'));
  let out = '';
  while (n > 0n) { out = BASE58[Number(n % 58n)] + out; n /= 58n; }
  for (const b of buf) { if (b === 0) out = '1' + out; else break; }
  return out;
}

function evmHexToTronB58(evmHex) {
  const a20 = Buffer.from(evmHex.replace(/^0x/, ''), 'hex');
  const a21 = Buffer.concat([Buffer.from([0x41]), a20]);
  const c1 = crypto.createHash('sha256').update(a21).digest();
  const c2 = crypto.createHash('sha256').update(c1).digest();
  return base58Encode(Buffer.concat([a21, c2.slice(0, 4)]));
}

function tronHexToEvm(tronHex) {
  const h = tronHex.replace(/^0x/, '');
  if (h.length !== 42 || !h.startsWith('41')) throw new Error(`bad tron hex: ${tronHex}`);
  return '0x' + h.slice(2).toLowerCase();
}

const wallet = new ethers.Wallet('0x' + PRIVATE_KEY);
const evmAddr = wallet.address.toLowerCase();
const ownerHex = '41' + evmAddr.slice(2);
const ownerB58 = evmHexToTronB58(evmAddr);

console.log('Tron full host:', FULL_HOST);
console.log('Deployer base58:', ownerB58);
console.log('Deployer 41-hex:', ownerHex);
console.log('Deployer EVM hex:', evmAddr);
console.log('INIT_SIGNERS:', INIT_SIGNERS);
console.log('INIT_CHAIN_ID:', INIT_CHAIN_ID);
console.log('INIT_GOV_CHAIN_ID:', INIT_GOV_CHAIN_ID);
console.log('INIT_GOV_CONTRACT:', INIT_GOV_CONTRACT);
console.log('INIT_EVM_CHAIN_ID:', INIT_EVM_CHAIN_ID);

async function rpc(p, body) {
  const headers = { 'Content-Type': 'application/json' };
  if (API_KEY) headers['TRON-PRO-API-KEY'] = API_KEY;
  const r = await fetch(`${FULL_HOST}${p}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body || {}),
  });
  if (!r.ok) throw new Error(`${FULL_HOST}${p} ${r.status}: ${await r.text()}`);
  return r.json();
}

function signTxId(txid) {
  const sk = new ethers.utils.SigningKey('0x' + PRIVATE_KEY);
  const sig = sk.signDigest('0x' + txid);
  return sig.r.slice(2) + sig.s.slice(2) + sig.recoveryParam.toString(16).padStart(2, '0');
}

async function waitForTx(txid, timeoutMs = 180_000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const info = await rpc('/wallet/gettransactioninfobyid', { value: txid });
    if (info && info.id) return info;
    await new Promise(r => setTimeout(r, 3000));
  }
  throw new Error(`timeout waiting for tx ${txid}`);
}

async function deploy(name, abi, bytecodeHex, ctorTypes, ctorArgs) {
  let data = bytecodeHex.replace(/^0x/, '');
  if (ctorTypes && ctorTypes.length) {
    data += ethers.utils.defaultAbiCoder.encode(ctorTypes, ctorArgs).slice(2);
  }
  console.log(`\nDeploying ${name} (${data.length / 2} bytes)...`);

  const created = await rpc('/wallet/deploycontract', {
    owner_address: ownerHex,
    abi: JSON.stringify(abi),
    bytecode: data,
    name,
    consume_user_resource_percent: USER_FEE_PERCENT,
    origin_energy_limit: ORIGIN_ENERGY_LIMIT,
    fee_limit: FEE_LIMIT_SUN,
    call_value: 0,
    visible: false,
  });
  if (created.Error) throw new Error(`create error: ${created.Error}`);
  if (!created.txID) throw new Error(`no txID: ${JSON.stringify(created)}`);

  created.signature = [signTxId(created.txID)];
  const b = await rpc('/wallet/broadcasttransaction', created);
  if (!b.result) {
    const m = b.message ? Buffer.from(b.message, 'hex').toString() : JSON.stringify(b);
    throw new Error(`broadcast failed: ${m}`);
  }
  console.log(`  txid: ${created.txID}`);

  const info = await waitForTx(created.txID);
  const result = info.receipt && info.receipt.result;
  if (result && result !== 'SUCCESS') {
    const m = info.resMessage ? Buffer.from(info.resMessage, 'hex').toString() : '';
    throw new Error(`tx failed: ${result} ${m}`);
  }
  if (!info.contract_address) {
    throw new Error(`no contract_address: ${JSON.stringify(info)}`);
  }
  const evm = tronHexToEvm(info.contract_address);
  const b58 = evmHexToTronB58(evm);
  console.log(`  ${b58} (${evm})`);
  console.log(`  energy: ${(info.receipt && info.receipt.energy_usage_total) || 0}`);
  return { tronHex: info.contract_address, b58, evm };
}

function loadArtifact(file, contract) {
  const p = path.join(__dirname, '..', 'build-forge', file, `${contract}.json`);
  const j = JSON.parse(fs.readFileSync(p, 'utf8'));
  const bc = j.bytecode && (j.bytecode.object || j.bytecode);
  if (!bc) throw new Error(`no bytecode in ${p}`);
  return { abi: j.abi, bytecode: bc };
}

(async () => {
  const acct = await rpc('/wallet/getaccount', { address: ownerHex });
  const balance = acct.balance || 0;
  console.log(`Deployer balance: ${balance / 1e6} TRX`);
  if (balance < 2_000_000_000) {
    console.warn('WARNING: balance < 2000 TRX. Implementation alone may need >1500 TRX of energy.');
  }

  const impl = loadArtifact('Implementation.sol', 'Implementation');
  const setup = loadArtifact('Setup.sol', 'Setup');
  const wormhole = loadArtifact('Wormhole.sol', 'Wormhole');

  const implAddr = await deploy('Implementation', impl.abi, impl.bytecode);
  const setupAddr = await deploy('Setup', setup.abi, setup.bytecode);

  const setupIface = new ethers.utils.Interface(setup.abi);
  const initData = setupIface.encodeFunctionData('setup', [
    implAddr.evm,
    INIT_SIGNERS,
    INIT_CHAIN_ID,
    INIT_GOV_CHAIN_ID,
    INIT_GOV_CONTRACT,
    INIT_EVM_CHAIN_ID,
  ]);

  const wormholeAddr = await deploy(
    'Wormhole',
    wormhole.abi,
    wormhole.bytecode,
    ['address', 'bytes'],
    [setupAddr.evm, initData],
  );

  console.log('\n-- Wormhole Core Addresses --');
  console.log(`Implementation: ${implAddr.b58}  (${implAddr.evm})`);
  console.log(`Setup:          ${setupAddr.b58}  (${setupAddr.evm})`);
  console.log(`Wormhole:       ${wormholeAddr.b58}  (${wormholeAddr.evm})`);

  const outDir = path.join(__dirname, '..', 'broadcast', 'DeployCoreTron');
  fs.mkdirSync(outDir, { recursive: true });
  const outFile = path.join(outDir, `${INIT_EVM_CHAIN_ID}.json`);
  fs.writeFileSync(outFile, JSON.stringify({
    network: FULL_HOST,
    deployer: { b58: ownerB58, evm: evmAddr },
    implementation: implAddr,
    setup: setupAddr,
    wormhole: wormholeAddr,
  }, null, 2));
  console.log(`\nWrote ${outFile}`);
})().catch(e => { console.error(e); process.exit(1); });
