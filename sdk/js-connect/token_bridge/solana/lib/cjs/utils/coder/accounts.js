"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenBridgeAccountsCoder = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
class TokenBridgeAccountsCoder {
    constructor(idl) {
        this.idl = idl;
    }
    async encode(accountName, account) {
        switch (accountName) {
            default: {
                throw new Error(`Invalid account name: ${accountName}`);
            }
        }
    }
    decode(accountName, ix) {
        return this.decodeUnchecked(accountName, ix);
    }
    decodeUnchecked(accountName, ix) {
        switch (accountName) {
            default: {
                throw new Error(`Invalid account name: ${accountName}`);
            }
        }
    }
    memcmp(accountName, _appendData) {
        switch (accountName) {
            default: {
                throw new Error(`Invalid account name: ${accountName}`);
            }
        }
    }
    size(idlAccount) {
        return connect_sdk_solana_1.anchor.accountSize(this.idl, idlAccount) ?? 0;
    }
}
exports.TokenBridgeAccountsCoder = TokenBridgeAccountsCoder;
