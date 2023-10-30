/// <reference types="node" />
import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
export declare function derivePostedVaaKey(wormholeProgramId: PublicKeyInitData, hash: Buffer): PublicKey;
