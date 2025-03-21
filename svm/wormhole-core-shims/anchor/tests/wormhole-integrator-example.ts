import {
  parseVaa,
  SignedVaa,
} from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";
import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import { WormholeIntegratorExample } from "../target/types/wormhole_integrator_example";
import { WormholePostMessageShim } from "../idls/wormhole_post_message_shim";
import { WormholeVerifyVaaShim } from "../idls/wormhole_verify_vaa_shim";
import { logCostAndCompute } from "./helpers";

// VAA from https://wormholescan.io/#/tx/AEa98mf68bcwUmT8Yheidw4C4KUVSG9732SVg5kqnfmH?view=advanced
const VAA =
  "AQAAAAQNAL1qji7v9KnngyX0VxK+3fCMVscWTLoYX8L48NWquq2WGrcHd4H0wYc0KF4ZOWjLD2okXoBjGQIDJzx4qIrbSzQBAQq69h+neXGb58VfhZgraPVCxJmnTj8JIDq5jqi3Qav1e+IW51mIJlOhSAdCRbEyQLzf6Z3C19WJJqSyt/z1XF0AAvFgDHkseyMZTE5vQjflu4tc5OLPJe2VYCxTJT15LA02YPrWgOM6HhfUhXDhFoG5AI/s2ApjK8jaqi7LGJILAUMBA6cp4vfko8hYyRvogqQWsdk9e20g0O6s60h4ewweapXCQHerQpoJYdDxlCehN4fuYnuudEhW+6FaXLjwNJBdqsoABDg9qXjXB47nBVCZAGns2eosVqpjkyDaCfo/p1x8AEjBA80CyC1/QlbG9L4zlnnDIfZWylsf3keJqx28+fZNC5oABi6XegfozgE8JKqvZLvd7apDhrJ6Qv+fMiynaXASkafeVJOqgFOFbCMXdMKehD38JXvz3JrlnZ92E+I5xOJaDVgABzDSte4mxUMBMJB9UUgJBeAVsokFvK4DOfvh6G3CVqqDJplLwmjUqFB7fAgRfGcA8PWNStRc+YDZiG66YxPnptwACe84S31Kh9voz2xRk1THMpqHQ4fqE7DizXPNWz6Z6ebEXGcd7UP9PBXoNNvjkLWZJZOdbkZyZqztaIiAo4dgWUABCobiuQP92WjTxOZz0KhfWVJ3YBVfsXUwaVQH4/p6khX0HCEVHR9VHmjvrAAGDMdJGWW+zu8mFQc4gPU6m4PZ6swADO7voA5GWZZPiztz22pftwxKINGvOjCPlLpM1Y2+Vq6AQuez/mlUAmaL0NKgs+5VYcM1SGBz0TL3ABRhKQAhUEMADWmiMo0J1Qaj8gElb+9711ZjvAY663GIyG/E6EdPW+nPKJI9iZE180sLct+krHj0J7PlC9BjDiO2y149oCOJ6FgAEcaVkYK43EpN7XqxrdpanX6R6TaqECgZTjvtN3L6AP2ceQr8mJJraYq+qY8pTfFvPKEqmW9CBYvnA5gIMpX59WsAEjIL9Hdnx+zFY0qSPB1hB9AhqWeBP/QfJjqzqafsczaeCN/rWUf6iNBgXI050ywtEp8JQ36rCn8w6dRhUusn+MEAZ32XyAAAAAAAFczO6yk0j3G90i/+9DoqGcH1teF8XMpUEVKRIBgmcq3lAAAAAAAC/1wAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC6Q7dAAAAAAAAAAAAAAAAAAoLhpkcYhizbB0Z1KLp6wzjYG60gAAgAAAAAAAAAAAAAAAInNTEvk5b/1WVF+JawF1smtAdicABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==";
const CORE_BRIDGE_PROGRAM_ID = new anchor.web3.PublicKey(
  "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
);
const GUARDIAN_SET_SEED = "GuardianSet";

