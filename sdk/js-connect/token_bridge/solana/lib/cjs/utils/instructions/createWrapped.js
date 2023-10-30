"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getCreateWrappedAccounts = exports.createCreateWrappedInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const spl_token_1 = require("@solana/spl-token");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const accounts_1 = require("../accounts");
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
function createCreateWrappedInstruction(connection, tokenBridgeProgramId, wormholeProgramId, payer, vaa) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId, connection).methods.createWrapped();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getCreateWrappedAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createCreateWrappedInstruction = createCreateWrappedInstruction;
function getCreateWrappedAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa) {
    const mint = (0, accounts_1.deriveWrappedMintKey)(tokenBridgeProgramId, (0, connect_sdk_1.toChainId)(vaa.payload.token.chain), vaa.payload.token.address.toUint8Array());
    return {
        payer: new web3_js_1.PublicKey(payer),
        config: (0, accounts_1.deriveTokenBridgeConfigKey)(tokenBridgeProgramId),
        endpoint: (0, accounts_1.deriveEndpointKey)(tokenBridgeProgramId, (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.emitterAddress.toUint8Array()),
        vaa: wormhole_connect_sdk_core_solana_1.utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: wormhole_connect_sdk_core_solana_1.utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.sequence),
        mint,
        wrappedMeta: (0, accounts_1.deriveWrappedMetaKey)(tokenBridgeProgramId, mint),
        splMetadata: connect_sdk_solana_1.utils.deriveSplTokenMetadataKey(mint),
        mintAuthority: (0, accounts_1.deriveMintAuthorityKey)(tokenBridgeProgramId),
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
        tokenProgram: spl_token_1.TOKEN_PROGRAM_ID,
        splMetadataProgram: connect_sdk_solana_1.utils.SplTokenMetadataProgram.programId,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
    };
}
exports.getCreateWrappedAccounts = getCreateWrappedAccounts;
