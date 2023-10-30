/// <reference types="node" />
import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createTransferWrappedWithPayloadInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, fromOwner: PublicKeyInitData, tokenChain: number, tokenAddress: Buffer | Uint8Array, nonce: number, amount: bigint, targetAddress: Buffer | Uint8Array, targetChain: number, payload: Buffer | Uint8Array): TransactionInstruction;
export interface TransferWrappedWithPayloadAccounts {
    payer: PublicKey;
    config: PublicKey;
    from: PublicKey;
    fromOwner: PublicKey;
    mint: PublicKey;
    wrappedMeta: PublicKey;
    authoritySigner: PublicKey;
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
export declare function getTransferWrappedWithPayloadAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, fromOwner: PublicKeyInitData, tokenChain: number, tokenAddress: Buffer | Uint8Array, cpiProgramId?: PublicKeyInitData): TransferWrappedWithPayloadAccounts;
