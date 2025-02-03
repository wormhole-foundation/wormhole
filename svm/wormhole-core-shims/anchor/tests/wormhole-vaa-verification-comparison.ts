import { postVaa } from "@certusone/wormhole-sdk/lib/cjs/solana/sendAndConfirmPostVaa";
import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import { WormholeVaaVerificationComparison } from "../target/types/wormhole_vaa_verification_comparison";
import { WormholeVerifyVaaShim } from "../idls/wormhole_verify_vaa_shim";
import { parseVaa } from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";
import { logCostAndCompute } from "./helpers";
import { SignedVaa } from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";

// VAA from https://wormholescan.io/#/tx/3abEhA94A2bqqizebHoDGGjAtJ2dXpoPs4BPvMyCtmqx51EsSCZikUquLmncM6KwuFpNqkqpUNNFy3bCeV2poavX
// Error: Transaction too large: 1250 > 1232 on post_vaa, but works with new verification
// const VAA =
//   "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";

// VAA from https://wormholescan.io/#/tx/AEa98mf68bcwUmT8Yheidw4C4KUVSG9732SVg5kqnfmH?view=advanced
const VAA =
  "AQAAAAQNAL1qji7v9KnngyX0VxK+3fCMVscWTLoYX8L48NWquq2WGrcHd4H0wYc0KF4ZOWjLD2okXoBjGQIDJzx4qIrbSzQBAQq69h+neXGb58VfhZgraPVCxJmnTj8JIDq5jqi3Qav1e+IW51mIJlOhSAdCRbEyQLzf6Z3C19WJJqSyt/z1XF0AAvFgDHkseyMZTE5vQjflu4tc5OLPJe2VYCxTJT15LA02YPrWgOM6HhfUhXDhFoG5AI/s2ApjK8jaqi7LGJILAUMBA6cp4vfko8hYyRvogqQWsdk9e20g0O6s60h4ewweapXCQHerQpoJYdDxlCehN4fuYnuudEhW+6FaXLjwNJBdqsoABDg9qXjXB47nBVCZAGns2eosVqpjkyDaCfo/p1x8AEjBA80CyC1/QlbG9L4zlnnDIfZWylsf3keJqx28+fZNC5oABi6XegfozgE8JKqvZLvd7apDhrJ6Qv+fMiynaXASkafeVJOqgFOFbCMXdMKehD38JXvz3JrlnZ92E+I5xOJaDVgABzDSte4mxUMBMJB9UUgJBeAVsokFvK4DOfvh6G3CVqqDJplLwmjUqFB7fAgRfGcA8PWNStRc+YDZiG66YxPnptwACe84S31Kh9voz2xRk1THMpqHQ4fqE7DizXPNWz6Z6ebEXGcd7UP9PBXoNNvjkLWZJZOdbkZyZqztaIiAo4dgWUABCobiuQP92WjTxOZz0KhfWVJ3YBVfsXUwaVQH4/p6khX0HCEVHR9VHmjvrAAGDMdJGWW+zu8mFQc4gPU6m4PZ6swADO7voA5GWZZPiztz22pftwxKINGvOjCPlLpM1Y2+Vq6AQuez/mlUAmaL0NKgs+5VYcM1SGBz0TL3ABRhKQAhUEMADWmiMo0J1Qaj8gElb+9711ZjvAY663GIyG/E6EdPW+nPKJI9iZE180sLct+krHj0J7PlC9BjDiO2y149oCOJ6FgAEcaVkYK43EpN7XqxrdpanX6R6TaqECgZTjvtN3L6AP2ceQr8mJJraYq+qY8pTfFvPKEqmW9CBYvnA5gIMpX59WsAEjIL9Hdnx+zFY0qSPB1hB9AhqWeBP/QfJjqzqafsczaeCN/rWUf6iNBgXI050ywtEp8JQ36rCn8w6dRhUusn+MEAZ32XyAAAAAAAFczO6yk0j3G90i/+9DoqGcH1teF8XMpUEVKRIBgmcq3lAAAAAAAC/1wAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAC6Q7dAAAAAAAAAAAAAAAAAAoLhpkcYhizbB0Z1KLp6wzjYG60gAAgAAAAAAAAAAAAAAAInNTEvk5b/1WVF+JawF1smtAdicABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==";
const CORE_BRIDGE_PROGRAM_ID = new anchor.web3.PublicKey([
  14, 10, 88, 154, 65, 165, 95, 189, 102, 197, 42, 71, 95, 45, 146, 166, 211,
  220, 155, 71, 71, 17, 76, 185, 175, 130, 90, 152, 181, 69, 211, 206,
]);
const SEED_PREFIX = "GuardianSet";

