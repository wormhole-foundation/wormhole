import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
import { TokenBridge } from '@wormhole-foundation/connect-sdk';
export declare function createCompleteTransferNativeInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, feeRecipient?: PublicKeyInitData): TransactionInstruction;
export interface CompleteTransferNativeAccounts {
    payer: PublicKey;
    config: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    endpoint: PublicKey;
    to: PublicKey;
    toFees: PublicKey;
    custody: PublicKey;
    mint: PublicKey;
    custodySigner: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
export declare function getCompleteTransferNativeAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, feeRecipient?: PublicKeyInitData): CompleteTransferNativeAccounts;
