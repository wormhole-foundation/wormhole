"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getUpgradeContractAccounts = exports.createUpgradeContractInstruction = exports.getRegisterChainAccounts = exports.createRegisterChainInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const program_1 = require("../program");
const wormhole_connect_sdk_core_solana_1 = require("@wormhole-foundation/wormhole-connect-sdk-core-solana");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const accounts_1 = require("../accounts");
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
function createRegisterChainInstruction(tokenBridgeProgramId, wormholeProgramId, payer, vaa) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId).methods.registerChain();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getRegisterChainAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createRegisterChainInstruction = createRegisterChainInstruction;
function getRegisterChainAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa) {
    return {
        payer: new web3_js_1.PublicKey(payer),
        config: (0, accounts_1.deriveTokenBridgeConfigKey)(tokenBridgeProgramId),
        endpoint: (0, accounts_1.deriveEndpointKey)(tokenBridgeProgramId, (0, connect_sdk_1.toChainId)(vaa.payload.foreignChain), vaa.payload.foreignAddress.toUint8Array()),
        vaa: wormhole_connect_sdk_core_solana_1.utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: wormhole_connect_sdk_core_solana_1.utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.sequence),
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
        wormholeProgram: new web3_js_1.PublicKey(wormholeProgramId),
    };
}
exports.getRegisterChainAccounts = getRegisterChainAccounts;
function createUpgradeContractInstruction(tokenBridgeProgramId, wormholeProgramId, payer, vaa, spill) {
    const methods = (0, program_1.createReadOnlyTokenBridgeProgramInterface)(tokenBridgeProgramId).methods.upgradeContract();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getUpgradeContractAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, spill),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
exports.createUpgradeContractInstruction = createUpgradeContractInstruction;
function getUpgradeContractAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, spill) {
    return {
        payer: new web3_js_1.PublicKey(payer),
        vaa: wormhole_connect_sdk_core_solana_1.utils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: wormhole_connect_sdk_core_solana_1.utils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), (0, connect_sdk_1.toChainId)(vaa.emitterChain), vaa.sequence),
        upgradeAuthority: wormhole_connect_sdk_core_solana_1.utils.deriveUpgradeAuthorityKey(tokenBridgeProgramId),
        spill: new web3_js_1.PublicKey(spill === undefined ? payer : spill),
        implementation: new web3_js_1.PublicKey(vaa.payload.newContract),
        programData: connect_sdk_solana_1.utils.deriveUpgradeableProgramKey(tokenBridgeProgramId),
        tokenBridgeProgram: new web3_js_1.PublicKey(tokenBridgeProgramId),
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        clock: web3_js_1.SYSVAR_CLOCK_PUBKEY,
        bpfLoaderUpgradeable: connect_sdk_solana_1.utils.BpfLoaderUpgradeable.programId,
        systemProgram: web3_js_1.SystemProgram.programId,
    };
}
exports.getUpgradeContractAccounts = getUpgradeContractAccounts;
