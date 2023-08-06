import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type InitMessageV1Context = {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
};

export type InitMessageV1Args = {
  cpiProgramId: PublicKey | null;
};

export async function initMessageV1Ix(
  program: CoreBridgeProgram,
  accounts: InitMessageV1Context,
  args: InitMessageV1Args
) {
  return program.methods.initMessageV1(args).accounts(accounts).instruction();
}
