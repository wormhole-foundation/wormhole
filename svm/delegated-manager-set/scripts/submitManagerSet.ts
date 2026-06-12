import {
  parseVaa,
  SignedVaa,
} from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";
import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import type { WormholeVerifyVaaShim } from "../idls/wormhole_verify_vaa_shim";
import WormholeVerifyVaaShimIdl from "../idls/wormhole_verify_vaa_shim.json";
import { DelegatedManagerSet } from "../target/types/delegated_manager_set";
import DelegatedManagerSetIdl from "../target/idl/delegated_manager_set.json";
import { keccak256 } from "@certusone/wormhole-sdk";
import * as fs from "fs";

const CORE_BRIDGE_PROGRAM_IDS = {
  mainnet: new anchor.web3.PublicKey(
    "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth",
  ),
  devnet: new anchor.web3.PublicKey(
    "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
  ),
};
const GUARDIAN_SET_SEED = "GuardianSet";

// VAA body offsets for extracting governance payload fields
const MANAGER_CHAIN_ID_OFFSET = 86;
const MANAGER_SET_INDEX_OFFSET = 88;
const MANAGER_SET_DATA_OFFSET = 92;

function vaaBody(vaa: SignedVaa): Buffer {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const sigStart = 6;
  const numSigners = signedVaa[5];
  const sigLength = 66;
  return signedVaa.subarray(sigStart + sigLength * numSigners);
}

