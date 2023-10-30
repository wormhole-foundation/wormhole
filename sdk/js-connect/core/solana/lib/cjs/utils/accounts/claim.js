"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getClaim = exports.deriveClaimKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveClaimKey(programId, emitterAddress, emitterChain, sequence) {
    const address = typeof emitterAddress == 'string'
        ? Buffer.from(emitterAddress, 'hex')
        : Buffer.from(emitterAddress);
    if (address.length != 32) {
        throw Error('address.length != 32');
    }
    const sequenceSerialized = Buffer.alloc(8);
    sequenceSerialized.writeBigUInt64BE(typeof sequence == 'number' ? BigInt(sequence) : sequence);
    return connect_sdk_solana_1.utils.deriveAddress([
        address,
        (() => {
            const buf = Buffer.alloc(2);
            buf.writeUInt16BE(emitterChain);
            return buf;
        })(),
        sequenceSerialized,
    ], programId);
}
exports.deriveClaimKey = deriveClaimKey;
function getClaim(connection, programId, emitterAddress, emitterChain, sequence, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return connection
            .getAccountInfo(deriveClaimKey(programId, emitterAddress, emitterChain, sequence), commitment)
            .then((info) => !!connect_sdk_solana_1.utils.getAccountData(info)[0]);
    });
}
exports.getClaim = getClaim;
