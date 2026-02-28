import {
  Connection,
  Keypair,
  PublicKey,
  sendAndConfirmTransaction,
  SystemProgram,
  SYSVAR_CLOCK_PUBKEY,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";
import * as bitcoin from "bitcoinjs-lib";
import {
  fetchManagerSignatures,
  loadManagerKeys,
  type ManagerSignaturesResponse,
} from "./manager";
import { buildRedeemScript, dogecoinTestnet } from "./redeem-script";
import {
  broadcastTx,
  DOGECOIN_CHAIN_ID,
  explorerTxUrl,
  FEE,
  fetchUtxos,
  KOINU_PER_DOGE,
} from "./shared";
import {
  ADDRESS_TYPE_P2PKH,
  ADDRESS_TYPE_P2SH,
  decodeUnlockPayload,
  encodeUnlockPayload,
} from "./vaa";

// Wormhole constants
const WORMHOLE_CHAIN_ID_SOLANA = 1;
const WORMHOLE_CHAIN_ID_DOGECOIN = 65;
const SOLANA_DEVNET_RPC = "https://api.devnet.solana.com";
const GUARDIAN_RPC = "http://136.119.196.246/";

// Program addresses
const WORMHOLE_CORE_BRIDGE_DEVNET = new PublicKey(
  "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
);
const POST_MESSAGE_SHIM = new PublicKey(
  "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
);

// Delegated manager set index (placeholder)
const DELEGATED_MANAGER_SET = 1;

// Post message shim instruction discriminator
const POST_MESSAGE_DISCRIMINATOR = Buffer.from([
  214, 50, 100, 209, 38, 34, 7, 76,
]);

// Finality enum
const FINALITY_CONFIRMED = 0;
const FINALITY_FINALIZED = 1;

// Load Solana keypair
async function loadSolanaKeypair(): Promise<Keypair> {
  const keypairJson = await Bun.file("solana-devnet-keypair.json").json();
  return Keypair.fromSecretKey(new Uint8Array(keypairJson.privateKeyArray));
}

// Derive PDAs for the post message shim
function deriveShimAccounts(emitter: PublicKey) {
  // Bridge config PDA (from Wormhole Core Bridge)
  const [bridge] = PublicKey.findProgramAddressSync(
    [Buffer.from("Bridge")],
    WORMHOLE_CORE_BRIDGE_DEVNET,
  );

  // Message PDA (from Post Message Shim, seeded by emitter)
  const [message] = PublicKey.findProgramAddressSync(
    [emitter.toBuffer()],
    POST_MESSAGE_SHIM,
  );

  // Sequence PDA (from Wormhole Core Bridge)
  const [sequence] = PublicKey.findProgramAddressSync(
    [Buffer.from("Sequence"), emitter.toBuffer()],
    WORMHOLE_CORE_BRIDGE_DEVNET,
  );

  // Fee collector PDA (from Wormhole Core Bridge)
  const [feeCollector] = PublicKey.findProgramAddressSync(
    [Buffer.from("fee_collector")],
    WORMHOLE_CORE_BRIDGE_DEVNET,
  );

  // Event authority PDA (from Post Message Shim)
  const [eventAuthority] = PublicKey.findProgramAddressSync(
    [Buffer.from("__event_authority")],
    POST_MESSAGE_SHIM,
  );

  return { bridge, message, sequence, feeCollector, eventAuthority };
}

// Build post message instruction
function buildPostMessageInstruction(
  payer: PublicKey,
  emitter: PublicKey,
  nonce: number,
  payload: Buffer,
  finality: number = FINALITY_FINALIZED,
): TransactionInstruction {
  const accounts = deriveShimAccounts(emitter);

  // Build instruction data: discriminator + nonce (u32 LE) + finality (u8) + payload (borsh bytes)
  const nonceBuffer = Buffer.alloc(4);
  nonceBuffer.writeUInt32LE(nonce);

  // Borsh-encode the payload (4-byte length prefix LE + data)
  const payloadLenBuffer = Buffer.alloc(4);
  payloadLenBuffer.writeUInt32LE(payload.length);

  const instructionData = Buffer.concat([
    new Uint8Array(POST_MESSAGE_DISCRIMINATOR),
    new Uint8Array(nonceBuffer),
    new Uint8Array([finality]), // Finality enum as u8
    new Uint8Array(payloadLenBuffer),
    new Uint8Array(payload),
  ]);

  return new TransactionInstruction({
    keys: [
      { pubkey: accounts.bridge, isSigner: false, isWritable: true },
      { pubkey: accounts.message, isSigner: false, isWritable: true },
      { pubkey: emitter, isSigner: true, isWritable: false },
      { pubkey: accounts.sequence, isSigner: false, isWritable: true },
      { pubkey: payer, isSigner: true, isWritable: true },
      { pubkey: accounts.feeCollector, isSigner: false, isWritable: true },
      { pubkey: SYSVAR_CLOCK_PUBKEY, isSigner: false, isWritable: false },
      { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
      {
        pubkey: WORMHOLE_CORE_BRIDGE_DEVNET,
        isSigner: false,
        isWritable: false,
      },
      { pubkey: accounts.eventAuthority, isSigner: false, isWritable: false },
      { pubkey: POST_MESSAGE_SHIM, isSigner: false, isWritable: false },
    ],
    programId: POST_MESSAGE_SHIM,
    data: instructionData,
  });
}

// Post message using the shim
async function postWormholeMessage(
  connection: Connection,
  payer: Keypair,
  payload: Buffer,
  nonce: number = 0,
): Promise<{ sequence: bigint; emitterAddress: string; signature: string }> {
  const emitter = payer.publicKey;
  const accounts = deriveShimAccounts(emitter);

  const instruction = buildPostMessageInstruction(
    payer.publicKey,
    emitter,
    nonce,
    payload,
    FINALITY_CONFIRMED,
  );

  const tx = new Transaction().add(
    SystemProgram.transfer({
      fromPubkey: payer.publicKey,
      toPubkey: accounts.feeCollector,
      lamports: 100, // TODO: dynamically read bridge config
    }),
    instruction,
  );

  console.log("Posting Wormhole message via shim...");
  console.log("  Emitter:", emitter.toBase58());
  console.log("  Message PDA:", accounts.message.toBase58());
  console.log("  Sequence PDA:", accounts.sequence.toBase58());

  const signature = await sendAndConfirmTransaction(connection, tx, [payer], {
    commitment: "confirmed",
  });
  console.log("Transaction signature:", signature);

  // TODO: read the sequence from the shim log
  // Fetch sequence from the sequence account
  const sequenceAccountInfo = await connection.getAccountInfo(
    accounts.sequence,
  );
  let sequence = BigInt(0);
  if (sequenceAccountInfo && sequenceAccountInfo.data.length >= 8) {
    // Sequence is stored as u64 LE at offset 0
    sequence = sequenceAccountInfo.data.readBigUInt64LE(0) - 1n;
  }

  // Emitter address as 32-byte hex (left-padded)
  const emitterAddress = emitter.toBuffer().toString("hex");

  return { sequence, emitterAddress, signature };
}

// ============================================================================
// Main Script
// ============================================================================

console.log("=== Wormhole TESTNET Withdraw ===\n");

// Load Solana keypair
const solanaKeypair = await loadSolanaKeypair();
console.log("Solana address:", solanaKeypair.publicKey.toBase58());

// Load manager keys
const {
  mThreshold,
  nTotal,
  pubkeys: managerPubkeys,
} = await loadManagerKeys(DOGECOIN_CHAIN_ID);

console.log("\nManager keys:");
for (let i = 0; i < managerPubkeys.length; i++) {
  console.log(`  ${i}: ${managerPubkeys[i]!.toString("hex")}`);
}

// Read deposit info
let depositInfo: {
  txid: string;
  amount: number;
  p2shAddress: string;
  redeemScript: string;
  senderAddress: string;
  timestamp: string;
};

try {
  depositInfo = await Bun.file("deposit-info.json").json();
  console.log("\nLoaded deposit info from deposit-info.json");
  console.log("  Deposit TXID:", depositInfo.txid);
  console.log("  Amount:", depositInfo.amount / KOINU_PER_DOGE, "DOGE");
} catch {
  console.error("Error: deposit-info.json not found. Run deposit.ts first.");
  process.exit(1);
}

// Get the recipient address (Solana pubkey as 32-byte hex)
const recipientAddress = solanaKeypair.publicKey.toBuffer().toString("hex");

// Fetch UTXOs to determine inputs
console.log("\nFetching UTXOs from P2SH address...");
const utxos = await fetchUtxos(depositInfo.p2shAddress);
console.log(`Found ${utxos.length} UTXOs`);

if (utxos.length === 0) {
  console.error("No UTXOs available in P2SH address.");
  process.exit(1);
}

// Calculate total available
const totalAvailable = utxos.reduce(
  (sum: number, utxo: any) => sum + utxo.value,
  0,
);
console.log(`Total available: ${totalAvailable / KOINU_PER_DOGE} DOGE`);

const amountToWithdraw = totalAvailable - FEE;
if (amountToWithdraw <= 0) {
  console.error("Insufficient funds to cover fee.");
  process.exit(1);
}

// Decode the destination address to get the pubkey hash
const destAddressDecoded = bitcoin.address.fromBase58Check(
  depositInfo.senderAddress,
);
const destPubKeyHash = Buffer.from(destAddressDecoded.hash).toString("hex");

// Build VAA payload
console.log("\n=== Building VAA Payload ===");
const payload = encodeUnlockPayload({
  destinationChain: WORMHOLE_CHAIN_ID_DOGECOIN,
  delegatedManagerSet: DELEGATED_MANAGER_SET,
  inputs: utxos.map((utxo: any) => ({
    originalRecipientAddress: recipientAddress,
    transactionId: utxo.txid,
    vout: utxo.vout,
  })),
  outputs: [
    {
      amount: BigInt(amountToWithdraw),
      addressType: ADDRESS_TYPE_P2PKH,
      address: destPubKeyHash,
    },
  ],
});

console.log("Payload length:", payload.length, "bytes");
console.log("Payload (hex):", payload.toString("hex"));

// Post message to Wormhole via shim
const connection = new Connection(SOLANA_DEVNET_RPC, "confirmed");
console.log("\n=== Posting Wormhole Message via Shim ===");

let sequence: bigint;
let emitterAddress: string;
let signature: string;
try {
  const result = await postWormholeMessage(
    connection,
    solanaKeypair,
    payload,
    Date.now() % 2 ** 32, // Use timestamp as nonce
  );
  sequence = result.sequence;
  emitterAddress = result.emitterAddress;
  signature = result.signature;
  console.log("\nMessage posted successfully!");
  console.log("  Emitter:", emitterAddress);
  console.log("  Sequence:", sequence.toString());
} catch (error) {
  console.error("Failed to post Wormhole message:", error);
  process.exit(1);
}

// Fetch signatures from Guardian Manager
console.log("\n=== Fetching Signatures from Guardian Manager ===");
let managerSignatures: ManagerSignaturesResponse | null = null;

try {
  managerSignatures = await fetchManagerSignatures(
    GUARDIAN_RPC,
    WORMHOLE_CHAIN_ID_SOLANA,
    emitterAddress,
    sequence,
    30, // max retries
    2000, // retry delay (2 seconds)
  );

  console.log("\nManager Signatures Response:");
  console.log("  VAA Hash:", managerSignatures.vaaHash);
  console.log("  VAA ID:", managerSignatures.vaaId);
  console.log("  Destination Chain:", managerSignatures.destinationChain);
  console.log("  Manager Set Index:", managerSignatures.managerSetIndex);
  console.log("  Required:", managerSignatures.required);
  console.log("  Total:", managerSignatures.total);
  console.log("  Signatures:");
  for (const sig of managerSignatures.signatures) {
    console.log(
      `    Signer ${sig.signerIndex}: ${sig.signatures[0]?.slice(0, 20)}...`,
    );
  }
} catch (error) {
  console.error("Failed to fetch manager signatures:", error);
  console.log("\nContinuing without signatures for testing...");
}

// Use the original payload we built (we don't need to parse it from VAA)
const vaaPayload = payload;

// Parse the VAA payload
console.log("\n=== Parsing VAA Payload ===");
const decodedPayload = decodeUnlockPayload(vaaPayload);
console.log("Destination Chain:", decodedPayload.destinationChain);
console.log("Delegated Manager Set:", decodedPayload.delegatedManagerSet);
console.log("Inputs:", decodedPayload.inputs.length);
for (const input of decodedPayload.inputs) {
  console.log("  - Recipient:", input.originalRecipientAddress);
  console.log("    TxID:", input.transactionId);
  console.log("    Vout:", input.vout);
}
console.log("Outputs:", decodedPayload.outputs.length);
for (const output of decodedPayload.outputs) {
  console.log("  - Amount:", output.amount.toString(), "koinu");
  console.log("    Type:", output.addressType === 0 ? "P2PKH" : "P2SH");
  console.log("    Address:", output.address);
}

// Build redeem script from VAA data
console.log("\n=== Building Redeem Script from VAA ===");
const vaaRecipientAddress = decodedPayload.inputs[0]?.originalRecipientAddress;
if (!vaaRecipientAddress) {
  console.error("No inputs in VAA payload");
  process.exit(1);
}

const redeemScript = buildRedeemScript({
  emitterChain: WORMHOLE_CHAIN_ID_SOLANA,
  emitterContract: vaaRecipientAddress,
  recipientAddress: vaaRecipientAddress,
  managerPubkeys,
  mThreshold,
  nTotal,
});

console.log("Redeem script length:", redeemScript.length, "bytes");
console.log("Redeem script (hex):", redeemScript.toString("hex"));

// Note: The redeem script won't match because deposit used different emitter chain
// For devnet testing, we use the deposit's redeem script
console.log("\nNote: Using deposit redeem script for signing...");
const depositRedeemScript = Buffer.from(depositInfo.redeemScript, "hex");

// Generate P2SH address from redeem script
const p2sh = bitcoin.payments.p2sh({
  redeem: { output: new Uint8Array(depositRedeemScript) },
  network: dogecoinTestnet,
});

console.log("P2SH Address:", p2sh.address);

// Build the transaction from VAA data
console.log("\n=== Building Dogecoin Transaction ===");
const tx = new bitcoin.Transaction();
tx.version = 1;

// Add inputs from VAA
for (const input of decodedPayload.inputs) {
  tx.addInput(
    new Uint8Array(Buffer.from(input.transactionId, "hex").reverse()),
    input.vout,
  );
}

// Add outputs from VAA
for (const output of decodedPayload.outputs) {
  let outputScript: Uint8Array;
  if (output.addressType === ADDRESS_TYPE_P2PKH) {
    // P2PKH: OP_DUP OP_HASH160 <20-byte hash> OP_EQUALVERIFY OP_CHECKSIG
    outputScript = bitcoin.script.compile([
      bitcoin.opcodes.OP_DUP,
      bitcoin.opcodes.OP_HASH160,
      new Uint8Array(Buffer.from(output.address, "hex")),
      bitcoin.opcodes.OP_EQUALVERIFY,
      bitcoin.opcodes.OP_CHECKSIG,
    ]);
  } else if (output.addressType === ADDRESS_TYPE_P2SH) {
    // P2SH: OP_HASH160 <20-byte hash> OP_EQUAL
    outputScript = bitcoin.script.compile([
      bitcoin.opcodes.OP_HASH160,
      new Uint8Array(Buffer.from(output.address, "hex")),
      bitcoin.opcodes.OP_EQUAL,
    ]);
  } else {
    throw new Error(`Unknown address type: ${output.addressType}`);
  }
  tx.addOutput(outputScript, output.amount);
}

console.log("\nTransaction built (unsigned)");
console.log("  Inputs:", tx.ins.length);
console.log("  Outputs:", tx.outs.length);

// Sign with manager signatures
if (managerSignatures && managerSignatures.isComplete) {
  console.log(
    `\n=== Signing with ${managerSignatures.required} of ${managerSignatures.total} manager signatures ===`,
  );

  // Sort signatures by signer index to match the order in the redeem script
  const sortedSigs = [...managerSignatures.signatures].sort(
    (a, b) => a.signerIndex - b.signerIndex,
  );

  // Take only the required number of signatures
  const requiredSigs = sortedSigs.slice(0, managerSignatures.required);

  console.log(
    "Using signatures from signers:",
    requiredSigs.map((s) => s.signerIndex).join(", "),
  );

  // Build the scriptSig for each input
  for (
    let inputIndex = 0;
    inputIndex < decodedPayload.inputs.length;
    inputIndex++
  ) {
    // Decode base64 signatures and convert to Uint8Array
    const signatures: Uint8Array[] = requiredSigs.map((sig) => {
      // Each signer may have multiple signatures (one per input), use the one for this input
      const sigBase64 = sig.signatures[inputIndex] ?? sig.signatures[0];
      if (!sigBase64) {
        throw new Error(
          `No signature found for signer ${sig.signerIndex} input ${inputIndex}`,
        );
      }
      return new Uint8Array(Buffer.from(sigBase64, "base64"));
    });

    const scriptSig = bitcoin.script.compile([
      bitcoin.opcodes.OP_0, // Required for CHECKMULTISIG bug
      ...signatures,
      new Uint8Array(depositRedeemScript),
    ]);

    tx.setInputScript(inputIndex, scriptSig);
    console.log(
      `  Input ${inputIndex}: Applied ${signatures.length} signatures`,
    );
  }

  // Serialize the transaction
  const txHex = tx.toHex();

  console.log("\nTransaction hex:", txHex);
  console.log("Transaction ID:", tx.getId());

  // Broadcast the transaction
  console.log("\nBroadcasting transaction...");
  try {
    const txid = await broadcastTx(txHex);
    console.log("Transaction broadcast successfully!");
    console.log("TXID:", txid);
    console.log(`\nView on explorer: ${explorerTxUrl(txid)}`);

    // Save withdraw info
    const withdrawInfo = {
      solTxHash: signature,
      sequence: sequence.toString(),
      emitterAddress,
      emitterChain: WORMHOLE_CHAIN_ID_SOLANA,
      payload: payload.toString("hex"),
      dogeTxId: txid,
      amount: Number(decodedPayload.outputs[0]?.amount ?? 0),
      destinationAddress: depositInfo.senderAddress,
      p2shAddress: p2sh.address,
      managerSetIndex: managerSignatures.managerSetIndex,
      signersUsed: requiredSigs.map((s) => s.signerIndex),
      timestamp: new Date().toISOString(),
    };
    await Bun.write(
      "withdraw-devnet-info.json",
      JSON.stringify(withdrawInfo, null, 2),
    );
    console.log("\nWithdraw info saved to withdraw-devnet-info.json");
  } catch (error) {
    console.error("Broadcast failed:", error);

    // Save failed withdraw info for debugging
    const withdrawInfo = {
      solTxHash: signature,
      sequence: sequence.toString(),
      emitterAddress,
      emitterChain: WORMHOLE_CHAIN_ID_SOLANA,
      payload: payload.toString("hex"),
      dogeTxHex: txHex,
      dogeTxId: tx.getId(),
      amount: Number(decodedPayload.outputs[0]?.amount ?? 0),
      destinationAddress: depositInfo.senderAddress,
      p2shAddress: p2sh.address,
      managerSetIndex: managerSignatures.managerSetIndex,
      signersUsed: requiredSigs.map((s) => s.signerIndex),
      error: String(error),
      timestamp: new Date().toISOString(),
    };
    await Bun.write(
      "withdraw-devnet-info.json",
      JSON.stringify(withdrawInfo, null, 2),
    );
    console.log("\nFailed withdraw info saved to withdraw-devnet-info.json");
  }
} else {
  console.log("\n=== No Manager Signatures Available ===");
  console.log("Cannot sign transaction without manager signatures.");
  console.log("Transaction ID (unsigned):", tx.getId());

  // Save unsigned withdraw info
  const withdrawInfo = {
    solTxHash: signature,
    sequence: sequence.toString(),
    emitterAddress,
    emitterChain: WORMHOLE_CHAIN_ID_SOLANA,
    payload: payload.toString("hex"),
    dogeTxId: tx.getId(),
    amount: Number(decodedPayload.outputs[0]?.amount ?? 0),
    destinationAddress: depositInfo.senderAddress,
    p2shAddress: p2sh.address,
    signed: false,
    timestamp: new Date().toISOString(),
  };
  await Bun.write(
    "withdraw-devnet-info.json",
    JSON.stringify(withdrawInfo, null, 2),
  );
  console.log("\nUnsigned withdraw info saved to withdraw-devnet-info.json");
}
