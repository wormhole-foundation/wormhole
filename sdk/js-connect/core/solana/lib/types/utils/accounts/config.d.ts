/// <reference types="node" />
import { Connection, PublicKey, Commitment, PublicKeyInitData } from '@solana/web3.js';
export declare function deriveWormholeBridgeDataKey(wormholeProgramId: PublicKeyInitData): PublicKey;
export declare function getWormholeBridgeData(connection: Connection, wormholeProgramId: PublicKeyInitData, commitment?: Commitment): Promise<BridgeData>;
export declare class BridgeConfig {
    guardianSetExpirationTime: number;
    fee: bigint;
    constructor(guardianSetExpirationTime: number, fee: bigint);
    static deserialize(data: Buffer): BridgeConfig;
}
export declare class BridgeData {
    guardianSetIndex: number;
    lastLamports: bigint;
    config: BridgeConfig;
    constructor(guardianSetIndex: number, lastLamports: bigint, config: BridgeConfig);
    static deserialize(data: Buffer): BridgeData;
}
