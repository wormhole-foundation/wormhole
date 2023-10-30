import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';

export function deriveEmitterSequenceKey(
  emitter: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress(
    [Buffer.from('Sequence'), new PublicKey(emitter).toBytes()],
    wormholeProgramId,
  );
}

export async function getSequenceTracker(
  connection: Connection,
  emitter: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  commitment?: Commitment,
): Promise<SequenceTracker> {
  return connection
    .getAccountInfo(
      deriveEmitterSequenceKey(emitter, wormholeProgramId),
      commitment,
    )
    .then((info) => SequenceTracker.deserialize(utils.getAccountData(info)));
}

export class SequenceTracker {
  sequence: bigint;

  constructor(sequence: bigint) {
    this.sequence = sequence;
  }

  static deserialize(data: Buffer): SequenceTracker {
    if (data.length != 8) {
      throw new Error('data.length != 8');
    }
    return new SequenceTracker(data.readBigUInt64LE(0));
  }

  value(): bigint {
    return this.sequence;
  }
}
