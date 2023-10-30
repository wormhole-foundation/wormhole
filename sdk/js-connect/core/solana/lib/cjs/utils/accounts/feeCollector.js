"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deriveFeeCollectorKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveFeeCollectorKey(wormholeProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('fee_collector')], wormholeProgramId);
}
exports.deriveFeeCollectorKey = deriveFeeCollectorKey;
