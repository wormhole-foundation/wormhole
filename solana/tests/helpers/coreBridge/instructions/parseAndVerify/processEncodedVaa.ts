import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type VerifyEncodedVaaV1 = {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;
  guardianSet: PublicKey;
};

export async function verifyEncodedVaaV1Ix(
  program: CoreBridgeProgram,
  accounts: VerifyEncodedVaaV1
) {
  return program.methods.verifyEncodedVaaV1().accounts(accounts).instruction();
}
