import { PublicKey, SystemProgram, SYSVAR_RENT_PUBKEY, } from '@solana/web3.js';
import { createReadOnlyTokenBridgeProgramInterface } from '../program';
import { deriveTokenBridgeConfigKey } from '../accounts';
export function createInitializeInstruction(tokenBridgeProgramId, payer, wormholeProgramId) {
    const methods = createReadOnlyTokenBridgeProgramInterface(tokenBridgeProgramId).methods.initialize(wormholeProgramId);
    // @ts-ignore
    return methods._ixFn(...methods._args, {
        accounts: getInitializeAccounts(tokenBridgeProgramId, payer),
        signers: undefined,
        remainingAccounts: undefined,
        preInstructions: undefined,
        postInstructions: undefined,
    });
}
export function getInitializeAccounts(tokenBridgeProgramId, payer) {
    return {
        payer: new PublicKey(payer),
        config: deriveTokenBridgeConfigKey(tokenBridgeProgramId),
        rent: SYSVAR_RENT_PUBKEY,
        systemProgram: SystemProgram.programId,
    };
}
