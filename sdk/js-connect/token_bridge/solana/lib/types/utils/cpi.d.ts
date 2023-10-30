/// <reference types="node" />
import { PublicKey, PublicKeyInitData } from '@solana/web3.js';
import { TokenBridge } from '@wormhole-foundation/connect-sdk';
export interface TokenBridgeBaseDerivedAccounts {
    /**
     * seeds = ["config"], seeds::program = tokenBridgeProgram
     */
    tokenBridgeConfig: PublicKey;
}
export interface TokenBridgeBaseNativeDerivedAccounts extends TokenBridgeBaseDerivedAccounts {
    /**
     * seeds = ["custody_signer"], seeds::program = tokenBridgeProgram
     */
    tokenBridgeCustodySigner: PublicKey;
}
export interface TokenBridgeBaseSenderDerivedAccounts extends TokenBridgeBaseDerivedAccounts {
    /**
     * seeds = ["authority_signer"], seeds::program = tokenBridgeProgram
     */
    tokenBridgeAuthoritySigner: PublicKey;
    /**
     * seeds = ["sender"], seeds::program = cpiProgramId
     */
    tokenBridgeSender: PublicKey;
    /**
     * seeds = ["Bridge"], seeds::program = wormholeProgram
     */
    wormholeBridge: PublicKey;
    /**
     * seeds = ["emitter"], seeds::program = tokenBridgeProgram
     */
    tokenBridgeEmitter: PublicKey;
    /**
     * seeds = ["Sequence", tokenBridgeEmitter], seeds::program = wormholeProgram
     */
    tokenBridgeSequence: PublicKey;
    /**
     * seeds = ["fee_collector"], seeds::program = wormholeProgram
     */
    wormholeFeeCollector: PublicKey;
}
export interface TokenBridgeNativeSenderDerivedAccounts extends TokenBridgeBaseNativeDerivedAccounts, TokenBridgeBaseSenderDerivedAccounts {
}
export interface TokenBridgeWrappedSenderDerivedAccounts extends TokenBridgeBaseSenderDerivedAccounts {
}
export interface TokenBridgeBaseRedeemerDerivedAccounts extends TokenBridgeBaseDerivedAccounts {
    /**
     * seeds = ["redeemer"], seeds::program = cpiProgramId
     */
    tokenBridgeRedeemer: PublicKey;
}
export interface TokenBridgeNativeRedeemerDerivedAccounts extends TokenBridgeBaseNativeDerivedAccounts, TokenBridgeBaseRedeemerDerivedAccounts {
}
export interface TokenBridgeWrappedRedeemerDerivedAccounts extends TokenBridgeBaseRedeemerDerivedAccounts {
    /**
     * seeds = ["mint_signer"], seeds::program = tokenBridgeProgram
     */
    tokenBridgeMintAuthority: PublicKey;
}
export interface TokenBridgeDerivedAccounts extends TokenBridgeNativeSenderDerivedAccounts, TokenBridgeWrappedSenderDerivedAccounts, TokenBridgeNativeRedeemerDerivedAccounts, TokenBridgeWrappedRedeemerDerivedAccounts {
}
/**
 * Generate Token Bridge PDAs.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @returns
 */
