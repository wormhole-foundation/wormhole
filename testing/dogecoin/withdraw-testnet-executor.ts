import { AnchorProvider, BN, Program, Wallet } from "@anchor-lang/core";
import {
  Connection,
  Keypair,
  PublicKey,
  sendAndConfirmTransaction,
  SystemProgram,
  SYSVAR_CLOCK_PUBKEY,
  Transaction,
} from "@solana/web3.js";
import * as bitcoin from "bitcoinjs-lib";
import type { WormholePostMessageShim } from "../../svm/wormhole-core-shims/anchor/idls/wormhole_post_message_shim";
import postMessageShimIdl from "../../svm/wormhole-core-shims/anchor/idls/wormhole_post_message_shim.json";
import type { Executor } from "./idls/executor";
import executorIdl from "./idls/executor.json";
import { FEE, fetchUtxos, KOINU_PER_DOGE } from "./shared";
import { ADDRESS_TYPE_P2PKH, encodeUnlockPayload } from "./vaa";
import { deserialize, serialize } from "binary-layout";
import { toBytes } from "viem";
import {
  RequestPrefix,
  signedQuoteLayout,
} from "@wormhole-foundation/sdk-definitions";
import { requestLayout } from "./requestForExecution";

// Wormhole constants
const WORMHOLE_CHAIN_ID_SOLANA = 1;
const WORMHOLE_CHAIN_ID_DOGECOIN = 65;
const SOLANA_DEVNET_RPC = "https://api.devnet.solana.com";

// Program addresses
const WORMHOLE_CORE_BRIDGE_DEVNET = new PublicKey(
  "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
);
const POST_MESSAGE_SHIM = new PublicKey(
  "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
);

// Executor API
const EXECUTOR_API = "https://executor-testnet.labsapis.com";

// Delegated manager set index
const DELEGATED_MANAGER_SET = 1;

// Load Solana keypair
async function loadSolanaKeypair(): Promise<Keypair> {
  const keypairJson = await Bun.file("solana-devnet-keypair.json").json();
  return Keypair.fromSecretKey(new Uint8Array(keypairJson.privateKeyArray));
}

// Load Solana keypair
const solanaKeypair = await loadSolanaKeypair();
console.log("Solana address:", solanaKeypair.publicKey.toBase58());

const connection = new Connection(SOLANA_DEVNET_RPC, "confirmed");

const provider = new AnchorProvider(connection, new Wallet(solanaKeypair));

// Create Anchor programs
const postMessageShimProgram = new Program<WormholePostMessageShim>(
  postMessageShimIdl as WormholePostMessageShim,
  provider,
);
const executorProgram = new Program<Executor>(
  executorIdl as Executor,
  provider,
);

// Derive PDAs for the post message shim
function deriveShimAccounts(emitter: PublicKey) {
  const [bridge] = PublicKey.findProgramAddressSync(
    [Buffer.from("Bridge")],
    WORMHOLE_CORE_BRIDGE_DEVNET,
  );

  const [message] = PublicKey.findProgramAddressSync(
    [emitter.toBuffer()],
    POST_MESSAGE_SHIM,
  );

  const [sequence] = PublicKey.findProgramAddressSync(
    [Buffer.from("Sequence"), emitter.toBuffer()],
    WORMHOLE_CORE_BRIDGE_DEVNET,
  );

  const [feeCollector] = PublicKey.findProgramAddressSync(
    [Buffer.from("fee_collector")],
    WORMHOLE_CORE_BRIDGE_DEVNET,
  );

  const [eventAuthority] = PublicKey.findProgramAddressSync(
    [Buffer.from("__event_authority")],
    POST_MESSAGE_SHIM,
  );

  return { bridge, message, sequence, feeCollector, eventAuthority };
}

