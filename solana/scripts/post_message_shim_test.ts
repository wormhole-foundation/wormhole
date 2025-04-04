import { deriveFeeCollectorKey } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import {
  AnchorProvider,
  Program,
  setProvider,
  Wallet,
  web3,
} from "@coral-xyz/anchor";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";
import { WormholePostMessageShim } from "../../svm/wormhole-core-shims/anchor/idls/wormhole_post_message_shim";
import WormholePostMessageShimIdl from "../../svm/wormhole-core-shims/anchor/idls/wormhole_post_message_shim.json";

// Usage:
// RPC_URL="https://api.devnet.solana.com" CORE_BRIDGE_PROGRAM_ID=3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 SOLANA_KEY="<full_path>.json" MSG="hello wormhole" npx tsx post_message.ts

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

  const MSG = process.env.MSG;
  if (!MSG) {
    throw new Error("MSG is required");
  }

  const connection = new web3.Connection(RPC_URL, "confirmed");

  const key = process.env.SOLANA_KEY;

  if (!key) {
    throw new Error("SOLANA_KEY is required");
  }

  const payer = web3.Keypair.fromSecretKey(
    key.endsWith(".json") ? new Uint8Array(require(key)) : bs58.decode(key)
  );
  const provider = new AnchorProvider(connection, new Wallet(payer));
  setProvider(provider);

  const program = new Program<WormholePostMessageShim>(
    WormholePostMessageShimIdl as WormholePostMessageShim
  );

  const tx = await program.methods
    .postMessage(0, { confirmed: {} }, Buffer.from(MSG, "ascii"))
    // there seems to be an extra "program" field that is not needed and is marked optional in `svm/wormhole-core-shims/tests/wormhole-post-message-shim.ts` using anchor 0.30.1
    // @ts-ignore
    .accounts({
      emitter: payer.publicKey,
      wormholeProgram: coreBridgeAddress,
    })
    .preInstructions([
      // gotta pay the fee
      web3.SystemProgram.transfer({
        fromPubkey: payer.publicKey,
        toPubkey: deriveFeeCollectorKey(coreBridgeAddress), // fee collector
        lamports: 100, // hardcoded for tilt in devnet_setup.sh
      }),
    ])
    .rpc();
  console.log(tx);
})();
