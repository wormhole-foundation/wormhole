import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";
import { PostedVaaV1 } from "../../legacy/state";
import { ethers } from "ethers";

export type PostVaaV1Context = {
  writeAuthority: PublicKey;
  vaa: PublicKey;
  postedVaa?: PublicKey;
  systemProgram: PublicKey;
};

export type PostVaaV1Directive = { tryOnce: {} };

export async function postVaaV1Ix(
  program: CoreBridgeProgram,
  accounts: PostVaaV1Context,
  directive: PostVaaV1Directive
) {
  let { writeAuthority, vaa, postedVaa, systemProgram } = accounts;

  if (postedVaa === undefined) {
    const vaaBuf = await program.account.encodedVaa
      .fetch(vaa)
      .then((vaaData) => vaaData.buf);
    const numSignatures = vaaBuf.readUInt8(5);
    const message = vaaBuf.subarray(6 + 66 * numSignatures);

    postedVaa = PostedVaaV1.address(
      program.programId,
      Array.from(ethers.utils.arrayify(ethers.utils.keccak256(message)))
    );
  }

  return program.methods
    .postVaaV1(directive)
    .accounts({ writeAuthority, vaa, postedVaa, systemProgram })
    .instruction();
}
