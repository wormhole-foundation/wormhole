import { Commitment, PublicKey } from "@solana/web3.js";
import { toMessageCommitment, CoreBridgeProgram } from "../..";

export type InitMessageV1Context = {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
};

export type InitMessageV1Args = {
  nonce: number;
  commitment: Commitment;
  cpiProgramId: PublicKey | null;
};

export async function initMessageV1Ix(
  program: CoreBridgeProgram,
  accounts: InitMessageV1Context,
  args: InitMessageV1Args
) {
  const { nonce, cpiProgramId, commitment: solanaCommitment } = args;
  const commitment = toMessageCommitment(solanaCommitment);
  return program.methods
    .initMessageV1({ nonce, commitment, cpiProgramId })
    .accounts(accounts)
    .instruction();
}
