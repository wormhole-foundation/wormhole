/// <reference types="node" />
import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createTransferNativeInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, mint: PublicKeyInitData, nonce: number, amount: bigint, fee: bigint, targetAddress: Buffer | Uint8Array, targetChain: number): TransactionInstruction;
export interface TransferNativeAccounts {
    payer: PublicKey;
    config: PublicKey;
    from: PublicKey;
    mint: PublicKey;
    custody: PublicKey;
    authoritySigner: PublicKey;
    custodySigner: PublicKey;
    wormholeBridge: PublicKey;
    wormholeMessage: PublicKey;
    wormholeEmitter: PublicKey;
    wormholeSequence: PublicKey;
    wormholeFeeCollector: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
export declare function getTransferNativeAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, mint: PublicKeyInitData): TransferNativeAccounts;
