/// <reference types="node" />
import { Connection, PublicKey, Commitment, PublicKeyInitData } from '@solana/web3.js';
export declare function deriveEmitterSequenceKey(emitter: PublicKeyInitData, wormholeProgramId: PublicKeyInitData): PublicKey;
export declare function getSequenceTracker(connection: Connection, emitter: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, commitment?: Commitment): Promise<SequenceTracker>;
export declare class SequenceTracker {
    sequence: bigint;
    constructor(sequence: bigint);
    static deserialize(data: Buffer): SequenceTracker;
    value(): bigint;
}
