import { Connection, PublicKey, PublicKeyInitData, TransactionInstruction } from '@solana/web3.js';
export declare function createAttestTokenInstruction(connection: Connection, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, mint: PublicKeyInitData, message: PublicKeyInitData, nonce: number): TransactionInstruction;
export interface AttestTokenAccounts {
    payer: PublicKey;
    config: PublicKey;
    mint: PublicKey;
    wrappedMeta: PublicKey;
    splMetadata: PublicKey;
    wormholeBridge: PublicKey;
    wormholeMessage: PublicKey;
    wormholeEmitter: PublicKey;
    wormholeSequence: PublicKey;
    wormholeFeeCollector: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    wormholeProgram: PublicKey;
}
export declare function getAttestTokenAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, mint: PublicKeyInitData, message: PublicKeyInitData): AttestTokenAccounts;
