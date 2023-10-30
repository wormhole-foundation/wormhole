"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCompleteTransferNativeAccounts = exports.createCompleteTransferNativeInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const spl_token_1 = require("@solana/spl-token");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const accounts_1 = require("../accounts");
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
function createCompleteTransferNativeInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId, connection).methods.completeNative();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getCompleteTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createCompleteTransferNativeInstruction = createCompleteTransferNativeInstruction;
function getCompleteTransferNativeAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, feeRecipient) {
    const mint = new web3_js_1.PublicKey(vaa.payload.token.address.toUint8Array());
    return {
        payer: new web3_js_1.PublicKey(payer),
        config: (0, accounts_1.deriveTokenBridgeConfigKey)(tokenBridgeProgramId),
        vaa: wormhole_connect_sdk_core_solana_1.utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: wormhole_connect_sdk_core_solana_1.utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.sequence),
        endpoint: (0, accounts_1.deriveEndpointKey)(tokenBridgeProgramId, (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.emitterAddress.toUint8Array()),
        to: new web3_js_1.PublicKey(vaa.payload.to.address.toUint8Array()),
        toFees: new web3_js_1.PublicKey(feeRecipient === undefined
            ? vaa.payload.to.address.toUint8Array()
            : feeRecipient),
        custody: (0, accounts_1.deriveCustodyKey)(tokenBridgeProgramId, mint),
        mint,
        custodySigner: (0, accounts_1.deriveCustodySignerKey)(tokenBridgeProgramId),
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
        tokenProgram: spl_token_1.TOKEN_PROGRAM_ID,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
    };
}
exports.getCompleteTransferNativeAccounts = getCompleteTransferNativeAccounts;
