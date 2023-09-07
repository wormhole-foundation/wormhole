import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type ProcessMessageV1Context = {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
  closeAccountDestination: PublicKey;
};

export async function closeMessageV1Ix(
  program: CoreBridgeProgram,
  accounts: ProcessMessageV1Context
) {
  return program.methods.closeMessageV1().accounts(accounts).instruction();
}
