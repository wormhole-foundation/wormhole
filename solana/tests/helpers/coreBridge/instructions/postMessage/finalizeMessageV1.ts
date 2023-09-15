import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type FinalizeMessageV1Context = {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
};

export async function finalizeMessageV1Ix(
  program: CoreBridgeProgram,
  accounts: FinalizeMessageV1Context
) {
  return program.methods.finalizeMessageV1().accounts(accounts).instruction();
}
