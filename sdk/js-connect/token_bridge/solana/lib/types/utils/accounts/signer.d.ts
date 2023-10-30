import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
export declare function deriveAuthoritySignerKey(tokenBridgeProgramId: PublicKeyInitData): PublicKey;
export declare function deriveCustodySignerKey(tokenBridgeProgramId: PublicKeyInitData): PublicKey;
export declare function deriveMintAuthorityKey(tokenBridgeProgramId: PublicKeyInitData): PublicKey;
