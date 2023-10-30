"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.derivePostedVaaKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function derivePostedVaaKey(wormholeProgramId, hash) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('PostedVAA'), hash], wormholeProgramId);
}
exports.derivePostedVaaKey = derivePostedVaaKey;
