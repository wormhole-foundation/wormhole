"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EndpointRegistration = exports.getEndpointRegistration = exports.deriveEndpointKey = void 0;
const web3_js_1 = require("@solana/web3.js");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
function deriveEndpointKey(tokenBridgeProgramId, emitterChain, emitterAddress) {
    if (emitterChain == (0, connect_sdk_1.toChainId)('Solana')) {
        throw new Error('emitterChain == CHAIN_ID_SOLANA cannot exist as foreign token bridge emitter');
    }
    if (typeof emitterAddress == 'string') {
        const parsedAddress = (0, connect_sdk_1.toNative)((0, connect_sdk_1.toChainName)(emitterChain), emitterAddress);
        emitterAddress = parsedAddress.toUint8Array();
    }
    return connect_sdk_solana_1.utils.deriveAddress([
        (() => {
            const buf = Buffer.alloc(2);
            buf.writeUInt16BE(emitterChain);
            return buf;
        })(),
        emitterAddress,
    ], tokenBridgeProgramId);
}
exports.deriveEndpointKey = deriveEndpointKey;
async function getEndpointRegistration(connection, endpointKey, commitment) {
    return connection
        .getAccountInfo(new web3_js_1.PublicKey(endpointKey), commitment)
        .then((info) => EndpointRegistration.deserialize(connect_sdk_solana_1.utils.getAccountData(info)));
}
exports.getEndpointRegistration = getEndpointRegistration;
class EndpointRegistration {
    constructor(chain, contract) {
        this.chain = chain;
        this.contract = contract;
    }
    static deserialize(data) {
        if (data.length != 34) {
            throw new Error('data.length != 34');
        }
        const chain = data.readUInt16LE(0);
        const contract = data.subarray(2, 34);
        return new EndpointRegistration(chain, contract);
    }
}
exports.EndpointRegistration = EndpointRegistration;
