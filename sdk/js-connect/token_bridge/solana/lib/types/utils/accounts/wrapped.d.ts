/// <reference types="node" />
import { Connection, PublicKey, Commitment, PublicKeyInitData } from '@solana/web3.js';
import { ChainId } from '@wormhole-foundation/connect-sdk';
export declare function deriveWrappedMintKey(tokenBridgeProgramId: PublicKeyInitData, tokenChain: number | ChainId, tokenAddress: Buffer | Uint8Array | string): PublicKey;
export declare function deriveWrappedMetaKey(tokenBridgeProgramId: PublicKeyInitData, mint: PublicKeyInitData): PublicKey;
export declare function getWrappedMeta(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, mint: PublicKeyInitData, commitment?: Commitment): Promise<WrappedMeta>;
export declare class WrappedMeta {
    chain: number;
    tokenAddress: Buffer;
    originalDecimals: number;
    constructor(chain: number, tokenAddress: Buffer, originalDecimals: number);
    static deserialize(data: Buffer): WrappedMeta;
}
