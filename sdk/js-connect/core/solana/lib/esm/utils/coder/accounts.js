var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { anchor } from '@wormhole-foundation/connect-sdk-solana';
export class WormholeAccountsCoder {
    constructor(idl) {
        this.idl = idl;
    }
    encode(accountName, account) {
        return __awaiter(this, void 0, void 0, function* () {
            switch (accountName) {
                default: {
                    throw new Error(`Invalid account name: ${accountName}`);
                }
            }
        });
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
            case 'postVaa': {
                return {
                    dataSize: 56, // + 4 + payload.length
                };
            }
            default: {
                throw new Error(`Invalid account name: ${accountName}`);
            }
        }
    }
    size(idlAccount) {
        var _a;
        return (_a = anchor.accountSize(this.idl, idlAccount)) !== null && _a !== void 0 ? _a : 0;
    }
}
export function encodePostVaaData(account) {
    const payload = account.payload;
    const serialized = Buffer.alloc(60 + payload.length);
    serialized.writeUInt8(account.version, 0);
    serialized.writeUInt32LE(account.guardianSetIndex, 1);
    serialized.writeUInt32LE(account.timestamp, 5);
    serialized.writeUInt32LE(account.nonce, 9);
    serialized.writeUInt16LE(account.emitterChain, 13);
    serialized.write(account.emitterAddress.toString('hex'), 15, 'hex');
    serialized.writeBigUInt64LE(account.sequence, 47);
    serialized.writeUInt8(account.consistencyLevel, 55);
    serialized.writeUInt32LE(payload.length, 56);
    serialized.write(payload.toString('hex'), 60, 'hex');
    return serialized;
}
export function decodePostVaaAccount(buf) {
    return {};
}
