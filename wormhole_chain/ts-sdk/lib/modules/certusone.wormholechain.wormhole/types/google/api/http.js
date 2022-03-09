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
import { Writer, Reader } from "protobufjs/minimal";
export var protobufPackage = "google.api";
var baseHttp = { fullyDecodeReservedExpansion: false };
export var Http = {
    encode: function (message, writer) {
        var e_1, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.rules), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                HttpRule.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_1) throw e_1.error; }
        }
        if (message.fullyDecodeReservedExpansion === true) {
            writer.uint32(16).bool(message.fullyDecodeReservedExpansion);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseHttp);
        message.rules = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.rules.push(HttpRule.decode(reader, reader.uint32()));
                    break;
                case 2:
                    message.fullyDecodeReservedExpansion = reader.bool();
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
        var message = __assign({}, baseHttp);
        message.rules = [];
        if (object.rules !== undefined && object.rules !== null) {
            try {
                for (var _b = __values(object.rules), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.rules.push(HttpRule.fromJSON(e));
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
        if (object.fullyDecodeReservedExpansion !== undefined &&
            object.fullyDecodeReservedExpansion !== null) {
            message.fullyDecodeReservedExpansion = Boolean(object.fullyDecodeReservedExpansion);
        }
        else {
            message.fullyDecodeReservedExpansion = false;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.rules) {
            obj.rules = message.rules.map(function (e) {
                return e ? HttpRule.toJSON(e) : undefined;
            });
        }
        else {
            obj.rules = [];
        }
        message.fullyDecodeReservedExpansion !== undefined &&
            (obj.fullyDecodeReservedExpansion = message.fullyDecodeReservedExpansion);
        return obj;
    },
    fromPartial: function (object) {
        var e_3, _a;
        var message = __assign({}, baseHttp);
        message.rules = [];
        if (object.rules !== undefined && object.rules !== null) {
            try {
                for (var _b = __values(object.rules), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.rules.push(HttpRule.fromPartial(e));
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
        if (object.fullyDecodeReservedExpansion !== undefined &&
            object.fullyDecodeReservedExpansion !== null) {
            message.fullyDecodeReservedExpansion =
                object.fullyDecodeReservedExpansion;
        }
        else {
            message.fullyDecodeReservedExpansion = false;
        }
        return message;
    },
};
var baseHttpRule = { selector: "", body: "", responseBody: "" };
export var HttpRule = {
    encode: function (message, writer) {
        var e_4, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.selector !== "") {
            writer.uint32(10).string(message.selector);
        }
        if (message.get !== undefined) {
            writer.uint32(18).string(message.get);
        }
        if (message.put !== undefined) {
            writer.uint32(26).string(message.put);
        }
        if (message.post !== undefined) {
            writer.uint32(34).string(message.post);
        }
        if (message.delete !== undefined) {
            writer.uint32(42).string(message.delete);
        }
        if (message.patch !== undefined) {
            writer.uint32(50).string(message.patch);
        }
        if (message.custom !== undefined) {
            CustomHttpPattern.encode(message.custom, writer.uint32(66).fork()).ldelim();
        }
        if (message.body !== "") {
            writer.uint32(58).string(message.body);
        }
        if (message.responseBody !== "") {
            writer.uint32(98).string(message.responseBody);
        }
        try {
            for (var _b = __values(message.additionalBindings), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                HttpRule.encode(v, writer.uint32(90).fork()).ldelim();
            }
        }
        catch (e_4_1) { e_4 = { error: e_4_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_4) throw e_4.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseHttpRule);
        message.additionalBindings = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.selector = reader.string();
                    break;
                case 2:
                    message.get = reader.string();
                    break;
                case 3:
                    message.put = reader.string();
                    break;
                case 4:
                    message.post = reader.string();
                    break;
                case 5:
                    message.delete = reader.string();
                    break;
                case 6:
                    message.patch = reader.string();
                    break;
                case 8:
                    message.custom = CustomHttpPattern.decode(reader, reader.uint32());
                    break;
                case 7:
                    message.body = reader.string();
                    break;
                case 12:
                    message.responseBody = reader.string();
                    break;
                case 11:
                    message.additionalBindings.push(HttpRule.decode(reader, reader.uint32()));
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
        var message = __assign({}, baseHttpRule);
        message.additionalBindings = [];
        if (object.selector !== undefined && object.selector !== null) {
            message.selector = String(object.selector);
        }
        else {
            message.selector = "";
        }
        if (object.get !== undefined && object.get !== null) {
            message.get = String(object.get);
        }
        else {
            message.get = undefined;
        }
        if (object.put !== undefined && object.put !== null) {
            message.put = String(object.put);
        }
        else {
            message.put = undefined;
        }
        if (object.post !== undefined && object.post !== null) {
            message.post = String(object.post);
        }
        else {
            message.post = undefined;
        }
        if (object.delete !== undefined && object.delete !== null) {
            message.delete = String(object.delete);
        }
        else {
            message.delete = undefined;
        }
        if (object.patch !== undefined && object.patch !== null) {
            message.patch = String(object.patch);
        }
        else {
            message.patch = undefined;
        }
        if (object.custom !== undefined && object.custom !== null) {
            message.custom = CustomHttpPattern.fromJSON(object.custom);
        }
        else {
            message.custom = undefined;
        }
        if (object.body !== undefined && object.body !== null) {
            message.body = String(object.body);
        }
        else {
            message.body = "";
        }
        if (object.responseBody !== undefined && object.responseBody !== null) {
            message.responseBody = String(object.responseBody);
        }
        else {
            message.responseBody = "";
        }
        if (object.additionalBindings !== undefined &&
            object.additionalBindings !== null) {
            try {
                for (var _b = __values(object.additionalBindings), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.additionalBindings.push(HttpRule.fromJSON(e));
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
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.selector !== undefined && (obj.selector = message.selector);
        message.get !== undefined && (obj.get = message.get);
        message.put !== undefined && (obj.put = message.put);
        message.post !== undefined && (obj.post = message.post);
        message.delete !== undefined && (obj.delete = message.delete);
        message.patch !== undefined && (obj.patch = message.patch);
        message.custom !== undefined &&
            (obj.custom = message.custom
                ? CustomHttpPattern.toJSON(message.custom)
                : undefined);
        message.body !== undefined && (obj.body = message.body);
        message.responseBody !== undefined &&
            (obj.responseBody = message.responseBody);
        if (message.additionalBindings) {
            obj.additionalBindings = message.additionalBindings.map(function (e) {
                return e ? HttpRule.toJSON(e) : undefined;
            });
        }
        else {
            obj.additionalBindings = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_6, _a;
        var message = __assign({}, baseHttpRule);
        message.additionalBindings = [];
        if (object.selector !== undefined && object.selector !== null) {
            message.selector = object.selector;
        }
        else {
            message.selector = "";
        }
        if (object.get !== undefined && object.get !== null) {
            message.get = object.get;
        }
        else {
            message.get = undefined;
        }
        if (object.put !== undefined && object.put !== null) {
            message.put = object.put;
        }
        else {
            message.put = undefined;
        }
        if (object.post !== undefined && object.post !== null) {
            message.post = object.post;
        }
        else {
            message.post = undefined;
        }
        if (object.delete !== undefined && object.delete !== null) {
            message.delete = object.delete;
        }
        else {
            message.delete = undefined;
        }
        if (object.patch !== undefined && object.patch !== null) {
            message.patch = object.patch;
        }
        else {
            message.patch = undefined;
        }
        if (object.custom !== undefined && object.custom !== null) {
            message.custom = CustomHttpPattern.fromPartial(object.custom);
        }
        else {
            message.custom = undefined;
        }
        if (object.body !== undefined && object.body !== null) {
            message.body = object.body;
        }
        else {
            message.body = "";
        }
        if (object.responseBody !== undefined && object.responseBody !== null) {
            message.responseBody = object.responseBody;
        }
        else {
            message.responseBody = "";
        }
        if (object.additionalBindings !== undefined &&
            object.additionalBindings !== null) {
            try {
                for (var _b = __values(object.additionalBindings), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.additionalBindings.push(HttpRule.fromPartial(e));
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
        return message;
    },
};
var baseCustomHttpPattern = { kind: "", path: "" };
export var CustomHttpPattern = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.kind !== "") {
            writer.uint32(10).string(message.kind);
        }
        if (message.path !== "") {
            writer.uint32(18).string(message.path);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseCustomHttpPattern);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.kind = reader.string();
                    break;
                case 2:
                    message.path = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseCustomHttpPattern);
        if (object.kind !== undefined && object.kind !== null) {
            message.kind = String(object.kind);
        }
        else {
            message.kind = "";
        }
        if (object.path !== undefined && object.path !== null) {
            message.path = String(object.path);
        }
        else {
            message.path = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.kind !== undefined && (obj.kind = message.kind);
        message.path !== undefined && (obj.path = message.path);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseCustomHttpPattern);
        if (object.kind !== undefined && object.kind !== null) {
            message.kind = object.kind;
        }
        else {
            message.kind = "";
        }
        if (object.path !== undefined && object.path !== null) {
            message.path = object.path;
        }
        else {
            message.path = "";
        }
        return message;
    },
};
