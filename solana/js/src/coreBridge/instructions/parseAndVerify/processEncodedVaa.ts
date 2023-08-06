import { Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { coreBridgeProgram } from "../../anchor";
import { ProgramId } from "../../consts";

export class ProcessEncodedVaaContext {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;
  guardianSet: PublicKey | null;

  private constructor(
    writeAuthority: PublicKeyInitData,
    encodedVaa: PublicKeyInitData,
    guardianSet: PublicKeyInitData | null
  ) {
    this.writeAuthority = new PublicKey(writeAuthority);
    this.encodedVaa = new PublicKey(encodedVaa);
    this.guardianSet = guardianSet === null ? null : new PublicKey(guardianSet);
  }

  static new(
    writeAuthority: PublicKeyInitData,
    encodedVaa: PublicKeyInitData,
    guardianSet: PublicKeyInitData | null
  ) {
    return new ProcessEncodedVaaContext(
      writeAuthority,
      encodedVaa,
      guardianSet
    );
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    writeAuthority: PublicKeyInitData,
    encodedVaa: PublicKeyInitData,
    guardianSet: PublicKeyInitData | null,
    directive: ProcessEncodedVaaDirective
  ) {
    return processEncodedVaaIx(
      connection,
      programId,
      ProcessEncodedVaaContext.new(writeAuthority, encodedVaa, guardianSet),
      directive
    );
  }
}

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
  connection: Connection,
  programId: ProgramId,
  accounts: ProcessEncodedVaaContext,
  directive: ProcessEncodedVaaDirective
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods
    .processEncodedVaa(directive)
    .accounts(accounts)
    .instruction();
}
