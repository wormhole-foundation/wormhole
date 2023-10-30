"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WrappedMeta = exports.getWrappedMeta = exports.deriveWrappedMetaKey = exports.deriveWrappedMintKey = void 0;
const web3_js_1 = require("@solana/web3.js");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
function deriveWrappedMintKey(tokenBridgeProgramId, tokenChain, tokenAddress) {
    if (tokenChain == (0, connect_sdk_1.toChainId)('Solana')) {
        throw new Error('tokenChain == CHAIN_ID_SOLANA does not have wrapped mint key');
    }
    if (typeof tokenAddress == 'string') {
        const parsedAddress = (0, connect_sdk_1.toNative)((0, connect_sdk_1.toChainName)(tokenChain), tokenAddress);
        tokenAddress = parsedAddress.toUint8Array();
    }
    return connect_sdk_solana_1.utils.deriveAddress([
        Buffer.from('wrapped'),
        (() => {
            const buf = Buffer.alloc(2);
            buf.writeUInt16BE(tokenChain);
            return buf;
        })(),
        tokenAddress,
    ], tokenBridgeProgramId);
}
exports.deriveWrappedMintKey = deriveWrappedMintKey;
function deriveWrappedMetaKey(tokenBridgeProgramId, mint) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('meta'), new web3_js_1.PublicKey(mint).toBuffer()], tokenBridgeProgramId);
}
exports.deriveWrappedMetaKey = deriveWrappedMetaKey;
async function getWrappedMeta(connection, tokenBridgeProgramId, mint, commitment) {
    return connection
        .getAccountInfo(deriveWrappedMetaKey(tokenBridgeProgramId, mint), commitment)
        .then((info) => WrappedMeta.deserialize(connect_sdk_solana_1.utils.getAccountData(info)));
}
exports.getWrappedMeta = getWrappedMeta;
class WrappedMeta {
    constructor(chain, tokenAddress, originalDecimals) {
        this.chain = chain;
        this.tokenAddress = tokenAddress;
        this.originalDecimals = originalDecimals;
    }
    static deserialize(data) {
        if (data.length != 35) {
            throw new Error('data.length != 35');
        }
        const chain = data.readUInt16LE(0);
        const tokenAddress = data.subarray(2, 34);
        const originalDecimals = data.readUInt8(34);
        return new WrappedMeta(chain, tokenAddress, originalDecimals);
    }
}
exports.WrappedMeta = WrappedMeta;
