import { PublicKey } from "@solana/web3.js";
import { CoreBridgeProgram } from "../..";

export type ProcessMessageV1Context = {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
  closeAccountDestination: PublicKey | null;
};

export type ProcessMessageV1Directive =
  | {
      closeMessageAccount: {};
    }
  | {
      write: {
        index: number;
        data: Buffer;
      };
    }
  | {
      finalize: {};
    };

export async function processMessageV1Ix(
  program: CoreBridgeProgram,
  accounts: ProcessMessageV1Context,
  directive: ProcessMessageV1Directive
) {
  return program.methods.processMessageV1(directive).accounts(accounts).instruction();
}
