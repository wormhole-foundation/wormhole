import {
  Connection,
  PublicKey,
  PublicKeyInitData,
  SystemProgram,
} from "@solana/web3.js";
import { coreBridgeProgram } from "../../anchor";
import { ProgramId } from "../../consts";

export class PostVaaV1Context {
  writeAuthority: PublicKey;
  vaa: PublicKey;
  postedVaa: PublicKey;
  systemProgram: PublicKey;

  private constructor(
    writeAuthority: PublicKeyInitData,
    vaa: PublicKeyInitData,
    postedVaa: PublicKeyInitData
  ) {
    this.writeAuthority = new PublicKey(writeAuthority);
    this.vaa = new PublicKey(vaa);
    this.postedVaa = new PublicKey(postedVaa);
    this.systemProgram = SystemProgram.programId;
  }

  static new(
    writeAuthority: PublicKeyInitData,
    vaa: PublicKeyInitData,
    postedVaa: PublicKeyInitData
  ) {
    return new PostVaaV1Context(writeAuthority, vaa, postedVaa);
  }

  static instruction(
    connection: Connection,
    programId: ProgramId,
    writeAuthority: PublicKeyInitData,
    vaa: PublicKeyInitData,
    postedVaa: PublicKeyInitData,
    directive: PostVaaV1Directive
  ) {
    return postVaaV1Ix(
      connection,
      programId,
      PostVaaV1Context.new(writeAuthority, vaa, postedVaa),
      directive
    );
  }
}

export type PostVaaV1Directive = { tryOnce: {} };

export async function postVaaV1Ix(
  connection: Connection,
  programId: ProgramId,
  accounts: PostVaaV1Context,
  directive: PostVaaV1Directive
) {
  const program = coreBridgeProgram(connection, programId);
  return program.methods.postVaaV1(directive).accounts(accounts).instruction();
}
