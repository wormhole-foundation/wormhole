/// <reference types="node" />
import { Connection, PublicKey, Commitment, PublicKeyInitData } from '@solana/web3.js';
export declare function deriveGuardianSetKey(wormholeProgramId: PublicKeyInitData, index: number): PublicKey;
export declare function getGuardianSet(connection: Connection, wormholeProgramId: PublicKeyInitData, index: number, commitment?: Commitment): Promise<GuardianSetData>;
export declare class GuardianSetData {
    index: number;
    keys: Buffer[];
    creationTime: number;
    expirationTime: number;
    constructor(index: number, keys: Buffer[], creationTime: number, expirationTime: number);
    static deserialize(data: Buffer): GuardianSetData;
}
