import { PublicKey, } from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { deriveAuthoritySignerKey, deriveSenderAccountKey, deriveTokenBridgeConfigKey, deriveWrappedMetaKey, deriveWrappedMintKey, } from '../accounts';
export function createTransferWrappedWithPayloadInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, message, from, fromOwner, tokenChain, tokenAddress, nonce, amount, targetAddress, targetChain, payload) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId, connection).methods.transferWrappedWithPayload(nonce, amount, Buffer.from(targetAddress), targetChain, Buffer.from(payload), null);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferWrappedWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, fromOwner, tokenChain, tokenAddress),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getTransferWrappedWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, fromOwner, tokenChain, tokenAddress, cpiProgramId) {
    const mint = deriveWrappedMintKey(tokenBridgeProgramId, tokenChain, tokenAddress);
    const { wormholeBridge, wormholeMessage, wormholeEmitter, wormholeSequence, wormholeFeeCollector, clock, rent, systemProgram, } = utils.getPostMessageCpiAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message);
    return {
        payer: new PublicKey(payer),
        config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        from: new PublicKey(from),
        fromOwner: new PublicKey(fromOwner),
        mint: mint,
        wrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
        authoritySigner: deriveAuthoritySignerKey(tokenBridgeProgramId),
        wormholeBridge,
        wormholeMessage: wormholeMessage,
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector,
        clock,
        sender: new PublicKey(cpiProgramId === undefined ? payer : deriveSenderAccountKey(cpiProgramId)),
        rent,
        systemProgram,
        wormholeProgram: new PublicKey(wormholeProgramId),
        tokenProgram: TOKEN_PROGRAM_ID,
    };
}
