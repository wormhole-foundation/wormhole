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
exports.GuardianSetData = exports.getGuardianSet = exports.deriveGuardianSetKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveGuardianSetKey(wormholeProgramId, index) {
    return connect_sdk_solana_1.utils.deriveAddress([
        Buffer.from('GuardianSet'),
        (() => {
            const buf = Buffer.alloc(4);
            buf.writeUInt32BE(index);
            return buf;
        })(),
    ], wormholeProgramId);
}
exports.deriveGuardianSetKey = deriveGuardianSetKey;
function getGuardianSet(connection, wormholeProgramId, index, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return connection
            .getAccountInfo(deriveGuardianSetKey(wormholeProgramId, index), commitment)
            .then((info) => GuardianSetData.deserialize(connect_sdk_solana_1.utils.getAccountData(info)));
    });
}
exports.getGuardianSet = getGuardianSet;
class GuardianSetData {
    constructor(index, keys, creationTime, expirationTime) {
        this.index = index;
        this.keys = keys;
        this.creationTime = creationTime;
        this.expirationTime = expirationTime;
    }
    static deserialize(data) {
        const index = data.readUInt32LE(0);
        const keysLen = data.readUInt32LE(4);
        const keysEnd = 8 + keysLen * connect_sdk_solana_1.utils.ETHEREUM_KEY_LENGTH;
        const creationTime = data.readUInt32LE(keysEnd);
        const expirationTime = data.readUInt32LE(4 + keysEnd);
        const keys = [];
        for (let i = 0; i < keysLen; ++i) {
            const start = 8 + i * connect_sdk_solana_1.utils.ETHEREUM_KEY_LENGTH;
            keys.push(data.subarray(start, start + connect_sdk_solana_1.utils.ETHEREUM_KEY_LENGTH));
        }
        return new GuardianSetData(index, keys, creationTime, expirationTime);
    }
}
exports.GuardianSetData = GuardianSetData;
