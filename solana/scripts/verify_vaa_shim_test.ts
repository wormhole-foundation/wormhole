import { keccak256 } from "@certusone/wormhole-sdk/lib/cjs/utils/keccak";
import { parseVaa } from "@certusone/wormhole-sdk/lib/cjs/vaa";
import {
  AnchorProvider,
  Program,
  setProvider,
  Wallet,
  web3,
} from "@coral-xyz/anchor";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";
import { WormholeVerifyVaaShim } from "../../svm/wormhole-core-shims/anchor/idls/wormhole_verify_vaa_shim";
import WormholeVerifyVaaShimIdl from "../../svm/wormhole-core-shims/anchor/idls/wormhole_verify_vaa_shim.json";

// Usage:
// RPC_URL="https://api.devnet.solana.com" CORE_BRIDGE_PROGRAM_ID=3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5 SOLANA_KEY="<full_path>.json" VAA="<base64 VAA from wormholescan>" npx tsx post_message.ts

const GUARDIAN_SET_SEED = "GuardianSet";

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

  const VAA = process.env.VAA;
  if (!VAA) {
    throw new Error("VAA is required");
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

  const program = new Program<WormholeVerifyVaaShim>(
    WormholeVerifyVaaShimIdl as WormholeVerifyVaaShim
  );
  const signatureKeypair = web3.Keypair.generate();
  console.log(`signature public key: ${signatureKeypair.publicKey.toString()}`);
  const buf = Buffer.from(VAA, "base64");
  const vaa = parseVaa(buf);
  const tx = await program.methods
    .postSignatures(
      vaa.guardianSetIndex,
      vaa.guardianSignatures.length,
      vaa.guardianSignatures.map((s) => [s.index, ...s.signature])
    )
    .accounts({ guardianSignatures: signatureKeypair.publicKey })
    .signers([signatureKeypair])
    .rpc();
  console.log(`verify tx1: ${tx}`);

  // Convert guardian_set_index to big-endian bytes
  const guardianSetIndex = vaa.guardianSetIndex;
  const indexBuffer = Buffer.alloc(4); // guardian_set_index is a u32
  indexBuffer.writeUInt32BE(guardianSetIndex);
  const [guardianSet, guardianSetBump] = web3.PublicKey.findProgramAddressSync(
    [Buffer.from(GUARDIAN_SET_SEED), indexBuffer],
    coreBridgeAddress
  );

  const tx2 = await program.methods
    .verifyHash(guardianSetBump, [...keccak256(vaa.hash)])
    .accounts({
      guardianSet,
      guardianSignatures: signatureKeypair.publicKey,
    })
    .preInstructions([
      web3.ComputeBudgetProgram.setComputeUnitLimit({
        units: 420_000,
      }),
    ])
    .postInstructions([
      await program.methods
        .closeSignatures()
        .accounts({ guardianSignatures: signatureKeypair.publicKey })
        .instruction(),
    ])
    .rpc();
  console.log(`verify tx2: ${tx2}`);
})();
