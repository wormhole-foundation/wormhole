"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getTransferWrappedAccounts = exports.createTransferWrappedInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const spl_token_1 = require("@solana/spl-token");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const accounts_1 = require("../accounts");
function createTransferWrappedInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, message, from, fromOwner, tokenChain, tokenAddress, nonce, amount, fee, targetAddress, targetChain) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId, connection).methods.transferWrapped(nonce, amount, fee, Buffer.from(targetAddress), targetChain);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferWrappedAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, fromOwner, tokenChain, tokenAddress),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createTransferWrappedInstruction = createTransferWrappedInstruction;
function getTransferWrappedAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, fromOwner, tokenChain, tokenAddress) {
    const mint = (0, accounts_1.deriveWrappedMintKey)(tokenBridgeProgramId, tokenChain, tokenAddress);
    const { wormholeBridge, wormholeMessage, wormholeEmitter, wormholeSequence, wormholeFeeCollector, clock, rent, systemProgram, } = wormhole_connect_sdk_core_solana_1.utils.getPostMessageCpiAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message);
    return {
        payer: new web3_js_1.PublicKey(payer),
        config: (0, accounts_1.deriveTokenBridgeConfigKey)(tokenBridgeProgramId),
        from: new web3_js_1.PublicKey(from),
        fromOwner: new web3_js_1.PublicKey(fromOwner),
        mint: mint,
        wrappedMeta: (0, accounts_1.deriveWrappedMetaKey)(tokenBridgeProgramId, mint),
        authoritySigner: (0, accounts_1.deriveAuthoritySignerKey)(tokenBridgeProgramId),
        wormholeBridge,
        wormholeMessage: wormholeMessage,
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector,
        clock,
        rent,
        systemProgram,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
        tokenProgram: spl_token_1.TOKEN_PROGRAM_ID,
    };
}
exports.getTransferWrappedAccounts = getTransferWrappedAccounts;
