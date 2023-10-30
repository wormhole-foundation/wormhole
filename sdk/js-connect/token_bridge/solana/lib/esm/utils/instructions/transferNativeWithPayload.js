import { PublicKey, } from '@solana/web3.js';
import { TOKEN_PROGRAM_ID } from '@solana/spl-token';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { deriveAuthoritySignerKey, deriveCustodySignerKey, deriveTokenBridgeConfigKey, deriveCustodyKey, deriveSenderAccountKey, } from '../accounts';
export function createTransferNativeWithPayloadInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint, nonce, amount, targetAddress, targetChain, payload) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId, connection).methods.transferNativeWithPayload(nonce, amount, Buffer.from(targetAddress), targetChain, Buffer.from(payload), null);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferNativeWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getTransferNativeWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint, cpiProgramId) {
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
        sender: new PublicKey(cpiProgramId === undefined ? payer : deriveSenderAccountKey(cpiProgramId)),
        rent,
        systemProgram,
        tokenProgram: TOKEN_PROGRAM_ID,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
