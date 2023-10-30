/// <reference types="node" />
import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createTransferNativeWithPayloadInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, mint: PublicKeyInitData, nonce: number, amount: bigint, targetAddress: Buffer | Uint8Array, targetChain: number, payload: Buffer | Uint8Array): TransactionInstruction;
export interface TransferNativeWithPayloadAccounts {
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
    sender: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
export declare function getTransferNativeWithPayloadAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, mint: PublicKeyInitData, cpiProgramId?: PublicKeyInitData): TransferNativeWithPayloadAccounts;
