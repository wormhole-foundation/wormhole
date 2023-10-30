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
exports.BridgeData = exports.BridgeConfig = exports.getWormholeBridgeData = exports.deriveWormholeBridgeDataKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
function deriveWormholeBridgeDataKey(wormholeProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('Bridge')], wormholeProgramId);
}
exports.deriveWormholeBridgeDataKey = deriveWormholeBridgeDataKey;
function getWormholeBridgeData(connection, wormholeProgramId, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return connection
            .getAccountInfo(deriveWormholeBridgeDataKey(wormholeProgramId), commitment)
            .then((info) => BridgeData.deserialize(connect_sdk_solana_1.utils.getAccountData(info)));
    });
}
exports.getWormholeBridgeData = getWormholeBridgeData;
class BridgeConfig {
    constructor(guardianSetExpirationTime, fee) {
        this.guardianSetExpirationTime = guardianSetExpirationTime;
        this.fee = fee;
    }
    static deserialize(data) {
        if (data.length != 12) {
            throw new Error('data.length != 12');
        }
        const guardianSetExpirationTime = data.readUInt32LE(0);
        const fee = data.readBigUInt64LE(4);
        return new BridgeConfig(guardianSetExpirationTime, fee);
    }
}
exports.BridgeConfig = BridgeConfig;
class BridgeData {
    constructor(guardianSetIndex, lastLamports, config) {
        this.guardianSetIndex = guardianSetIndex;
        this.lastLamports = lastLamports;
        this.config = config;
    }
    static deserialize(data) {
        if (data.length != 24) {
            throw new Error('data.length != 24');
        }
        const guardianSetIndex = data.readUInt32LE(0);
        const lastLamports = data.readBigUInt64LE(4);
        const config = BridgeConfig.deserialize(data.subarray(12));
        return new BridgeData(guardianSetIndex, lastLamports, config);
    }
}
exports.BridgeData = BridgeData;
