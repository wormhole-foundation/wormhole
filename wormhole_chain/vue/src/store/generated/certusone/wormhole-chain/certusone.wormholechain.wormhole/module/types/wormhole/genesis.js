/* eslint-disable */
import { GuardianSet } from "../wormhole/guardian_set";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "certusone.wormholechain.wormhole";
const baseGenesisState = { guardianSetCount: 0 };
export const GenesisState = {
    encode(message, writer = Writer.create()) {
        for (const v of message.guardianSetList) {
            GuardianSet.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.guardianSetCount !== 0) {
            writer.uint32(16).uint32(message.guardianSetCount);
        }
        if (message.config !== undefined) {
            Config.encode(message.config, writer.uint32(26).fork()).ldelim();
        }
        for (const v of message.replayProtectionList) {
            ReplayProtection.encode(v, writer.uint32(34).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.guardianSetList = [];
        message.replayProtectionList = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianSetList.push(GuardianSet.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.guardianSetCount = reader.uint32();
                    break;
                case 3:
                    message.config = Config.decode(reader, reader.uint32());
                    break;
                case 4:
                    message.replayProtectionList.push(ReplayProtection.decode(reader, reader.uint32()));
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
        if (object.guardianSetList !== undefined &&
            object.guardianSetList !== null) {
            for (const e of object.guardianSetList) {
                message.guardianSetList.push(GuardianSet.fromJSON(e));
            }
        }
        if (object.guardianSetCount !== undefined &&
            object.guardianSetCount !== null) {
            message.guardianSetCount = Number(object.guardianSetCount);
        }
        else {
            message.guardianSetCount = 0;
        }
        if (object.config !== undefined && object.config !== null) {
            message.config = Config.fromJSON(object.config);
        }
        else {
            message.config = undefined;
        }
        if (object.replayProtectionList !== undefined &&
            object.replayProtectionList !== null) {
            for (const e of object.replayProtectionList) {
                message.replayProtectionList.push(ReplayProtection.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.guardianSetList) {
            obj.guardianSetList = message.guardianSetList.map((e) => e ? GuardianSet.toJSON(e) : undefined);
        }
        else {
            obj.guardianSetList = [];
        }
        message.guardianSetCount !== undefined &&
            (obj.guardianSetCount = message.guardianSetCount);
        message.config !== undefined &&
            (obj.config = message.config ? Config.toJSON(message.config) : undefined);
        if (message.replayProtectionList) {
            obj.replayProtectionList = message.replayProtectionList.map((e) => e ? ReplayProtection.toJSON(e) : undefined);
        }
        else {
            obj.replayProtectionList = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.guardianSetList = [];
        message.replayProtectionList = [];
        if (object.guardianSetList !== undefined &&
            object.guardianSetList !== null) {
            for (const e of object.guardianSetList) {
                message.guardianSetList.push(GuardianSet.fromPartial(e));
            }
        }
        if (object.guardianSetCount !== undefined &&
            object.guardianSetCount !== null) {
            message.guardianSetCount = object.guardianSetCount;
        }
        else {
            message.guardianSetCount = 0;
        }
        if (object.config !== undefined && object.config !== null) {
            message.config = Config.fromPartial(object.config);
        }
        else {
            message.config = undefined;
        }
        if (object.replayProtectionList !== undefined &&
            object.replayProtectionList !== null) {
            for (const e of object.replayProtectionList) {
                message.replayProtectionList.push(ReplayProtection.fromPartial(e));
            }
        }
        return message;
    },
};
