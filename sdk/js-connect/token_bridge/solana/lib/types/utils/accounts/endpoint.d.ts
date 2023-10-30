/// <reference types="node" />
import { Connection, PublicKey, Commitment, PublicKeyInitData } from '@solana/web3.js';
import { ChainId } from '@wormhole-foundation/connect-sdk';
export declare function deriveEndpointKey(tokenBridgeProgramId: PublicKeyInitData, emitterChain: number | ChainId, emitterAddress: Buffer | Uint8Array | string): PublicKey;
export declare function getEndpointRegistration(connection: Connection, endpointKey: PublicKeyInitData, commitment?: Commitment): Promise<EndpointRegistration>;
export declare class EndpointRegistration {
    chain: ChainId;
    contract: Buffer;
    constructor(chain: number, contract: Buffer);
    static deserialize(data: Buffer): EndpointRegistration;
}