describe("wormhole-vaa-verification-comparison", () => {
  // Configure the client to use the local cluster.
  anchor.setProvider(anchor.AnchorProvider.env());

  const program = anchor.workspace
    .WormholeVaaVerificationComparison as Program<WormholeVaaVerificationComparison>;
  const verifyShimProgram = anchor.workspace
    .WormholeVerifyVaaShim as Program<WormholeVerifyVaaShim>;

  it("Consumes a Posted VAA from the Core Bridge!", async () => {
    const payer = anchor.web3.Keypair.generate();
    {
      const tx = await program.provider.connection.requestAirdrop(
        payer.publicKey,
        10000000000
      );
      await program.provider.connection.confirmTransaction({
        ...(await program.provider.connection.getLatestBlockhash()),
        signature: tx,
      });
    }
    const buf = Buffer.from(VAA, "base64");
    const txs = await postVaa(
      program.provider.connection,
      async (transaction) => {
        await new Promise(function (resolve) {
          setTimeout(function () {
            resolve(500);
          });
        });
        transaction.partialSign(payer);
        return transaction;
      },
      "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth",
      payer.publicKey.toString(),
      buf,
      undefined,
      false
    );
    const vaa = parseVaa(buf);
    txs.push({
      signature: await program.methods
        .consumeCorePostedVaa([...vaa.hash])
        .rpc(),
      response: null,
    });
    for (const tx of txs) {
      await logCostAndCompute("core", tx.signature);
    }
  });

  it("Consumes a VAA directly!", async () => {
    const signatureKeypair = anchor.web3.Keypair.generate();
    const buf = Buffer.from(VAA, "base64");
    const vaa = parseVaa(buf);
    const tx = await program.methods
      .postSignatures(
        vaa.guardianSignatures.map((s) => [s.index, ...s.signature]),
        vaa.guardianSignatures.length
      )
      .accounts({ guardianSignatures: signatureKeypair.publicKey })
      .signers([signatureKeypair])
      .rpc();
    await logCostAndCompute("self", tx);

    // Convert guardian_set_index to big-endian bytes
    const guardianSetIndex = vaa.guardianSetIndex;
    const indexBuffer = Buffer.alloc(4); // guardian_set_index is a u32
    indexBuffer.writeUInt32BE(guardianSetIndex);
    const [guardianSetPDA] = anchor.web3.PublicKey.findProgramAddressSync(
      [Buffer.from(SEED_PREFIX), indexBuffer],
      CORE_BRIDGE_PROGRAM_ID
    );
    const tx2 = await program.methods
      .consumeVaa(vaaBody(buf), vaa.guardianSetIndex)
      .accounts({
        guardianSignatures: signatureKeypair.publicKey,
      })
      .accountsPartial({ guardianSet: guardianSetPDA })
      .preInstructions([
        anchor.web3.ComputeBudgetProgram.setComputeUnitLimit({
          units: 420_000,
        }),
      ])
      .rpc();
    await logCostAndCompute("self", tx2);
  });

  it("Consumes a VAA via the shim!", async () => {
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
    await logCostAndCompute("shim", tx);

    // Convert guardian_set_index to big-endian bytes
    const guardianSetIndex = vaa.guardianSetIndex;
    const indexBuffer = Buffer.alloc(4); // guardian_set_index is a u32
    indexBuffer.writeUInt32BE(guardianSetIndex);
    const [guardianSetPDA, guardianSetBump] =
      anchor.web3.PublicKey.findProgramAddressSync(
        [Buffer.from(SEED_PREFIX), indexBuffer],
        CORE_BRIDGE_PROGRAM_ID
      );
    const tx2 = await program.methods
      .consumeVaaViaShim(guardianSetBump, vaaBody(buf))
      .accounts({
        guardianSignatures: signatureKeypair.publicKey,
      })
      .accountsPartial({ guardianSet: guardianSetPDA })
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
    await logCostAndCompute("shim", tx2);
  });
});

function vaaBody(vaa: SignedVaa): Buffer {
  const signedVaa = Buffer.isBuffer(vaa) ? vaa : Buffer.from(vaa as Uint8Array);
  const sigStart = 6;
  const numSigners = signedVaa[5];
  const sigLength = 66;
  return signedVaa.subarray(sigStart + sigLength * numSigners);
}
