"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPostMessageCpiAccounts = exports.getWormholeDerivedAccounts = void 0;
const accounts_1 = require("./accounts");
const instructions_1 = require("./instructions");
/**
 * Generate Wormhole PDAs.
 *
 * @param cpiProgramId
 * @param wormholeProgramId
 * @returns
 */
function getWormholeDerivedAccounts(cpiProgramId, wormholeProgramId) {
    const { emitter: wormholeEmitter, sequence: wormholeSequence } = (0, accounts_1.getEmitterKeys)(cpiProgramId, wormholeProgramId);
    return {
        wormholeBridge: (0, accounts_1.deriveWormholeBridgeDataKey)(wormholeProgramId),
        wormholeEmitter,
        wormholeSequence,
        wormholeFeeCollector: (0, accounts_1.deriveFeeCollectorKey)(wormholeProgramId),
    };
}
exports.getWormholeDerivedAccounts = getWormholeDerivedAccounts;
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
function getPostMessageCpiAccounts(cpiProgramId, wormholeProgramId, payer, message) {
    const accounts = (0, instructions_1.getPostMessageAccounts)(wormholeProgramId, payer, cpiProgramId, message);
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
exports.getPostMessageCpiAccounts = getPostMessageCpiAccounts;
