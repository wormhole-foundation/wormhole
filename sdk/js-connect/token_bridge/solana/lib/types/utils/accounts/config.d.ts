/// <reference types="node" />
import { Connection, PublicKey, Commitment, PublicKeyInitData } from '@solana/web3.js';
export declare function deriveTokenBridgeConfigKey(tokenBridgeProgramId: PublicKeyInitData): PublicKey;
export declare function getTokenBridgeConfig(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, commitment?: Commitment): Promise<TokenBridgeConfig>;
export declare class TokenBridgeConfig {
    wormhole: PublicKey;
    constructor(wormholeProgramId: Buffer);
    static deserialize(data: Buffer): TokenBridgeConfig;
}
