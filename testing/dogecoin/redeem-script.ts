import * as bitcoin from "bitcoinjs-lib";
import * as ecc from "tiny-secp256k1";
import { ECPairFactory } from "ecpair";
import bs58 from "bs58";

bitcoin.initEccLib(ecc);
export const ECPair = ECPairFactory(ecc);

// Dogecoin Testnet network parameters
export const dogecoinTestnet: bitcoin.Network = {
  messagePrefix: "\x19Dogecoin Signed Message:\n",
  bech32: "tdge",
  bip32: {
    public: 0x043587cf,
    private: 0x04358394,
  },
  pubKeyHash: 0x71,
  scriptHash: 0xc4,
  wif: 0xf1,
};

// Redeem script constants
export const EMITTER_CHAIN = 1; // u16

export interface RedeemScriptParams {
  emitterChain: number;
  emitterContract: string; // 32 bytes hex
  recipientAddress: string; // 32 bytes hex
  managerPubkeys: Buffer[];
  mThreshold: number;
  nTotal: number;
}

// Helper to get OP_N opcode for small integers (1-16)
function opN(n: number): number {
  if (n < 1 || n > 16) throw new Error(`Invalid OP_N value: ${n}`);
  return bitcoin.opcodes.OP_1 + (n - 1);
}

// Build the custom redeem script
export function buildRedeemScript(params: RedeemScriptParams): Buffer {
  const {
    emitterChain,
    emitterContract,
    recipientAddress,
    managerPubkeys,
    mThreshold,
    nTotal,
  } = params;

  // Prepare data buffers
  const emitterChainBuf = Buffer.alloc(2);
  emitterChainBuf.writeUInt16BE(emitterChain);
  const emitterContractBuf = Buffer.from(emitterContract, "hex");
  const recipientAddressBuf = Buffer.from(recipientAddress, "hex");

  // Build script using bitcoin.script.compile for proper push opcodes
  const compiled = bitcoin.script.compile([
    // Push emitter_chain (2 bytes, u16 BE)
    new Uint8Array(emitterChainBuf),
    // Push emitter_contract (32 bytes)
    new Uint8Array(emitterContractBuf),
    // OP_2DROP - drops top 2 stack items
    bitcoin.opcodes.OP_2DROP,
    // Push recipient_address (32 bytes)
    new Uint8Array(recipientAddressBuf),
    // OP_DROP - drops top stack item
    bitcoin.opcodes.OP_DROP,
    // OP_M (threshold)
    opN(mThreshold),
    // Push each pubkey in index order (33 bytes compressed secp256k1)
    ...managerPubkeys.map((pk) => new Uint8Array(pk)),
    // OP_N (total keys)
    opN(nTotal),
    // OP_CHECKMULTISIG
    bitcoin.opcodes.OP_CHECKMULTISIG,
  ]);

  return Buffer.from(compiled);
}

// Load Solana keypair and derive emitter contract
export async function loadSolanaEmitterContract(): Promise<string> {
  const solanaKeypair = await Bun.file("solana-devnet-keypair.json").json();
  return Buffer.from(bs58.decode(solanaKeypair.publicKey)).toString("hex");
}
