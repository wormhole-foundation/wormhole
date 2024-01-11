import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from "@solana/web3.js";
import { deriveAddress, getAccountData } from "../../utils";

export function deriveEmitterSequenceKey(
  emitter: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData
): PublicKey {
  return deriveAddress(
    [Buffer.from("Sequence"), new PublicKey(emitter).toBytes()],
    wormholeProgramId
  );
}

export async function getSequenceTracker(
  connection: Connection,
  emitter: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  commitment?: Commitment
): Promise<SequenceTracker> {
  return connection
    .getAccountInfo(
      deriveEmitterSequenceKey(emitter, wormholeProgramId),
      commitment
    )
    .then((info) => SequenceTracker.deserialize(getAccountData(info)));
}

export class SequenceTracker {
  sequence: bigint;
  bump?: number;
  emitterType?: number;

  constructor(sequence: bigint, bump?: number, emitterType?: number) {
    this.sequence = sequence;
    this.bump = bump;
    this.emitterType = emitterType;
  }

  static deserialize(data: Buffer): SequenceTracker {
    if (data.length !== 8 && data.length !== 10) {
      throw new Error("data.length != 8 or data.length != 10");
    }

    let bump, emitterType;
    const sequence = data.readBigUInt64LE(0);

    if (data.length === 10) {
      bump = data[8];
      emitterType = data[9];
    }

    return new SequenceTracker(sequence, bump, emitterType);
  }

  value(): bigint {
    return this.sequence;
  }
}
