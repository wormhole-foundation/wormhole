export class TokenBridgeTypesCoder {
    constructor(_idl) { }
    encode(_name, _type) {
        throw new Error('Token Bridge program does not have user-defined types');
    }
    decode(_name, _typeData) {
        throw new Error('Token Bridge program does not have user-defined types');
    }
}
