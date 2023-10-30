export class WormholeTypesCoder {
    constructor(_idl) { }
    encode(_name, _type) {
        throw new Error('Wormhole program does not have user-defined types');
    }
    decode(_name, _typeData) {
        throw new Error('Wormhole program does not have user-defined types');
    }
}
