"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getInitializeAccounts = exports.createInitializeInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const program_1 = require("../program");
const accounts_1 = require("../accounts");
function createInitializeInstruction(connection, wormholeProgramId, payer, guardianSetExpirationTime, fee, initialGuardians) {
    const methods = (0, program_1.createReadOnlyWormholeProgramInterface)(wormholeProgramId, connection).methods.initialize(guardianSetExpirationTime, BigInt(fee.toString()), [
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
exports.createInitializeInstruction = createInitializeInstruction;
function getInitializeAccounts(wormholeProgramId, payer) {
    return {
        bridge: (0, accounts_1.deriveWormholeBridgeDataKey)(wormholeProgramId),
        guardianSet: (0, accounts_1.deriveGuardianSetKey)(wormholeProgramId, 0),
        feeCollector: (0, accounts_1.deriveFeeCollectorKey)(wormholeProgramId),
        payer: new web3_js_1.PublicKey(payer),
        clock: web3_js_1.SYSVAR_CLOCK_PUBKEY,
        rent: web3_js_1.SYSVAR_RENT_PUBKEY,
        systemProgram: web3_js_1.SystemProgram.programId,
    };
}
exports.getInitializeAccounts = getInitializeAccounts;