// Build post message instruction using Anchor
function buildPostMessageInstruction(
  payer: PublicKey,
  emitter: PublicKey,
  nonce: number,
  payload: Buffer,
  consistencyLevel: { confirmed: {} } | { finalized: {} } = { confirmed: {} },
) {
  const accounts = deriveShimAccounts(emitter);

  return postMessageShimProgram.methods
    .postMessage(nonce, consistencyLevel, Buffer.from(payload))
    .accountsPartial({
      bridge: accounts.bridge,
      message: accounts.message,
      emitter,
      sequence: accounts.sequence,
      payer,
      feeCollector: accounts.feeCollector,
      clock: SYSVAR_CLOCK_PUBKEY,
      systemProgram: SystemProgram.programId,
      wormholeProgram: WORMHOLE_CORE_BRIDGE_DEVNET,
      eventAuthority: accounts.eventAuthority,
      program: POST_MESSAGE_SHIM,
    })
    .instruction();
}

// ============================================================================
// Executor API Types and Functions
// ============================================================================

interface ExecutorQuote {
  signedQuote: string;
  estimatedCost: string;
}

interface ExecutorStatusTx {
  txHash: string;
  chainId: number;
  blockNumber: string;
  blockTime: string;
  cost: string;
}

interface ExecutorStatus {
  chainId: number;
  estimatedCost: string;
  id: string;
  indexedAt: string;
  status: string;
  txHash: string;
  txs: ExecutorStatusTx[];
}

// Fetch execution status from executor API
async function fetchExecutorStatus(
  txHash: string,
  chainId: number,
): Promise<ExecutorStatus[]> {
  const url = `${EXECUTOR_API}/v0/status/tx`;

  const response = await fetch(url, {
    method: "POST",
    headers: {
      accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      txHash,
      chainId,
    }),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to fetch executor status: ${error}`);
  }

  return response.json();
}

// Poll executor status until completion or timeout
async function pollExecutorStatus(
  txHash: string,
  chainId: number,
  maxAttempts: number = 30,
  intervalMs: number = 10000,
): Promise<ExecutorStatus[]> {
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    console.log(`  Polling status (attempt ${attempt}/${maxAttempts})...`);

    const statuses = await fetchExecutorStatus(txHash, chainId);

    if (statuses.length > 0 && statuses[0]) {
      const status = statuses[0];
      console.log(`    Status: ${status.status}`);

      if (status.txs && status.txs.length > 0) {
        console.log(`    Destination transactions:`);
        for (const tx of status.txs) {
          console.log(`      Chain ${tx.chainId}: ${tx.txHash}`);
          console.log(
            `        Block: ${tx.blockNumber}, Time: ${tx.blockTime}`,
          );
        }
      }

      // Check if execution is complete
      if (status.status === "submitted" || status.status === "completed") {
        if (status.txs && status.txs.length > 0) {
          console.log(`\n  Execution complete!`);
          return statuses;
        }
      }

      // Check for failure states
      if (status.status === "failed" || status.status === "expired") {
        console.log(`\n  Execution ${status.status}.`);
        return statuses;
      }
    } else {
      console.log(`    No status found yet...`);
    }

    if (attempt < maxAttempts) {
      await new Promise((resolve) => setTimeout(resolve, intervalMs));
    }
  }

  console.log(`  Max polling attempts reached.`);
  return fetchExecutorStatus(txHash, chainId);
}

// Fetch quote from executor API
async function fetchExecutorQuote(
  srcChain: number,
  dstChain: number,
  relayInstructions: string = "0x",
): Promise<ExecutorQuote> {
  const url = `${EXECUTOR_API}/v0/quote`;
  console.log(`Fetching executor quote from ${url}...`);

  const response = await fetch(url, {
    method: "POST",
    headers: {
      accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      srcChain,
      dstChain,
      relayInstructions,
    }),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Failed to fetch executor quote: ${error}`);
  }

  return response.json();
}

// Build request_for_execution instruction using Anchor
function buildRequestForExecutionInstruction(
  payer: PublicKey,
  payee: PublicKey,
  amount: bigint,
  dstChain: number,
  refundAddr: PublicKey,
  signedQuoteBytes: Buffer,
  requestBytes: Buffer,
  relayInstructions: Buffer = Buffer.alloc(0),
) {
  // dst_addr ([u8; 32]) - not used for Dogecoin, leave as zeros
  const dstAddr = Array.from(Buffer.alloc(32));

  return executorProgram.methods
    .requestForExecution({
      amount: new BN(amount.toString()),
      dstChain,
      dstAddr,
      refundAddr,
      signedQuoteBytes,
      requestBytes,
      relayInstructions,
    })
    .accountsPartial({
      payer,
      payee,
      systemProgram: SystemProgram.programId,
    })
    .instruction();
}

