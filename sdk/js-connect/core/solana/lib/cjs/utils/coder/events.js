"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WormholeEventsCoder = void 0;
class WormholeEventsCoder {
    constructor(_idl) { }
    decode(_log) {
        throw new Error('Wormhole program does not have events');
    }
}
exports.WormholeEventsCoder = WormholeEventsCoder;
