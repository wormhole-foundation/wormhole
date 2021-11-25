/* eslint-disable */
import { Config } from "../tokenbridge/config";
import { ReplayProtection } from "../tokenbridge/replay_protection";
import { ChainRegistration } from "../tokenbridge/chain_registration";
import { Writer, Reader } from "protobufjs/minimal";
export const protobufPackage = "certusone.wormholechain.tokenbridge";
const baseGenesisState = {};
export const GenesisState = {
    encode(message, writer = Writer.create()) {
        if (message.config !== undefined) {
            Config.encode(message.config, writer.uint32(10).fork()).ldelim();
        }
        for (const v of message.replayProtectionList) {
            ReplayProtection.encode(v, writer.uint32(18).fork()).ldelim();
        }
        for (const v of message.chainRegistrationList) {
            ChainRegistration.encode(v, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseGenesisState };
        message.replayProtectionList = [];
        message.chainRegistrationList = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.config = Config.decode(reader, reader.uint32());
                    break;
                case 2:
                    message.replayProtectionList.push(ReplayProtection.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.chainRegistrationList.push(ChainRegistration.decode(reader, reader.uint32()));
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
        if (object.chainRegistrationList !== undefined &&
            object.chainRegistrationList !== null) {
            for (const e of object.chainRegistrationList) {
                message.chainRegistrationList.push(ChainRegistration.fromJSON(e));
            }
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.config !== undefined &&
            (obj.config = message.config ? Config.toJSON(message.config) : undefined);
        if (message.replayProtectionList) {
            obj.replayProtectionList = message.replayProtectionList.map((e) => e ? ReplayProtection.toJSON(e) : undefined);
        }
        else {
            obj.replayProtectionList = [];
        }
        if (message.chainRegistrationList) {
            obj.chainRegistrationList = message.chainRegistrationList.map((e) => e ? ChainRegistration.toJSON(e) : undefined);
        }
        else {
            obj.chainRegistrationList = [];
        }
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseGenesisState };
        message.replayProtectionList = [];
        message.chainRegistrationList = [];
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
        if (object.chainRegistrationList !== undefined &&
            object.chainRegistrationList !== null) {
            for (const e of object.chainRegistrationList) {
                message.chainRegistrationList.push(ChainRegistration.fromPartial(e));
            }
        }
        return message;
    },
};
