import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type InitEncodedVaaContext = {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;
};

export async function initEncodedVaaIx(
  program: CoreBridgeProgram,
  accounts: InitEncodedVaaContext
) {
  return program.methods.initEncodedVaa().accounts(accounts).instruction();
}
