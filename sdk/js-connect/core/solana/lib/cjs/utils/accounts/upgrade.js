"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deriveUpgradeAuthorityKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveUpgradeAuthorityKey(wormholeProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('upgrade')], wormholeProgramId);
}
exports.deriveUpgradeAuthorityKey = deriveUpgradeAuthorityKey;
