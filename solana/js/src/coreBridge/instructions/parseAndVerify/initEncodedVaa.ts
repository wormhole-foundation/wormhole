import { Connection, PublicKey, PublicKeyInitData } from "@solana/web3.js";
import { coreBridgeProgram } from "../../anchor";
import { ProgramId } from "../../consts";

export class InitEncodedVaaContext {
  writeAuthority: PublicKey;
  encodedVaa: PublicKey;

  private constructor(
    writeAuthority: PublicKeyInitData,
    encodedVaa: PublicKeyInitData
  ) {
    this.writeAuthority = new PublicKey(writeAuthority);
    this.encodedVaa = new PublicKey(encodedVaa);
  }

  static new(writeAuthority: PublicKeyInitData, encodedVaa: PublicKeyInitData) {
    return new InitEncodedVaaContext(writeAuthority, encodedVaa);
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    writeAuthority: PublicKeyInitData,
    encodedVaa: PublicKeyInitData
  ) {
    return initEncodedVaaIx(
      connection,
      programId,
      InitEncodedVaaContext.new(writeAuthority, encodedVaa)
    );
  }
}

export async function initEncodedVaaIx(
  connection: Connection,
  programId: ProgramId,
  accounts: InitEncodedVaaContext
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods.initEncodedVaa().accounts(accounts).instruction();
}
