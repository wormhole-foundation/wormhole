"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.createApproveAuthoritySignerInstruction = void 0;
const spl_token_1 = require("@solana/spl-token");
const web3_js_1 = require("@solana/web3.js");
const accounts_1 = require("../accounts");
function createApproveAuthoritySignerInstruction(tokenBridgeProgramId, tokenAccount, owner, amount) {
    return (0, spl_token_1.createApproveInstruction)(new web3_js_1.PublicKey(tokenAccount), (0, accounts_1.deriveAuthoritySignerKey)(tokenBridgeProgramId), new web3_js_1.PublicKey(owner), amount);
}
exports.createApproveAuthoritySignerInstruction = createApproveAuthoritySignerInstruction;
