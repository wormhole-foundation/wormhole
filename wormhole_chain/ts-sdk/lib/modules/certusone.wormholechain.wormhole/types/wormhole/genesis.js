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
import { GuardianSet } from "../wormhole/guardian_set";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";
import { Writer, Reader } from "protobufjs/minimal";
export var protobufPackage = "certusone.wormholechain.wormhole";
var baseGenesisState = { guardianSetCount: 0 };
export var GenesisState = {
    encode: function (message, writer) {
        var e_1, _a, e_2, _b, e_3, _c;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _d = __values(message.guardianSetList), _e = _d.next(); !_e.done; _e = _d.next()) {
                var v = _e.value;
                GuardianSet.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
            }
            finally { if (e_1) throw e_1.error; }
        }
        if (message.guardianSetCount !== 0) {
            writer.uint32(16).uint32(message.guardianSetCount);
        }
        if (message.config !== undefined) {
            Config.encode(message.config, writer.uint32(26).fork()).ldelim();
        }
        try {
            for (var _f = __values(message.replayProtectionList), _g = _f.next(); !_g.done; _g = _f.next()) {
                var v = _g.value;
                ReplayProtection.encode(v, writer.uint32(34).fork()).ldelim();
            }
        }
        catch (e_2_1) { e_2 = { error: e_2_1 }; }
        finally {
            try {
                if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
            }
            finally { if (e_2) throw e_2.error; }
        }
        try {
            for (var _h = __values(message.sequenceCounterList), _j = _h.next(); !_j.done; _j = _h.next()) {
                var v = _j.value;
                SequenceCounter.encode(v, writer.uint32(42).fork()).ldelim();
            }
        }
        catch (e_3_1) { e_3 = { error: e_3_1 }; }
        finally {
            try {
                if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
            }
            finally { if (e_3) throw e_3.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseGenesisState);
        message.guardianSetList = [];
        message.replayProtectionList = [];
        message.sequenceCounterList = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
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
                case 5:
                    message.sequenceCounterList.push(SequenceCounter.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_4, _a, e_5, _b, e_6, _c;
        var message = __assign({}, baseGenesisState);
        message.guardianSetList = [];
        message.replayProtectionList = [];
        message.sequenceCounterList = [];
        if (object.guardianSetList !== undefined &&
            object.guardianSetList !== null) {
            try {
                for (var _d = __values(object.guardianSetList), _e = _d.next(); !_e.done; _e = _d.next()) {
                    var e = _e.value;
                    message.guardianSetList.push(GuardianSet.fromJSON(e));
                }
            }
            catch (e_4_1) { e_4 = { error: e_4_1 }; }
            finally {
                try {
                    if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
                }
                finally { if (e_4) throw e_4.error; }
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
            try {
                for (var _f = __values(object.replayProtectionList), _g = _f.next(); !_g.done; _g = _f.next()) {
                    var e = _g.value;
                    message.replayProtectionList.push(ReplayProtection.fromJSON(e));
                }
            }
            catch (e_5_1) { e_5 = { error: e_5_1 }; }
            finally {
                try {
                    if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
                }
                finally { if (e_5) throw e_5.error; }
            }
        }
        if (object.sequenceCounterList !== undefined &&
            object.sequenceCounterList !== null) {
            try {
                for (var _h = __values(object.sequenceCounterList), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.sequenceCounterList.push(SequenceCounter.fromJSON(e));
                }
            }
            catch (e_6_1) { e_6 = { error: e_6_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
                }
                finally { if (e_6) throw e_6.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.guardianSetList) {
            obj.guardianSetList = message.guardianSetList.map(function (e) {
                return e ? GuardianSet.toJSON(e) : undefined;
            });
        }
        else {
            obj.guardianSetList = [];
        }
        message.guardianSetCount !== undefined &&
            (obj.guardianSetCount = message.guardianSetCount);
        message.config !== undefined &&
            (obj.config = message.config ? Config.toJSON(message.config) : undefined);
        if (message.replayProtectionList) {
            obj.replayProtectionList = message.replayProtectionList.map(function (e) {
                return e ? ReplayProtection.toJSON(e) : undefined;
            });
        }
        else {
            obj.replayProtectionList = [];
        }
        if (message.sequenceCounterList) {
            obj.sequenceCounterList = message.sequenceCounterList.map(function (e) {
                return e ? SequenceCounter.toJSON(e) : undefined;
            });
        }
        else {
            obj.sequenceCounterList = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_7, _a, e_8, _b, e_9, _c;
        var message = __assign({}, baseGenesisState);
        message.guardianSetList = [];
        message.replayProtectionList = [];
        message.sequenceCounterList = [];
        if (object.guardianSetList !== undefined &&
            object.guardianSetList !== null) {
            try {
                for (var _d = __values(object.guardianSetList), _e = _d.next(); !_e.done; _e = _d.next()) {
                    var e = _e.value;
                    message.guardianSetList.push(GuardianSet.fromPartial(e));
                }
            }
            catch (e_7_1) { e_7 = { error: e_7_1 }; }
            finally {
                try {
                    if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
                }
                finally { if (e_7) throw e_7.error; }
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
            try {
                for (var _f = __values(object.replayProtectionList), _g = _f.next(); !_g.done; _g = _f.next()) {
                    var e = _g.value;
                    message.replayProtectionList.push(ReplayProtection.fromPartial(e));
                }
            }
            catch (e_8_1) { e_8 = { error: e_8_1 }; }
            finally {
                try {
                    if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
                }
                finally { if (e_8) throw e_8.error; }
            }
        }
        if (object.sequenceCounterList !== undefined &&
            object.sequenceCounterList !== null) {
            try {
                for (var _h = __values(object.sequenceCounterList), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.sequenceCounterList.push(SequenceCounter.fromPartial(e));
                }
            }
            catch (e_9_1) { e_9 = { error: e_9_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
                }
                finally { if (e_9) throw e_9.error; }
            }
        }
        return message;
    },
};
