/// <reference types="node" />
import { AccountsCoder, Idl } from '@project-serum/anchor';
import { anchor } from '@wormhole-foundation/connect-sdk-solana';
export declare class WormholeAccountsCoder<A extends string = string> implements AccountsCoder {
    private idl;
    constructor(idl: Idl);
    encode<T = any>(accountName: A, account: T): Promise<Buffer>;
    decode<T = any>(accountName: A, ix: Buffer): T;
    decodeUnchecked<T = any>(accountName: A, ix: Buffer): T;
    memcmp(accountName: A, _appendData?: Buffer): any;
    size(idlAccount: anchor.IdlTypeDef): number;
}
export interface PostVAAData {
    version: number;
    guardianSetIndex: number;
    timestamp: number;
    nonce: number;
    emitterChain: number;
    emitterAddress: Buffer;
    sequence: bigint;
    consistencyLevel: number;
    payload: Buffer;
}
export declare function encodePostVaaData(account: PostVAAData): Buffer;
export declare function decodePostVaaAccount<T = any>(buf: Buffer): T;
