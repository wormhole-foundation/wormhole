"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.GenesisState = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const guardian_set_1 = require("../wormhole/guardian_set");
const config_1 = require("../wormhole/config");
const replay_protection_1 = require("../wormhole/replay_protection");
const sequence_counter_1 = require("../wormhole/sequence_counter");
const active_guardian_set_index_1 = require("../wormhole/active_guardian_set_index");
const guardian_validator_1 = require("../wormhole/guardian_validator");
const minimal_1 = require("protobufjs/minimal");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseGenesisState = {};
exports.GenesisState = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.guardianSetList) {
            guardian_set_1.GuardianSet.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.config !== undefined) {
            config_1.Config.encode(message.config, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.replayProtectionList) {
            replay_protection_1.ReplayProtection.encode(v, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.sequenceCounterList) {
            sequence_counter_1.SequenceCounter.encode(v, writer.uint32(34).fork()).ldelim();
        }
        if (message.activeGuardianSetIndex !== undefined) {
            active_guardian_set_index_1.ActiveGuardianSetIndex.encode(message.activeGuardianSetIndex, writer.uint32(42).fork()).ldelim();
        }
        for (const v of message.guardianValidatorList) {
            guardian_validator_1.GuardianValidator.encode(v, writer.uint32(50).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.guardianSetList = [];
        message.replayProtectionList = [];
        message.sequenceCounterList = [];
        message.guardianValidatorList = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianSetList.push(guardian_set_1.GuardianSet.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.config = config_1.Config.decode(reader, reader.uint32());
                    break;
                case 3:
                    message.replayProtectionList.push(replay_protection_1.ReplayProtection.decode(reader, reader.uint32()));
                    break;
                case 4:
                    message.sequenceCounterList.push(sequence_counter_1.SequenceCounter.decode(reader, reader.uint32()));
                    break;
                case 5:
                    message.activeGuardianSetIndex = active_guardian_set_index_1.ActiveGuardianSetIndex.decode(reader, reader.uint32());
                    break;
                case 6:
                    message.guardianValidatorList.push(guardian_validator_1.GuardianValidator.decode(reader, reader.uint32()));
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
        message.guardianSetList = [];
        message.replayProtectionList = [];
        message.sequenceCounterList = [];
        message.guardianValidatorList = [];
        if (object.guardianSetList !== undefined &&
            object.guardianSetList !== null) {
            for (const e of object.guardianSetList) {
                message.guardianSetList.push(guardian_set_1.GuardianSet.fromJSON(e));
            }
        }
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
        if (object.sequenceCounterList !== undefined &&
            object.sequenceCounterList !== null) {
            for (const e of object.sequenceCounterList) {
                message.sequenceCounterList.push(sequence_counter_1.SequenceCounter.fromJSON(e));
            }
        }
        if (object.activeGuardianSetIndex !== undefined &&
            object.activeGuardianSetIndex !== null) {
            message.activeGuardianSetIndex = active_guardian_set_index_1.ActiveGuardianSetIndex.fromJSON(object.activeGuardianSetIndex);
        }
        else {
            message.activeGuardianSetIndex = undefined;
        }
        if (object.guardianValidatorList !== undefined &&
            object.guardianValidatorList !== null) {
            for (const e of object.guardianValidatorList) {
                message.guardianValidatorList.push(guardian_validator_1.GuardianValidator.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.guardianSetList) {
            obj.guardianSetList = message.guardianSetList.map((e) => e ? guardian_set_1.GuardianSet.toJSON(e) : undefined);
        }
        else {
            obj.guardianSetList = [];
        }
        message.config !== undefined &&
            (obj.config = message.config ? config_1.Config.toJSON(message.config) : undefined);
        if (message.replayProtectionList) {
            obj.replayProtectionList = message.replayProtectionList.map((e) => e ? replay_protection_1.ReplayProtection.toJSON(e) : undefined);
        }
        else {
            obj.replayProtectionList = [];
        }
        if (message.sequenceCounterList) {
            obj.sequenceCounterList = message.sequenceCounterList.map((e) => e ? sequence_counter_1.SequenceCounter.toJSON(e) : undefined);
        }
        else {
            obj.sequenceCounterList = [];
        }
        message.activeGuardianSetIndex !== undefined &&
            (obj.activeGuardianSetIndex = message.activeGuardianSetIndex
                ? active_guardian_set_index_1.ActiveGuardianSetIndex.toJSON(message.activeGuardianSetIndex)
                : undefined);
        if (message.guardianValidatorList) {
            obj.guardianValidatorList = message.guardianValidatorList.map((e) => e ? guardian_validator_1.GuardianValidator.toJSON(e) : undefined);
        }
        else {
            obj.guardianValidatorList = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.guardianSetList = [];
        message.replayProtectionList = [];
        message.sequenceCounterList = [];
        message.guardianValidatorList = [];
        if (object.guardianSetList !== undefined &&
            object.guardianSetList !== null) {
            for (const e of object.guardianSetList) {
                message.guardianSetList.push(guardian_set_1.GuardianSet.fromPartial(e));
            }
        }
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
        if (object.sequenceCounterList !== undefined &&
            object.sequenceCounterList !== null) {
            for (const e of object.sequenceCounterList) {
                message.sequenceCounterList.push(sequence_counter_1.SequenceCounter.fromPartial(e));
            }
        }
        if (object.activeGuardianSetIndex !== undefined &&
            object.activeGuardianSetIndex !== null) {
            message.activeGuardianSetIndex = active_guardian_set_index_1.ActiveGuardianSetIndex.fromPartial(object.activeGuardianSetIndex);
        }
        else {
            message.activeGuardianSetIndex = undefined;
        }
        if (object.guardianValidatorList !== undefined &&
            object.guardianValidatorList !== null) {
            for (const e of object.guardianValidatorList) {
                message.guardianValidatorList.push(guardian_validator_1.GuardianValidator.fromPartial(e));
            }
        }
        return message;
    },
};
