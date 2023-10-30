import { PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createInitializeInstruction(tokenBridgeProgramId: PublicKeyInitData, payer: PublicKeyInitData, wormholeProgramId: PublicKeyInitData): TransactionInstruction;
export interface InitializeAccounts {
    payer: PublicKey;
    config: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
}
export declare function getInitializeAccounts(tokenBridgeProgramId: PublicKeyInitData, payer: PublicKeyInitData): InitializeAccounts;
