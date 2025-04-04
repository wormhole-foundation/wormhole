import { createInitializeInstruction } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { AnchorProvider, Wallet, web3 } from "@coral-xyz/anchor";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";

// Usage:
// RPC_URL="https://api.devnet.solana.com" CORE_BRIDGE_PROGRAM_ID=3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 SOLANA_KEY="<full_path>.json" npx tsx post_message.ts

(async () => {
  const RPC_URL = process.env.RPC_URL;
  if (!RPC_URL) {
    throw new Error("RPC_URL is required");
  }

  const CORE_BRIDGE_PROGRAM_ID = process.env.CORE_BRIDGE_PROGRAM_ID;
  if (!CORE_BRIDGE_PROGRAM_ID) {
    throw new Error("CORE_BRIDGE_PROGRAM_ID is required");
  }

  const coreBridgeAddress = new web3.PublicKey(CORE_BRIDGE_PROGRAM_ID);

  const connection = new web3.Connection(RPC_URL, "confirmed");

  const key = process.env.SOLANA_KEY;

  if (!key) {
    throw new Error("SOLANA_KEY is required");
  }

  const payer = web3.Keypair.fromSecretKey(
    key.endsWith(".json") ? new Uint8Array(require(key)) : bs58.decode(key)
  );
  const provider = new AnchorProvider(connection, new Wallet(payer));

  const ix = createInitializeInstruction(
    coreBridgeAddress,
    payer.publicKey.toString(),
    86400,
    BigInt(100),
    [Buffer.from("13947Bd48b18E53fdAeEe77F3473391aC727C638", "hex")]
  );
  const transaction = new web3.Transaction();
  transaction.add(ix);
  const tx = await provider.sendAndConfirm(transaction);
  console.log(tx);
})();