export declare function getTokenBridgeDerivedAccounts(cpiProgramId: PublicKeyInitData, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData): TokenBridgeDerivedAccounts;
export interface TransferNativeWithPayloadCpiAccounts extends TokenBridgeNativeSenderDerivedAccounts {
    payer: PublicKey;
    /**
     * seeds = [mint], seeds::program = tokenBridgeProgram
     */
    tokenBridgeCustody: PublicKey;
    /**
     * Token account where tokens reside
     */
    fromTokenAccount: PublicKey;
    mint: PublicKey;
    wormholeMessage: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
/**
 * Generate accounts needed to perform `transfer_wrapped_with_payload` instruction
 * as cross-program invocation.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param message
 * @param fromTokenAccount
 * @param mint
 * @returns
 */
export declare function getTransferNativeWithPayloadCpiAccounts(cpiProgramId: PublicKeyInitData, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, fromTokenAccount: PublicKeyInitData, mint: PublicKeyInitData): TransferNativeWithPayloadCpiAccounts;
export interface TransferWrappedWithPayloadCpiAccounts extends TokenBridgeWrappedSenderDerivedAccounts {
    payer: PublicKey;
    /**
     * Token account where tokens reside
     */
    fromTokenAccount: PublicKey;
    /**
     * Token account owner (usually cpiProgramId)
     */
    fromTokenAccountOwner: PublicKey;
    /**
     * seeds = ["wrapped", token_chain, token_address], seeds::program = tokenBridgeProgram
     */
    tokenBridgeWrappedMint: PublicKey;
    /**
     * seeds = ["meta", mint], seeds::program = tokenBridgeProgram
     */
    tokenBridgeWrappedMeta: PublicKey;
    wormholeMessage: PublicKey;
    clock: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
/**
 * Generate accounts needed to perform `transfer_wrapped_with_payload` instruction
 * as cross-program invocation.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param message
 * @param fromTokenAccount
 * @param tokenChain
 * @param tokenAddress
 * @param [fromTokenAccountOwner]
 * @returns
 */
export declare function getTransferWrappedWithPayloadCpiAccounts(cpiProgramId: PublicKeyInitData, tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, message: PublicKeyInitData, fromTokenAccount: PublicKeyInitData, tokenChain: number, tokenAddress: Buffer | Uint8Array, fromTokenAccountOwner?: PublicKeyInitData): TransferWrappedWithPayloadCpiAccounts;
export interface CompleteTransferNativeWithPayloadCpiAccounts extends TokenBridgeNativeRedeemerDerivedAccounts {
    payer: PublicKey;
    /**
     * seeds = ["PostedVAA", vaa_hash], seeds::program = wormholeProgram
     */
    vaa: PublicKey;
    /**
     * seeds = [emitter_address, emitter_chain, sequence], seeds::program = tokenBridgeProgram
     */
    tokenBridgeClaim: PublicKey;
    /**
     * seeds = [emitter_chain, emitter_address], seeds::program = tokenBridgeProgram
     */
    tokenBridgeForeignEndpoint: PublicKey;
    /**
     * Token account to receive tokens
     */
    toTokenAccount: PublicKey;
    toFeesTokenAccount: PublicKey;
    /**
     * seeds = [mint], seeds::program = tokenBridgeProgram
     */
    tokenBridgeCustody: PublicKey;
    mint: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
/**
 * Generate accounts needed to perform `complete_native_with_payload` instruction
 * as cross-program invocation.
 *
 * Note: `toFeesTokenAccount` is the same as `toTokenAccount`. For your program,
 * you only need to pass your `toTokenAccount` into the complete transfer
 * instruction for the `toFeesTokenAccount`.
 *
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param vaa
 * @param toTokenAccount
 * @returns
 */
export declare function getCompleteTransferNativeWithPayloadCpiAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, toTokenAccount: PublicKeyInitData): CompleteTransferNativeWithPayloadCpiAccounts;
export interface CompleteTransferWrappedWithPayloadCpiAccounts extends TokenBridgeWrappedRedeemerDerivedAccounts {
    payer: PublicKey;
    /**
     * seeds = ["PostedVAA", vaa_hash], seeds::program = wormholeProgram
     */
    vaa: PublicKey;
    /**
     * seeds = [emitter_address, emitter_chain, sequence], seeds::program = tokenBridgeProgram
     */
    tokenBridgeClaim: PublicKey;
    /**
     * seeds = [emitter_chain, emitter_address], seeds::program = tokenBridgeProgram
     */
    tokenBridgeForeignEndpoint: PublicKey;
    /**
     * Token account to receive tokens
     */
    toTokenAccount: PublicKey;
    toFeesTokenAccount: PublicKey;
    /**
     * seeds = ["wrapped", token_chain, token_address], seeds::program = tokenBridgeProgram
     */
    tokenBridgeWrappedMint: PublicKey;
    /**
     * seeds = ["meta", mint], seeds::program = tokenBridgeProgram
     */
    tokenBridgeWrappedMeta: PublicKey;
    rent: PublicKey;
    systemProgram: PublicKey;
    tokenProgram: PublicKey;
    wormholeProgram: PublicKey;
}
/**
 * Generate accounts needed to perform `complete_wrapped_with_payload` instruction
 * as cross-program invocation.
 *
 * Note: `toFeesTokenAccount` is the same as `toTokenAccount`. For your program,
 * you only need to pass your `toTokenAccount` into the complete transfer
 * instruction for the `toFeesTokenAccount`.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @param payer
 * @param vaa
 * @returns
 */
export declare function getCompleteTransferWrappedWithPayloadCpiAccounts(tokenBridgeProgramId: PublicKeyInitData, wormholeProgramId: PublicKeyInitData, payer: PublicKeyInitData, vaa: TokenBridge.VAA<'Transfer' | 'TransferWithPayload'>, toTokenAccount: PublicKeyInitData): CompleteTransferWrappedWithPayloadCpiAccounts;
