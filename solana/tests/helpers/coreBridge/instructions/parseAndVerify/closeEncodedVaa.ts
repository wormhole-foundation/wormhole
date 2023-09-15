import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type CloseEncodedVaaContext = {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;
};

export async function closeEncodedVaaIx(
  program: CoreBridgeProgram,
  accounts: CloseEncodedVaaContext
) {
  return program.methods.closeEncodedVaa().accounts(accounts).instruction();
}
