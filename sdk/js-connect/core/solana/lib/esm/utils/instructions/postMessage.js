import { PublicKey, SYSVAR_CLOCK_PUBKEY, SYSVAR_RENT_PUBKEY, SystemProgram, } from '@solana/web3.js';
import { deriveWormholeBridgeDataKey, deriveFeeCollectorKey, getEmitterKeys, } from '../accounts';
export function getPostMessageAccounts(wormholeProgramId, payer, emitterProgramId, message) {
    const { emitter, sequence } = getEmitterKeys(emitterProgramId, wormholeProgramId);
    return {
        bridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        message: new PublicKey(message),
        emitter,
        sequence,
        payer: new PublicKey(payer),
        feeCollector: deriveFeeCollectorKey(wormholeProgramId),
        clock: SYSVAR_CLOCK_PUBKEY,
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
    };
}
