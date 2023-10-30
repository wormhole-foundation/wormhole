"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getAttestTokenAccounts = exports.createAttestTokenInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const accounts_1 = require("../accounts");
function createAttestTokenInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, mint, message, nonce) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId, connection).methods.attestToken(nonce);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getAttestTokenAccounts(tokenBridgeProgramId, wormholeProgramId, payer, mint, message),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createAttestTokenInstruction = createAttestTokenInstruction;
function getAttestTokenAccounts(tokenBridgeProgramId, wormholeProgramId, payer, mint, message) {
    const { bridge: wormholeBridge, emitter: wormholeEmitter, sequence: wormholeSequence, feeCollector: wormholeFeeCollector, clock, rent, systemProgram, } = wormhole_connect_sdk_core_solana_1.utils.getPostMessageAccounts(wormholeProgramId, payer, tokenBridgeProgramId, message);
    return {
        payer: new web3_js_1.PublicKey(payer),
        config: (0, accounts_1.deriveTokenBridgeConfigKey)(tokenBridgeProgramId),
        mint: new web3_js_1.PublicKey(mint),
        wrappedMeta: (0, accounts_1.deriveWrappedMetaKey)(tokenBridgeProgramId, mint),
        splMetadata: connect_sdk_solana_1.utils.deriveSplTokenMetadataKey(mint),
        wormholeBridge,
        wormholeMessage: new web3_js_1.PublicKey(message),
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector,
        clock,
        rent,
        systemProgram,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
    };
}
exports.getAttestTokenAccounts = getAttestTokenAccounts;
