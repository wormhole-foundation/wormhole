import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type WriteEncodedVaaContext = {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;
};

export type WriteEncodedVaaArgs = {
  index: number;
  data: Buffer;
};

export async function writeEncodedVaaIx(
  program: CoreBridgeProgram,
  accounts: WriteEncodedVaaContext,
  args: WriteEncodedVaaArgs
) {
  return program.methods.writeEncodedVaa(args).accounts(accounts).instruction();
}