// ============================================================================
// Main Script
// ============================================================================

console.log("=== Wormhole TESTNET Withdraw via Executor ===\n");

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

// Fetch executor quote
console.log("\n=== Fetching Executor Quote ===");
let quote: ExecutorQuote;
let payee: PublicKey;
try {
  quote = await fetchExecutorQuote(
    WORMHOLE_CHAIN_ID_SOLANA,
    WORMHOLE_CHAIN_ID_DOGECOIN,
    "0x", // Empty relay instructions for Dogecoin
  );
  payee = new PublicKey(
    deserialize(signedQuoteLayout, toBytes(quote.signedQuote)).quote
      .payeeAddress,
  );
} catch (error) {
  console.error("Failed to fetch executor quote:", error);
  process.exit(1);
}

// Build the transaction with both instructions
console.log("\n=== Building Solana Transaction ===");

const emitter = solanaKeypair.publicKey;
const accounts = deriveShimAccounts(emitter);

// Read current sequence to predict the next one
const sequenceAccountInfo = await connection.getAccountInfo(accounts.sequence);
let nextSequence = BigInt(0);
if (sequenceAccountInfo && sequenceAccountInfo.data.length >= 8) {
  nextSequence = sequenceAccountInfo.data.readBigUInt64LE(0);
}
console.log("Next sequence:", nextSequence.toString());

// Build ERV1 request bytes
const requestBytes = serialize(requestLayout, {
  request: {
    prefix: RequestPrefix.ERV1,
    chain: WORMHOLE_CHAIN_ID_SOLANA,
    address: emitter.toBytes(),
    sequence: nextSequence,
  },
});
console.log("Request bytes (ERV1):", requestBytes);

// Instruction 1: Post Wormhole message
const postMessageIx = await buildPostMessageInstruction(
  solanaKeypair.publicKey,
  emitter,
  Date.now() % 2 ** 32, // Use timestamp as nonce
  payload,
  { confirmed: {} },
);

// Instruction 2: Request for execution
const signedQuoteBytes = Buffer.from(
  quote.signedQuote.replace("0x", ""),
  "hex",
);

const requestForExecutionIx = await buildRequestForExecutionInstruction(
  solanaKeypair.publicKey,
  payee,
  BigInt(quote.estimatedCost),
  WORMHOLE_CHAIN_ID_DOGECOIN,
  solanaKeypair.publicKey, // refund to self
  signedQuoteBytes,
  Buffer.from(requestBytes), // ERV1 request bytes
  Buffer.alloc(0), // Empty relay instructions
);

// Build transaction with both instructions
const tx = new Transaction().add(
  // Fee transfer to Wormhole
  SystemProgram.transfer({
    fromPubkey: solanaKeypair.publicKey,
    toPubkey: accounts.feeCollector,
    lamports: 100,
  }),
  // Post Wormhole message
  postMessageIx,
  // Request executor to relay
  requestForExecutionIx,
);

// Send the transaction
console.log("\n=== Submitting Transaction ===");
let signature: string;
try {
  signature = await sendAndConfirmTransaction(connection, tx, [solanaKeypair], {
    commitment: "confirmed",
  });
  console.log("Transaction confirmed!");
  console.log("  Signature:", signature);
} catch (error) {
  console.error("Transaction failed:", error);
  process.exit(1);
}

// Build executor explorer URL
const executorExplorerUrl = `https://wormholelabs-xyz.github.io/executor-explorer/#/chain/${WORMHOLE_CHAIN_ID_SOLANA}/tx/${signature}?endpoint=https%3A%2F%2Fexecutor-testnet.labsapis.com&env=Testnet`;
console.log("\nExecutor Explorer:", executorExplorerUrl);

await pollExecutorStatus(
  signature,
  WORMHOLE_CHAIN_ID_SOLANA,
  30, // max attempts
  10000, // 10 second interval
);
