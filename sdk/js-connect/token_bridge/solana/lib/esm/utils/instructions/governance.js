import { PublicKey, SystemProgram, SYSVAR_CLOCK_PUBKEY, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { utils as CoreUtils } from '@wormhole-foundation/wormhole-connect-sdk-core-solana';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import { deriveEndpointKey, deriveTokenBridgeConfigKey } from '../accounts';
import { toChainId } from '@wormhole-foundation/connect-sdk';
export function createRegisterChainInstruction(tokenBridgeProgramId, wormholeProgramId, payer, vaa) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId).methods.registerChain();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getRegisterChainAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getRegisterChainAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa) {
    return {
        payer: new PublicKey(payer),
        config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        endpoint: deriveEndpointKey(tokenBridgeProgramId, toChainId(vaa.payload.foreignChain), vaa.payload.foreignAddress.toUint8Array()),
        vaa: CoreUtils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: CoreUtils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), toChainId(vaa.emitterChain), vaa.sequence),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
        wormholeProgram: new PublicKey(wormholeProgramId),
    };
}
export function createUpgradeContractInstruction(tokenBridgeProgramId, wormholeProgramId, payer, vaa, spill) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId).methods.upgradeContract();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getUpgradeContractAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, spill),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getUpgradeContractAccounts(tokenBridgeProgramId, wormholeProgramId, payer, vaa, spill) {
    return {
        payer: new PublicKey(payer),
        vaa: CoreUtils.derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: CoreUtils.deriveClaimKey(tokenBridgeProgramId, vaa.emitterAddress.toUint8Array(), toChainId(vaa.emitterChain), vaa.sequence),
        upgradeAuthority: CoreUtils.deriveUpgradeAuthorityKey(tokenBridgeProgramId),
        spill: new PublicKey(spill === undefined ? payer : spill),
        implementation: new PublicKey(vaa.payload.newContract),
        programData: utils.deriveUpgradeableProgramKey(tokenBridgeProgramId),
        tokenBridgeProgram: new PublicKey(tokenBridgeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        clock: SYSVAR_CLOCK_PUBKEY,
        bpfLoaderUpgradeable: utils.BpfLoaderUpgradeable.programId,
        systemProgram: SystemProgram.programId,
    };
}
