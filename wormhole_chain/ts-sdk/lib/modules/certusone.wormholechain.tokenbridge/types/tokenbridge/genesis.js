"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GenesisState = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const config_1 = require("../tokenbridge/config");
const replay_protection_1 = require("../tokenbridge/replay_protection");
const chain_registration_1 = require("../tokenbridge/chain_registration");
const coin_meta_rollback_protection_1 = require("../tokenbridge/coin_meta_rollback_protection");
const minimal_1 = require("protobufjs/minimal");
exports.protobufPackage = "certusone.wormholechain.tokenbridge";
const baseGenesisState = {};
exports.GenesisState = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.config !== undefined) {
            config_1.Config.encode(message.config, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.replayProtectionList) {
            replay_protection_1.ReplayProtection.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.chainRegistrationList) {
            chain_registration_1.ChainRegistration.encode(v, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.coinMetaRollbackProtectionList) {
            coin_meta_rollback_protection_1.CoinMetaRollbackProtection.encode(v, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.replayProtectionList = [];
        message.chainRegistrationList = [];
        message.coinMetaRollbackProtectionList = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.config = config_1.Config.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.replayProtectionList.push(replay_protection_1.ReplayProtection.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.chainRegistrationList.push(chain_registration_1.ChainRegistration.decode(reader, reader.uint32()));
                    break;
                case 4:
                    message.coinMetaRollbackProtectionList.push(coin_meta_rollback_protection_1.CoinMetaRollbackProtection.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseGenesisState };
        message.replayProtectionList = [];
        message.chainRegistrationList = [];
        message.coinMetaRollbackProtectionList = [];
        if (object.config !== undefined && object.config !== null) {
            message.config = config_1.Config.fromJSON(object.config);
        }
        else {
            message.config = undefined;
        }
        if (object.replayProtectionList !== undefined &&
            object.replayProtectionList !== null) {
            for (const e of object.replayProtectionList) {
                message.replayProtectionList.push(replay_protection_1.ReplayProtection.fromJSON(e));
            }
        }
        if (object.chainRegistrationList !== undefined &&
            object.chainRegistrationList !== null) {
            for (const e of object.chainRegistrationList) {
                message.chainRegistrationList.push(chain_registration_1.ChainRegistration.fromJSON(e));
            }
        }
        if (object.coinMetaRollbackProtectionList !== undefined &&
            object.coinMetaRollbackProtectionList !== null) {
            for (const e of object.coinMetaRollbackProtectionList) {
                message.coinMetaRollbackProtectionList.push(coin_meta_rollback_protection_1.CoinMetaRollbackProtection.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.config !== undefined &&
            (obj.config = message.config ? config_1.Config.toJSON(message.config) : undefined);
        if (message.replayProtectionList) {
            obj.replayProtectionList = message.replayProtectionList.map((e) => e ? replay_protection_1.ReplayProtection.toJSON(e) : undefined);
        }
        else {
            obj.replayProtectionList = [];
        }
        if (message.chainRegistrationList) {
            obj.chainRegistrationList = message.chainRegistrationList.map((e) => e ? chain_registration_1.ChainRegistration.toJSON(e) : undefined);
        }
        else {
            obj.chainRegistrationList = [];
        }
        if (message.coinMetaRollbackProtectionList) {
            obj.coinMetaRollbackProtectionList = message.coinMetaRollbackProtectionList.map((e) => (e ? coin_meta_rollback_protection_1.CoinMetaRollbackProtection.toJSON(e) : undefined));
        }
        else {
            obj.coinMetaRollbackProtectionList = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.replayProtectionList = [];
        message.chainRegistrationList = [];
        message.coinMetaRollbackProtectionList = [];
        if (object.config !== undefined && object.config !== null) {
            message.config = config_1.Config.fromPartial(object.config);
        }
        else {
            message.config = undefined;
        }
        if (object.replayProtectionList !== undefined &&
            object.replayProtectionList !== null) {
            for (const e of object.replayProtectionList) {
                message.replayProtectionList.push(replay_protection_1.ReplayProtection.fromPartial(e));
            }
        }
        if (object.chainRegistrationList !== undefined &&
            object.chainRegistrationList !== null) {
            for (const e of object.chainRegistrationList) {
                message.chainRegistrationList.push(chain_registration_1.ChainRegistration.fromPartial(e));
            }
        }
        if (object.coinMetaRollbackProtectionList !== undefined &&
            object.coinMetaRollbackProtectionList !== null) {
            for (const e of object.coinMetaRollbackProtectionList) {
                message.coinMetaRollbackProtectionList.push(coin_meta_rollback_protection_1.CoinMetaRollbackProtection.fromPartial(e));
            }
        }
        return message;
    },
};
