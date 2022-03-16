"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.QueryAllGuardianValidatorResponse = exports.QueryAllGuardianValidatorRequest = exports.QueryGetGuardianValidatorResponse = exports.QueryGetGuardianValidatorRequest = exports.QueryGetActiveGuardianSetIndexResponse = exports.QueryGetActiveGuardianSetIndexRequest = exports.QueryAllSequenceCounterResponse = exports.QueryAllSequenceCounterRequest = exports.QueryGetSequenceCounterResponse = exports.QueryGetSequenceCounterRequest = exports.QueryAllReplayProtectionResponse = exports.QueryAllReplayProtectionRequest = exports.QueryGetReplayProtectionResponse = exports.QueryGetReplayProtectionRequest = exports.QueryGetConfigResponse = exports.QueryGetConfigRequest = exports.QueryAllGuardianSetResponse = exports.QueryAllGuardianSetRequest = exports.QueryGetGuardianSetResponse = exports.QueryGetGuardianSetRequest = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const minimal_1 = require("protobufjs/minimal");
const guardian_set_1 = require("../wormhole/guardian_set");
const pagination_1 = require("../cosmos/base/query/v1beta1/pagination");
const config_1 = require("../wormhole/config");
const replay_protection_1 = require("../wormhole/replay_protection");
const sequence_counter_1 = require("../wormhole/sequence_counter");
const active_guardian_set_index_1 = require("../wormhole/active_guardian_set_index");
const guardian_validator_1 = require("../wormhole/guardian_validator");
exports.protobufPackage = "certusone.wormholechain.wormhole";
const baseQueryGetGuardianSetRequest = { index: 0 };
exports.QueryGetGuardianSetRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.index !== 0) {
            writer.uint32(8).uint32(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetGuardianSetRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.uint32();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetGuardianSetRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = Number(object.index);
        }
        else {
            message.index = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetGuardianSetRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = 0;
        }
        return message;
    },
};
const baseQueryGetGuardianSetResponse = {};
exports.QueryGetGuardianSetResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.GuardianSet !== undefined) {
            guardian_set_1.GuardianSet.encode(message.GuardianSet, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetGuardianSetResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.GuardianSet = guardian_set_1.GuardianSet.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetGuardianSetResponse,
        };
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            message.GuardianSet = guardian_set_1.GuardianSet.fromJSON(object.GuardianSet);
        }
        else {
            message.GuardianSet = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.GuardianSet !== undefined &&
            (obj.GuardianSet = message.GuardianSet
                ? guardian_set_1.GuardianSet.toJSON(message.GuardianSet)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetGuardianSetResponse,
        };
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            message.GuardianSet = guardian_set_1.GuardianSet.fromPartial(object.GuardianSet);
        }
        else {
            message.GuardianSet = undefined;
        }
        return message;
    },
};
const baseQueryAllGuardianSetRequest = {};
exports.QueryAllGuardianSetRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllGuardianSetRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllGuardianSetRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllGuardianSetRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllGuardianSetResponse = {};
exports.QueryAllGuardianSetResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.GuardianSet) {
            guardian_set_1.GuardianSet.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllGuardianSetResponse,
        };
        message.GuardianSet = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.GuardianSet.push(guardian_set_1.GuardianSet.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllGuardianSetResponse,
        };
        message.GuardianSet = [];
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            for (const e of object.GuardianSet) {
                message.GuardianSet.push(guardian_set_1.GuardianSet.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.GuardianSet) {
            obj.GuardianSet = message.GuardianSet.map((e) => e ? guardian_set_1.GuardianSet.toJSON(e) : undefined);
        }
        else {
            obj.GuardianSet = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllGuardianSetResponse,
        };
        message.GuardianSet = [];
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            for (const e of object.GuardianSet) {
                message.GuardianSet.push(guardian_set_1.GuardianSet.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryGetConfigRequest = {};
exports.QueryGetConfigRequest = {
    encode(_, writer = minimal_1.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryGetConfigRequest };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = { ...baseQueryGetConfigRequest };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = { ...baseQueryGetConfigRequest };
        return message;
    },
};
const baseQueryGetConfigResponse = {};
exports.QueryGetConfigResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.Config !== undefined) {
            config_1.Config.encode(message.Config, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryGetConfigResponse };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.Config = config_1.Config.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = { ...baseQueryGetConfigResponse };
        if (object.Config !== undefined && object.Config !== null) {
            message.Config = config_1.Config.fromJSON(object.Config);
        }
        else {
            message.Config = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.Config !== undefined &&
            (obj.Config = message.Config ? config_1.Config.toJSON(message.Config) : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryGetConfigResponse };
        if (object.Config !== undefined && object.Config !== null) {
            message.Config = config_1.Config.fromPartial(object.Config);
        }
        else {
            message.Config = undefined;
        }
        return message;
    },
};
const baseQueryGetReplayProtectionRequest = { index: "" };
exports.QueryGetReplayProtectionRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetReplayProtectionRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetReplayProtectionRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetReplayProtectionRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
const baseQueryGetReplayProtectionResponse = {};
exports.QueryGetReplayProtectionResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.replayProtection !== undefined) {
            replay_protection_1.ReplayProtection.encode(message.replayProtection, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetReplayProtectionResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.replayProtection = replay_protection_1.ReplayProtection.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetReplayProtectionResponse,
        };
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            message.replayProtection = replay_protection_1.ReplayProtection.fromJSON(object.replayProtection);
        }
        else {
            message.replayProtection = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.replayProtection !== undefined &&
            (obj.replayProtection = message.replayProtection
                ? replay_protection_1.ReplayProtection.toJSON(message.replayProtection)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetReplayProtectionResponse,
        };
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            message.replayProtection = replay_protection_1.ReplayProtection.fromPartial(object.replayProtection);
        }
        else {
            message.replayProtection = undefined;
        }
        return message;
    },
};
const baseQueryAllReplayProtectionRequest = {};
exports.QueryAllReplayProtectionRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllReplayProtectionRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllReplayProtectionRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllReplayProtectionRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllReplayProtectionResponse = {};
exports.QueryAllReplayProtectionResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.replayProtection) {
            replay_protection_1.ReplayProtection.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllReplayProtectionResponse,
        };
        message.replayProtection = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.replayProtection.push(replay_protection_1.ReplayProtection.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllReplayProtectionResponse,
        };
        message.replayProtection = [];
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            for (const e of object.replayProtection) {
                message.replayProtection.push(replay_protection_1.ReplayProtection.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.replayProtection) {
            obj.replayProtection = message.replayProtection.map((e) => e ? replay_protection_1.ReplayProtection.toJSON(e) : undefined);
        }
        else {
            obj.replayProtection = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllReplayProtectionResponse,
        };
        message.replayProtection = [];
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            for (const e of object.replayProtection) {
                message.replayProtection.push(replay_protection_1.ReplayProtection.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryGetSequenceCounterRequest = { index: "" };
exports.QueryGetSequenceCounterRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetSequenceCounterRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.index = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetSequenceCounterRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetSequenceCounterRequest,
        };
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
const baseQueryGetSequenceCounterResponse = {};
exports.QueryGetSequenceCounterResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.sequenceCounter !== undefined) {
            sequence_counter_1.SequenceCounter.encode(message.sequenceCounter, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetSequenceCounterResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.sequenceCounter = sequence_counter_1.SequenceCounter.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetSequenceCounterResponse,
        };
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            message.sequenceCounter = sequence_counter_1.SequenceCounter.fromJSON(object.sequenceCounter);
        }
        else {
            message.sequenceCounter = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.sequenceCounter !== undefined &&
            (obj.sequenceCounter = message.sequenceCounter
                ? sequence_counter_1.SequenceCounter.toJSON(message.sequenceCounter)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetSequenceCounterResponse,
        };
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            message.sequenceCounter = sequence_counter_1.SequenceCounter.fromPartial(object.sequenceCounter);
        }
        else {
            message.sequenceCounter = undefined;
        }
        return message;
    },
};
const baseQueryAllSequenceCounterRequest = {};
exports.QueryAllSequenceCounterRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllSequenceCounterRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllSequenceCounterRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllSequenceCounterRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllSequenceCounterResponse = {};
exports.QueryAllSequenceCounterResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.sequenceCounter) {
            sequence_counter_1.SequenceCounter.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllSequenceCounterResponse,
        };
        message.sequenceCounter = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.sequenceCounter.push(sequence_counter_1.SequenceCounter.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllSequenceCounterResponse,
        };
        message.sequenceCounter = [];
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            for (const e of object.sequenceCounter) {
                message.sequenceCounter.push(sequence_counter_1.SequenceCounter.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.sequenceCounter) {
            obj.sequenceCounter = message.sequenceCounter.map((e) => e ? sequence_counter_1.SequenceCounter.toJSON(e) : undefined);
        }
        else {
            obj.sequenceCounter = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllSequenceCounterResponse,
        };
        message.sequenceCounter = [];
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            for (const e of object.sequenceCounter) {
                message.sequenceCounter.push(sequence_counter_1.SequenceCounter.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryGetActiveGuardianSetIndexRequest = {};
exports.QueryGetActiveGuardianSetIndexRequest = {
    encode(_, writer = minimal_1.Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetActiveGuardianSetIndexRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(_) {
        const message = {
            ...baseQueryGetActiveGuardianSetIndexRequest,
        };
        return message;
    },
    toJSON(_) {
        const obj = {};
        return obj;
    },
    fromPartial(_) {
        const message = {
            ...baseQueryGetActiveGuardianSetIndexRequest,
        };
        return message;
    },
};
const baseQueryGetActiveGuardianSetIndexResponse = {};
exports.QueryGetActiveGuardianSetIndexResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.ActiveGuardianSetIndex !== undefined) {
            active_guardian_set_index_1.ActiveGuardianSetIndex.encode(message.ActiveGuardianSetIndex, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetActiveGuardianSetIndexResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.ActiveGuardianSetIndex = active_guardian_set_index_1.ActiveGuardianSetIndex.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetActiveGuardianSetIndexResponse,
        };
        if (object.ActiveGuardianSetIndex !== undefined &&
            object.ActiveGuardianSetIndex !== null) {
            message.ActiveGuardianSetIndex = active_guardian_set_index_1.ActiveGuardianSetIndex.fromJSON(object.ActiveGuardianSetIndex);
        }
        else {
            message.ActiveGuardianSetIndex = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.ActiveGuardianSetIndex !== undefined &&
            (obj.ActiveGuardianSetIndex = message.ActiveGuardianSetIndex
                ? active_guardian_set_index_1.ActiveGuardianSetIndex.toJSON(message.ActiveGuardianSetIndex)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetActiveGuardianSetIndexResponse,
        };
        if (object.ActiveGuardianSetIndex !== undefined &&
            object.ActiveGuardianSetIndex !== null) {
            message.ActiveGuardianSetIndex = active_guardian_set_index_1.ActiveGuardianSetIndex.fromPartial(object.ActiveGuardianSetIndex);
        }
        else {
            message.ActiveGuardianSetIndex = undefined;
        }
        return message;
    },
};
const baseQueryGetGuardianValidatorRequest = {};
exports.QueryGetGuardianValidatorRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.guardianKey.length !== 0) {
            writer.uint32(10).bytes(message.guardianKey);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetGuardianValidatorRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianKey = reader.bytes();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetGuardianValidatorRequest,
        };
        if (object.guardianKey !== undefined && object.guardianKey !== null) {
            message.guardianKey = bytesFromBase64(object.guardianKey);
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.guardianKey !== undefined &&
            (obj.guardianKey = base64FromBytes(message.guardianKey !== undefined
                ? message.guardianKey
                : new Uint8Array()));
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetGuardianValidatorRequest,
        };
        if (object.guardianKey !== undefined && object.guardianKey !== null) {
            message.guardianKey = object.guardianKey;
        }
        else {
            message.guardianKey = new Uint8Array();
        }
        return message;
    },
};
const baseQueryGetGuardianValidatorResponse = {};
exports.QueryGetGuardianValidatorResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.guardianValidator !== undefined) {
            guardian_validator_1.GuardianValidator.encode(message.guardianValidator, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetGuardianValidatorResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianValidator = guardian_validator_1.GuardianValidator.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryGetGuardianValidatorResponse,
        };
        if (object.guardianValidator !== undefined &&
            object.guardianValidator !== null) {
            message.guardianValidator = guardian_validator_1.GuardianValidator.fromJSON(object.guardianValidator);
        }
        else {
            message.guardianValidator = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.guardianValidator !== undefined &&
            (obj.guardianValidator = message.guardianValidator
                ? guardian_validator_1.GuardianValidator.toJSON(message.guardianValidator)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetGuardianValidatorResponse,
        };
        if (object.guardianValidator !== undefined &&
            object.guardianValidator !== null) {
            message.guardianValidator = guardian_validator_1.GuardianValidator.fromPartial(object.guardianValidator);
        }
        else {
            message.guardianValidator = undefined;
        }
        return message;
    },
};
const baseQueryAllGuardianValidatorRequest = {};
exports.QueryAllGuardianValidatorRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.pagination !== undefined) {
            pagination_1.PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllGuardianValidatorRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = pagination_1.PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllGuardianValidatorRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllGuardianValidatorRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllGuardianValidatorResponse = {};
exports.QueryAllGuardianValidatorResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.guardianValidator) {
            guardian_validator_1.GuardianValidator.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            pagination_1.PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllGuardianValidatorResponse,
        };
        message.guardianValidator = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.guardianValidator.push(guardian_validator_1.GuardianValidator.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = pagination_1.PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON(object) {
        const message = {
            ...baseQueryAllGuardianValidatorResponse,
        };
        message.guardianValidator = [];
        if (object.guardianValidator !== undefined &&
            object.guardianValidator !== null) {
            for (const e of object.guardianValidator) {
                message.guardianValidator.push(guardian_validator_1.GuardianValidator.fromJSON(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        if (message.guardianValidator) {
            obj.guardianValidator = message.guardianValidator.map((e) => e ? guardian_validator_1.GuardianValidator.toJSON(e) : undefined);
        }
        else {
            obj.guardianValidator = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllGuardianValidatorResponse,
        };
        message.guardianValidator = [];
        if (object.guardianValidator !== undefined &&
            object.guardianValidator !== null) {
            for (const e of object.guardianValidator) {
                message.guardianValidator.push(guardian_validator_1.GuardianValidator.fromPartial(e));
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = pagination_1.PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
class QueryClientImpl {
    rpc;
    constructor(rpc) {
        this.rpc = rpc;
    }
    GuardianSet(request) {
        const data = exports.QueryGetGuardianSetRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianSet", data);
        return promise.then((data) => exports.QueryGetGuardianSetResponse.decode(new minimal_1.Reader(data)));
    }
    GuardianSetAll(request) {
        const data = exports.QueryAllGuardianSetRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianSetAll", data);
        return promise.then((data) => exports.QueryAllGuardianSetResponse.decode(new minimal_1.Reader(data)));
    }
    Config(request) {
        const data = exports.QueryGetConfigRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "Config", data);
        return promise.then((data) => exports.QueryGetConfigResponse.decode(new minimal_1.Reader(data)));
    }
    ReplayProtection(request) {
        const data = exports.QueryGetReplayProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "ReplayProtection", data);
        return promise.then((data) => exports.QueryGetReplayProtectionResponse.decode(new minimal_1.Reader(data)));
    }
    ReplayProtectionAll(request) {
        const data = exports.QueryAllReplayProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "ReplayProtectionAll", data);
        return promise.then((data) => exports.QueryAllReplayProtectionResponse.decode(new minimal_1.Reader(data)));
    }
    SequenceCounter(request) {
        const data = exports.QueryGetSequenceCounterRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "SequenceCounter", data);
        return promise.then((data) => exports.QueryGetSequenceCounterResponse.decode(new minimal_1.Reader(data)));
    }
    SequenceCounterAll(request) {
        const data = exports.QueryAllSequenceCounterRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "SequenceCounterAll", data);
        return promise.then((data) => exports.QueryAllSequenceCounterResponse.decode(new minimal_1.Reader(data)));
    }
    ActiveGuardianSetIndex(request) {
        const data = exports.QueryGetActiveGuardianSetIndexRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "ActiveGuardianSetIndex", data);
        return promise.then((data) => exports.QueryGetActiveGuardianSetIndexResponse.decode(new minimal_1.Reader(data)));
    }
    GuardianValidator(request) {
        const data = exports.QueryGetGuardianValidatorRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianValidator", data);
        return promise.then((data) => exports.QueryGetGuardianValidatorResponse.decode(new minimal_1.Reader(data)));
    }
    GuardianValidatorAll(request) {
        const data = exports.QueryAllGuardianValidatorRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianValidatorAll", data);
        return promise.then((data) => exports.QueryAllGuardianValidatorResponse.decode(new minimal_1.Reader(data)));
    }
}
exports.QueryClientImpl = QueryClientImpl;
var globalThis = (() => {
    if (typeof globalThis !== "undefined")
        return globalThis;
    if (typeof self !== "undefined")
        return self;
    if (typeof window !== "undefined")
        return window;
    if (typeof global !== "undefined")
        return global;
    throw "Unable to locate global object";
})();
const atob = globalThis.atob ||
    ((b64) => globalThis.Buffer.from(b64, "base64").toString("binary"));
function bytesFromBase64(b64) {
    const bin = atob(b64);
    const arr = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; ++i) {
        arr[i] = bin.charCodeAt(i);
    }
    return arr;
}
const btoa = globalThis.btoa ||
    ((bin) => globalThis.Buffer.from(bin, "binary").toString("base64"));
function base64FromBytes(arr) {
    const bin = [];
    for (let i = 0; i < arr.byteLength; ++i) {
        bin.push(String.fromCharCode(arr[i]));
    }
    return btoa(bin.join(""));
}
