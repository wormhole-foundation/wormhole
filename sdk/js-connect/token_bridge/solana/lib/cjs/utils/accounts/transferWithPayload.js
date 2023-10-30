"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deriveRedeemerAccountKey = exports.deriveSenderAccountKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveSenderAccountKey(cpiProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('sender')], cpiProgramId);
}
exports.deriveSenderAccountKey = deriveSenderAccountKey;
function deriveRedeemerAccountKey(cpiProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('redeemer')], cpiProgramId);
}
exports.deriveRedeemerAccountKey = deriveRedeemerAccountKey;
