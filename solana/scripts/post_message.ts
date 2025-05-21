import {
  deriveFeeCollectorKey,
  deriveWormholeBridgeDataKey,
} from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
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

  const acct = new web3.Keypair();
  const data = Buffer.from("01000000000b00000068656c6c6f20776f726c6400", "hex");
  const transaction = new web3.Transaction();
  transaction.add(
    web3.SystemProgram.transfer({
      fromPubkey: provider.publicKey,
      toPubkey: deriveFeeCollectorKey(coreBridgeAddress), // fee collector
      lamports: 100, // hardcoded for tilt in devnet_setup.sh
    })
  );
  transaction.add(
    new web3.TransactionInstruction({
      keys: [
        {
          // config
          isSigner: false,
          isWritable: true,
          pubkey: deriveWormholeBridgeDataKey(coreBridgeAddress),
        },
        {
          // message
          isSigner: true,
          isWritable: true,
          pubkey: acct.publicKey,
        },
        {
          // emitter
          isSigner: true,
          isWritable: false,
          pubkey: provider.publicKey,
        },
        {
          // sequence
          isSigner: false,
          isWritable: true,
          pubkey: web3.PublicKey.findProgramAddressSync(
            [Buffer.from("Sequence"), provider.publicKey.toBuffer()],
            coreBridgeAddress
          )[0],
        },
        {
          // payer
          isSigner: true,
          isWritable: true,
          pubkey: provider.publicKey,
        },
        {
          // fee collector
          isSigner: false,
          isWritable: true,
          pubkey: deriveFeeCollectorKey(coreBridgeAddress),
        },
        {
          // clock
          isSigner: false,
          isWritable: false,
          pubkey: new web3.PublicKey(
            "SysvarC1ock11111111111111111111111111111111"
          ),
        },
        {
          // system program
          isSigner: false,
          isWritable: false,
          pubkey: new web3.PublicKey("11111111111111111111111111111111"),
        },
      ],
      programId: coreBridgeAddress,
      data,
    })
  );
  const tx = await provider.sendAndConfirm(transaction, [acct]);
  console.log(tx);
})();
