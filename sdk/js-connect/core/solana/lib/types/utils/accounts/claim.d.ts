/// <reference types="node" />
import { Commitment, Connection, PublicKey, PublicKeyInitData } from '@solana/web3.js';
export declare function deriveClaimKey(programId: PublicKeyInitData, emitterAddress: Buffer | Uint8Array | string, emitterChain: number, sequence: bigint | number): PublicKey;
export declare function getClaim(connection: Connection, programId: PublicKeyInitData, emitterAddress: Buffer | Uint8Array | string, emitterChain: number, sequence: bigint | number, commitment?: Commitment): Promise<boolean>;
