import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
import { TokenBridge } from '@wormhole-foundation/connect-sdk';
export declare function createCompleteTransferWrappedInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, feeRecipient?: PublicKeyInitData): TransactionInstruction;
export interface CompleteTransferWrappedAccounts {
    payer: PublicKey;
    config: PublicKey;
    vaa: PublicKey;
    claim: PublicKey;
    endpoint: PublicKey;
    to: PublicKey;
    toFees: PublicKey;
    mint: PublicKey;
    wrappedMeta: PublicKey;
    mintAuthority: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
export declare function getCompleteTransferWrappedAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, feeRecipient?: PublicKeyInitData): CompleteTransferWrappedAccounts;
