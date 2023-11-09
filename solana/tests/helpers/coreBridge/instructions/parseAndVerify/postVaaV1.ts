import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";
import { PostedVaaV1 } from "../../legacy/state";
import { ethers } from "ethers";

export type PostVaaV1Context = {
  payer: PublicKey;
  encodedVaa: PublicKey;
  postedVaa?: PublicKey;
};

export async function postVaaV1Ix(program: CoreBridgeProgram, accounts: PostVaaV1Context) {
  let { payer, encodedVaa, postedVaa } = accounts;

  if (postedVaa === undefined) {
    const vaaBuf = await program.account.encodedVaa
      .fetch(encodedVaa)
      .then((vaaData) => vaaData.buf);
    const numSignatures = vaaBuf.readUInt8(5);
    const message = vaaBuf.subarray(6 + 66 * numSignatures);

    postedVaa = PostedVaaV1.address(
      program.programId,
      Array.from(ethers.utils.arrayify(ethers.utils.keccak256(message)))
    );
  }

  return program.methods.postVaaV1().accounts({ payer, encodedVaa, postedVaa }).instruction();
}