describe("wormhole-integrator-example", () => {
  // Configure the client to use the local cluster.
  anchor.setProvider(anchor.AnchorProvider.env());

  const program = anchor.workspace
    .WormholeIntegratorExample as Program<WormholeIntegratorExample>;
  const verifyShimProgram = anchor.workspace
    .WormholeVerifyVaaShim as Program<WormholeVerifyVaaShim>;
  const postShimProgram = anchor.workspace
    .WormholePostMessageShim as Program<WormholePostMessageShim>;

  it("Initializes!", async () => {
    const tx = await program.methods
      .initialize()
      .accountsPartial({
        sequence: anchor.web3.PublicKey.findProgramAddressSync(
          [
            Buffer.from("Sequence"),
            anchor.web3.PublicKey.findProgramAddressSync(
              [Buffer.from("emitter")],
              program.programId
            )[0].toBuffer(),
          ],
          CORE_BRIDGE_PROGRAM_ID
        )[0],
        wormholePostMessageShimEa: anchor.web3.PublicKey.findProgramAddressSync(
          [Buffer.from("__event_authority")],
          postShimProgram.programId
        )[0],
        programData: anchor.web3.PublicKey.findProgramAddressSync(
          [program.programId.toBuffer()],
          new anchor.web3.PublicKey(
            "BPFLoaderUpgradeab1e11111111111111111111111"
          )
        )[0],
      })
      .preInstructions([
        // gotta pay the fee
        anchor.web3.SystemProgram.transfer({
          fromPubkey: program.provider.publicKey,
          toPubkey: new anchor.web3.PublicKey(
            "9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy"
          ), // fee collector
          lamports: 100, // hardcoded for tilt in devnet_setup.sh
        }),
      ])
      .rpc();
    await logCostAndCompute("init", tx);
  });

  it("Posts a message!", async () => {
    const tx = await program.methods
      .postMessage()
      .accounts({
        sequence: anchor.web3.PublicKey.findProgramAddressSync(
          [
            Buffer.from("Sequence"),
            anchor.web3.PublicKey.findProgramAddressSync(
              [Buffer.from("emitter")],
              program.programId
            )[0].toBuffer(),
          ],
          CORE_BRIDGE_PROGRAM_ID
        )[0],
        wormholePostMessageShimEa: anchor.web3.PublicKey.findProgramAddressSync(
          [Buffer.from("__event_authority")],
          postShimProgram.programId
        )[0],
      })
      .preInstructions([
        // gotta pay the fee
        anchor.web3.SystemProgram.transfer({
          fromPubkey: program.provider.publicKey,
          toPubkey: new anchor.web3.PublicKey(
            "9bFNrXNb2WTx8fMHXCheaZqkLZ3YCCaiqTftHxeintHy"
          ), // fee collector
          lamports: 100, // hardcoded for tilt in devnet_setup.sh
        }),
      ])
      .rpc();
    await logCostAndCompute("shim post", tx);
  });

  it("Consumes a VAA!", async () => {
    const signatureKeypair = anchor.web3.Keypair.generate();
    const buf = Buffer.from(VAA, "base64");
    const vaa = parseVaa(buf);
    const tx = await verifyShimProgram.methods
      .postSignatures(
        vaa.guardianSetIndex,
        vaa.guardianSignatures.length,
        vaa.guardianSignatures.map((s) => [s.index, ...s.signature])
      )
      .accounts({ guardianSignatures: signatureKeypair.publicKey })
      .signers([signatureKeypair])
      .rpc();
    await logCostAndCompute("shim verify (1/2)", tx);

    // Convert guardian_set_index to big-endian bytes
    const guardianSetIndex = vaa.guardianSetIndex;
    const indexBuffer = Buffer.alloc(4); // guardian_set_index is a u32
    indexBuffer.writeUInt32BE(guardianSetIndex);
    const [guardianSet, guardianSetBump] =
      anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from(GUARDIAN_SET_SEED), indexBuffer],
        CORE_BRIDGE_PROGRAM_ID
      );
    const tx2 = await program.methods
      .consumeVaa(guardianSetBump, vaaBody(buf))
      .accountsPartial({
        guardianSignatures: signatureKeypair.publicKey,
        guardianSet,
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
    await logCostAndCompute("shim verify (2/2)", tx2);
  });
});

function vaaBody(vaa: SignedVaa): Buffer {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const sigStart = 6;
  const numSigners = signedVaa[5];
  const sigLength = 66;
  return signedVaa.subarray(sigStart + sigLength * numSigners);
}
