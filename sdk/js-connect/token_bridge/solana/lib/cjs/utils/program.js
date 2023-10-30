"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.coder = exports.createReadOnlyTokenBridgeProgramInterface = exports.createTokenBridgeProgramInterface = void 0;
const web3_js_1 = require("@solana/web3.js");
const anchor_1 = require("@project-serum/anchor");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const coder_1 = require("./coder");
const token_bridge_json_1 = __importDefault(require("../anchor-idl/token_bridge.json"));
function createTokenBridgeProgramInterface(programId, provider) {
    return new anchor_1.Program(token_bridge_json_1.default, new web3_js_1.PublicKey(programId), provider === undefined ? { connection: null } : provider, coder());
}
exports.createTokenBridgeProgramInterface = createTokenBridgeProgramInterface;
function createReadOnlyTokenBridgeProgramInterface(programId, connection) {
    return createTokenBridgeProgramInterface(programId, connect_sdk_solana_1.utils.createReadOnlyProvider(connection));
}
exports.createReadOnlyTokenBridgeProgramInterface = createReadOnlyTokenBridgeProgramInterface;
function coder() {
    return new coder_1.TokenBridgeCoder(token_bridge_json_1.default);
}
exports.coder = coder;
