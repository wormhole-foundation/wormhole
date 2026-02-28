import * as bitcoin from "bitcoinjs-lib";
import { loadManagerKeys } from "./manager";
import {
  ECPair,
  EMITTER_CHAIN,
  buildRedeemScript,
  dogecoinTestnet,
  loadSolanaEmitterContract,
} from "./redeem-script";
import {
  DOGECOIN_CHAIN_ID,
  FEE,
  KOINU_PER_DOGE,
  broadcastTx,
  explorerTxUrl,
  fetchRawTx,
  fetchUtxos,
} from "./shared";

// Load emitter contract and manager keys
const emitterContract = await loadSolanaEmitterContract();

const {
  mThreshold,
  nTotal,
  pubkeys: managerPubkeys,
} = await loadManagerKeys(DOGECOIN_CHAIN_ID);

console.log("\nManager keys:");
for (let i = 0; i < managerPubkeys.length; i++) {
  console.log(`  ${i}: ${managerPubkeys[i]!.toString("hex")}`);
}

// Build the redeem script (using Solana pubkey as recipient_address)
const redeemScript = buildRedeemScript({
  emitterChain: EMITTER_CHAIN,
  emitterContract,
  recipientAddress: emitterContract, // Use Solana public key as recipient_address
  managerPubkeys,
  mThreshold,
  nTotal,
});

console.log("\nRedeem script length:", redeemScript.length, "bytes");
console.log("Redeem script (hex):", redeemScript.toString("hex"));

// Generate P2SH address from redeem script
const p2sh = bitcoin.payments.p2sh({
  redeem: { output: new Uint8Array(redeemScript) },
  network: dogecoinTestnet,
});

console.log("\nP2SH Address:", p2sh.address);
console.log(
  "Script Hash:",
  p2sh.hash ? Buffer.from(p2sh.hash).toString("hex") : undefined,
);

// Read sender's keypair
const senderKeypair = await Bun.file("dogecoin-testnet-keypair.json").json();
const senderKeyPair = ECPair.fromWIF(
  senderKeypair.privateKeyWIF,
  dogecoinTestnet,
);
const senderAddress = senderKeypair.address;

console.log("\nSender address:", senderAddress);

// Amount to send (in koinu) - 1 DOGE = 100,000,000 koinu
const AMOUNT_TO_SEND = KOINU_PER_DOGE; // 1 DOGE

console.log("\nFetching UTXOs...");
const utxos = await fetchUtxos(senderAddress);
console.log(`Found ${utxos.length} UTXOs`);

if (utxos.length === 0) {
  console.error("No UTXOs available. Please fund the sender address first.");
  console.log(`Get testnet coins from: https://faucet.doge.toys/`);
  process.exit(1);
}

// Calculate total available
const totalAvailable = utxos.reduce(
  (sum: number, utxo: any) => sum + utxo.value,
  0,
);
console.log(`Total available: ${totalAvailable / KOINU_PER_DOGE} DOGE`);

if (totalAvailable < AMOUNT_TO_SEND + FEE) {
  console.error(
    `Insufficient funds. Need ${(AMOUNT_TO_SEND + FEE) / KOINU_PER_DOGE} DOGE`,
  );
  process.exit(1);
}

// Build transaction
const psbt = new bitcoin.Psbt({ network: dogecoinTestnet });

// Add inputs
let inputSum = 0;
for (const utxo of utxos) {
  // Fetch raw transaction for the UTXO
  const rawTxHex = await fetchRawTx(utxo.txid);

  psbt.addInput({
    hash: utxo.txid,
    index: utxo.vout,
    nonWitnessUtxo: new Uint8Array(Buffer.from(rawTxHex, "hex")),
  });

  inputSum += utxo.value;
  if (inputSum >= AMOUNT_TO_SEND + FEE) break;
}

// Add output to P2SH address
psbt.addOutput({
  address: p2sh.address!,
  value: BigInt(AMOUNT_TO_SEND),
});

// Add change output if needed
const change = inputSum - AMOUNT_TO_SEND - FEE;
if (change > 0) {
  psbt.addOutput({
    address: senderAddress,
    value: BigInt(change),
  });
}

// Sign all inputs
psbt.signAllInputs(senderKeyPair);
psbt.finalizeAllInputs();

// Extract and broadcast
const tx = psbt.extractTransaction();
const txHex = tx.toHex();

console.log("\nTransaction hex:", txHex);
console.log("Transaction ID:", tx.getId());

console.log("\nBroadcasting transaction...");
try {
  const txid = await broadcastTx(txHex);
  console.log("Transaction broadcast successfully!");
  console.log("TXID:", txid);
  console.log(`\nView on explorer: ${explorerTxUrl(txid)}`);

  // Save deposit info
  const depositInfo = {
    txid: tx.getId(),
    amount: AMOUNT_TO_SEND,
    p2shAddress: p2sh.address,
    redeemScript: redeemScript.toString("hex"),
    senderAddress,
    timestamp: new Date().toISOString(),
  };
  await Bun.write("deposit-info.json", JSON.stringify(depositInfo, null, 2));
  console.log("\nDeposit info saved to deposit-info.json");
} catch (error) {
  console.error("Broadcast failed:", error);
}
