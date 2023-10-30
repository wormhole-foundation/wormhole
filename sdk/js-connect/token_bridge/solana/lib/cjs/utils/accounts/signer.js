"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deriveMintAuthorityKey = exports.deriveCustodySignerKey = exports.deriveAuthoritySignerKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveAuthoritySignerKey(tokenBridgeProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('authority_signer')], tokenBridgeProgramId);
}
exports.deriveAuthoritySignerKey = deriveAuthoritySignerKey;
function deriveCustodySignerKey(tokenBridgeProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('custody_signer')], tokenBridgeProgramId);
}
exports.deriveCustodySignerKey = deriveCustodySignerKey;
function deriveMintAuthorityKey(tokenBridgeProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('mint_signer')], tokenBridgeProgramId);
}
exports.deriveMintAuthorityKey = deriveMintAuthorityKey;
