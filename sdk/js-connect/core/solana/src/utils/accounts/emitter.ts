import {
  Commitment,
  Connection,
  PublicKey,
  PublicKeyInitData,
} from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import {
  deriveEmitterSequenceKey,
  getSequenceTracker,
  SequenceTracker,
} from './sequence';

export interface EmitterAccounts {
  emitter: PublicKey;
  sequence: PublicKey;
}

export function deriveWormholeEmitterKey(
  emitterProgramId: PublicKeyInitData,
): PublicKey {
  return utils.deriveAddress([Buffer.from('emitter')], emitterProgramId);
}

export function getEmitterKeys(
  emitterProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
): EmitterAccounts {
  const emitter = deriveWormholeEmitterKey(emitterProgramId);
  return {
    emitter,
    sequence: deriveEmitterSequenceKey(emitter, wormholeProgramId),
  };
}

export async function getProgramSequenceTracker(
  connection: Connection,
  emitterProgramId: PublicKeyInitData,
  wormholeProgramId: PublicKeyInitData,
  commitment?: Commitment,
): Promise<SequenceTracker> {
  return getSequenceTracker(
    connection,
    deriveWormholeEmitterKey(emitterProgramId),
    wormholeProgramId,
    commitment,
  );
}
