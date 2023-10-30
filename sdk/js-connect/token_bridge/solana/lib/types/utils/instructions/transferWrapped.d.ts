/// <reference types="node" />
import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createTransferWrappedInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, fromOwner: PublicKeyInitData, tokenChain: number, tokenAddress: Buffer | Uint8Array, nonce: number, amount: bigint, fee: bigint, targetAddress: Buffer | Uint8Array, targetChain: number): TransactionInstruction;
export interface TransferWrappedAccounts {
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
    rent: PublicKey;
    systemProgram: PublicKey;
    wormholeProgram: PublicKey;
    tokenProgram: PublicKey;
}
export declare function getTransferWrappedAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, from: PublicKeyInitData, fromOwner: PublicKeyInitData, tokenChain: number, tokenAddress: Buffer | Uint8Array): TransferWrappedAccounts;
