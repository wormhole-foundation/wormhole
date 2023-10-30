import { PublicKey, } from '@solana/web3.js';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils as CoreUtils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import { deriveTokenBridgeConfigKey, deriveWrappedMetaKey } from '../accounts';
export function createAttestTokenInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, mint, message, nonce) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId, connection).methods.attestToken(nonce);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getAttestTokenAccounts(tokenBridgeProgramId, wormholeProgramId, payer, mint, message),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getAttestTokenAccounts(tokenBridgeProgramId, wormholeProgramId, payer, mint, message) {
    const { bridge: wormholeBridge, emitter: wormholeEmitter, sequence: wormholeSequence, feeCollector: wormholeFeeCollector, clock, rent, systemProgram, } = CoreUtils.getPostMessageAccounts(wormholeProgramId, payer, tokenBridgeProgramId, message);
    return {
        payer: new PublicKey(payer),
        config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        mint: new PublicKey(mint),
        wrappedMeta: deriveWrappedMetaKey(tokenBridgeProgramId, mint),
        splMetadata: utils.deriveSplTokenMetadataKey(mint),
        wormholeBridge,
        wormholeMessage: new PublicKey(message),
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector,
        clock,
        rent,
        systemProgram,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
