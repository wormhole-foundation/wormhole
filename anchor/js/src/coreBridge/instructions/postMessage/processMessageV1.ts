import { Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { coreBridgeProgram } from "../../anchor";
import { ProgramId } from "../../consts";

export class ProcessMessageV1Context {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;
  closeAccountDestination: PublicKey | null;

  private constructor(
    emitterAuthority: PublicKeyInitData,
    draftMessage: PublicKeyInitData,
    closeAccountDestination: PublicKeyInitData | null
  ) {
    this.emitterAuthority = new PublicKey(emitterAuthority);
    this.draftMessage = new PublicKey(draftMessage);
    this.closeAccountDestination =
      closeAccountDestination === null
        ? null
        : new PublicKey(closeAccountDestination);
  }

  static new(
    emitterAuthority: PublicKeyInitData,
    draftMessage: PublicKeyInitData,
    closeAccountDestination: PublicKeyInitData | null
  ) {
    return new ProcessMessageV1Context(
      emitterAuthority,
      draftMessage,
      closeAccountDestination
    );
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    emitterAuthority: PublicKeyInitData,
    draftMessage: PublicKeyInitData,
    closeAccountDestination: PublicKeyInitData | null,
    directive: ProcessMessageV1Directive
  ) {
    return processMessageV1Ix(
      connection,
      programId,
      ProcessMessageV1Context.new(
        emitterAuthority,
        draftMessage,
        closeAccountDestination
      ),
      directive
    );
  }
}

export type ProcessMessageV1Directive =
  | {
      closeMessageAccount: {};
    }
  | {
      write: {
        index: number;
        data: Buffer;
      };
    };

export async function processMessageV1Ix(
  connection: Connection,
  programId: ProgramId,
  accounts: ProcessMessageV1Context,
  directive: ProcessMessageV1Directive
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods
    .processMessageV1(directive)
    .accounts(accounts)
    .instruction();
}
