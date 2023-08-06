import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type ProcessEncodedVaaContext = {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;
  guardianSet: PublicKey | null;
};

export type ProcessEncodedVaaDirective =
  | {
      closeVaaAccount: {};
    }
  | {
      write: {
        index: number;
        data: Buffer;
      };
    }
  | { verifySignaturesV1: {} };

export async function processEncodedVaaIx(
  program: CoreBridgeProgram,
  accounts: ProcessEncodedVaaContext,
  directive: ProcessEncodedVaaDirective
) {
  return program.methods
    .processEncodedVaa(directive)
    .accounts(accounts)
    .instruction();
}
