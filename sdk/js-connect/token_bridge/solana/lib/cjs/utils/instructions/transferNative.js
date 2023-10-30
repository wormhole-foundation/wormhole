"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getTransferNativeAccounts = exports.createTransferNativeInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const spl_token_1 = require("@solana/spl-token");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const accounts_1 = require("../accounts");
function createTransferNativeInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint, nonce, amount, fee, targetAddress, targetChain) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId, connection).methods.transferNative(nonce, amount, fee, Buffer.from(targetAddress), targetChain);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createTransferNativeInstruction = createTransferNativeInstruction;
function getTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint) {
    const { wormholeBridge, wormholeMessage, wormholeEmitter, wormholeSequence, wormholeFeeCollector, clock, rent, systemProgram, } = wormhole_connect_sdk_core_solana_1.utils.getPostMessageCpiAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message);
    return {
        payer: new web3_js_1.PublicKey(payer),
        config: (0, accounts_1.deriveTokenBridgeConfigKey)(tokenBridgeProgramId),
        from: new web3_js_1.PublicKey(from),
        mint: new web3_js_1.PublicKey(mint),
        custody: (0, accounts_1.deriveCustodyKey)(tokenBridgeProgramId, mint),
        authoritySigner: (0, accounts_1.deriveAuthoritySignerKey)(tokenBridgeProgramId),
        custodySigner: (0, accounts_1.deriveCustodySignerKey)(tokenBridgeProgramId),
        wormholeBridge,
        wormholeMessage: wormholeMessage,
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector,
        clock,
        rent,
        systemProgram,
        tokenProgram: spl_token_1.TOKEN_PROGRAM_ID,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
    };
}
exports.getTransferNativeAccounts = getTransferNativeAccounts;
