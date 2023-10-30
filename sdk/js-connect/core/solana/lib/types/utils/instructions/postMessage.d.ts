import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
/** All accounts required to make a cross-program invocation with the Core Bridge program */
export interface PostMessageAccounts {
    bridge: PublicKey;
    message: PublicKey;
    emitter: PublicKey;
    sequence: PublicKey;
    payer: PublicKey;
    feeCollector: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
}
export declare function getPostMessageAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, emitterProgramId: PublicKeyInitData, message: PublicKeyInitData): PostMessageAccounts;
