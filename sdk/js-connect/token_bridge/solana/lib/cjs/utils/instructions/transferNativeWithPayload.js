"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getTransferNativeWithPayloadAccounts = exports.createTransferNativeWithPayloadInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const spl_token_1 = require("@solana/spl-token");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const accounts_1 = require("../accounts");
function createTransferNativeWithPayloadInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint, nonce, amount, targetAddress, targetChain, payload) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId, connection).methods.transferNativeWithPayload(nonce, amount, Buffer.from(targetAddress), targetChain, Buffer.from(payload), null);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferNativeWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createTransferNativeWithPayloadInstruction = createTransferNativeWithPayloadInstruction;
function getTransferNativeWithPayloadAccounts(tokenBridgeProgramId, wormholeProgramId, payer, message, from, mint, cpiProgramId) {
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
        sender: new web3_js_1.PublicKey(cpiProgramId === undefined ? payer : (0, accounts_1.deriveSenderAccountKey)(cpiProgramId)),
        rent,
        systemProgram,
        tokenProgram: spl_token_1.TOKEN_PROGRAM_ID,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
    };
}
exports.getTransferNativeWithPayloadAccounts = getTransferNativeWithPayloadAccounts;
