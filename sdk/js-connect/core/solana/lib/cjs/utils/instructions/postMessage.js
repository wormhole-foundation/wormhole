"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPostMessageAccounts = void 0;
const web3_js_1 = require("@solana/web3.js");
const accounts_1 = require("../accounts");
function getPostMessageAccounts(wormholeProgramId, payer, emitterProgramId, message) {
    const { emitter, sequence } = (0, accounts_1.getEmitterKeys)(emitterProgramId, wormholeProgramId);
    return {
        bridge: (0, accounts_1.deriveWormholeBridgeDataKey)(wormholeProgramId),
        message: new web3_js_1.PublicKey(message),
        emitter,
        sequence,
        payer: new web3_js_1.PublicKey(payer),
        feeCollector: (0, accounts_1.deriveFeeCollectorKey)(wormholeProgramId),
        clock: web3_js_1.SYSVAR_CLOCK_PUBKEY,
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
    };
}
exports.getPostMessageAccounts = getPostMessageAccounts;
