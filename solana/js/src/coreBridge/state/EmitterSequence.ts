import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
  PublicKeyInitData,
} from "@solana/web3.js";
import { ProgramId } from "../consts";
import { getProgramPubkey } from "../utils";

export class EmitterSequence {
  sequence: BN;

  private constructor(sequence: BN) {
    this.sequence = sequence;
  }

  static address(programId: ProgramId, emitter: PublicKeyInitData): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("Sequence"), new PublicKey(emitter).toBuffer()],
      getProgramPubkey(programId)
    )[0];
  }

  static fromAccountInfo(info: AccountInfo<Buffer>): EmitterSequence {
    return EmitterSequence.deserialize(info.data);
  }

  static async fromAccountAddress(
    connection: Connection,
    address: PublicKey,
    commitmentOrConfig?: Commitment | GetAccountInfoConfig
  ): Promise<EmitterSequence> {
    const accountInfo = await connection.getAccountInfo(
      address,
      commitmentOrConfig
    );
    if (accountInfo == null) {
      throw new Error(`Unable to find EmitterSequence account at ${address}`);
    }
    return EmitterSequence.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: ProgramId,
    emitter: PublicKeyInitData
  ): Promise<EmitterSequence> {
    return EmitterSequence.fromAccountAddress(
      connection,
      EmitterSequence.address(programId, emitter)
    );
  }

  static deserialize(data: Buffer): EmitterSequence {
    if (data.length != 8) {
      throw new Error("data.length != 8");
    }

    const sequence = new BN(data.subarray(0), undefined, "le");
    return new EmitterSequence(sequence);
  }
}
