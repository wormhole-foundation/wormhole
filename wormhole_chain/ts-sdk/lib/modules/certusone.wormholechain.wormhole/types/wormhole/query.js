var __assign = (this && this.__assign) || function () {
    __assign = Object.assign || function(t) {
        for (var s, i = 1, n = arguments.length; i < n; i++) {
            s = arguments[i];
            for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p))
                t[p] = s[p];
        }
        return t;
    };
    return __assign.apply(this, arguments);
};
var __values = (this && this.__values) || function(o) {
    var s = typeof Symbol === "function" && Symbol.iterator, m = s && o[s], i = 0;
    if (m) return m.call(o);
    if (o && typeof o.length === "number") return {
        next: function () {
            if (o && i >= o.length) o = void 0;
            return { value: o && o[i++], done: !o };
        }
    };
    throw new TypeError(s ? "Object is not iterable." : "Symbol.iterator is not defined.");
};
//@ts-nocheck
/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { GuardianSet } from "../wormhole/guardian_set";
import { PageRequest, PageResponse, } from "../cosmos/base/query/v1beta1/pagination";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";
export var protobufPackage = "certusone.wormholechain.wormhole";
var baseQueryGetGuardianSetRequest = { index: 0 };
export var QueryGetGuardianSetRequest = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.index !== 0) {
            writer.uint32(8).uint32(message.index);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetGuardianSetRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
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
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetGuardianSetRequest);
        if (object.index !== undefined && object.index !== null) {
            message.index = Number(object.index);
        }
        else {
            message.index = 0;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetGuardianSetRequest);
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = 0;
        }
        return message;
    },
};
var baseQueryGetGuardianSetResponse = {};
export var QueryGetGuardianSetResponse = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.GuardianSet !== undefined) {
            GuardianSet.encode(message.GuardianSet, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetGuardianSetResponse);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.GuardianSet = GuardianSet.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetGuardianSetResponse);
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            message.GuardianSet = GuardianSet.fromJSON(object.GuardianSet);
        }
        else {
            message.GuardianSet = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.GuardianSet !== undefined &&
            (obj.GuardianSet = message.GuardianSet
                ? GuardianSet.toJSON(message.GuardianSet)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetGuardianSetResponse);
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            message.GuardianSet = GuardianSet.fromPartial(object.GuardianSet);
        }
        else {
            message.GuardianSet = undefined;
        }
        return message;
    },
};
var baseQueryAllGuardianSetRequest = {};
export var QueryAllGuardianSetRequest = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryAllGuardianSetRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryAllGuardianSetRequest);
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryAllGuardianSetRequest);
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
var baseQueryAllGuardianSetResponse = {};
export var QueryAllGuardianSetResponse = {
    encode: function (message, writer) {
        var e_1, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.GuardianSet), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                GuardianSet.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_1) throw e_1.error; }
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryAllGuardianSetResponse);
        message.GuardianSet = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.GuardianSet.push(GuardianSet.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_2, _a;
        var message = __assign({}, baseQueryAllGuardianSetResponse);
        message.GuardianSet = [];
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            try {
                for (var _b = __values(object.GuardianSet), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.GuardianSet.push(GuardianSet.fromJSON(e));
                }
            }
            catch (e_2_1) { e_2 = { error: e_2_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_2) throw e_2.error; }
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.GuardianSet) {
            obj.GuardianSet = message.GuardianSet.map(function (e) {
                return e ? GuardianSet.toJSON(e) : undefined;
            });
        }
        else {
            obj.GuardianSet = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var e_3, _a;
        var message = __assign({}, baseQueryAllGuardianSetResponse);
        message.GuardianSet = [];
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            try {
                for (var _b = __values(object.GuardianSet), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.GuardianSet.push(GuardianSet.fromPartial(e));
                }
            }
            catch (e_3_1) { e_3 = { error: e_3_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_3) throw e_3.error; }
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
var baseQueryGetConfigRequest = {};
export var QueryGetConfigRequest = {
    encode: function (_, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetConfigRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (_) {
        var message = __assign({}, baseQueryGetConfigRequest);
        return message;
    },
    toJSON: function (_) {
        var obj = {};
        return obj;
    },
    fromPartial: function (_) {
        var message = __assign({}, baseQueryGetConfigRequest);
        return message;
    },
};
var baseQueryGetConfigResponse = {};
export var QueryGetConfigResponse = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.Config !== undefined) {
            Config.encode(message.Config, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetConfigResponse);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.Config = Config.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetConfigResponse);
        if (object.Config !== undefined && object.Config !== null) {
            message.Config = Config.fromJSON(object.Config);
        }
        else {
            message.Config = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.Config !== undefined &&
            (obj.Config = message.Config ? Config.toJSON(message.Config) : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetConfigResponse);
        if (object.Config !== undefined && object.Config !== null) {
            message.Config = Config.fromPartial(object.Config);
        }
        else {
            message.Config = undefined;
        }
        return message;
    },
};
var baseQueryGetReplayProtectionRequest = { index: "" };
export var QueryGetReplayProtectionRequest = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetReplayProtectionRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
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
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetReplayProtectionRequest);
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetReplayProtectionRequest);
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
var baseQueryGetReplayProtectionResponse = {};
export var QueryGetReplayProtectionResponse = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.replayProtection !== undefined) {
            ReplayProtection.encode(message.replayProtection, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetReplayProtectionResponse);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.replayProtection = ReplayProtection.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetReplayProtectionResponse);
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            message.replayProtection = ReplayProtection.fromJSON(object.replayProtection);
        }
        else {
            message.replayProtection = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.replayProtection !== undefined &&
            (obj.replayProtection = message.replayProtection
                ? ReplayProtection.toJSON(message.replayProtection)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetReplayProtectionResponse);
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            message.replayProtection = ReplayProtection.fromPartial(object.replayProtection);
        }
        else {
            message.replayProtection = undefined;
        }
        return message;
    },
};
var baseQueryAllReplayProtectionRequest = {};
export var QueryAllReplayProtectionRequest = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryAllReplayProtectionRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryAllReplayProtectionRequest);
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryAllReplayProtectionRequest);
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
var baseQueryAllReplayProtectionResponse = {};
export var QueryAllReplayProtectionResponse = {
    encode: function (message, writer) {
        var e_4, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.replayProtection), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                ReplayProtection.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_4_1) { e_4 = { error: e_4_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_4) throw e_4.error; }
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryAllReplayProtectionResponse);
        message.replayProtection = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.replayProtection.push(ReplayProtection.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_5, _a;
        var message = __assign({}, baseQueryAllReplayProtectionResponse);
        message.replayProtection = [];
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            try {
                for (var _b = __values(object.replayProtection), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.replayProtection.push(ReplayProtection.fromJSON(e));
                }
            }
            catch (e_5_1) { e_5 = { error: e_5_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_5) throw e_5.error; }
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.replayProtection) {
            obj.replayProtection = message.replayProtection.map(function (e) {
                return e ? ReplayProtection.toJSON(e) : undefined;
            });
        }
        else {
            obj.replayProtection = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var e_6, _a;
        var message = __assign({}, baseQueryAllReplayProtectionResponse);
        message.replayProtection = [];
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            try {
                for (var _b = __values(object.replayProtection), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.replayProtection.push(ReplayProtection.fromPartial(e));
                }
            }
            catch (e_6_1) { e_6 = { error: e_6_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_6) throw e_6.error; }
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
var baseQueryGetSequenceCounterRequest = { index: "" };
export var QueryGetSequenceCounterRequest = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetSequenceCounterRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
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
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetSequenceCounterRequest);
        if (object.index !== undefined && object.index !== null) {
            message.index = String(object.index);
        }
        else {
            message.index = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.index !== undefined && (obj.index = message.index);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetSequenceCounterRequest);
        if (object.index !== undefined && object.index !== null) {
            message.index = object.index;
        }
        else {
            message.index = "";
        }
        return message;
    },
};
var baseQueryGetSequenceCounterResponse = {};
export var QueryGetSequenceCounterResponse = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.sequenceCounter !== undefined) {
            SequenceCounter.encode(message.sequenceCounter, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryGetSequenceCounterResponse);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.sequenceCounter = SequenceCounter.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryGetSequenceCounterResponse);
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            message.sequenceCounter = SequenceCounter.fromJSON(object.sequenceCounter);
        }
        else {
            message.sequenceCounter = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.sequenceCounter !== undefined &&
            (obj.sequenceCounter = message.sequenceCounter
                ? SequenceCounter.toJSON(message.sequenceCounter)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryGetSequenceCounterResponse);
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            message.sequenceCounter = SequenceCounter.fromPartial(object.sequenceCounter);
        }
        else {
            message.sequenceCounter = undefined;
        }
        return message;
    },
};
var baseQueryAllSequenceCounterRequest = {};
export var QueryAllSequenceCounterRequest = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryAllSequenceCounterRequest);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.pagination = PageRequest.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseQueryAllSequenceCounterRequest);
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseQueryAllSequenceCounterRequest);
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
var baseQueryAllSequenceCounterResponse = {};
export var QueryAllSequenceCounterResponse = {
    encode: function (message, writer) {
        var e_7, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.sequenceCounter), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                SequenceCounter.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_7_1) { e_7 = { error: e_7_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_7) throw e_7.error; }
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseQueryAllSequenceCounterResponse);
        message.sequenceCounter = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.sequenceCounter.push(SequenceCounter.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.pagination = PageResponse.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_8, _a;
        var message = __assign({}, baseQueryAllSequenceCounterResponse);
        message.sequenceCounter = [];
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            try {
                for (var _b = __values(object.sequenceCounter), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.sequenceCounter.push(SequenceCounter.fromJSON(e));
                }
            }
            catch (e_8_1) { e_8 = { error: e_8_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_8) throw e_8.error; }
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromJSON(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.sequenceCounter) {
            obj.sequenceCounter = message.sequenceCounter.map(function (e) {
                return e ? SequenceCounter.toJSON(e) : undefined;
            });
        }
        else {
            obj.sequenceCounter = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var e_9, _a;
        var message = __assign({}, baseQueryAllSequenceCounterResponse);
        message.sequenceCounter = [];
        if (object.sequenceCounter !== undefined &&
            object.sequenceCounter !== null) {
            try {
                for (var _b = __values(object.sequenceCounter), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.sequenceCounter.push(SequenceCounter.fromPartial(e));
                }
            }
            catch (e_9_1) { e_9 = { error: e_9_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_9) throw e_9.error; }
            }
        }
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageResponse.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
var QueryClientImpl = /** @class */ (function () {
    function QueryClientImpl(rpc) {
        this.rpc = rpc;
    }
    QueryClientImpl.prototype.GuardianSet = function (request) {
        var data = QueryGetGuardianSetRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianSet", data);
        return promise.then(function (data) {
            return QueryGetGuardianSetResponse.decode(new Reader(data));
        });
    };
    QueryClientImpl.prototype.GuardianSetAll = function (request) {
        var data = QueryAllGuardianSetRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianSetAll", data);
        return promise.then(function (data) {
            return QueryAllGuardianSetResponse.decode(new Reader(data));
        });
    };
    QueryClientImpl.prototype.Config = function (request) {
        var data = QueryGetConfigRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "Config", data);
        return promise.then(function (data) {
            return QueryGetConfigResponse.decode(new Reader(data));
        });
    };
    QueryClientImpl.prototype.ReplayProtection = function (request) {
        var data = QueryGetReplayProtectionRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "ReplayProtection", data);
        return promise.then(function (data) {
            return QueryGetReplayProtectionResponse.decode(new Reader(data));
        });
    };
    QueryClientImpl.prototype.ReplayProtectionAll = function (request) {
        var data = QueryAllReplayProtectionRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "ReplayProtectionAll", data);
        return promise.then(function (data) {
            return QueryAllReplayProtectionResponse.decode(new Reader(data));
        });
    };
    QueryClientImpl.prototype.SequenceCounter = function (request) {
        var data = QueryGetSequenceCounterRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "SequenceCounter", data);
        return promise.then(function (data) {
            return QueryGetSequenceCounterResponse.decode(new Reader(data));
        });
    };
    QueryClientImpl.prototype.SequenceCounterAll = function (request) {
        var data = QueryAllSequenceCounterRequest.encode(request).finish();
        var promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "SequenceCounterAll", data);
        return promise.then(function (data) {
            return QueryAllSequenceCounterResponse.decode(new Reader(data));
        });
    };
    return QueryClientImpl;
}());
export { QueryClientImpl };
