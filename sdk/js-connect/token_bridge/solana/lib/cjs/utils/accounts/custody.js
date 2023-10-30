"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deriveCustodyKey = void 0;
const web3_js_1 = require("@solana/web3.js");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveCustodyKey(tokenBridgeProgramId, mint) {
    return connect_sdk_solana_1.utils.deriveAddress([new web3_js_1.PublicKey(mint).toBuffer()], tokenBridgeProgramId);
}
exports.deriveCustodyKey = deriveCustodyKey;
