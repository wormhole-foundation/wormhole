import { PublicKey, SystemProgram, SYSVAR_CLOCK_PUBKEY, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { toChainId } from '@wormhole-foundation/connect-sdk';
import { createReadOnlyWormholeProgramInterface } from '../program';
import { deriveWormholeBridgeDataKey, deriveClaimKey, deriveFeeCollectorKey, deriveGuardianSetKey, derivePostedVaaKey, deriveUpgradeAuthorityKey, } from '../accounts';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function createSetFeesInstruction(connection, wormholeProgramId, payer, vaa) {
    const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId, connection).methods.setFees();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getSetFeesAccounts(wormholeProgramId, payer, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getSetFeesAccounts(wormholeProgramId, payer, vaa) {
    return {
        payer: new PublicKey(payer),
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        vaa: derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: deriveClaimKey(wormholeProgramId, vaa.emitterAddress.toString(), toChainId(vaa.emitterChain), vaa.sequence),
        systemProgram: SystemProgram.programId,
    };
}
export function createTransferFeesInstruction(connection, wormholeProgramId, payer, recipient, vaa) {
    const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId, connection).methods.transferFees();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getTransferFeesAccounts(wormholeProgramId, payer, recipient, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getTransferFeesAccounts(wormholeProgramId, payer, recipient, vaa) {
    return {
        payer: new PublicKey(payer),
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        vaa: derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: deriveClaimKey(wormholeProgramId, vaa.emitterAddress.toString(), toChainId(vaa.emitterChain), vaa.sequence),
        feeCollector: deriveFeeCollectorKey(wormholeProgramId),
        recipient: new PublicKey(recipient),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
    };
}
export function createUpgradeGuardianSetInstruction(connection, wormholeProgramId, payer, vaa) {
    const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId, connection).methods.upgradeGuardianSet();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getUpgradeGuardianSetAccounts(wormholeProgramId, payer, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getUpgradeGuardianSetAccounts(wormholeProgramId, payer, vaa) {
    return {
        payer: new PublicKey(payer),
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        vaa: derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: deriveClaimKey(wormholeProgramId, vaa.emitterAddress.toString(), toChainId(vaa.emitterChain), vaa.sequence),
        guardianSetOld: deriveGuardianSetKey(wormholeProgramId, vaa.guardianSet),
        guardianSetNew: deriveGuardianSetKey(wormholeProgramId, vaa.guardianSet + 1),
        systemProgram: SystemProgram.programId,
    };
}
export function createUpgradeContractInstruction(connection, wormholeProgramId, payer, vaa) {
    const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId, connection).methods.upgradeContract();
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getUpgradeContractAccounts(wormholeProgramId, payer, vaa),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getUpgradeContractAccounts(wormholeProgramId, payer, vaa, spill) {
    const { newContract } = vaa.payload;
    return {
        payer: new PublicKey(payer),
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        vaa: derivePostedVaaKey(wormholeProgramId, Buffer.from(vaa.hash)),
        claim: deriveClaimKey(wormholeProgramId, vaa.emitterAddress.toString(), toChainId(vaa.emitterChain), vaa.sequence),
        upgradeAuthority: deriveUpgradeAuthorityKey(wormholeProgramId),
        spill: new PublicKey(spill === undefined ? payer : spill),
        implementation: newContract.toNative('Solana').unwrap(),
        programData: utils.deriveUpgradeableProgramKey(wormholeProgramId),
        wormholeProgram: new PublicKey(wormholeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        clock: SYSVAR_CLOCK_PUBKEY,
        bpfLoaderUpgradeable: utils.BpfLoaderUpgradeable.programId,
        systemProgram: SystemProgram.programId,
    };
}
