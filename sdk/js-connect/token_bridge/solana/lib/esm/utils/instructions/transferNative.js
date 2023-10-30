import { PublicKey, } from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { deriveAuthoritySignerKey, deriveCustodySignerKey, deriveTokenBridgeConfigKey, deriveCustodyKey, } from '../accounts';
export function createTransferNativeInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint, nonce, amount, fee, targetAddress, targetChain) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId, connection).methods.transferNative(nonce, amount, fee, Buffer.from(targetAddress), targetChain);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint) {
    const { wormholeBridge, wormholeMessage, wormholeEmitter, wormholeSequence, wormholeFeeCollector, clock, rent, systemProgram, } = utils.getPostMessageCpiAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message);
    return {
        payer: new PublicKey(payer),
        config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        from: new PublicKey(from),
        mint: new PublicKey(mint),
        custody: deriveCustodyKey(tokenBridgeProgramId, mint),
        authoritySigner: deriveAuthoritySignerKey(tokenBridgeProgramId),
        custodySigner: deriveCustodySignerKey(tokenBridgeProgramId),
        wormholeBridge,
        wormholeMessage: wormholeMessage,
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector,
        clock,
        rent,
        systemProgram,
        tokenProgram: TOKEN_PROGRAM_ID,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
