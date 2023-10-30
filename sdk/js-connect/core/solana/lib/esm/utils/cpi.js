import { deriveFeeCollectorKey, deriveWormholeBridgeDataKey, getEmitterKeys, } from './accounts';
import { getPostMessageAccounts } from './instructions';
/**
 * Generate Wormhole PDAs.
 *
 * @param cpiProgramId
 * @param wormholeProgramId
 * @returns
 */
export function getWormholeDerivedAccounts(cpiProgramId, wormholeProgramId) {
    const { emitter: wormholeEmitter, sequence: wormholeSequence } = getEmitterKeys(cpiProgramId, wormholeProgramId);
    return {
        wormholeBridge: deriveWormholeBridgeDataKey(wormholeProgramId),
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector: deriveFeeCollectorKey(wormholeProgramId),
    };
}
/**
 * Generate accounts needed to perform `post_message` instruction
 * as cross-program invocation.
 *
 * @param cpiProgramId
 * @param wormholeProgramId
 * @param payer
 * @param message
 * @returns
 */
export function getPostMessageCpiAccounts(cpiProgramId, wormholeProgramId, payer, message) {
    const accounts = getPostMessageAccounts(wormholeProgramId, payer, cpiProgramId, message);
    return {
        payer: accounts.payer,
        wormholeBridge: accounts.bridge,
        wormholeMessage: accounts.message,
        wormholeEmitter: accounts.emitter,
        wormholeSequence: accounts.sequence,
        wormholeFeeCollector: accounts.feeCollector,
        clock: accounts.clock,
        rent: accounts.rent,
        systemProgram: accounts.systemProgram,
    };
}