async function main() {
  // Parse command line arguments
  const args = process.argv.slice(2);
  let network: "mainnet" | "devnet" = "mainnet";
  let vaaHex: string | undefined;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--mainnet" || args[i] === "-m") {
      network = "mainnet";
    } else if (args[i] === "--devnet" || args[i] === "-d") {
      network = "devnet";
    } else if (!vaaHex) {
      vaaHex = args[i];
    }
  }

  if (!vaaHex) {
    console.error(
      "Usage: bun run scripts/submitManagerSet.ts [--mainnet|-m | --devnet|-d] <VAA_HEX>",
    );
    console.error("");
    console.error("Options:");
    console.error(
      "  --mainnet, -m  Use mainnet Core Bridge program ID (default)",
    );
    console.error("  --devnet, -d   Use devnet Core Bridge program ID");
    console.error("");
    console.error("Environment variables:");
    console.error("  SOLANA_KEY - Path to keypair file (required)");
    console.error(
      "  RPC_URL - Solana RPC URL (optional, defaults to localhost)",
    );
    process.exit(1);
  }

  const coreBridgeProgramId = CORE_BRIDGE_PROGRAM_IDS[network];
  console.log(`Network: ${network}`);
  console.log(`Core Bridge Program ID: ${coreBridgeProgramId.toBase58()}`);

  // Load keypair from environment
  const keyPath = process.env.SOLANA_KEY;
  if (!keyPath) {
    console.error("Error: SOLANA_KEY environment variable not set");
    process.exit(1);
  }

  const keypairData = JSON.parse(fs.readFileSync(keyPath, "utf-8"));
  const payer = anchor.web3.Keypair.fromSecretKey(Uint8Array.from(keypairData));

  // Setup connection and provider
  const rpcUrl = process.env.RPC_URL || "http://localhost:8899";
  const connection = new anchor.web3.Connection(rpcUrl, "confirmed");
  const wallet = new anchor.Wallet(payer);
  const provider = new anchor.AnchorProvider(connection, wallet, {
    commitment: "confirmed",
  });
  anchor.setProvider(provider);

  // Initialize programs
  const program = new Program<DelegatedManagerSet>(
    DelegatedManagerSetIdl as DelegatedManagerSet,
    provider,
  );
  const verifyShimProgram = new Program<WormholeVerifyVaaShim>(
    WormholeVerifyVaaShimIdl as WormholeVerifyVaaShim,
    provider,
  );

  // Parse VAA
  const buf = Buffer.from(vaaHex, "hex");
  const vaa = parseVaa(buf);

  console.log("Parsed VAA:");
  console.log("  Guardian Set Index:", vaa.guardianSetIndex);
  console.log("  Num Signatures:", vaa.guardianSignatures.length);
  console.log("  Emitter Chain:", vaa.emitterChain);
  console.log("  Sequence:", vaa.sequence.toString());

  // Derive guardian set PDA
  const guardianSetIndex = vaa.guardianSetIndex;
  const indexBuffer = Buffer.alloc(4);
  indexBuffer.writeUInt32BE(guardianSetIndex);
  const [guardianSet, guardianSetBump] =
    anchor.web3.PublicKey.findProgramAddressSync(
      [Buffer.from(GUARDIAN_SET_SEED), indexBuffer],
      coreBridgeProgramId,
    );

  // Extract the VAA body
  const vaaBodyBytes = vaaBody(buf);
  const vaaDigest = keccak256(keccak256(vaaBodyBytes));

  // Extract manager chain ID and set index from VAA body (big-endian)
  const managerChainIdBytes = vaaBodyBytes.subarray(
    MANAGER_CHAIN_ID_OFFSET,
    MANAGER_CHAIN_ID_OFFSET + 2,
  );
  const managerSetIndexBytes = vaaBodyBytes.subarray(
    MANAGER_SET_INDEX_OFFSET,
    MANAGER_SET_INDEX_OFFSET + 4,
  );
  const managerChainId = managerChainIdBytes.readUInt16BE(0);
  const managerSetIndex = managerSetIndexBytes.readUInt32BE(0);
  const managerSetData = vaaBodyBytes.subarray(MANAGER_SET_DATA_OFFSET);

  console.log("\nGovernance Payload:");
  console.log("  Manager Chain ID:", managerChainId);
  console.log("  Manager Set Index:", managerSetIndex);
  console.log("  Manager Set Data Length:", managerSetData.length, "bytes");

  // Derive manager_set_index PDA
  const [managerSetIndexPda] = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("manager_set_index"), managerChainIdBytes],
    program.programId,
  );

  // Derive manager_set PDA
  const [managerSetPda] = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("manager_set"), managerChainIdBytes, managerSetIndexBytes],
    program.programId,
  );

  // Derive consumed PDA
  const [consumedPda] = anchor.web3.PublicKey.findProgramAddressSync(
    [Buffer.from("consumed_vaa"), Buffer.from(vaaDigest)],
    program.programId,
  );

  console.log("\nPDAs:");
  console.log("  Guardian Set:", guardianSet.toBase58());
  console.log("  Manager Set Index:", managerSetIndexPda.toBase58());
  console.log("  Manager Set:", managerSetPda.toBase58());
  console.log("  Consumed VAA:", consumedPda.toBase58());

  // Generate keypair for signatures account
  const signatureKeypair = anchor.web3.Keypair.generate();

  console.log("\nStep 1: Posting guardian signatures...");
  try {
    const tx1 = await verifyShimProgram.methods
      .postSignatures(
        vaa.guardianSetIndex,
        vaa.guardianSignatures.length,
        vaa.guardianSignatures.map((s) => [s.index, ...s.signature]),
      )
      .accounts({ guardianSignatures: signatureKeypair.publicKey })
      .signers([signatureKeypair])
      .rpc();
    console.log("  Transaction:", tx1);
  } catch (e) {
    console.error("  Failed to post signatures:", e.message);
    process.exit(1);
  }

  console.log("\nStep 2: Submitting new manager set...");
  try {
    const tx2 = await program.methods
      .submitNewManagerSet({
        guardianSetBump,
        digest: [...vaaDigest],
        vaaBody: vaaBodyBytes,
      })
      .accountsPartial({
        guardianSignatures: signatureKeypair.publicKey,
        guardianSet,
        managerSetIndex: managerSetIndexPda,
        managerSet: managerSetPda,
      })
      .preInstructions([
        anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({
          units: 420_000,
        }),
      ])
      .postInstructions([
        await verifyShimProgram.methods
          .closeSignatures()
          .accounts({ guardianSignatures: signatureKeypair.publicKey })
          .instruction(),
      ])
      .rpc();
    console.log("  Transaction:", tx2);
  } catch (e) {
    console.error("  Failed to submit manager set:", e.message);
    process.exit(1);
  }

  console.log("\nStep 3: Verifying accounts...");
  try {
    const managerSetIndexAccount =
      await program.account.managerSetIndex.fetch(managerSetIndexPda);
    console.log("  Manager Set Index Account:");
    console.log("    Manager Chain ID:", managerSetIndexAccount.managerChainId);
    console.log("    Current Index:", managerSetIndexAccount.currentIndex);

    const managerSetAccount =
      await program.account.managerSet.fetch(managerSetPda);
    console.log("  Manager Set Account:");
    console.log("    Manager Chain ID:", managerSetAccount.managerChainId);
    console.log("    Index:", managerSetAccount.index);
    console.log(
      "    Manager Set:",
      Buffer.from(managerSetAccount.managerSet).toString("hex"),
    );

    // Verify data matches
    if (
      managerSetIndexAccount.managerChainId === managerChainId &&
      managerSetIndexAccount.currentIndex === managerSetIndex &&
      managerSetAccount.managerChainId === managerChainId &&
      managerSetAccount.index === managerSetIndex &&
      Buffer.from(managerSetAccount.managerSet).equals(managerSetData)
    ) {
      console.log("\nSuccess! Manager set submitted and verified.");
    } else {
      console.error("\nError: Account data does not match expected values.");
      process.exit(1);
    }
  } catch (e) {
    console.error("  Failed to verify accounts:", e.message);
    process.exit(1);
  }
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
