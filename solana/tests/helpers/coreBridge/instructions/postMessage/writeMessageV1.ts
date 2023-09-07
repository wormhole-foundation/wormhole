import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type WriteMessageV1Context = {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
};

export type WriteMessageV1Args = {
  index: number;
  data: Buffer;
};

export async function writeMessageV1Ix(
  program: CoreBridgeProgram,
  accounts: WriteMessageV1Context,
  args: WriteMessageV1Args
) {
  return program.methods.writeMessageV1(args).accounts(accounts).instruction();
}
