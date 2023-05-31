import { Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { coreBridgeProgram } from "../../anchor";
import { ProgramId } from "../../consts";

export class InitMessageV1Context {
  emitterAuthority: PublicKey;
  draftMessage: PublicKey;

  private constructor(
    emitterAuthority: PublicKeyInitData,
    draftMessage: PublicKeyInitData
  ) {
    this.emitterAuthority = new PublicKey(emitterAuthority);
    this.draftMessage = new PublicKey(draftMessage);
  }

  static new(
    emitterAuthority: PublicKeyInitData,
    draftMessage: PublicKeyInitData
  ) {
    return new InitMessageV1Context(emitterAuthority, draftMessage);
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    emitterAuthority: PublicKeyInitData,
    draftMessage: PublicKeyInitData,
    args: InitMessageV1Args
  ) {
    return initMessageV1Ix(
      connection,
      programId,
      InitMessageV1Context.new(emitterAuthority, draftMessage),
      args
    );
  }
}

export type InitMessageV1Args = {
  cpiProgramId: PublicKey | null;
};

export async function initMessageV1Ix(
  connection: Connection,
  programId: ProgramId,
  accounts: InitMessageV1Context,
  args: InitMessageV1Args
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods.initMessageV1(args).accounts(accounts).instruction();
}
