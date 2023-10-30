import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { PublicKey, SystemProgram, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { deriveAuthoritySignerKey, deriveCustodyKey, deriveCustodySignerKey, deriveEndpointKey, deriveMintAuthorityKey, deriveRedeemerAccountKey, deriveSenderAccountKey, deriveTokenBridgeConfigKey, deriveWrappedMetaKey, deriveWrappedMintKey, } from './accounts';
import { getTransferNativeWithPayloadAccounts, getTransferWrappedWithPayloadAccounts, } from './instructions';
import { toChainId } from '@wormhole-foundation/connect-sdk';
/**
 * Generate Token Bridge PDAs.
 *
 * @param cpiProgramId
 * @param tokenBridgeProgramId
 * @param wormholeProgramId
 * @returns
 */
export function getTokenBridgeDerivedAccounts(cpiProgramId, tokenBridgeProgramId, wormholeProgramId) {
    const { wormholeEmitter: tokenBridgeEmitter, wormholeBridge, wormholeFeeCollector, wormholeSequence: tokenBridgeSequence, } = utils.getWormholeDerivedAccounts(tokenBridgeProgramId, wormholeProgramId);
    return {
        tokenBridgeConfig: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        tokenBridgeAuthoritySigner: deriveAuthoritySignerKey(tokenBridgeProgramId),
        tokenBridgeCustodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
        tokenBridgeMintAuthority: deriveMintAuthorityKey(tokenBridgeProgramId),
        tokenBridgeSender: deriveSenderAccountKey(cpiProgramId),
        tokenBridgeRedeemer: deriveRedeemerAccountKey(cpiProgramId),
        wormholeBridge,
        tokenBridgeEmitter,
        wormholeFeeCollector,
        tokenBridgeSequence,
    };
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
export function getTransferNativeWithPayloadCpiAccounts(cpiProgramId, tokenBridgeProgramId, wormholeProgramId, payer, message, fromTokenAccount, mint) {
    const accounts = getTransferNativeWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, fromTokenAccount, mint, cpiProgramId);
    return {
        payer: accounts.payer,
        tokenBridgeConfig: accounts.config,
        fromTokenAccount: accounts.from,
        mint: accounts.mint,
        tokenBridgeCustody: accounts.custody,
        tokenBridgeAuthoritySigner: accounts.authoritySigner,
        tokenBridgeCustodySigner: accounts.custodySigner,
        wormholeBridge: accounts.wormholeBridge,
        wormholeMessage: accounts.wormholeMessage,
        tokenBridgeEmitter: accounts.wormholeEmitter,
        tokenBridgeSequence: accounts.wormholeSequence,
        wormholeFeeCollector: accounts.wormholeFeeCollector,
        clock: accounts.clock,
        tokenBridgeSender: accounts.sender,
        rent: accounts.rent,
        systemProgram: accounts.systemProgram,
        tokenProgram: accounts.tokenProgram,
        wormholeProgram: accounts.wormholeProgram,
    };
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
export function getTransferWrappedWithPayloadCpiAccounts(cpiProgramId, tokenBridgeProgramId, wormholeProgramId, payer, message, fromTokenAccount, tokenChain, tokenAddress, fromTokenAccountOwner) {
    const accounts = getTransferWrappedWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, fromTokenAccount, fromTokenAccountOwner === undefined ? cpiProgramId : fromTokenAccountOwner, tokenChain, tokenAddress, cpiProgramId);
    return {
        payer: accounts.payer,
        tokenBridgeConfig: accounts.config,
        fromTokenAccount: accounts.from,
        fromTokenAccountOwner: accounts.fromOwner,
        tokenBridgeWrappedMint: accounts.mint,
        tokenBridgeWrappedMeta: accounts.wrappedMeta,
        tokenBridgeAuthoritySigner: accounts.authoritySigner,
        wormholeBridge: accounts.wormholeBridge,
        wormholeMessage: accounts.wormholeMessage,
        tokenBridgeEmitter: accounts.wormholeEmitter,
        tokenBridgeSequence: accounts.wormholeSequence,
        wormholeFeeCollector: accounts.wormholeFeeCollector,
        clock: accounts.clock,
        tokenBridgeSender: accounts.sender,
        rent: accounts.rent,
        systemProgram: accounts.systemProgram,
        tokenProgram: accounts.tokenProgram,
        wormholeProgram: accounts.wormholeProgram,
    };
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
export function getCompleteTransferNativeWithPayloadCpiAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, toTokenAccount) {
    const mint = new PublicKey(vaa.payload.token.address.toUint8Array());
    const cpiProgramId = new PublicKey(vaa.payload.to.address.toUint8Array());
    return {
        payer: new PublicKey(payer),
        tokenBridgeConfig: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        vaa: utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        tokenBridgeClaim: utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), toChainId(vaa.emitterChain), vaa.sequence),
        tokenBridgeForeignEndpoint: deriveEndpointKey(tokenBridgeProgramId, toChainId(vaa.emitterChain), vaa.emitterAddress.toUint8Array()),
        toTokenAccount: new PublicKey(toTokenAccount),
        tokenBridgeRedeemer: deriveRedeemerAccountKey(cpiProgramId),
        toFeesTokenAccount: new PublicKey(toTokenAccount),
        tokenBridgeCustody: deriveCustodyKey(tokenBridgeProgramId, mint),
        mint,
        tokenBridgeCustodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
        tokenProgram: TOKEN_PROGRAM_ID,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
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
export function getCompleteTransferWrappedWithPayloadCpiAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, toTokenAccount) {
    const mint = deriveWrappedMintKey(tokenBridgeProgramId, toChainId(vaa.payload.token.chain), vaa.payload.token.address.toUint8Array());
    const cpiProgramId = new PublicKey(vaa.payload.to.address.toUint8Array());
    return {
        payer: new PublicKey(payer),
        tokenBridgeConfig: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        vaa: utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        tokenBridgeClaim: utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), toChainId(vaa.emitterChain), vaa.sequence),
        tokenBridgeForeignEndpoint: deriveEndpointKey(tokenBridgeProgramId, toChainId(vaa.emitterChain), vaa.emitterAddress.toUint8Array()),
        toTokenAccount: new PublicKey(toTokenAccount),
        tokenBridgeRedeemer: deriveRedeemerAccountKey(cpiProgramId),
        toFeesTokenAccount: new PublicKey(toTokenAccount),
        tokenBridgeWrappedMint: mint,
        tokenBridgeWrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
        tokenBridgeMintAuthority: deriveMintAuthorityKey(tokenBridgeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
        tokenProgram: TOKEN_PROGRAM_ID,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
