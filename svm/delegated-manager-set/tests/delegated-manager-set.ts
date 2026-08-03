import {
  parseVaa,
  SignedVaa,
} from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";
import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import type { WormholeVerifyVaaShim } from "../idls/wormhole_verify_vaa_shim";
import WormholeVerifyVaaShimIdl from "../idls/wormhole_verify_vaa_shim.json";
import { DelegatedManagerSet } from "../target/types/delegated_manager_set";
import { expect } from "chai";
import { keccak256 } from "@certusone/wormhole-sdk";

const VAA =
  "01000000000100332d786f5f8daec0ec99d08c7df7777d57cf45605b1667d868bd0b07a8f816f4723bd5578381173c5ef2ed4b6f2224726439ea4a7227cf8b2742b143bf3f8fed01000000008d53cf9f000100000000000000000000000000000000000000000000000000000000000000041e1a7bad048d464a200000000000000000000000000000000044656c6567617465644d616e6167657201000000410000000101050702349de56ca5dd06db8660419d6f150662e0f04febdbf6512d7cfe78c23b51491c035163bfd9518b0a536a17f330a1589fe21d7404b51f525a0a990a65a701952ebb036d40b0b85bca49e41f05a26950578bb13a424507ce34a80f83d3cf601e25818b0307681002ae28b9399e828d0f46d54c31d5d6ff187b3bdddc6615987a466455f50375abc8955c8a8c875ee1febd157132adcc1b992d69a946e83485b8360e23a277030212d206546216917a75533ed6c975f8f794ba0d8a7fb84dedf65ebb20e64841037ff483369b52bd87a73f23413dd8fcace71de7f7823c5c9120f1e9cfe5733a88";
const CORE_BRIDGE_PROGRAM_ID = new anchor.web3.PublicKey(
  "3u8hJUVTA4jH1wYAyUur7FFZVQ8H635K3tSHHF4ssjQ5",
);
const GUARDIAN_SET_SEED = "GuardianSet";

// VAA body offsets for extracting governance payload fields
const MANAGER_CHAIN_ID_OFFSET = 86;
const MANAGER_SET_INDEX_OFFSET = 88;
const MANAGER_SET_DATA_OFFSET = 92;

describe("delegated-manager-set", () => {
  // Configure the client to use the local cluster.
  anchor.setProvider(anchor.AnchorProvider.env());

  const program = anchor.workspace
    .delegatedManagerSet as Program<DelegatedManagerSet>;
  const verifyShimProgram = new Program<WormholeVerifyVaaShim>(
    WormholeVerifyVaaShimIdl as WormholeVerifyVaaShim,
  );

  const signatureKeypair = anchor.web3.Keypair.generate();
  const buf = Buffer.from(VAA, "hex");
  const vaa = parseVaa(buf);
  // Convert guardian_set_index to big-endian bytes
  const guardianSetIndex = vaa.guardianSetIndex;
  const indexBuffer = Buffer.alloc(4); // guardian_set_index is a u32
  indexBuffer.writeUInt32BE(guardianSetIndex);
  const [guardianSet, guardianSetBump] =
    anchor.web3.PublicKey.findProgramAddressSync(
      [Buffer.from(GUARDIAN_SET_SEED), indexBuffer],
      CORE_BRIDGE_PROGRAM_ID,
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

  it("Posts signatures (pre-test)", async () => {
    const tx = await verifyShimProgram.methods
      .postSignatures(
        vaa.guardianSetIndex,
        vaa.guardianSignatures.length,
        vaa.guardianSignatures.map((s) => [s.index, ...s.signature]),
      )
      .accounts({ guardianSignatures: signatureKeypair.publicKey })
      .signers([signatureKeypair])
      .rpc();
  });

  it("Rejects a digest mismatch", async () => {
    let expectedError = "";
    try {
      const tx3 = await program.methods
        .submitNewManagerSet({
          guardianSetBump,
          digest: Array(32).fill(0),
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
        .rpc();
    } catch (e) {
      expectedError = e.message;
    }
    expect(expectedError).to.include(
      "Error Code: DigestMismatch. Error Number: 6000. Error Message: Digest argument does not match computed digest from VAA body.",
    );
  });

  it("Submits a new manager set!", async () => {
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
      // This would be done in practice, but omitting for the replay test
      // .postInstructions([
      //   await verifyShimProgram.methods
      //     .closeSignatures()
      //     .accounts({ guardianSignatures: signatureKeypair.publicKey })
      //     .instruction(),
      // ])
      .rpc();

    // Verify the manager_set_index account was correctly populated
    const managerSetIndexAccount = await program.account.managerSetIndex.fetch(
      managerSetIndexPda,
    );
    expect(managerSetIndexAccount.managerChainId).to.equal(managerChainId);
    expect(managerSetIndexAccount.currentIndex).to.equal(managerSetIndex);

    // Verify the manager_set account was correctly populated
    const managerSetAccount = await program.account.managerSet.fetch(
      managerSetPda,
    );
    expect(managerSetAccount.managerChainId).to.equal(managerChainId);
    expect(managerSetAccount.index).to.equal(managerSetIndex);
    expect(Buffer.from(managerSetAccount.managerSet)).to.deep.equal(
      managerSetData,
    );
  });

  it("Rejects a repeat submission", async () => {
    let expectedError = "";
    try {
      const tx3 = await program.methods
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
        .rpc();
    } catch (e) {
      expectedError = e.message;
    }
    expect(expectedError).to.include(
      "Allocate: account Address { address: H1N5AXQhNUAzBF1qQjcstb1Q2b64Qvun8eyz9Cit1BAu, base: None } already in use",
    );
  });
});

function vaaBody(vaa: SignedVaa): Buffer {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const sigStart = 6;
  const numSigners = signedVaa[5];
  const sigLength = 66;
  return signedVaa.subarray(sigStart + sigLength * numSigners);
}
