import { BN } from "@coral-xyz/anchor";
import {
  AccountInfo,
  Commitment,
  Connection,
  GetAccountInfoConfig,
  PublicKey,
} from "@solana/web3.js";

export enum EmitterType {
  Unset,
  Legacy,
  Executable,
}

export class EmitterSequence {
  sequence: BN;
  bump?: number;
  emitterType?: EmitterType;

  private constructor(sequence: BN, bump?: number, emitterType?: EmitterType) {
    this.sequence = sequence;
    this.bump = bump;
    this.emitterType = emitterType;
  }

  static address(programId: PublicKey, emitter: PublicKey): PublicKey {
    return PublicKey.findProgramAddressSync(
      [Buffer.from("Sequence"), emitter.toBuffer()],
      programId
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
    const accountInfo = await connection.getAccountInfo(address, commitmentOrConfig);
    if (accountInfo == null) {
      throw new Error(`Unable to find EmitterSequence account at ${address}`);
    }
    return EmitterSequence.fromAccountInfo(accountInfo);
  }

  static async fromPda(
    connection: Connection,
    programId: PublicKey,
    emitter: PublicKey
  ): Promise<EmitterSequence> {
    return EmitterSequence.fromAccountAddress(
      connection,
      EmitterSequence.address(programId, emitter)
    );
  }

  static deserialize(data: Buffer): EmitterSequence {
    if (data.length == 8) {
      const sequence = new BN(data.subarray(0, 8), undefined, "le");
      return new EmitterSequence(sequence);
    } else if (data.length == 10) {
      const sequence = new BN(data.subarray(0, 8), undefined, "le");
      const bump = data[8];
      const emitterType = data[9];
      return new EmitterSequence(sequence, bump, emitterType);
    } else {
      throw new Error("data.length != 8 or data.length != 10");
    }
  }
}
