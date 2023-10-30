/// <reference types="node" />
import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createInitializeInstruction(connection: Connection, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, guardianSetExpirationTime: number, fee: bigint, initialGuardians: Buffer[]): TransactionInstruction;
export interface InitializeAccounts {
    bridge: PublicKey;
    guardianSet: PublicKey;
    feeCollector: PublicKey;
    payer: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
}
export declare function getInitializeAccounts(wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData): InitializeAccounts;
