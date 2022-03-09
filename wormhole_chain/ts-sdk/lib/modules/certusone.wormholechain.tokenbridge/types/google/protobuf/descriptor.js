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
import * as Long from "long";
import { util, configure, Writer, Reader } from "protobufjs/minimal";
export var protobufPackage = "google.protobuf";
export var FieldDescriptorProto_Type;
(function (FieldDescriptorProto_Type) {
    /**
     * TYPE_DOUBLE - 0 is reserved for errors.
     * Order is weird for historical reasons.
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_DOUBLE"] = 1] = "TYPE_DOUBLE";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_FLOAT"] = 2] = "TYPE_FLOAT";
    /**
     * TYPE_INT64 - Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT64 if
     * negative values are likely.
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_INT64"] = 3] = "TYPE_INT64";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_UINT64"] = 4] = "TYPE_UINT64";
    /**
     * TYPE_INT32 - Not ZigZag encoded.  Negative numbers take 10 bytes.  Use TYPE_SINT32 if
     * negative values are likely.
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_INT32"] = 5] = "TYPE_INT32";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_FIXED64"] = 6] = "TYPE_FIXED64";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_FIXED32"] = 7] = "TYPE_FIXED32";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_BOOL"] = 8] = "TYPE_BOOL";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_STRING"] = 9] = "TYPE_STRING";
    /**
     * TYPE_GROUP - Tag-delimited aggregate.
     * Group type is deprecated and not supported in proto3. However, Proto3
     * implementations should still be able to parse the group wire format and
     * treat group fields as unknown fields.
     */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_GROUP"] = 10] = "TYPE_GROUP";
    /** TYPE_MESSAGE - Length-delimited aggregate. */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_MESSAGE"] = 11] = "TYPE_MESSAGE";
    /** TYPE_BYTES - New in version 2. */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_BYTES"] = 12] = "TYPE_BYTES";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_UINT32"] = 13] = "TYPE_UINT32";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_ENUM"] = 14] = "TYPE_ENUM";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_SFIXED32"] = 15] = "TYPE_SFIXED32";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_SFIXED64"] = 16] = "TYPE_SFIXED64";
    /** TYPE_SINT32 - Uses ZigZag encoding. */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_SINT32"] = 17] = "TYPE_SINT32";
    /** TYPE_SINT64 - Uses ZigZag encoding. */
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["TYPE_SINT64"] = 18] = "TYPE_SINT64";
    FieldDescriptorProto_Type[FieldDescriptorProto_Type["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(FieldDescriptorProto_Type || (FieldDescriptorProto_Type = {}));
export function fieldDescriptorProto_TypeFromJSON(object) {
    switch (object) {
        case 1:
        case "TYPE_DOUBLE":
            return FieldDescriptorProto_Type.TYPE_DOUBLE;
        case 2:
        case "TYPE_FLOAT":
            return FieldDescriptorProto_Type.TYPE_FLOAT;
        case 3:
        case "TYPE_INT64":
            return FieldDescriptorProto_Type.TYPE_INT64;
        case 4:
        case "TYPE_UINT64":
            return FieldDescriptorProto_Type.TYPE_UINT64;
        case 5:
        case "TYPE_INT32":
            return FieldDescriptorProto_Type.TYPE_INT32;
        case 6:
        case "TYPE_FIXED64":
            return FieldDescriptorProto_Type.TYPE_FIXED64;
        case 7:
        case "TYPE_FIXED32":
            return FieldDescriptorProto_Type.TYPE_FIXED32;
        case 8:
        case "TYPE_BOOL":
            return FieldDescriptorProto_Type.TYPE_BOOL;
        case 9:
        case "TYPE_STRING":
            return FieldDescriptorProto_Type.TYPE_STRING;
        case 10:
        case "TYPE_GROUP":
            return FieldDescriptorProto_Type.TYPE_GROUP;
        case 11:
        case "TYPE_MESSAGE":
            return FieldDescriptorProto_Type.TYPE_MESSAGE;
        case 12:
        case "TYPE_BYTES":
            return FieldDescriptorProto_Type.TYPE_BYTES;
        case 13:
        case "TYPE_UINT32":
            return FieldDescriptorProto_Type.TYPE_UINT32;
        case 14:
        case "TYPE_ENUM":
            return FieldDescriptorProto_Type.TYPE_ENUM;
        case 15:
        case "TYPE_SFIXED32":
            return FieldDescriptorProto_Type.TYPE_SFIXED32;
        case 16:
        case "TYPE_SFIXED64":
            return FieldDescriptorProto_Type.TYPE_SFIXED64;
        case 17:
        case "TYPE_SINT32":
            return FieldDescriptorProto_Type.TYPE_SINT32;
        case 18:
        case "TYPE_SINT64":
            return FieldDescriptorProto_Type.TYPE_SINT64;
        case -1:
        case "UNRECOGNIZED":
        default:
            return FieldDescriptorProto_Type.UNRECOGNIZED;
    }
}
export function fieldDescriptorProto_TypeToJSON(object) {
    switch (object) {
        case FieldDescriptorProto_Type.TYPE_DOUBLE:
            return "TYPE_DOUBLE";
        case FieldDescriptorProto_Type.TYPE_FLOAT:
            return "TYPE_FLOAT";
        case FieldDescriptorProto_Type.TYPE_INT64:
            return "TYPE_INT64";
        case FieldDescriptorProto_Type.TYPE_UINT64:
            return "TYPE_UINT64";
        case FieldDescriptorProto_Type.TYPE_INT32:
            return "TYPE_INT32";
        case FieldDescriptorProto_Type.TYPE_FIXED64:
            return "TYPE_FIXED64";
        case FieldDescriptorProto_Type.TYPE_FIXED32:
            return "TYPE_FIXED32";
        case FieldDescriptorProto_Type.TYPE_BOOL:
            return "TYPE_BOOL";
        case FieldDescriptorProto_Type.TYPE_STRING:
            return "TYPE_STRING";
        case FieldDescriptorProto_Type.TYPE_GROUP:
            return "TYPE_GROUP";
        case FieldDescriptorProto_Type.TYPE_MESSAGE:
            return "TYPE_MESSAGE";
        case FieldDescriptorProto_Type.TYPE_BYTES:
            return "TYPE_BYTES";
        case FieldDescriptorProto_Type.TYPE_UINT32:
            return "TYPE_UINT32";
        case FieldDescriptorProto_Type.TYPE_ENUM:
            return "TYPE_ENUM";
        case FieldDescriptorProto_Type.TYPE_SFIXED32:
            return "TYPE_SFIXED32";
        case FieldDescriptorProto_Type.TYPE_SFIXED64:
            return "TYPE_SFIXED64";
        case FieldDescriptorProto_Type.TYPE_SINT32:
            return "TYPE_SINT32";
        case FieldDescriptorProto_Type.TYPE_SINT64:
            return "TYPE_SINT64";
        default:
            return "UNKNOWN";
    }
}
export var FieldDescriptorProto_Label;
(function (FieldDescriptorProto_Label) {
    /** LABEL_OPTIONAL - 0 is reserved for errors */
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["LABEL_OPTIONAL"] = 1] = "LABEL_OPTIONAL";
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["LABEL_REQUIRED"] = 2] = "LABEL_REQUIRED";
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["LABEL_REPEATED"] = 3] = "LABEL_REPEATED";
    FieldDescriptorProto_Label[FieldDescriptorProto_Label["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(FieldDescriptorProto_Label || (FieldDescriptorProto_Label = {}));
export function fieldDescriptorProto_LabelFromJSON(object) {
    switch (object) {
        case 1:
        case "LABEL_OPTIONAL":
            return FieldDescriptorProto_Label.LABEL_OPTIONAL;
        case 2:
        case "LABEL_REQUIRED":
            return FieldDescriptorProto_Label.LABEL_REQUIRED;
        case 3:
        case "LABEL_REPEATED":
            return FieldDescriptorProto_Label.LABEL_REPEATED;
        case -1:
        case "UNRECOGNIZED":
        default:
            return FieldDescriptorProto_Label.UNRECOGNIZED;
    }
}
export function fieldDescriptorProto_LabelToJSON(object) {
    switch (object) {
        case FieldDescriptorProto_Label.LABEL_OPTIONAL:
            return "LABEL_OPTIONAL";
        case FieldDescriptorProto_Label.LABEL_REQUIRED:
            return "LABEL_REQUIRED";
        case FieldDescriptorProto_Label.LABEL_REPEATED:
            return "LABEL_REPEATED";
        default:
            return "UNKNOWN";
    }
}
/** Generated classes can be optimized for speed or code size. */
export var FileOptions_OptimizeMode;
(function (FileOptions_OptimizeMode) {
    /** SPEED - Generate complete code for parsing, serialization, */
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["SPEED"] = 1] = "SPEED";
    /** CODE_SIZE - etc. */
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["CODE_SIZE"] = 2] = "CODE_SIZE";
    /** LITE_RUNTIME - Generate code using MessageLite and the lite runtime. */
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["LITE_RUNTIME"] = 3] = "LITE_RUNTIME";
    FileOptions_OptimizeMode[FileOptions_OptimizeMode["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(FileOptions_OptimizeMode || (FileOptions_OptimizeMode = {}));
export function fileOptions_OptimizeModeFromJSON(object) {
    switch (object) {
        case 1:
        case "SPEED":
            return FileOptions_OptimizeMode.SPEED;
        case 2:
        case "CODE_SIZE":
            return FileOptions_OptimizeMode.CODE_SIZE;
        case 3:
        case "LITE_RUNTIME":
            return FileOptions_OptimizeMode.LITE_RUNTIME;
        case -1:
        case "UNRECOGNIZED":
        default:
            return FileOptions_OptimizeMode.UNRECOGNIZED;
    }
}
export function fileOptions_OptimizeModeToJSON(object) {
    switch (object) {
        case FileOptions_OptimizeMode.SPEED:
            return "SPEED";
        case FileOptions_OptimizeMode.CODE_SIZE:
            return "CODE_SIZE";
        case FileOptions_OptimizeMode.LITE_RUNTIME:
            return "LITE_RUNTIME";
        default:
            return "UNKNOWN";
    }
}
export var FieldOptions_CType;
(function (FieldOptions_CType) {
    /** STRING - Default mode. */
    FieldOptions_CType[FieldOptions_CType["STRING"] = 0] = "STRING";
    FieldOptions_CType[FieldOptions_CType["CORD"] = 1] = "CORD";
    FieldOptions_CType[FieldOptions_CType["STRING_PIECE"] = 2] = "STRING_PIECE";
    FieldOptions_CType[FieldOptions_CType["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(FieldOptions_CType || (FieldOptions_CType = {}));
export function fieldOptions_CTypeFromJSON(object) {
    switch (object) {
        case 0:
        case "STRING":
            return FieldOptions_CType.STRING;
        case 1:
        case "CORD":
            return FieldOptions_CType.CORD;
        case 2:
        case "STRING_PIECE":
            return FieldOptions_CType.STRING_PIECE;
        case -1:
        case "UNRECOGNIZED":
        default:
            return FieldOptions_CType.UNRECOGNIZED;
    }
}
export function fieldOptions_CTypeToJSON(object) {
    switch (object) {
        case FieldOptions_CType.STRING:
            return "STRING";
        case FieldOptions_CType.CORD:
            return "CORD";
        case FieldOptions_CType.STRING_PIECE:
            return "STRING_PIECE";
        default:
            return "UNKNOWN";
    }
}
export var FieldOptions_JSType;
(function (FieldOptions_JSType) {
    /** JS_NORMAL - Use the default type. */
    FieldOptions_JSType[FieldOptions_JSType["JS_NORMAL"] = 0] = "JS_NORMAL";
    /** JS_STRING - Use JavaScript strings. */
    FieldOptions_JSType[FieldOptions_JSType["JS_STRING"] = 1] = "JS_STRING";
    /** JS_NUMBER - Use JavaScript numbers. */
    FieldOptions_JSType[FieldOptions_JSType["JS_NUMBER"] = 2] = "JS_NUMBER";
    FieldOptions_JSType[FieldOptions_JSType["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(FieldOptions_JSType || (FieldOptions_JSType = {}));
export function fieldOptions_JSTypeFromJSON(object) {
    switch (object) {
        case 0:
        case "JS_NORMAL":
            return FieldOptions_JSType.JS_NORMAL;
        case 1:
        case "JS_STRING":
            return FieldOptions_JSType.JS_STRING;
        case 2:
        case "JS_NUMBER":
            return FieldOptions_JSType.JS_NUMBER;
        case -1:
        case "UNRECOGNIZED":
        default:
            return FieldOptions_JSType.UNRECOGNIZED;
    }
}
export function fieldOptions_JSTypeToJSON(object) {
    switch (object) {
        case FieldOptions_JSType.JS_NORMAL:
            return "JS_NORMAL";
        case FieldOptions_JSType.JS_STRING:
            return "JS_STRING";
        case FieldOptions_JSType.JS_NUMBER:
            return "JS_NUMBER";
        default:
            return "UNKNOWN";
    }
}
/**
 * Is this method side-effect-free (or safe in HTTP parlance), or idempotent,
 * or neither? HTTP based RPC implementation may choose GET verb for safe
 * methods, and PUT verb for idempotent methods instead of the default POST.
 */
export var MethodOptions_IdempotencyLevel;
(function (MethodOptions_IdempotencyLevel) {
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["IDEMPOTENCY_UNKNOWN"] = 0] = "IDEMPOTENCY_UNKNOWN";
    /** NO_SIDE_EFFECTS - implies idempotent */
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["NO_SIDE_EFFECTS"] = 1] = "NO_SIDE_EFFECTS";
    /** IDEMPOTENT - idempotent, but may have side effects */
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["IDEMPOTENT"] = 2] = "IDEMPOTENT";
    MethodOptions_IdempotencyLevel[MethodOptions_IdempotencyLevel["UNRECOGNIZED"] = -1] = "UNRECOGNIZED";
})(MethodOptions_IdempotencyLevel || (MethodOptions_IdempotencyLevel = {}));
export function methodOptions_IdempotencyLevelFromJSON(object) {
    switch (object) {
        case 0:
        case "IDEMPOTENCY_UNKNOWN":
            return MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN;
        case 1:
        case "NO_SIDE_EFFECTS":
            return MethodOptions_IdempotencyLevel.NO_SIDE_EFFECTS;
        case 2:
        case "IDEMPOTENT":
            return MethodOptions_IdempotencyLevel.IDEMPOTENT;
        case -1:
        case "UNRECOGNIZED":
        default:
            return MethodOptions_IdempotencyLevel.UNRECOGNIZED;
    }
}
export function methodOptions_IdempotencyLevelToJSON(object) {
    switch (object) {
        case MethodOptions_IdempotencyLevel.IDEMPOTENCY_UNKNOWN:
            return "IDEMPOTENCY_UNKNOWN";
        case MethodOptions_IdempotencyLevel.NO_SIDE_EFFECTS:
            return "NO_SIDE_EFFECTS";
        case MethodOptions_IdempotencyLevel.IDEMPOTENT:
            return "IDEMPOTENT";
        default:
            return "UNKNOWN";
    }
}
var baseFileDescriptorSet = {};
export var FileDescriptorSet = {
    encode: function (message, writer) {
        var e_1, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.file), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                FileDescriptorProto.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_1_1) { e_1 = { error: e_1_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_1) throw e_1.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseFileDescriptorSet);
        message.file = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.file.push(FileDescriptorProto.decode(reader, reader.uint32()));
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
        var message = __assign({}, baseFileDescriptorSet);
        message.file = [];
        if (object.file !== undefined && object.file !== null) {
            try {
                for (var _b = __values(object.file), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.file.push(FileDescriptorProto.fromJSON(e));
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
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.file) {
            obj.file = message.file.map(function (e) {
                return e ? FileDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.file = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_3, _a;
        var message = __assign({}, baseFileDescriptorSet);
        message.file = [];
        if (object.file !== undefined && object.file !== null) {
            try {
                for (var _b = __values(object.file), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.file.push(FileDescriptorProto.fromPartial(e));
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
        return message;
    },
};
var baseFileDescriptorProto = {
    name: "",
    package: "",
    dependency: "",
    publicDependency: 0,
    weakDependency: 0,
    syntax: "",
};
export var FileDescriptorProto = {
    encode: function (message, writer) {
        var e_4, _a, e_5, _b, e_6, _c, e_7, _d, e_8, _e, e_9, _f, e_10, _g;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.package !== "") {
            writer.uint32(18).string(message.package);
        }
        try {
            for (var _h = __values(message.dependency), _j = _h.next(); !_j.done; _j = _h.next()) {
                var v = _j.value;
                writer.uint32(26).string(v);
            }
        }
        catch (e_4_1) { e_4 = { error: e_4_1 }; }
        finally {
            try {
                if (_j && !_j.done && (_a = _h.return)) _a.call(_h);
            }
            finally { if (e_4) throw e_4.error; }
        }
        writer.uint32(82).fork();
        try {
            for (var _k = __values(message.publicDependency), _l = _k.next(); !_l.done; _l = _k.next()) {
                var v = _l.value;
                writer.int32(v);
            }
        }
        catch (e_5_1) { e_5 = { error: e_5_1 }; }
        finally {
            try {
                if (_l && !_l.done && (_b = _k.return)) _b.call(_k);
            }
            finally { if (e_5) throw e_5.error; }
        }
        writer.ldelim();
        writer.uint32(90).fork();
        try {
            for (var _m = __values(message.weakDependency), _o = _m.next(); !_o.done; _o = _m.next()) {
                var v = _o.value;
                writer.int32(v);
            }
        }
        catch (e_6_1) { e_6 = { error: e_6_1 }; }
        finally {
            try {
                if (_o && !_o.done && (_c = _m.return)) _c.call(_m);
            }
            finally { if (e_6) throw e_6.error; }
        }
        writer.ldelim();
        try {
            for (var _p = __values(message.messageType), _q = _p.next(); !_q.done; _q = _p.next()) {
                var v = _q.value;
                DescriptorProto.encode(v, writer.uint32(34).fork()).ldelim();
            }
        }
        catch (e_7_1) { e_7 = { error: e_7_1 }; }
        finally {
            try {
                if (_q && !_q.done && (_d = _p.return)) _d.call(_p);
            }
            finally { if (e_7) throw e_7.error; }
        }
        try {
            for (var _r = __values(message.enumType), _s = _r.next(); !_s.done; _s = _r.next()) {
                var v = _s.value;
                EnumDescriptorProto.encode(v, writer.uint32(42).fork()).ldelim();
            }
        }
        catch (e_8_1) { e_8 = { error: e_8_1 }; }
        finally {
            try {
                if (_s && !_s.done && (_e = _r.return)) _e.call(_r);
            }
            finally { if (e_8) throw e_8.error; }
        }
        try {
            for (var _t = __values(message.service), _u = _t.next(); !_u.done; _u = _t.next()) {
                var v = _u.value;
                ServiceDescriptorProto.encode(v, writer.uint32(50).fork()).ldelim();
            }
        }
        catch (e_9_1) { e_9 = { error: e_9_1 }; }
        finally {
            try {
                if (_u && !_u.done && (_f = _t.return)) _f.call(_t);
            }
            finally { if (e_9) throw e_9.error; }
        }
        try {
            for (var _v = __values(message.extension), _w = _v.next(); !_w.done; _w = _v.next()) {
                var v = _w.value;
                FieldDescriptorProto.encode(v, writer.uint32(58).fork()).ldelim();
            }
        }
        catch (e_10_1) { e_10 = { error: e_10_1 }; }
        finally {
            try {
                if (_w && !_w.done && (_g = _v.return)) _g.call(_v);
            }
            finally { if (e_10) throw e_10.error; }
        }
        if (message.options !== undefined) {
            FileOptions.encode(message.options, writer.uint32(66).fork()).ldelim();
        }
        if (message.sourceCodeInfo !== undefined) {
            SourceCodeInfo.encode(message.sourceCodeInfo, writer.uint32(74).fork()).ldelim();
        }
        if (message.syntax !== "") {
            writer.uint32(98).string(message.syntax);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseFileDescriptorProto);
        message.dependency = [];
        message.publicDependency = [];
        message.weakDependency = [];
        message.messageType = [];
        message.enumType = [];
        message.service = [];
        message.extension = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.package = reader.string();
                    break;
                case 3:
                    message.dependency.push(reader.string());
                    break;
                case 10:
                    if ((tag & 7) === 2) {
                        var end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.publicDependency.push(reader.int32());
                        }
                    }
                    else {
                        message.publicDependency.push(reader.int32());
                    }
                    break;
                case 11:
                    if ((tag & 7) === 2) {
                        var end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.weakDependency.push(reader.int32());
                        }
                    }
                    else {
                        message.weakDependency.push(reader.int32());
                    }
                    break;
                case 4:
                    message.messageType.push(DescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 5:
                    message.enumType.push(EnumDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 6:
                    message.service.push(ServiceDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 7:
                    message.extension.push(FieldDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 8:
                    message.options = FileOptions.decode(reader, reader.uint32());
                    break;
                case 9:
                    message.sourceCodeInfo = SourceCodeInfo.decode(reader, reader.uint32());
                    break;
                case 12:
                    message.syntax = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_11, _a, e_12, _b, e_13, _c, e_14, _d, e_15, _e, e_16, _f, e_17, _g;
        var message = __assign({}, baseFileDescriptorProto);
        message.dependency = [];
        message.publicDependency = [];
        message.weakDependency = [];
        message.messageType = [];
        message.enumType = [];
        message.service = [];
        message.extension = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.package !== undefined && object.package !== null) {
            message.package = String(object.package);
        }
        else {
            message.package = "";
        }
        if (object.dependency !== undefined && object.dependency !== null) {
            try {
                for (var _h = __values(object.dependency), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.dependency.push(String(e));
                }
            }
            catch (e_11_1) { e_11 = { error: e_11_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_a = _h.return)) _a.call(_h);
                }
                finally { if (e_11) throw e_11.error; }
            }
        }
        if (object.publicDependency !== undefined &&
            object.publicDependency !== null) {
            try {
                for (var _k = __values(object.publicDependency), _l = _k.next(); !_l.done; _l = _k.next()) {
                    var e = _l.value;
                    message.publicDependency.push(Number(e));
                }
            }
            catch (e_12_1) { e_12 = { error: e_12_1 }; }
            finally {
                try {
                    if (_l && !_l.done && (_b = _k.return)) _b.call(_k);
                }
                finally { if (e_12) throw e_12.error; }
            }
        }
        if (object.weakDependency !== undefined && object.weakDependency !== null) {
            try {
                for (var _m = __values(object.weakDependency), _o = _m.next(); !_o.done; _o = _m.next()) {
                    var e = _o.value;
                    message.weakDependency.push(Number(e));
                }
            }
            catch (e_13_1) { e_13 = { error: e_13_1 }; }
            finally {
                try {
                    if (_o && !_o.done && (_c = _m.return)) _c.call(_m);
                }
                finally { if (e_13) throw e_13.error; }
            }
        }
        if (object.messageType !== undefined && object.messageType !== null) {
            try {
                for (var _p = __values(object.messageType), _q = _p.next(); !_q.done; _q = _p.next()) {
                    var e = _q.value;
                    message.messageType.push(DescriptorProto.fromJSON(e));
                }
            }
            catch (e_14_1) { e_14 = { error: e_14_1 }; }
            finally {
                try {
                    if (_q && !_q.done && (_d = _p.return)) _d.call(_p);
                }
                finally { if (e_14) throw e_14.error; }
            }
        }
        if (object.enumType !== undefined && object.enumType !== null) {
            try {
                for (var _r = __values(object.enumType), _s = _r.next(); !_s.done; _s = _r.next()) {
                    var e = _s.value;
                    message.enumType.push(EnumDescriptorProto.fromJSON(e));
                }
            }
            catch (e_15_1) { e_15 = { error: e_15_1 }; }
            finally {
                try {
                    if (_s && !_s.done && (_e = _r.return)) _e.call(_r);
                }
                finally { if (e_15) throw e_15.error; }
            }
        }
        if (object.service !== undefined && object.service !== null) {
            try {
                for (var _t = __values(object.service), _u = _t.next(); !_u.done; _u = _t.next()) {
                    var e = _u.value;
                    message.service.push(ServiceDescriptorProto.fromJSON(e));
                }
            }
            catch (e_16_1) { e_16 = { error: e_16_1 }; }
            finally {
                try {
                    if (_u && !_u.done && (_f = _t.return)) _f.call(_t);
                }
                finally { if (e_16) throw e_16.error; }
            }
        }
        if (object.extension !== undefined && object.extension !== null) {
            try {
                for (var _v = __values(object.extension), _w = _v.next(); !_w.done; _w = _v.next()) {
                    var e = _w.value;
                    message.extension.push(FieldDescriptorProto.fromJSON(e));
                }
            }
            catch (e_17_1) { e_17 = { error: e_17_1 }; }
            finally {
                try {
                    if (_w && !_w.done && (_g = _v.return)) _g.call(_v);
                }
                finally { if (e_17) throw e_17.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = FileOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.sourceCodeInfo !== undefined && object.sourceCodeInfo !== null) {
            message.sourceCodeInfo = SourceCodeInfo.fromJSON(object.sourceCodeInfo);
        }
        else {
            message.sourceCodeInfo = undefined;
        }
        if (object.syntax !== undefined && object.syntax !== null) {
            message.syntax = String(object.syntax);
        }
        else {
            message.syntax = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.package !== undefined && (obj.package = message.package);
        if (message.dependency) {
            obj.dependency = message.dependency.map(function (e) { return e; });
        }
        else {
            obj.dependency = [];
        }
        if (message.publicDependency) {
            obj.publicDependency = message.publicDependency.map(function (e) { return e; });
        }
        else {
            obj.publicDependency = [];
        }
        if (message.weakDependency) {
            obj.weakDependency = message.weakDependency.map(function (e) { return e; });
        }
        else {
            obj.weakDependency = [];
        }
        if (message.messageType) {
            obj.messageType = message.messageType.map(function (e) {
                return e ? DescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.messageType = [];
        }
        if (message.enumType) {
            obj.enumType = message.enumType.map(function (e) {
                return e ? EnumDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.enumType = [];
        }
        if (message.service) {
            obj.service = message.service.map(function (e) {
                return e ? ServiceDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.service = [];
        }
        if (message.extension) {
            obj.extension = message.extension.map(function (e) {
                return e ? FieldDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.extension = [];
        }
        message.options !== undefined &&
            (obj.options = message.options
                ? FileOptions.toJSON(message.options)
                : undefined);
        message.sourceCodeInfo !== undefined &&
            (obj.sourceCodeInfo = message.sourceCodeInfo
                ? SourceCodeInfo.toJSON(message.sourceCodeInfo)
                : undefined);
        message.syntax !== undefined && (obj.syntax = message.syntax);
        return obj;
    },
    fromPartial: function (object) {
        var e_18, _a, e_19, _b, e_20, _c, e_21, _d, e_22, _e, e_23, _f, e_24, _g;
        var message = __assign({}, baseFileDescriptorProto);
        message.dependency = [];
        message.publicDependency = [];
        message.weakDependency = [];
        message.messageType = [];
        message.enumType = [];
        message.service = [];
        message.extension = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.package !== undefined && object.package !== null) {
            message.package = object.package;
        }
        else {
            message.package = "";
        }
        if (object.dependency !== undefined && object.dependency !== null) {
            try {
                for (var _h = __values(object.dependency), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.dependency.push(e);
                }
            }
            catch (e_18_1) { e_18 = { error: e_18_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_a = _h.return)) _a.call(_h);
                }
                finally { if (e_18) throw e_18.error; }
            }
        }
        if (object.publicDependency !== undefined &&
            object.publicDependency !== null) {
            try {
                for (var _k = __values(object.publicDependency), _l = _k.next(); !_l.done; _l = _k.next()) {
                    var e = _l.value;
                    message.publicDependency.push(e);
                }
            }
            catch (e_19_1) { e_19 = { error: e_19_1 }; }
            finally {
                try {
                    if (_l && !_l.done && (_b = _k.return)) _b.call(_k);
                }
                finally { if (e_19) throw e_19.error; }
            }
        }
        if (object.weakDependency !== undefined && object.weakDependency !== null) {
            try {
                for (var _m = __values(object.weakDependency), _o = _m.next(); !_o.done; _o = _m.next()) {
                    var e = _o.value;
                    message.weakDependency.push(e);
                }
            }
            catch (e_20_1) { e_20 = { error: e_20_1 }; }
            finally {
                try {
                    if (_o && !_o.done && (_c = _m.return)) _c.call(_m);
                }
                finally { if (e_20) throw e_20.error; }
            }
        }
        if (object.messageType !== undefined && object.messageType !== null) {
            try {
                for (var _p = __values(object.messageType), _q = _p.next(); !_q.done; _q = _p.next()) {
                    var e = _q.value;
                    message.messageType.push(DescriptorProto.fromPartial(e));
                }
            }
            catch (e_21_1) { e_21 = { error: e_21_1 }; }
            finally {
                try {
                    if (_q && !_q.done && (_d = _p.return)) _d.call(_p);
                }
                finally { if (e_21) throw e_21.error; }
            }
        }
        if (object.enumType !== undefined && object.enumType !== null) {
            try {
                for (var _r = __values(object.enumType), _s = _r.next(); !_s.done; _s = _r.next()) {
                    var e = _s.value;
                    message.enumType.push(EnumDescriptorProto.fromPartial(e));
                }
            }
            catch (e_22_1) { e_22 = { error: e_22_1 }; }
            finally {
                try {
                    if (_s && !_s.done && (_e = _r.return)) _e.call(_r);
                }
                finally { if (e_22) throw e_22.error; }
            }
        }
        if (object.service !== undefined && object.service !== null) {
            try {
                for (var _t = __values(object.service), _u = _t.next(); !_u.done; _u = _t.next()) {
                    var e = _u.value;
                    message.service.push(ServiceDescriptorProto.fromPartial(e));
                }
            }
            catch (e_23_1) { e_23 = { error: e_23_1 }; }
            finally {
                try {
                    if (_u && !_u.done && (_f = _t.return)) _f.call(_t);
                }
                finally { if (e_23) throw e_23.error; }
            }
        }
        if (object.extension !== undefined && object.extension !== null) {
            try {
                for (var _v = __values(object.extension), _w = _v.next(); !_w.done; _w = _v.next()) {
                    var e = _w.value;
                    message.extension.push(FieldDescriptorProto.fromPartial(e));
                }
            }
            catch (e_24_1) { e_24 = { error: e_24_1 }; }
            finally {
                try {
                    if (_w && !_w.done && (_g = _v.return)) _g.call(_v);
                }
                finally { if (e_24) throw e_24.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = FileOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.sourceCodeInfo !== undefined && object.sourceCodeInfo !== null) {
            message.sourceCodeInfo = SourceCodeInfo.fromPartial(object.sourceCodeInfo);
        }
        else {
            message.sourceCodeInfo = undefined;
        }
        if (object.syntax !== undefined && object.syntax !== null) {
            message.syntax = object.syntax;
        }
        else {
            message.syntax = "";
        }
        return message;
    },
};
var baseDescriptorProto = { name: "", reservedName: "" };
export var DescriptorProto = {
    encode: function (message, writer) {
        var e_25, _a, e_26, _b, e_27, _c, e_28, _d, e_29, _e, e_30, _f, e_31, _g, e_32, _h;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        try {
            for (var _j = __values(message.field), _k = _j.next(); !_k.done; _k = _j.next()) {
                var v = _k.value;
                FieldDescriptorProto.encode(v, writer.uint32(18).fork()).ldelim();
            }
        }
        catch (e_25_1) { e_25 = { error: e_25_1 }; }
        finally {
            try {
                if (_k && !_k.done && (_a = _j.return)) _a.call(_j);
            }
            finally { if (e_25) throw e_25.error; }
        }
        try {
            for (var _l = __values(message.extension), _m = _l.next(); !_m.done; _m = _l.next()) {
                var v = _m.value;
                FieldDescriptorProto.encode(v, writer.uint32(50).fork()).ldelim();
            }
        }
        catch (e_26_1) { e_26 = { error: e_26_1 }; }
        finally {
            try {
                if (_m && !_m.done && (_b = _l.return)) _b.call(_l);
            }
            finally { if (e_26) throw e_26.error; }
        }
        try {
            for (var _o = __values(message.nestedType), _p = _o.next(); !_p.done; _p = _o.next()) {
                var v = _p.value;
                DescriptorProto.encode(v, writer.uint32(26).fork()).ldelim();
            }
        }
        catch (e_27_1) { e_27 = { error: e_27_1 }; }
        finally {
            try {
                if (_p && !_p.done && (_c = _o.return)) _c.call(_o);
            }
            finally { if (e_27) throw e_27.error; }
        }
        try {
            for (var _q = __values(message.enumType), _r = _q.next(); !_r.done; _r = _q.next()) {
                var v = _r.value;
                EnumDescriptorProto.encode(v, writer.uint32(34).fork()).ldelim();
            }
        }
        catch (e_28_1) { e_28 = { error: e_28_1 }; }
        finally {
            try {
                if (_r && !_r.done && (_d = _q.return)) _d.call(_q);
            }
            finally { if (e_28) throw e_28.error; }
        }
        try {
            for (var _s = __values(message.extensionRange), _t = _s.next(); !_t.done; _t = _s.next()) {
                var v = _t.value;
                DescriptorProto_ExtensionRange.encode(v, writer.uint32(42).fork()).ldelim();
            }
        }
        catch (e_29_1) { e_29 = { error: e_29_1 }; }
        finally {
            try {
                if (_t && !_t.done && (_e = _s.return)) _e.call(_s);
            }
            finally { if (e_29) throw e_29.error; }
        }
        try {
            for (var _u = __values(message.oneofDecl), _v = _u.next(); !_v.done; _v = _u.next()) {
                var v = _v.value;
                OneofDescriptorProto.encode(v, writer.uint32(66).fork()).ldelim();
            }
        }
        catch (e_30_1) { e_30 = { error: e_30_1 }; }
        finally {
            try {
                if (_v && !_v.done && (_f = _u.return)) _f.call(_u);
            }
            finally { if (e_30) throw e_30.error; }
        }
        if (message.options !== undefined) {
            MessageOptions.encode(message.options, writer.uint32(58).fork()).ldelim();
        }
        try {
            for (var _w = __values(message.reservedRange), _x = _w.next(); !_x.done; _x = _w.next()) {
                var v = _x.value;
                DescriptorProto_ReservedRange.encode(v, writer.uint32(74).fork()).ldelim();
            }
        }
        catch (e_31_1) { e_31 = { error: e_31_1 }; }
        finally {
            try {
                if (_x && !_x.done && (_g = _w.return)) _g.call(_w);
            }
            finally { if (e_31) throw e_31.error; }
        }
        try {
            for (var _y = __values(message.reservedName), _z = _y.next(); !_z.done; _z = _y.next()) {
                var v = _z.value;
                writer.uint32(82).string(v);
            }
        }
        catch (e_32_1) { e_32 = { error: e_32_1 }; }
        finally {
            try {
                if (_z && !_z.done && (_h = _y.return)) _h.call(_y);
            }
            finally { if (e_32) throw e_32.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseDescriptorProto);
        message.field = [];
        message.extension = [];
        message.nestedType = [];
        message.enumType = [];
        message.extensionRange = [];
        message.oneofDecl = [];
        message.reservedRange = [];
        message.reservedName = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.field.push(FieldDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 6:
                    message.extension.push(FieldDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.nestedType.push(DescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 4:
                    message.enumType.push(EnumDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 5:
                    message.extensionRange.push(DescriptorProto_ExtensionRange.decode(reader, reader.uint32()));
                    break;
                case 8:
                    message.oneofDecl.push(OneofDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 7:
                    message.options = MessageOptions.decode(reader, reader.uint32());
                    break;
                case 9:
                    message.reservedRange.push(DescriptorProto_ReservedRange.decode(reader, reader.uint32()));
                    break;
                case 10:
                    message.reservedName.push(reader.string());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_33, _a, e_34, _b, e_35, _c, e_36, _d, e_37, _e, e_38, _f, e_39, _g, e_40, _h;
        var message = __assign({}, baseDescriptorProto);
        message.field = [];
        message.extension = [];
        message.nestedType = [];
        message.enumType = [];
        message.extensionRange = [];
        message.oneofDecl = [];
        message.reservedRange = [];
        message.reservedName = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.field !== undefined && object.field !== null) {
            try {
                for (var _j = __values(object.field), _k = _j.next(); !_k.done; _k = _j.next()) {
                    var e = _k.value;
                    message.field.push(FieldDescriptorProto.fromJSON(e));
                }
            }
            catch (e_33_1) { e_33 = { error: e_33_1 }; }
            finally {
                try {
                    if (_k && !_k.done && (_a = _j.return)) _a.call(_j);
                }
                finally { if (e_33) throw e_33.error; }
            }
        }
        if (object.extension !== undefined && object.extension !== null) {
            try {
                for (var _l = __values(object.extension), _m = _l.next(); !_m.done; _m = _l.next()) {
                    var e = _m.value;
                    message.extension.push(FieldDescriptorProto.fromJSON(e));
                }
            }
            catch (e_34_1) { e_34 = { error: e_34_1 }; }
            finally {
                try {
                    if (_m && !_m.done && (_b = _l.return)) _b.call(_l);
                }
                finally { if (e_34) throw e_34.error; }
            }
        }
        if (object.nestedType !== undefined && object.nestedType !== null) {
            try {
                for (var _o = __values(object.nestedType), _p = _o.next(); !_p.done; _p = _o.next()) {
                    var e = _p.value;
                    message.nestedType.push(DescriptorProto.fromJSON(e));
                }
            }
            catch (e_35_1) { e_35 = { error: e_35_1 }; }
            finally {
                try {
                    if (_p && !_p.done && (_c = _o.return)) _c.call(_o);
                }
                finally { if (e_35) throw e_35.error; }
            }
        }
        if (object.enumType !== undefined && object.enumType !== null) {
            try {
                for (var _q = __values(object.enumType), _r = _q.next(); !_r.done; _r = _q.next()) {
                    var e = _r.value;
                    message.enumType.push(EnumDescriptorProto.fromJSON(e));
                }
            }
            catch (e_36_1) { e_36 = { error: e_36_1 }; }
            finally {
                try {
                    if (_r && !_r.done && (_d = _q.return)) _d.call(_q);
                }
                finally { if (e_36) throw e_36.error; }
            }
        }
        if (object.extensionRange !== undefined && object.extensionRange !== null) {
            try {
                for (var _s = __values(object.extensionRange), _t = _s.next(); !_t.done; _t = _s.next()) {
                    var e = _t.value;
                    message.extensionRange.push(DescriptorProto_ExtensionRange.fromJSON(e));
                }
            }
            catch (e_37_1) { e_37 = { error: e_37_1 }; }
            finally {
                try {
                    if (_t && !_t.done && (_e = _s.return)) _e.call(_s);
                }
                finally { if (e_37) throw e_37.error; }
            }
        }
        if (object.oneofDecl !== undefined && object.oneofDecl !== null) {
            try {
                for (var _u = __values(object.oneofDecl), _v = _u.next(); !_v.done; _v = _u.next()) {
                    var e = _v.value;
                    message.oneofDecl.push(OneofDescriptorProto.fromJSON(e));
                }
            }
            catch (e_38_1) { e_38 = { error: e_38_1 }; }
            finally {
                try {
                    if (_v && !_v.done && (_f = _u.return)) _f.call(_u);
                }
                finally { if (e_38) throw e_38.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = MessageOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.reservedRange !== undefined && object.reservedRange !== null) {
            try {
                for (var _w = __values(object.reservedRange), _x = _w.next(); !_x.done; _x = _w.next()) {
                    var e = _x.value;
                    message.reservedRange.push(DescriptorProto_ReservedRange.fromJSON(e));
                }
            }
            catch (e_39_1) { e_39 = { error: e_39_1 }; }
            finally {
                try {
                    if (_x && !_x.done && (_g = _w.return)) _g.call(_w);
                }
                finally { if (e_39) throw e_39.error; }
            }
        }
        if (object.reservedName !== undefined && object.reservedName !== null) {
            try {
                for (var _y = __values(object.reservedName), _z = _y.next(); !_z.done; _z = _y.next()) {
                    var e = _z.value;
                    message.reservedName.push(String(e));
                }
            }
            catch (e_40_1) { e_40 = { error: e_40_1 }; }
            finally {
                try {
                    if (_z && !_z.done && (_h = _y.return)) _h.call(_y);
                }
                finally { if (e_40) throw e_40.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        if (message.field) {
            obj.field = message.field.map(function (e) {
                return e ? FieldDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.field = [];
        }
        if (message.extension) {
            obj.extension = message.extension.map(function (e) {
                return e ? FieldDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.extension = [];
        }
        if (message.nestedType) {
            obj.nestedType = message.nestedType.map(function (e) {
                return e ? DescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.nestedType = [];
        }
        if (message.enumType) {
            obj.enumType = message.enumType.map(function (e) {
                return e ? EnumDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.enumType = [];
        }
        if (message.extensionRange) {
            obj.extensionRange = message.extensionRange.map(function (e) {
                return e ? DescriptorProto_ExtensionRange.toJSON(e) : undefined;
            });
        }
        else {
            obj.extensionRange = [];
        }
        if (message.oneofDecl) {
            obj.oneofDecl = message.oneofDecl.map(function (e) {
                return e ? OneofDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.oneofDecl = [];
        }
        message.options !== undefined &&
            (obj.options = message.options
                ? MessageOptions.toJSON(message.options)
                : undefined);
        if (message.reservedRange) {
            obj.reservedRange = message.reservedRange.map(function (e) {
                return e ? DescriptorProto_ReservedRange.toJSON(e) : undefined;
            });
        }
        else {
            obj.reservedRange = [];
        }
        if (message.reservedName) {
            obj.reservedName = message.reservedName.map(function (e) { return e; });
        }
        else {
            obj.reservedName = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_41, _a, e_42, _b, e_43, _c, e_44, _d, e_45, _e, e_46, _f, e_47, _g, e_48, _h;
        var message = __assign({}, baseDescriptorProto);
        message.field = [];
        message.extension = [];
        message.nestedType = [];
        message.enumType = [];
        message.extensionRange = [];
        message.oneofDecl = [];
        message.reservedRange = [];
        message.reservedName = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.field !== undefined && object.field !== null) {
            try {
                for (var _j = __values(object.field), _k = _j.next(); !_k.done; _k = _j.next()) {
                    var e = _k.value;
                    message.field.push(FieldDescriptorProto.fromPartial(e));
                }
            }
            catch (e_41_1) { e_41 = { error: e_41_1 }; }
            finally {
                try {
                    if (_k && !_k.done && (_a = _j.return)) _a.call(_j);
                }
                finally { if (e_41) throw e_41.error; }
            }
        }
        if (object.extension !== undefined && object.extension !== null) {
            try {
                for (var _l = __values(object.extension), _m = _l.next(); !_m.done; _m = _l.next()) {
                    var e = _m.value;
                    message.extension.push(FieldDescriptorProto.fromPartial(e));
                }
            }
            catch (e_42_1) { e_42 = { error: e_42_1 }; }
            finally {
                try {
                    if (_m && !_m.done && (_b = _l.return)) _b.call(_l);
                }
                finally { if (e_42) throw e_42.error; }
            }
        }
        if (object.nestedType !== undefined && object.nestedType !== null) {
            try {
                for (var _o = __values(object.nestedType), _p = _o.next(); !_p.done; _p = _o.next()) {
                    var e = _p.value;
                    message.nestedType.push(DescriptorProto.fromPartial(e));
                }
            }
            catch (e_43_1) { e_43 = { error: e_43_1 }; }
            finally {
                try {
                    if (_p && !_p.done && (_c = _o.return)) _c.call(_o);
                }
                finally { if (e_43) throw e_43.error; }
            }
        }
        if (object.enumType !== undefined && object.enumType !== null) {
            try {
                for (var _q = __values(object.enumType), _r = _q.next(); !_r.done; _r = _q.next()) {
                    var e = _r.value;
                    message.enumType.push(EnumDescriptorProto.fromPartial(e));
                }
            }
            catch (e_44_1) { e_44 = { error: e_44_1 }; }
            finally {
                try {
                    if (_r && !_r.done && (_d = _q.return)) _d.call(_q);
                }
                finally { if (e_44) throw e_44.error; }
            }
        }
        if (object.extensionRange !== undefined && object.extensionRange !== null) {
            try {
                for (var _s = __values(object.extensionRange), _t = _s.next(); !_t.done; _t = _s.next()) {
                    var e = _t.value;
                    message.extensionRange.push(DescriptorProto_ExtensionRange.fromPartial(e));
                }
            }
            catch (e_45_1) { e_45 = { error: e_45_1 }; }
            finally {
                try {
                    if (_t && !_t.done && (_e = _s.return)) _e.call(_s);
                }
                finally { if (e_45) throw e_45.error; }
            }
        }
        if (object.oneofDecl !== undefined && object.oneofDecl !== null) {
            try {
                for (var _u = __values(object.oneofDecl), _v = _u.next(); !_v.done; _v = _u.next()) {
                    var e = _v.value;
                    message.oneofDecl.push(OneofDescriptorProto.fromPartial(e));
                }
            }
            catch (e_46_1) { e_46 = { error: e_46_1 }; }
            finally {
                try {
                    if (_v && !_v.done && (_f = _u.return)) _f.call(_u);
                }
                finally { if (e_46) throw e_46.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = MessageOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.reservedRange !== undefined && object.reservedRange !== null) {
            try {
                for (var _w = __values(object.reservedRange), _x = _w.next(); !_x.done; _x = _w.next()) {
                    var e = _x.value;
                    message.reservedRange.push(DescriptorProto_ReservedRange.fromPartial(e));
                }
            }
            catch (e_47_1) { e_47 = { error: e_47_1 }; }
            finally {
                try {
                    if (_x && !_x.done && (_g = _w.return)) _g.call(_w);
                }
                finally { if (e_47) throw e_47.error; }
            }
        }
        if (object.reservedName !== undefined && object.reservedName !== null) {
            try {
                for (var _y = __values(object.reservedName), _z = _y.next(); !_z.done; _z = _y.next()) {
                    var e = _z.value;
                    message.reservedName.push(e);
                }
            }
            catch (e_48_1) { e_48 = { error: e_48_1 }; }
            finally {
                try {
                    if (_z && !_z.done && (_h = _y.return)) _h.call(_y);
                }
                finally { if (e_48) throw e_48.error; }
            }
        }
        return message;
    },
};
var baseDescriptorProto_ExtensionRange = { start: 0, end: 0 };
export var DescriptorProto_ExtensionRange = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.start !== 0) {
            writer.uint32(8).int32(message.start);
        }
        if (message.end !== 0) {
            writer.uint32(16).int32(message.end);
        }
        if (message.options !== undefined) {
            ExtensionRangeOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseDescriptorProto_ExtensionRange);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.start = reader.int32();
                    break;
                case 2:
                    message.end = reader.int32();
                    break;
                case 3:
                    message.options = ExtensionRangeOptions.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseDescriptorProto_ExtensionRange);
        if (object.start !== undefined && object.start !== null) {
            message.start = Number(object.start);
        }
        else {
            message.start = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = Number(object.end);
        }
        else {
            message.end = 0;
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = ExtensionRangeOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.start !== undefined && (obj.start = message.start);
        message.end !== undefined && (obj.end = message.end);
        message.options !== undefined &&
            (obj.options = message.options
                ? ExtensionRangeOptions.toJSON(message.options)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseDescriptorProto_ExtensionRange);
        if (object.start !== undefined && object.start !== null) {
            message.start = object.start;
        }
        else {
            message.start = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = object.end;
        }
        else {
            message.end = 0;
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = ExtensionRangeOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
};
var baseDescriptorProto_ReservedRange = { start: 0, end: 0 };
export var DescriptorProto_ReservedRange = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.start !== 0) {
            writer.uint32(8).int32(message.start);
        }
        if (message.end !== 0) {
            writer.uint32(16).int32(message.end);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseDescriptorProto_ReservedRange);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.start = reader.int32();
                    break;
                case 2:
                    message.end = reader.int32();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseDescriptorProto_ReservedRange);
        if (object.start !== undefined && object.start !== null) {
            message.start = Number(object.start);
        }
        else {
            message.start = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = Number(object.end);
        }
        else {
            message.end = 0;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.start !== undefined && (obj.start = message.start);
        message.end !== undefined && (obj.end = message.end);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseDescriptorProto_ReservedRange);
        if (object.start !== undefined && object.start !== null) {
            message.start = object.start;
        }
        else {
            message.start = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = object.end;
        }
        else {
            message.end = 0;
        }
        return message;
    },
};
var baseExtensionRangeOptions = {};
export var ExtensionRangeOptions = {
    encode: function (message, writer) {
        var e_49, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_49_1) { e_49 = { error: e_49_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_49) throw e_49.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseExtensionRangeOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_50, _a;
        var message = __assign({}, baseExtensionRangeOptions);
        message.uninterpretedOption = [];
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_50_1) { e_50 = { error: e_50_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_50) throw e_50.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_51, _a;
        var message = __assign({}, baseExtensionRangeOptions);
        message.uninterpretedOption = [];
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_51_1) { e_51 = { error: e_51_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_51) throw e_51.error; }
            }
        }
        return message;
    },
};
var baseFieldDescriptorProto = {
    name: "",
    number: 0,
    label: 1,
    type: 1,
    typeName: "",
    extendee: "",
    defaultValue: "",
    oneofIndex: 0,
    jsonName: "",
    proto3Optional: false,
};
export var FieldDescriptorProto = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.number !== 0) {
            writer.uint32(24).int32(message.number);
        }
        if (message.label !== 1) {
            writer.uint32(32).int32(message.label);
        }
        if (message.type !== 1) {
            writer.uint32(40).int32(message.type);
        }
        if (message.typeName !== "") {
            writer.uint32(50).string(message.typeName);
        }
        if (message.extendee !== "") {
            writer.uint32(18).string(message.extendee);
        }
        if (message.defaultValue !== "") {
            writer.uint32(58).string(message.defaultValue);
        }
        if (message.oneofIndex !== 0) {
            writer.uint32(72).int32(message.oneofIndex);
        }
        if (message.jsonName !== "") {
            writer.uint32(82).string(message.jsonName);
        }
        if (message.options !== undefined) {
            FieldOptions.encode(message.options, writer.uint32(66).fork()).ldelim();
        }
        if (message.proto3Optional === true) {
            writer.uint32(136).bool(message.proto3Optional);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseFieldDescriptorProto);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 3:
                    message.number = reader.int32();
                    break;
                case 4:
                    message.label = reader.int32();
                    break;
                case 5:
                    message.type = reader.int32();
                    break;
                case 6:
                    message.typeName = reader.string();
                    break;
                case 2:
                    message.extendee = reader.string();
                    break;
                case 7:
                    message.defaultValue = reader.string();
                    break;
                case 9:
                    message.oneofIndex = reader.int32();
                    break;
                case 10:
                    message.jsonName = reader.string();
                    break;
                case 8:
                    message.options = FieldOptions.decode(reader, reader.uint32());
                    break;
                case 17:
                    message.proto3Optional = reader.bool();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseFieldDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.number !== undefined && object.number !== null) {
            message.number = Number(object.number);
        }
        else {
            message.number = 0;
        }
        if (object.label !== undefined && object.label !== null) {
            message.label = fieldDescriptorProto_LabelFromJSON(object.label);
        }
        else {
            message.label = 1;
        }
        if (object.type !== undefined && object.type !== null) {
            message.type = fieldDescriptorProto_TypeFromJSON(object.type);
        }
        else {
            message.type = 1;
        }
        if (object.typeName !== undefined && object.typeName !== null) {
            message.typeName = String(object.typeName);
        }
        else {
            message.typeName = "";
        }
        if (object.extendee !== undefined && object.extendee !== null) {
            message.extendee = String(object.extendee);
        }
        else {
            message.extendee = "";
        }
        if (object.defaultValue !== undefined && object.defaultValue !== null) {
            message.defaultValue = String(object.defaultValue);
        }
        else {
            message.defaultValue = "";
        }
        if (object.oneofIndex !== undefined && object.oneofIndex !== null) {
            message.oneofIndex = Number(object.oneofIndex);
        }
        else {
            message.oneofIndex = 0;
        }
        if (object.jsonName !== undefined && object.jsonName !== null) {
            message.jsonName = String(object.jsonName);
        }
        else {
            message.jsonName = "";
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = FieldOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.proto3Optional !== undefined && object.proto3Optional !== null) {
            message.proto3Optional = Boolean(object.proto3Optional);
        }
        else {
            message.proto3Optional = false;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.number !== undefined && (obj.number = message.number);
        message.label !== undefined &&
            (obj.label = fieldDescriptorProto_LabelToJSON(message.label));
        message.type !== undefined &&
            (obj.type = fieldDescriptorProto_TypeToJSON(message.type));
        message.typeName !== undefined && (obj.typeName = message.typeName);
        message.extendee !== undefined && (obj.extendee = message.extendee);
        message.defaultValue !== undefined &&
            (obj.defaultValue = message.defaultValue);
        message.oneofIndex !== undefined && (obj.oneofIndex = message.oneofIndex);
        message.jsonName !== undefined && (obj.jsonName = message.jsonName);
        message.options !== undefined &&
            (obj.options = message.options
                ? FieldOptions.toJSON(message.options)
                : undefined);
        message.proto3Optional !== undefined &&
            (obj.proto3Optional = message.proto3Optional);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseFieldDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.number !== undefined && object.number !== null) {
            message.number = object.number;
        }
        else {
            message.number = 0;
        }
        if (object.label !== undefined && object.label !== null) {
            message.label = object.label;
        }
        else {
            message.label = 1;
        }
        if (object.type !== undefined && object.type !== null) {
            message.type = object.type;
        }
        else {
            message.type = 1;
        }
        if (object.typeName !== undefined && object.typeName !== null) {
            message.typeName = object.typeName;
        }
        else {
            message.typeName = "";
        }
        if (object.extendee !== undefined && object.extendee !== null) {
            message.extendee = object.extendee;
        }
        else {
            message.extendee = "";
        }
        if (object.defaultValue !== undefined && object.defaultValue !== null) {
            message.defaultValue = object.defaultValue;
        }
        else {
            message.defaultValue = "";
        }
        if (object.oneofIndex !== undefined && object.oneofIndex !== null) {
            message.oneofIndex = object.oneofIndex;
        }
        else {
            message.oneofIndex = 0;
        }
        if (object.jsonName !== undefined && object.jsonName !== null) {
            message.jsonName = object.jsonName;
        }
        else {
            message.jsonName = "";
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = FieldOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.proto3Optional !== undefined && object.proto3Optional !== null) {
            message.proto3Optional = object.proto3Optional;
        }
        else {
            message.proto3Optional = false;
        }
        return message;
    },
};
var baseOneofDescriptorProto = { name: "" };
export var OneofDescriptorProto = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.options !== undefined) {
            OneofOptions.encode(message.options, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseOneofDescriptorProto);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.options = OneofOptions.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseOneofDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = OneofOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.options !== undefined &&
            (obj.options = message.options
                ? OneofOptions.toJSON(message.options)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseOneofDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = OneofOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
};
var baseEnumDescriptorProto = { name: "", reservedName: "" };
export var EnumDescriptorProto = {
    encode: function (message, writer) {
        var e_52, _a, e_53, _b, e_54, _c;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        try {
            for (var _d = __values(message.value), _e = _d.next(); !_e.done; _e = _d.next()) {
                var v = _e.value;
                EnumValueDescriptorProto.encode(v, writer.uint32(18).fork()).ldelim();
            }
        }
        catch (e_52_1) { e_52 = { error: e_52_1 }; }
        finally {
            try {
                if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
            }
            finally { if (e_52) throw e_52.error; }
        }
        if (message.options !== undefined) {
            EnumOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
        }
        try {
            for (var _f = __values(message.reservedRange), _g = _f.next(); !_g.done; _g = _f.next()) {
                var v = _g.value;
                EnumDescriptorProto_EnumReservedRange.encode(v, writer.uint32(34).fork()).ldelim();
            }
        }
        catch (e_53_1) { e_53 = { error: e_53_1 }; }
        finally {
            try {
                if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
            }
            finally { if (e_53) throw e_53.error; }
        }
        try {
            for (var _h = __values(message.reservedName), _j = _h.next(); !_j.done; _j = _h.next()) {
                var v = _j.value;
                writer.uint32(42).string(v);
            }
        }
        catch (e_54_1) { e_54 = { error: e_54_1 }; }
        finally {
            try {
                if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
            }
            finally { if (e_54) throw e_54.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEnumDescriptorProto);
        message.value = [];
        message.reservedRange = [];
        message.reservedName = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.value.push(EnumValueDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.options = EnumOptions.decode(reader, reader.uint32());
                    break;
                case 4:
                    message.reservedRange.push(EnumDescriptorProto_EnumReservedRange.decode(reader, reader.uint32()));
                    break;
                case 5:
                    message.reservedName.push(reader.string());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_55, _a, e_56, _b, e_57, _c;
        var message = __assign({}, baseEnumDescriptorProto);
        message.value = [];
        message.reservedRange = [];
        message.reservedName = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.value !== undefined && object.value !== null) {
            try {
                for (var _d = __values(object.value), _e = _d.next(); !_e.done; _e = _d.next()) {
                    var e = _e.value;
                    message.value.push(EnumValueDescriptorProto.fromJSON(e));
                }
            }
            catch (e_55_1) { e_55 = { error: e_55_1 }; }
            finally {
                try {
                    if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
                }
                finally { if (e_55) throw e_55.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = EnumOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.reservedRange !== undefined && object.reservedRange !== null) {
            try {
                for (var _f = __values(object.reservedRange), _g = _f.next(); !_g.done; _g = _f.next()) {
                    var e = _g.value;
                    message.reservedRange.push(EnumDescriptorProto_EnumReservedRange.fromJSON(e));
                }
            }
            catch (e_56_1) { e_56 = { error: e_56_1 }; }
            finally {
                try {
                    if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
                }
                finally { if (e_56) throw e_56.error; }
            }
        }
        if (object.reservedName !== undefined && object.reservedName !== null) {
            try {
                for (var _h = __values(object.reservedName), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.reservedName.push(String(e));
                }
            }
            catch (e_57_1) { e_57 = { error: e_57_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
                }
                finally { if (e_57) throw e_57.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        if (message.value) {
            obj.value = message.value.map(function (e) {
                return e ? EnumValueDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.value = [];
        }
        message.options !== undefined &&
            (obj.options = message.options
                ? EnumOptions.toJSON(message.options)
                : undefined);
        if (message.reservedRange) {
            obj.reservedRange = message.reservedRange.map(function (e) {
                return e ? EnumDescriptorProto_EnumReservedRange.toJSON(e) : undefined;
            });
        }
        else {
            obj.reservedRange = [];
        }
        if (message.reservedName) {
            obj.reservedName = message.reservedName.map(function (e) { return e; });
        }
        else {
            obj.reservedName = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_58, _a, e_59, _b, e_60, _c;
        var message = __assign({}, baseEnumDescriptorProto);
        message.value = [];
        message.reservedRange = [];
        message.reservedName = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.value !== undefined && object.value !== null) {
            try {
                for (var _d = __values(object.value), _e = _d.next(); !_e.done; _e = _d.next()) {
                    var e = _e.value;
                    message.value.push(EnumValueDescriptorProto.fromPartial(e));
                }
            }
            catch (e_58_1) { e_58 = { error: e_58_1 }; }
            finally {
                try {
                    if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
                }
                finally { if (e_58) throw e_58.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = EnumOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.reservedRange !== undefined && object.reservedRange !== null) {
            try {
                for (var _f = __values(object.reservedRange), _g = _f.next(); !_g.done; _g = _f.next()) {
                    var e = _g.value;
                    message.reservedRange.push(EnumDescriptorProto_EnumReservedRange.fromPartial(e));
                }
            }
            catch (e_59_1) { e_59 = { error: e_59_1 }; }
            finally {
                try {
                    if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
                }
                finally { if (e_59) throw e_59.error; }
            }
        }
        if (object.reservedName !== undefined && object.reservedName !== null) {
            try {
                for (var _h = __values(object.reservedName), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.reservedName.push(e);
                }
            }
            catch (e_60_1) { e_60 = { error: e_60_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
                }
                finally { if (e_60) throw e_60.error; }
            }
        }
        return message;
    },
};
var baseEnumDescriptorProto_EnumReservedRange = { start: 0, end: 0 };
export var EnumDescriptorProto_EnumReservedRange = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.start !== 0) {
            writer.uint32(8).int32(message.start);
        }
        if (message.end !== 0) {
            writer.uint32(16).int32(message.end);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEnumDescriptorProto_EnumReservedRange);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.start = reader.int32();
                    break;
                case 2:
                    message.end = reader.int32();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseEnumDescriptorProto_EnumReservedRange);
        if (object.start !== undefined && object.start !== null) {
            message.start = Number(object.start);
        }
        else {
            message.start = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = Number(object.end);
        }
        else {
            message.end = 0;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.start !== undefined && (obj.start = message.start);
        message.end !== undefined && (obj.end = message.end);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEnumDescriptorProto_EnumReservedRange);
        if (object.start !== undefined && object.start !== null) {
            message.start = object.start;
        }
        else {
            message.start = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = object.end;
        }
        else {
            message.end = 0;
        }
        return message;
    },
};
var baseEnumValueDescriptorProto = { name: "", number: 0 };
export var EnumValueDescriptorProto = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.number !== 0) {
            writer.uint32(16).int32(message.number);
        }
        if (message.options !== undefined) {
            EnumValueOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEnumValueDescriptorProto);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.number = reader.int32();
                    break;
                case 3:
                    message.options = EnumValueOptions.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseEnumValueDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.number !== undefined && object.number !== null) {
            message.number = Number(object.number);
        }
        else {
            message.number = 0;
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = EnumValueOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.number !== undefined && (obj.number = message.number);
        message.options !== undefined &&
            (obj.options = message.options
                ? EnumValueOptions.toJSON(message.options)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseEnumValueDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.number !== undefined && object.number !== null) {
            message.number = object.number;
        }
        else {
            message.number = 0;
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = EnumValueOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
};
var baseServiceDescriptorProto = { name: "" };
export var ServiceDescriptorProto = {
    encode: function (message, writer) {
        var e_61, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        try {
            for (var _b = __values(message.method), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                MethodDescriptorProto.encode(v, writer.uint32(18).fork()).ldelim();
            }
        }
        catch (e_61_1) { e_61 = { error: e_61_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_61) throw e_61.error; }
        }
        if (message.options !== undefined) {
            ServiceOptions.encode(message.options, writer.uint32(26).fork()).ldelim();
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseServiceDescriptorProto);
        message.method = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.method.push(MethodDescriptorProto.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.options = ServiceOptions.decode(reader, reader.uint32());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_62, _a;
        var message = __assign({}, baseServiceDescriptorProto);
        message.method = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.method !== undefined && object.method !== null) {
            try {
                for (var _b = __values(object.method), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.method.push(MethodDescriptorProto.fromJSON(e));
                }
            }
            catch (e_62_1) { e_62 = { error: e_62_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_62) throw e_62.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = ServiceOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        if (message.method) {
            obj.method = message.method.map(function (e) {
                return e ? MethodDescriptorProto.toJSON(e) : undefined;
            });
        }
        else {
            obj.method = [];
        }
        message.options !== undefined &&
            (obj.options = message.options
                ? ServiceOptions.toJSON(message.options)
                : undefined);
        return obj;
    },
    fromPartial: function (object) {
        var e_63, _a;
        var message = __assign({}, baseServiceDescriptorProto);
        message.method = [];
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.method !== undefined && object.method !== null) {
            try {
                for (var _b = __values(object.method), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.method.push(MethodDescriptorProto.fromPartial(e));
                }
            }
            catch (e_63_1) { e_63 = { error: e_63_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_63) throw e_63.error; }
            }
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = ServiceOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        return message;
    },
};
var baseMethodDescriptorProto = {
    name: "",
    inputType: "",
    outputType: "",
    clientStreaming: false,
    serverStreaming: false,
};
export var MethodDescriptorProto = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.name !== "") {
            writer.uint32(10).string(message.name);
        }
        if (message.inputType !== "") {
            writer.uint32(18).string(message.inputType);
        }
        if (message.outputType !== "") {
            writer.uint32(26).string(message.outputType);
        }
        if (message.options !== undefined) {
            MethodOptions.encode(message.options, writer.uint32(34).fork()).ldelim();
        }
        if (message.clientStreaming === true) {
            writer.uint32(40).bool(message.clientStreaming);
        }
        if (message.serverStreaming === true) {
            writer.uint32(48).bool(message.serverStreaming);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseMethodDescriptorProto);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.name = reader.string();
                    break;
                case 2:
                    message.inputType = reader.string();
                    break;
                case 3:
                    message.outputType = reader.string();
                    break;
                case 4:
                    message.options = MethodOptions.decode(reader, reader.uint32());
                    break;
                case 5:
                    message.clientStreaming = reader.bool();
                    break;
                case 6:
                    message.serverStreaming = reader.bool();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseMethodDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = String(object.name);
        }
        else {
            message.name = "";
        }
        if (object.inputType !== undefined && object.inputType !== null) {
            message.inputType = String(object.inputType);
        }
        else {
            message.inputType = "";
        }
        if (object.outputType !== undefined && object.outputType !== null) {
            message.outputType = String(object.outputType);
        }
        else {
            message.outputType = "";
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = MethodOptions.fromJSON(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.clientStreaming !== undefined &&
            object.clientStreaming !== null) {
            message.clientStreaming = Boolean(object.clientStreaming);
        }
        else {
            message.clientStreaming = false;
        }
        if (object.serverStreaming !== undefined &&
            object.serverStreaming !== null) {
            message.serverStreaming = Boolean(object.serverStreaming);
        }
        else {
            message.serverStreaming = false;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.name !== undefined && (obj.name = message.name);
        message.inputType !== undefined && (obj.inputType = message.inputType);
        message.outputType !== undefined && (obj.outputType = message.outputType);
        message.options !== undefined &&
            (obj.options = message.options
                ? MethodOptions.toJSON(message.options)
                : undefined);
        message.clientStreaming !== undefined &&
            (obj.clientStreaming = message.clientStreaming);
        message.serverStreaming !== undefined &&
            (obj.serverStreaming = message.serverStreaming);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseMethodDescriptorProto);
        if (object.name !== undefined && object.name !== null) {
            message.name = object.name;
        }
        else {
            message.name = "";
        }
        if (object.inputType !== undefined && object.inputType !== null) {
            message.inputType = object.inputType;
        }
        else {
            message.inputType = "";
        }
        if (object.outputType !== undefined && object.outputType !== null) {
            message.outputType = object.outputType;
        }
        else {
            message.outputType = "";
        }
        if (object.options !== undefined && object.options !== null) {
            message.options = MethodOptions.fromPartial(object.options);
        }
        else {
            message.options = undefined;
        }
        if (object.clientStreaming !== undefined &&
            object.clientStreaming !== null) {
            message.clientStreaming = object.clientStreaming;
        }
        else {
            message.clientStreaming = false;
        }
        if (object.serverStreaming !== undefined &&
            object.serverStreaming !== null) {
            message.serverStreaming = object.serverStreaming;
        }
        else {
            message.serverStreaming = false;
        }
        return message;
    },
};
var baseFileOptions = {
    javaPackage: "",
    javaOuterClassname: "",
    javaMultipleFiles: false,
    javaGenerateEqualsAndHash: false,
    javaStringCheckUtf8: false,
    optimizeFor: 1,
    goPackage: "",
    ccGenericServices: false,
    javaGenericServices: false,
    pyGenericServices: false,
    phpGenericServices: false,
    deprecated: false,
    ccEnableArenas: false,
    objcClassPrefix: "",
    csharpNamespace: "",
    swiftPrefix: "",
    phpClassPrefix: "",
    phpNamespace: "",
    phpMetadataNamespace: "",
    rubyPackage: "",
};
export var FileOptions = {
    encode: function (message, writer) {
        var e_64, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.javaPackage !== "") {
            writer.uint32(10).string(message.javaPackage);
        }
        if (message.javaOuterClassname !== "") {
            writer.uint32(66).string(message.javaOuterClassname);
        }
        if (message.javaMultipleFiles === true) {
            writer.uint32(80).bool(message.javaMultipleFiles);
        }
        if (message.javaGenerateEqualsAndHash === true) {
            writer.uint32(160).bool(message.javaGenerateEqualsAndHash);
        }
        if (message.javaStringCheckUtf8 === true) {
            writer.uint32(216).bool(message.javaStringCheckUtf8);
        }
        if (message.optimizeFor !== 1) {
            writer.uint32(72).int32(message.optimizeFor);
        }
        if (message.goPackage !== "") {
            writer.uint32(90).string(message.goPackage);
        }
        if (message.ccGenericServices === true) {
            writer.uint32(128).bool(message.ccGenericServices);
        }
        if (message.javaGenericServices === true) {
            writer.uint32(136).bool(message.javaGenericServices);
        }
        if (message.pyGenericServices === true) {
            writer.uint32(144).bool(message.pyGenericServices);
        }
        if (message.phpGenericServices === true) {
            writer.uint32(336).bool(message.phpGenericServices);
        }
        if (message.deprecated === true) {
            writer.uint32(184).bool(message.deprecated);
        }
        if (message.ccEnableArenas === true) {
            writer.uint32(248).bool(message.ccEnableArenas);
        }
        if (message.objcClassPrefix !== "") {
            writer.uint32(290).string(message.objcClassPrefix);
        }
        if (message.csharpNamespace !== "") {
            writer.uint32(298).string(message.csharpNamespace);
        }
        if (message.swiftPrefix !== "") {
            writer.uint32(314).string(message.swiftPrefix);
        }
        if (message.phpClassPrefix !== "") {
            writer.uint32(322).string(message.phpClassPrefix);
        }
        if (message.phpNamespace !== "") {
            writer.uint32(330).string(message.phpNamespace);
        }
        if (message.phpMetadataNamespace !== "") {
            writer.uint32(354).string(message.phpMetadataNamespace);
        }
        if (message.rubyPackage !== "") {
            writer.uint32(362).string(message.rubyPackage);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_64_1) { e_64 = { error: e_64_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_64) throw e_64.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseFileOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.javaPackage = reader.string();
                    break;
                case 8:
                    message.javaOuterClassname = reader.string();
                    break;
                case 10:
                    message.javaMultipleFiles = reader.bool();
                    break;
                case 20:
                    message.javaGenerateEqualsAndHash = reader.bool();
                    break;
                case 27:
                    message.javaStringCheckUtf8 = reader.bool();
                    break;
                case 9:
                    message.optimizeFor = reader.int32();
                    break;
                case 11:
                    message.goPackage = reader.string();
                    break;
                case 16:
                    message.ccGenericServices = reader.bool();
                    break;
                case 17:
                    message.javaGenericServices = reader.bool();
                    break;
                case 18:
                    message.pyGenericServices = reader.bool();
                    break;
                case 42:
                    message.phpGenericServices = reader.bool();
                    break;
                case 23:
                    message.deprecated = reader.bool();
                    break;
                case 31:
                    message.ccEnableArenas = reader.bool();
                    break;
                case 36:
                    message.objcClassPrefix = reader.string();
                    break;
                case 37:
                    message.csharpNamespace = reader.string();
                    break;
                case 39:
                    message.swiftPrefix = reader.string();
                    break;
                case 40:
                    message.phpClassPrefix = reader.string();
                    break;
                case 41:
                    message.phpNamespace = reader.string();
                    break;
                case 44:
                    message.phpMetadataNamespace = reader.string();
                    break;
                case 45:
                    message.rubyPackage = reader.string();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_65, _a;
        var message = __assign({}, baseFileOptions);
        message.uninterpretedOption = [];
        if (object.javaPackage !== undefined && object.javaPackage !== null) {
            message.javaPackage = String(object.javaPackage);
        }
        else {
            message.javaPackage = "";
        }
        if (object.javaOuterClassname !== undefined &&
            object.javaOuterClassname !== null) {
            message.javaOuterClassname = String(object.javaOuterClassname);
        }
        else {
            message.javaOuterClassname = "";
        }
        if (object.javaMultipleFiles !== undefined &&
            object.javaMultipleFiles !== null) {
            message.javaMultipleFiles = Boolean(object.javaMultipleFiles);
        }
        else {
            message.javaMultipleFiles = false;
        }
        if (object.javaGenerateEqualsAndHash !== undefined &&
            object.javaGenerateEqualsAndHash !== null) {
            message.javaGenerateEqualsAndHash = Boolean(object.javaGenerateEqualsAndHash);
        }
        else {
            message.javaGenerateEqualsAndHash = false;
        }
        if (object.javaStringCheckUtf8 !== undefined &&
            object.javaStringCheckUtf8 !== null) {
            message.javaStringCheckUtf8 = Boolean(object.javaStringCheckUtf8);
        }
        else {
            message.javaStringCheckUtf8 = false;
        }
        if (object.optimizeFor !== undefined && object.optimizeFor !== null) {
            message.optimizeFor = fileOptions_OptimizeModeFromJSON(object.optimizeFor);
        }
        else {
            message.optimizeFor = 1;
        }
        if (object.goPackage !== undefined && object.goPackage !== null) {
            message.goPackage = String(object.goPackage);
        }
        else {
            message.goPackage = "";
        }
        if (object.ccGenericServices !== undefined &&
            object.ccGenericServices !== null) {
            message.ccGenericServices = Boolean(object.ccGenericServices);
        }
        else {
            message.ccGenericServices = false;
        }
        if (object.javaGenericServices !== undefined &&
            object.javaGenericServices !== null) {
            message.javaGenericServices = Boolean(object.javaGenericServices);
        }
        else {
            message.javaGenericServices = false;
        }
        if (object.pyGenericServices !== undefined &&
            object.pyGenericServices !== null) {
            message.pyGenericServices = Boolean(object.pyGenericServices);
        }
        else {
            message.pyGenericServices = false;
        }
        if (object.phpGenericServices !== undefined &&
            object.phpGenericServices !== null) {
            message.phpGenericServices = Boolean(object.phpGenericServices);
        }
        else {
            message.phpGenericServices = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.ccEnableArenas !== undefined && object.ccEnableArenas !== null) {
            message.ccEnableArenas = Boolean(object.ccEnableArenas);
        }
        else {
            message.ccEnableArenas = false;
        }
        if (object.objcClassPrefix !== undefined &&
            object.objcClassPrefix !== null) {
            message.objcClassPrefix = String(object.objcClassPrefix);
        }
        else {
            message.objcClassPrefix = "";
        }
        if (object.csharpNamespace !== undefined &&
            object.csharpNamespace !== null) {
            message.csharpNamespace = String(object.csharpNamespace);
        }
        else {
            message.csharpNamespace = "";
        }
        if (object.swiftPrefix !== undefined && object.swiftPrefix !== null) {
            message.swiftPrefix = String(object.swiftPrefix);
        }
        else {
            message.swiftPrefix = "";
        }
        if (object.phpClassPrefix !== undefined && object.phpClassPrefix !== null) {
            message.phpClassPrefix = String(object.phpClassPrefix);
        }
        else {
            message.phpClassPrefix = "";
        }
        if (object.phpNamespace !== undefined && object.phpNamespace !== null) {
            message.phpNamespace = String(object.phpNamespace);
        }
        else {
            message.phpNamespace = "";
        }
        if (object.phpMetadataNamespace !== undefined &&
            object.phpMetadataNamespace !== null) {
            message.phpMetadataNamespace = String(object.phpMetadataNamespace);
        }
        else {
            message.phpMetadataNamespace = "";
        }
        if (object.rubyPackage !== undefined && object.rubyPackage !== null) {
            message.rubyPackage = String(object.rubyPackage);
        }
        else {
            message.rubyPackage = "";
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_65_1) { e_65 = { error: e_65_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_65) throw e_65.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.javaPackage !== undefined &&
            (obj.javaPackage = message.javaPackage);
        message.javaOuterClassname !== undefined &&
            (obj.javaOuterClassname = message.javaOuterClassname);
        message.javaMultipleFiles !== undefined &&
            (obj.javaMultipleFiles = message.javaMultipleFiles);
        message.javaGenerateEqualsAndHash !== undefined &&
            (obj.javaGenerateEqualsAndHash = message.javaGenerateEqualsAndHash);
        message.javaStringCheckUtf8 !== undefined &&
            (obj.javaStringCheckUtf8 = message.javaStringCheckUtf8);
        message.optimizeFor !== undefined &&
            (obj.optimizeFor = fileOptions_OptimizeModeToJSON(message.optimizeFor));
        message.goPackage !== undefined && (obj.goPackage = message.goPackage);
        message.ccGenericServices !== undefined &&
            (obj.ccGenericServices = message.ccGenericServices);
        message.javaGenericServices !== undefined &&
            (obj.javaGenericServices = message.javaGenericServices);
        message.pyGenericServices !== undefined &&
            (obj.pyGenericServices = message.pyGenericServices);
        message.phpGenericServices !== undefined &&
            (obj.phpGenericServices = message.phpGenericServices);
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        message.ccEnableArenas !== undefined &&
            (obj.ccEnableArenas = message.ccEnableArenas);
        message.objcClassPrefix !== undefined &&
            (obj.objcClassPrefix = message.objcClassPrefix);
        message.csharpNamespace !== undefined &&
            (obj.csharpNamespace = message.csharpNamespace);
        message.swiftPrefix !== undefined &&
            (obj.swiftPrefix = message.swiftPrefix);
        message.phpClassPrefix !== undefined &&
            (obj.phpClassPrefix = message.phpClassPrefix);
        message.phpNamespace !== undefined &&
            (obj.phpNamespace = message.phpNamespace);
        message.phpMetadataNamespace !== undefined &&
            (obj.phpMetadataNamespace = message.phpMetadataNamespace);
        message.rubyPackage !== undefined &&
            (obj.rubyPackage = message.rubyPackage);
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_66, _a;
        var message = __assign({}, baseFileOptions);
        message.uninterpretedOption = [];
        if (object.javaPackage !== undefined && object.javaPackage !== null) {
            message.javaPackage = object.javaPackage;
        }
        else {
            message.javaPackage = "";
        }
        if (object.javaOuterClassname !== undefined &&
            object.javaOuterClassname !== null) {
            message.javaOuterClassname = object.javaOuterClassname;
        }
        else {
            message.javaOuterClassname = "";
        }
        if (object.javaMultipleFiles !== undefined &&
            object.javaMultipleFiles !== null) {
            message.javaMultipleFiles = object.javaMultipleFiles;
        }
        else {
            message.javaMultipleFiles = false;
        }
        if (object.javaGenerateEqualsAndHash !== undefined &&
            object.javaGenerateEqualsAndHash !== null) {
            message.javaGenerateEqualsAndHash = object.javaGenerateEqualsAndHash;
        }
        else {
            message.javaGenerateEqualsAndHash = false;
        }
        if (object.javaStringCheckUtf8 !== undefined &&
            object.javaStringCheckUtf8 !== null) {
            message.javaStringCheckUtf8 = object.javaStringCheckUtf8;
        }
        else {
            message.javaStringCheckUtf8 = false;
        }
        if (object.optimizeFor !== undefined && object.optimizeFor !== null) {
            message.optimizeFor = object.optimizeFor;
        }
        else {
            message.optimizeFor = 1;
        }
        if (object.goPackage !== undefined && object.goPackage !== null) {
            message.goPackage = object.goPackage;
        }
        else {
            message.goPackage = "";
        }
        if (object.ccGenericServices !== undefined &&
            object.ccGenericServices !== null) {
            message.ccGenericServices = object.ccGenericServices;
        }
        else {
            message.ccGenericServices = false;
        }
        if (object.javaGenericServices !== undefined &&
            object.javaGenericServices !== null) {
            message.javaGenericServices = object.javaGenericServices;
        }
        else {
            message.javaGenericServices = false;
        }
        if (object.pyGenericServices !== undefined &&
            object.pyGenericServices !== null) {
            message.pyGenericServices = object.pyGenericServices;
        }
        else {
            message.pyGenericServices = false;
        }
        if (object.phpGenericServices !== undefined &&
            object.phpGenericServices !== null) {
            message.phpGenericServices = object.phpGenericServices;
        }
        else {
            message.phpGenericServices = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.ccEnableArenas !== undefined && object.ccEnableArenas !== null) {
            message.ccEnableArenas = object.ccEnableArenas;
        }
        else {
            message.ccEnableArenas = false;
        }
        if (object.objcClassPrefix !== undefined &&
            object.objcClassPrefix !== null) {
            message.objcClassPrefix = object.objcClassPrefix;
        }
        else {
            message.objcClassPrefix = "";
        }
        if (object.csharpNamespace !== undefined &&
            object.csharpNamespace !== null) {
            message.csharpNamespace = object.csharpNamespace;
        }
        else {
            message.csharpNamespace = "";
        }
        if (object.swiftPrefix !== undefined && object.swiftPrefix !== null) {
            message.swiftPrefix = object.swiftPrefix;
        }
        else {
            message.swiftPrefix = "";
        }
        if (object.phpClassPrefix !== undefined && object.phpClassPrefix !== null) {
            message.phpClassPrefix = object.phpClassPrefix;
        }
        else {
            message.phpClassPrefix = "";
        }
        if (object.phpNamespace !== undefined && object.phpNamespace !== null) {
            message.phpNamespace = object.phpNamespace;
        }
        else {
            message.phpNamespace = "";
        }
        if (object.phpMetadataNamespace !== undefined &&
            object.phpMetadataNamespace !== null) {
            message.phpMetadataNamespace = object.phpMetadataNamespace;
        }
        else {
            message.phpMetadataNamespace = "";
        }
        if (object.rubyPackage !== undefined && object.rubyPackage !== null) {
            message.rubyPackage = object.rubyPackage;
        }
        else {
            message.rubyPackage = "";
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_66_1) { e_66 = { error: e_66_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_66) throw e_66.error; }
            }
        }
        return message;
    },
};
var baseMessageOptions = {
    messageSetWireFormat: false,
    noStandardDescriptorAccessor: false,
    deprecated: false,
    mapEntry: false,
};
export var MessageOptions = {
    encode: function (message, writer) {
        var e_67, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.messageSetWireFormat === true) {
            writer.uint32(8).bool(message.messageSetWireFormat);
        }
        if (message.noStandardDescriptorAccessor === true) {
            writer.uint32(16).bool(message.noStandardDescriptorAccessor);
        }
        if (message.deprecated === true) {
            writer.uint32(24).bool(message.deprecated);
        }
        if (message.mapEntry === true) {
            writer.uint32(56).bool(message.mapEntry);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_67_1) { e_67 = { error: e_67_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_67) throw e_67.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseMessageOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.messageSetWireFormat = reader.bool();
                    break;
                case 2:
                    message.noStandardDescriptorAccessor = reader.bool();
                    break;
                case 3:
                    message.deprecated = reader.bool();
                    break;
                case 7:
                    message.mapEntry = reader.bool();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_68, _a;
        var message = __assign({}, baseMessageOptions);
        message.uninterpretedOption = [];
        if (object.messageSetWireFormat !== undefined &&
            object.messageSetWireFormat !== null) {
            message.messageSetWireFormat = Boolean(object.messageSetWireFormat);
        }
        else {
            message.messageSetWireFormat = false;
        }
        if (object.noStandardDescriptorAccessor !== undefined &&
            object.noStandardDescriptorAccessor !== null) {
            message.noStandardDescriptorAccessor = Boolean(object.noStandardDescriptorAccessor);
        }
        else {
            message.noStandardDescriptorAccessor = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.mapEntry !== undefined && object.mapEntry !== null) {
            message.mapEntry = Boolean(object.mapEntry);
        }
        else {
            message.mapEntry = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_68_1) { e_68 = { error: e_68_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_68) throw e_68.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.messageSetWireFormat !== undefined &&
            (obj.messageSetWireFormat = message.messageSetWireFormat);
        message.noStandardDescriptorAccessor !== undefined &&
            (obj.noStandardDescriptorAccessor = message.noStandardDescriptorAccessor);
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        message.mapEntry !== undefined && (obj.mapEntry = message.mapEntry);
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_69, _a;
        var message = __assign({}, baseMessageOptions);
        message.uninterpretedOption = [];
        if (object.messageSetWireFormat !== undefined &&
            object.messageSetWireFormat !== null) {
            message.messageSetWireFormat = object.messageSetWireFormat;
        }
        else {
            message.messageSetWireFormat = false;
        }
        if (object.noStandardDescriptorAccessor !== undefined &&
            object.noStandardDescriptorAccessor !== null) {
            message.noStandardDescriptorAccessor =
                object.noStandardDescriptorAccessor;
        }
        else {
            message.noStandardDescriptorAccessor = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.mapEntry !== undefined && object.mapEntry !== null) {
            message.mapEntry = object.mapEntry;
        }
        else {
            message.mapEntry = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_69_1) { e_69 = { error: e_69_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_69) throw e_69.error; }
            }
        }
        return message;
    },
};
var baseFieldOptions = {
    ctype: 0,
    packed: false,
    jstype: 0,
    lazy: false,
    deprecated: false,
    weak: false,
};
export var FieldOptions = {
    encode: function (message, writer) {
        var e_70, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.ctype !== 0) {
            writer.uint32(8).int32(message.ctype);
        }
        if (message.packed === true) {
            writer.uint32(16).bool(message.packed);
        }
        if (message.jstype !== 0) {
            writer.uint32(48).int32(message.jstype);
        }
        if (message.lazy === true) {
            writer.uint32(40).bool(message.lazy);
        }
        if (message.deprecated === true) {
            writer.uint32(24).bool(message.deprecated);
        }
        if (message.weak === true) {
            writer.uint32(80).bool(message.weak);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_70_1) { e_70 = { error: e_70_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_70) throw e_70.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseFieldOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.ctype = reader.int32();
                    break;
                case 2:
                    message.packed = reader.bool();
                    break;
                case 6:
                    message.jstype = reader.int32();
                    break;
                case 5:
                    message.lazy = reader.bool();
                    break;
                case 3:
                    message.deprecated = reader.bool();
                    break;
                case 10:
                    message.weak = reader.bool();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_71, _a;
        var message = __assign({}, baseFieldOptions);
        message.uninterpretedOption = [];
        if (object.ctype !== undefined && object.ctype !== null) {
            message.ctype = fieldOptions_CTypeFromJSON(object.ctype);
        }
        else {
            message.ctype = 0;
        }
        if (object.packed !== undefined && object.packed !== null) {
            message.packed = Boolean(object.packed);
        }
        else {
            message.packed = false;
        }
        if (object.jstype !== undefined && object.jstype !== null) {
            message.jstype = fieldOptions_JSTypeFromJSON(object.jstype);
        }
        else {
            message.jstype = 0;
        }
        if (object.lazy !== undefined && object.lazy !== null) {
            message.lazy = Boolean(object.lazy);
        }
        else {
            message.lazy = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.weak !== undefined && object.weak !== null) {
            message.weak = Boolean(object.weak);
        }
        else {
            message.weak = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_71_1) { e_71 = { error: e_71_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_71) throw e_71.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.ctype !== undefined &&
            (obj.ctype = fieldOptions_CTypeToJSON(message.ctype));
        message.packed !== undefined && (obj.packed = message.packed);
        message.jstype !== undefined &&
            (obj.jstype = fieldOptions_JSTypeToJSON(message.jstype));
        message.lazy !== undefined && (obj.lazy = message.lazy);
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        message.weak !== undefined && (obj.weak = message.weak);
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_72, _a;
        var message = __assign({}, baseFieldOptions);
        message.uninterpretedOption = [];
        if (object.ctype !== undefined && object.ctype !== null) {
            message.ctype = object.ctype;
        }
        else {
            message.ctype = 0;
        }
        if (object.packed !== undefined && object.packed !== null) {
            message.packed = object.packed;
        }
        else {
            message.packed = false;
        }
        if (object.jstype !== undefined && object.jstype !== null) {
            message.jstype = object.jstype;
        }
        else {
            message.jstype = 0;
        }
        if (object.lazy !== undefined && object.lazy !== null) {
            message.lazy = object.lazy;
        }
        else {
            message.lazy = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.weak !== undefined && object.weak !== null) {
            message.weak = object.weak;
        }
        else {
            message.weak = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_72_1) { e_72 = { error: e_72_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_72) throw e_72.error; }
            }
        }
        return message;
    },
};
var baseOneofOptions = {};
export var OneofOptions = {
    encode: function (message, writer) {
        var e_73, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_73_1) { e_73 = { error: e_73_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_73) throw e_73.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseOneofOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_74, _a;
        var message = __assign({}, baseOneofOptions);
        message.uninterpretedOption = [];
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_74_1) { e_74 = { error: e_74_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_74) throw e_74.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_75, _a;
        var message = __assign({}, baseOneofOptions);
        message.uninterpretedOption = [];
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_75_1) { e_75 = { error: e_75_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_75) throw e_75.error; }
            }
        }
        return message;
    },
};
var baseEnumOptions = { allowAlias: false, deprecated: false };
export var EnumOptions = {
    encode: function (message, writer) {
        var e_76, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.allowAlias === true) {
            writer.uint32(16).bool(message.allowAlias);
        }
        if (message.deprecated === true) {
            writer.uint32(24).bool(message.deprecated);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_76_1) { e_76 = { error: e_76_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_76) throw e_76.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEnumOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 2:
                    message.allowAlias = reader.bool();
                    break;
                case 3:
                    message.deprecated = reader.bool();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_77, _a;
        var message = __assign({}, baseEnumOptions);
        message.uninterpretedOption = [];
        if (object.allowAlias !== undefined && object.allowAlias !== null) {
            message.allowAlias = Boolean(object.allowAlias);
        }
        else {
            message.allowAlias = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_77_1) { e_77 = { error: e_77_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_77) throw e_77.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.allowAlias !== undefined && (obj.allowAlias = message.allowAlias);
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_78, _a;
        var message = __assign({}, baseEnumOptions);
        message.uninterpretedOption = [];
        if (object.allowAlias !== undefined && object.allowAlias !== null) {
            message.allowAlias = object.allowAlias;
        }
        else {
            message.allowAlias = false;
        }
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_78_1) { e_78 = { error: e_78_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_78) throw e_78.error; }
            }
        }
        return message;
    },
};
var baseEnumValueOptions = { deprecated: false };
export var EnumValueOptions = {
    encode: function (message, writer) {
        var e_79, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.deprecated === true) {
            writer.uint32(8).bool(message.deprecated);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_79_1) { e_79 = { error: e_79_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_79) throw e_79.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseEnumValueOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.deprecated = reader.bool();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_80, _a;
        var message = __assign({}, baseEnumValueOptions);
        message.uninterpretedOption = [];
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_80_1) { e_80 = { error: e_80_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_80) throw e_80.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_81, _a;
        var message = __assign({}, baseEnumValueOptions);
        message.uninterpretedOption = [];
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_81_1) { e_81 = { error: e_81_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_81) throw e_81.error; }
            }
        }
        return message;
    },
};
var baseServiceOptions = { deprecated: false };
export var ServiceOptions = {
    encode: function (message, writer) {
        var e_82, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.deprecated === true) {
            writer.uint32(264).bool(message.deprecated);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_82_1) { e_82 = { error: e_82_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_82) throw e_82.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseServiceOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 33:
                    message.deprecated = reader.bool();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_83, _a;
        var message = __assign({}, baseServiceOptions);
        message.uninterpretedOption = [];
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_83_1) { e_83 = { error: e_83_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_83) throw e_83.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_84, _a;
        var message = __assign({}, baseServiceOptions);
        message.uninterpretedOption = [];
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_84_1) { e_84 = { error: e_84_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_84) throw e_84.error; }
            }
        }
        return message;
    },
};
var baseMethodOptions = { deprecated: false, idempotencyLevel: 0 };
export var MethodOptions = {
    encode: function (message, writer) {
        var e_85, _a;
        if (writer === void 0) { writer = Writer.create(); }
        if (message.deprecated === true) {
            writer.uint32(264).bool(message.deprecated);
        }
        if (message.idempotencyLevel !== 0) {
            writer.uint32(272).int32(message.idempotencyLevel);
        }
        try {
            for (var _b = __values(message.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption.encode(v, writer.uint32(7994).fork()).ldelim();
            }
        }
        catch (e_85_1) { e_85 = { error: e_85_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_85) throw e_85.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseMethodOptions);
        message.uninterpretedOption = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 33:
                    message.deprecated = reader.bool();
                    break;
                case 34:
                    message.idempotencyLevel = reader.int32();
                    break;
                case 999:
                    message.uninterpretedOption.push(UninterpretedOption.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_86, _a;
        var message = __assign({}, baseMethodOptions);
        message.uninterpretedOption = [];
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = Boolean(object.deprecated);
        }
        else {
            message.deprecated = false;
        }
        if (object.idempotencyLevel !== undefined &&
            object.idempotencyLevel !== null) {
            message.idempotencyLevel = methodOptions_IdempotencyLevelFromJSON(object.idempotencyLevel);
        }
        else {
            message.idempotencyLevel = 0;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromJSON(e));
                }
            }
            catch (e_86_1) { e_86 = { error: e_86_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_86) throw e_86.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.deprecated !== undefined && (obj.deprecated = message.deprecated);
        message.idempotencyLevel !== undefined &&
            (obj.idempotencyLevel = methodOptions_IdempotencyLevelToJSON(message.idempotencyLevel));
        if (message.uninterpretedOption) {
            obj.uninterpretedOption = message.uninterpretedOption.map(function (e) {
                return e ? UninterpretedOption.toJSON(e) : undefined;
            });
        }
        else {
            obj.uninterpretedOption = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_87, _a;
        var message = __assign({}, baseMethodOptions);
        message.uninterpretedOption = [];
        if (object.deprecated !== undefined && object.deprecated !== null) {
            message.deprecated = object.deprecated;
        }
        else {
            message.deprecated = false;
        }
        if (object.idempotencyLevel !== undefined &&
            object.idempotencyLevel !== null) {
            message.idempotencyLevel = object.idempotencyLevel;
        }
        else {
            message.idempotencyLevel = 0;
        }
        if (object.uninterpretedOption !== undefined &&
            object.uninterpretedOption !== null) {
            try {
                for (var _b = __values(object.uninterpretedOption), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.uninterpretedOption.push(UninterpretedOption.fromPartial(e));
                }
            }
            catch (e_87_1) { e_87 = { error: e_87_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_87) throw e_87.error; }
            }
        }
        return message;
    },
};
var baseUninterpretedOption = {
    identifierValue: "",
    positiveIntValue: 0,
    negativeIntValue: 0,
    doubleValue: 0,
    aggregateValue: "",
};
export var UninterpretedOption = {
    encode: function (message, writer) {
        var e_88, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.name), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                UninterpretedOption_NamePart.encode(v, writer.uint32(18).fork()).ldelim();
            }
        }
        catch (e_88_1) { e_88 = { error: e_88_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_88) throw e_88.error; }
        }
        if (message.identifierValue !== "") {
            writer.uint32(26).string(message.identifierValue);
        }
        if (message.positiveIntValue !== 0) {
            writer.uint32(32).uint64(message.positiveIntValue);
        }
        if (message.negativeIntValue !== 0) {
            writer.uint32(40).int64(message.negativeIntValue);
        }
        if (message.doubleValue !== 0) {
            writer.uint32(49).double(message.doubleValue);
        }
        if (message.stringValue.length !== 0) {
            writer.uint32(58).bytes(message.stringValue);
        }
        if (message.aggregateValue !== "") {
            writer.uint32(66).string(message.aggregateValue);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseUninterpretedOption);
        message.name = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 2:
                    message.name.push(UninterpretedOption_NamePart.decode(reader, reader.uint32()));
                    break;
                case 3:
                    message.identifierValue = reader.string();
                    break;
                case 4:
                    message.positiveIntValue = longToNumber(reader.uint64());
                    break;
                case 5:
                    message.negativeIntValue = longToNumber(reader.int64());
                    break;
                case 6:
                    message.doubleValue = reader.double();
                    break;
                case 7:
                    message.stringValue = reader.bytes();
                    break;
                case 8:
                    message.aggregateValue = reader.string();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_89, _a;
        var message = __assign({}, baseUninterpretedOption);
        message.name = [];
        if (object.name !== undefined && object.name !== null) {
            try {
                for (var _b = __values(object.name), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.name.push(UninterpretedOption_NamePart.fromJSON(e));
                }
            }
            catch (e_89_1) { e_89 = { error: e_89_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_89) throw e_89.error; }
            }
        }
        if (object.identifierValue !== undefined &&
            object.identifierValue !== null) {
            message.identifierValue = String(object.identifierValue);
        }
        else {
            message.identifierValue = "";
        }
        if (object.positiveIntValue !== undefined &&
            object.positiveIntValue !== null) {
            message.positiveIntValue = Number(object.positiveIntValue);
        }
        else {
            message.positiveIntValue = 0;
        }
        if (object.negativeIntValue !== undefined &&
            object.negativeIntValue !== null) {
            message.negativeIntValue = Number(object.negativeIntValue);
        }
        else {
            message.negativeIntValue = 0;
        }
        if (object.doubleValue !== undefined && object.doubleValue !== null) {
            message.doubleValue = Number(object.doubleValue);
        }
        else {
            message.doubleValue = 0;
        }
        if (object.stringValue !== undefined && object.stringValue !== null) {
            message.stringValue = bytesFromBase64(object.stringValue);
        }
        if (object.aggregateValue !== undefined && object.aggregateValue !== null) {
            message.aggregateValue = String(object.aggregateValue);
        }
        else {
            message.aggregateValue = "";
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.name) {
            obj.name = message.name.map(function (e) {
                return e ? UninterpretedOption_NamePart.toJSON(e) : undefined;
            });
        }
        else {
            obj.name = [];
        }
        message.identifierValue !== undefined &&
            (obj.identifierValue = message.identifierValue);
        message.positiveIntValue !== undefined &&
            (obj.positiveIntValue = message.positiveIntValue);
        message.negativeIntValue !== undefined &&
            (obj.negativeIntValue = message.negativeIntValue);
        message.doubleValue !== undefined &&
            (obj.doubleValue = message.doubleValue);
        message.stringValue !== undefined &&
            (obj.stringValue = base64FromBytes(message.stringValue !== undefined
                ? message.stringValue
                : new Uint8Array()));
        message.aggregateValue !== undefined &&
            (obj.aggregateValue = message.aggregateValue);
        return obj;
    },
    fromPartial: function (object) {
        var e_90, _a;
        var message = __assign({}, baseUninterpretedOption);
        message.name = [];
        if (object.name !== undefined && object.name !== null) {
            try {
                for (var _b = __values(object.name), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.name.push(UninterpretedOption_NamePart.fromPartial(e));
                }
            }
            catch (e_90_1) { e_90 = { error: e_90_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_90) throw e_90.error; }
            }
        }
        if (object.identifierValue !== undefined &&
            object.identifierValue !== null) {
            message.identifierValue = object.identifierValue;
        }
        else {
            message.identifierValue = "";
        }
        if (object.positiveIntValue !== undefined &&
            object.positiveIntValue !== null) {
            message.positiveIntValue = object.positiveIntValue;
        }
        else {
            message.positiveIntValue = 0;
        }
        if (object.negativeIntValue !== undefined &&
            object.negativeIntValue !== null) {
            message.negativeIntValue = object.negativeIntValue;
        }
        else {
            message.negativeIntValue = 0;
        }
        if (object.doubleValue !== undefined && object.doubleValue !== null) {
            message.doubleValue = object.doubleValue;
        }
        else {
            message.doubleValue = 0;
        }
        if (object.stringValue !== undefined && object.stringValue !== null) {
            message.stringValue = object.stringValue;
        }
        else {
            message.stringValue = new Uint8Array();
        }
        if (object.aggregateValue !== undefined && object.aggregateValue !== null) {
            message.aggregateValue = object.aggregateValue;
        }
        else {
            message.aggregateValue = "";
        }
        return message;
    },
};
var baseUninterpretedOption_NamePart = {
    namePart: "",
    isExtension: false,
};
export var UninterpretedOption_NamePart = {
    encode: function (message, writer) {
        if (writer === void 0) { writer = Writer.create(); }
        if (message.namePart !== "") {
            writer.uint32(10).string(message.namePart);
        }
        if (message.isExtension === true) {
            writer.uint32(16).bool(message.isExtension);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseUninterpretedOption_NamePart);
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.namePart = reader.string();
                    break;
                case 2:
                    message.isExtension = reader.bool();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var message = __assign({}, baseUninterpretedOption_NamePart);
        if (object.namePart !== undefined && object.namePart !== null) {
            message.namePart = String(object.namePart);
        }
        else {
            message.namePart = "";
        }
        if (object.isExtension !== undefined && object.isExtension !== null) {
            message.isExtension = Boolean(object.isExtension);
        }
        else {
            message.isExtension = false;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        message.namePart !== undefined && (obj.namePart = message.namePart);
        message.isExtension !== undefined &&
            (obj.isExtension = message.isExtension);
        return obj;
    },
    fromPartial: function (object) {
        var message = __assign({}, baseUninterpretedOption_NamePart);
        if (object.namePart !== undefined && object.namePart !== null) {
            message.namePart = object.namePart;
        }
        else {
            message.namePart = "";
        }
        if (object.isExtension !== undefined && object.isExtension !== null) {
            message.isExtension = object.isExtension;
        }
        else {
            message.isExtension = false;
        }
        return message;
    },
};
var baseSourceCodeInfo = {};
export var SourceCodeInfo = {
    encode: function (message, writer) {
        var e_91, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.location), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                SourceCodeInfo_Location.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_91_1) { e_91 = { error: e_91_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_91) throw e_91.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseSourceCodeInfo);
        message.location = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.location.push(SourceCodeInfo_Location.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_92, _a;
        var message = __assign({}, baseSourceCodeInfo);
        message.location = [];
        if (object.location !== undefined && object.location !== null) {
            try {
                for (var _b = __values(object.location), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.location.push(SourceCodeInfo_Location.fromJSON(e));
                }
            }
            catch (e_92_1) { e_92 = { error: e_92_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_92) throw e_92.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.location) {
            obj.location = message.location.map(function (e) {
                return e ? SourceCodeInfo_Location.toJSON(e) : undefined;
            });
        }
        else {
            obj.location = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_93, _a;
        var message = __assign({}, baseSourceCodeInfo);
        message.location = [];
        if (object.location !== undefined && object.location !== null) {
            try {
                for (var _b = __values(object.location), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.location.push(SourceCodeInfo_Location.fromPartial(e));
                }
            }
            catch (e_93_1) { e_93 = { error: e_93_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_93) throw e_93.error; }
            }
        }
        return message;
    },
};
var baseSourceCodeInfo_Location = {
    path: 0,
    span: 0,
    leadingComments: "",
    trailingComments: "",
    leadingDetachedComments: "",
};
export var SourceCodeInfo_Location = {
    encode: function (message, writer) {
        var e_94, _a, e_95, _b, e_96, _c;
        if (writer === void 0) { writer = Writer.create(); }
        writer.uint32(10).fork();
        try {
            for (var _d = __values(message.path), _e = _d.next(); !_e.done; _e = _d.next()) {
                var v = _e.value;
                writer.int32(v);
            }
        }
        catch (e_94_1) { e_94 = { error: e_94_1 }; }
        finally {
            try {
                if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
            }
            finally { if (e_94) throw e_94.error; }
        }
        writer.ldelim();
        writer.uint32(18).fork();
        try {
            for (var _f = __values(message.span), _g = _f.next(); !_g.done; _g = _f.next()) {
                var v = _g.value;
                writer.int32(v);
            }
        }
        catch (e_95_1) { e_95 = { error: e_95_1 }; }
        finally {
            try {
                if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
            }
            finally { if (e_95) throw e_95.error; }
        }
        writer.ldelim();
        if (message.leadingComments !== "") {
            writer.uint32(26).string(message.leadingComments);
        }
        if (message.trailingComments !== "") {
            writer.uint32(34).string(message.trailingComments);
        }
        try {
            for (var _h = __values(message.leadingDetachedComments), _j = _h.next(); !_j.done; _j = _h.next()) {
                var v = _j.value;
                writer.uint32(50).string(v);
            }
        }
        catch (e_96_1) { e_96 = { error: e_96_1 }; }
        finally {
            try {
                if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
            }
            finally { if (e_96) throw e_96.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseSourceCodeInfo_Location);
        message.path = [];
        message.span = [];
        message.leadingDetachedComments = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if ((tag & 7) === 2) {
                        var end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.path.push(reader.int32());
                        }
                    }
                    else {
                        message.path.push(reader.int32());
                    }
                    break;
                case 2:
                    if ((tag & 7) === 2) {
                        var end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.span.push(reader.int32());
                        }
                    }
                    else {
                        message.span.push(reader.int32());
                    }
                    break;
                case 3:
                    message.leadingComments = reader.string();
                    break;
                case 4:
                    message.trailingComments = reader.string();
                    break;
                case 6:
                    message.leadingDetachedComments.push(reader.string());
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_97, _a, e_98, _b, e_99, _c;
        var message = __assign({}, baseSourceCodeInfo_Location);
        message.path = [];
        message.span = [];
        message.leadingDetachedComments = [];
        if (object.path !== undefined && object.path !== null) {
            try {
                for (var _d = __values(object.path), _e = _d.next(); !_e.done; _e = _d.next()) {
                    var e = _e.value;
                    message.path.push(Number(e));
                }
            }
            catch (e_97_1) { e_97 = { error: e_97_1 }; }
            finally {
                try {
                    if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
                }
                finally { if (e_97) throw e_97.error; }
            }
        }
        if (object.span !== undefined && object.span !== null) {
            try {
                for (var _f = __values(object.span), _g = _f.next(); !_g.done; _g = _f.next()) {
                    var e = _g.value;
                    message.span.push(Number(e));
                }
            }
            catch (e_98_1) { e_98 = { error: e_98_1 }; }
            finally {
                try {
                    if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
                }
                finally { if (e_98) throw e_98.error; }
            }
        }
        if (object.leadingComments !== undefined &&
            object.leadingComments !== null) {
            message.leadingComments = String(object.leadingComments);
        }
        else {
            message.leadingComments = "";
        }
        if (object.trailingComments !== undefined &&
            object.trailingComments !== null) {
            message.trailingComments = String(object.trailingComments);
        }
        else {
            message.trailingComments = "";
        }
        if (object.leadingDetachedComments !== undefined &&
            object.leadingDetachedComments !== null) {
            try {
                for (var _h = __values(object.leadingDetachedComments), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.leadingDetachedComments.push(String(e));
                }
            }
            catch (e_99_1) { e_99 = { error: e_99_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
                }
                finally { if (e_99) throw e_99.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.path) {
            obj.path = message.path.map(function (e) { return e; });
        }
        else {
            obj.path = [];
        }
        if (message.span) {
            obj.span = message.span.map(function (e) { return e; });
        }
        else {
            obj.span = [];
        }
        message.leadingComments !== undefined &&
            (obj.leadingComments = message.leadingComments);
        message.trailingComments !== undefined &&
            (obj.trailingComments = message.trailingComments);
        if (message.leadingDetachedComments) {
            obj.leadingDetachedComments = message.leadingDetachedComments.map(function (e) { return e; });
        }
        else {
            obj.leadingDetachedComments = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_100, _a, e_101, _b, e_102, _c;
        var message = __assign({}, baseSourceCodeInfo_Location);
        message.path = [];
        message.span = [];
        message.leadingDetachedComments = [];
        if (object.path !== undefined && object.path !== null) {
            try {
                for (var _d = __values(object.path), _e = _d.next(); !_e.done; _e = _d.next()) {
                    var e = _e.value;
                    message.path.push(e);
                }
            }
            catch (e_100_1) { e_100 = { error: e_100_1 }; }
            finally {
                try {
                    if (_e && !_e.done && (_a = _d.return)) _a.call(_d);
                }
                finally { if (e_100) throw e_100.error; }
            }
        }
        if (object.span !== undefined && object.span !== null) {
            try {
                for (var _f = __values(object.span), _g = _f.next(); !_g.done; _g = _f.next()) {
                    var e = _g.value;
                    message.span.push(e);
                }
            }
            catch (e_101_1) { e_101 = { error: e_101_1 }; }
            finally {
                try {
                    if (_g && !_g.done && (_b = _f.return)) _b.call(_f);
                }
                finally { if (e_101) throw e_101.error; }
            }
        }
        if (object.leadingComments !== undefined &&
            object.leadingComments !== null) {
            message.leadingComments = object.leadingComments;
        }
        else {
            message.leadingComments = "";
        }
        if (object.trailingComments !== undefined &&
            object.trailingComments !== null) {
            message.trailingComments = object.trailingComments;
        }
        else {
            message.trailingComments = "";
        }
        if (object.leadingDetachedComments !== undefined &&
            object.leadingDetachedComments !== null) {
            try {
                for (var _h = __values(object.leadingDetachedComments), _j = _h.next(); !_j.done; _j = _h.next()) {
                    var e = _j.value;
                    message.leadingDetachedComments.push(e);
                }
            }
            catch (e_102_1) { e_102 = { error: e_102_1 }; }
            finally {
                try {
                    if (_j && !_j.done && (_c = _h.return)) _c.call(_h);
                }
                finally { if (e_102) throw e_102.error; }
            }
        }
        return message;
    },
};
var baseGeneratedCodeInfo = {};
export var GeneratedCodeInfo = {
    encode: function (message, writer) {
        var e_103, _a;
        if (writer === void 0) { writer = Writer.create(); }
        try {
            for (var _b = __values(message.annotation), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                GeneratedCodeInfo_Annotation.encode(v, writer.uint32(10).fork()).ldelim();
            }
        }
        catch (e_103_1) { e_103 = { error: e_103_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_103) throw e_103.error; }
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseGeneratedCodeInfo);
        message.annotation = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.annotation.push(GeneratedCodeInfo_Annotation.decode(reader, reader.uint32()));
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_104, _a;
        var message = __assign({}, baseGeneratedCodeInfo);
        message.annotation = [];
        if (object.annotation !== undefined && object.annotation !== null) {
            try {
                for (var _b = __values(object.annotation), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.annotation.push(GeneratedCodeInfo_Annotation.fromJSON(e));
                }
            }
            catch (e_104_1) { e_104 = { error: e_104_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_104) throw e_104.error; }
            }
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.annotation) {
            obj.annotation = message.annotation.map(function (e) {
                return e ? GeneratedCodeInfo_Annotation.toJSON(e) : undefined;
            });
        }
        else {
            obj.annotation = [];
        }
        return obj;
    },
    fromPartial: function (object) {
        var e_105, _a;
        var message = __assign({}, baseGeneratedCodeInfo);
        message.annotation = [];
        if (object.annotation !== undefined && object.annotation !== null) {
            try {
                for (var _b = __values(object.annotation), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.annotation.push(GeneratedCodeInfo_Annotation.fromPartial(e));
                }
            }
            catch (e_105_1) { e_105 = { error: e_105_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_105) throw e_105.error; }
            }
        }
        return message;
    },
};
var baseGeneratedCodeInfo_Annotation = {
    path: 0,
    sourceFile: "",
    begin: 0,
    end: 0,
};
export var GeneratedCodeInfo_Annotation = {
    encode: function (message, writer) {
        var e_106, _a;
        if (writer === void 0) { writer = Writer.create(); }
        writer.uint32(10).fork();
        try {
            for (var _b = __values(message.path), _c = _b.next(); !_c.done; _c = _b.next()) {
                var v = _c.value;
                writer.int32(v);
            }
        }
        catch (e_106_1) { e_106 = { error: e_106_1 }; }
        finally {
            try {
                if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
            }
            finally { if (e_106) throw e_106.error; }
        }
        writer.ldelim();
        if (message.sourceFile !== "") {
            writer.uint32(18).string(message.sourceFile);
        }
        if (message.begin !== 0) {
            writer.uint32(24).int32(message.begin);
        }
        if (message.end !== 0) {
            writer.uint32(32).int32(message.end);
        }
        return writer;
    },
    decode: function (input, length) {
        var reader = input instanceof Uint8Array ? new Reader(input) : input;
        var end = length === undefined ? reader.len : reader.pos + length;
        var message = __assign({}, baseGeneratedCodeInfo_Annotation);
        message.path = [];
        while (reader.pos < end) {
            var tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    if ((tag & 7) === 2) {
                        var end2 = reader.uint32() + reader.pos;
                        while (reader.pos < end2) {
                            message.path.push(reader.int32());
                        }
                    }
                    else {
                        message.path.push(reader.int32());
                    }
                    break;
                case 2:
                    message.sourceFile = reader.string();
                    break;
                case 3:
                    message.begin = reader.int32();
                    break;
                case 4:
                    message.end = reader.int32();
                    break;
                default:
                    reader.skipType(tag & 7);
                    break;
            }
        }
        return message;
    },
    fromJSON: function (object) {
        var e_107, _a;
        var message = __assign({}, baseGeneratedCodeInfo_Annotation);
        message.path = [];
        if (object.path !== undefined && object.path !== null) {
            try {
                for (var _b = __values(object.path), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.path.push(Number(e));
                }
            }
            catch (e_107_1) { e_107 = { error: e_107_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_107) throw e_107.error; }
            }
        }
        if (object.sourceFile !== undefined && object.sourceFile !== null) {
            message.sourceFile = String(object.sourceFile);
        }
        else {
            message.sourceFile = "";
        }
        if (object.begin !== undefined && object.begin !== null) {
            message.begin = Number(object.begin);
        }
        else {
            message.begin = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = Number(object.end);
        }
        else {
            message.end = 0;
        }
        return message;
    },
    toJSON: function (message) {
        var obj = {};
        if (message.path) {
            obj.path = message.path.map(function (e) { return e; });
        }
        else {
            obj.path = [];
        }
        message.sourceFile !== undefined && (obj.sourceFile = message.sourceFile);
        message.begin !== undefined && (obj.begin = message.begin);
        message.end !== undefined && (obj.end = message.end);
        return obj;
    },
    fromPartial: function (object) {
        var e_108, _a;
        var message = __assign({}, baseGeneratedCodeInfo_Annotation);
        message.path = [];
        if (object.path !== undefined && object.path !== null) {
            try {
                for (var _b = __values(object.path), _c = _b.next(); !_c.done; _c = _b.next()) {
                    var e = _c.value;
                    message.path.push(e);
                }
            }
            catch (e_108_1) { e_108 = { error: e_108_1 }; }
            finally {
                try {
                    if (_c && !_c.done && (_a = _b.return)) _a.call(_b);
                }
                finally { if (e_108) throw e_108.error; }
            }
        }
        if (object.sourceFile !== undefined && object.sourceFile !== null) {
            message.sourceFile = object.sourceFile;
        }
        else {
            message.sourceFile = "";
        }
        if (object.begin !== undefined && object.begin !== null) {
            message.begin = object.begin;
        }
        else {
            message.begin = 0;
        }
        if (object.end !== undefined && object.end !== null) {
            message.end = object.end;
        }
        else {
            message.end = 0;
        }
        return message;
    },
};
var globalThis = (function () {
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
var atob = globalThis.atob ||
    (function (b64) { return globalThis.Buffer.from(b64, "base64").toString("binary"); });
function bytesFromBase64(b64) {
    var bin = atob(b64);
    var arr = new Uint8Array(bin.length);
    for (var i = 0; i < bin.length; ++i) {
        arr[i] = bin.charCodeAt(i);
    }
    return arr;
}
var btoa = globalThis.btoa ||
    (function (bin) { return globalThis.Buffer.from(bin, "binary").toString("base64"); });
function base64FromBytes(arr) {
    var bin = [];
    for (var i = 0; i < arr.byteLength; ++i) {
        bin.push(String.fromCharCode(arr[i]));
    }
    return btoa(bin.join(""));
}
function longToNumber(long) {
    if (long.gt(Number.MAX_SAFE_INTEGER)) {
        throw new globalThis.Error("Value is larger than Number.MAX_SAFE_INTEGER");
    }
    return long.toNumber();
}
if (util.Long !== Long) {
    util.Long = Long;
    configure();
}
