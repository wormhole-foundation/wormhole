export class WormholeStateCoder {
    constructor(_idl) { }
    encode(_name, _account) {
        throw new Error('Wormhole program does not have state');
    }
    decode(_ix) {
        throw new Error('Wormhole program does not have state');
    }
}
