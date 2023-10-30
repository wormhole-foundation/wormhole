import { Commitment, Connection, PublicKey, PublicKeyInitData } from '@solana/web3.js';
import { SequenceTracker } from './sequence';
export interface EmitterAccounts {
    emitter: PublicKey;
    sequence: PublicKey;
}
export declare function deriveWormholeEmitterKey(emitterProgramId: PublicKeyInitData): PublicKey;
export declare function getEmitterKeys(emitterProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData): EmitterAccounts;
export declare function getProgramSequenceTracker(connection: Connection, emitterProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, commitment?: Commitment): Promise<SequenceTracker>;
