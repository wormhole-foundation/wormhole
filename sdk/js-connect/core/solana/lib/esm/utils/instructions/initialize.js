import { PublicKey, SystemProgram, SYSVAR_CLOCK_PUBKEY, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { createReadOnlyWormholeProgramInterface } from '../program';
import { deriveFeeCollectorKey, deriveGuardianSetKey, deriveWormholeBridgeDataKey, } from '../accounts';
export function createInitializeInstruction(connection, wormholeProgramId, payer, guardianSetExpirationTime, fee, initialGuardians) {
    const methods = createReadOnlyWormholeProgramInterface(wormholeProgramId, connection).methods.initialize(guardianSetExpirationTime, BigInt(fee.toString()), [
        ...initialGuardians,
    ]);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getInitializeAccounts(wormholeProgramId, payer),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getInitializeAccounts(wormholeProgramId, payer) {
    return {
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        guardianSet: deriveGuardianSetKey(wormholeProgramId, 0),
        feeCollector: deriveFeeCollectorKey(wormholeProgramId),
        payer: new PublicKey(payer),
        clock: SYSVAR_CLOCK_PUBKEY,
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
    };
}
