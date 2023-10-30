"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WormholeTypesCoder = void 0;
class WormholeTypesCoder {
    constructor(_idl) { }
    encode(_name, _type) {
        throw new Error('Wormhole program does not have user-defined types');
    }
    decode(_name, _typeData) {
        throw new Error('Wormhole program does not have user-defined types');
    }
}
exports.WormholeTypesCoder = WormholeTypesCoder;
