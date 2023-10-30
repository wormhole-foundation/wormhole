"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenBridgeStateCoder = void 0;
class TokenBridgeStateCoder {
    constructor(_idl) { }
    encode(_name, _account) {
        throw new Error('Token Bridge program does not have state');
    }
    decode(_ix) {
        throw new Error('Token Bridge program does not have state');
    }
}
exports.TokenBridgeStateCoder = TokenBridgeStateCoder;
