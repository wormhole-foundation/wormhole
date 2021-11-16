/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { GuardianSet } from "../wormhole/guardian_set";
import { PageRequest, PageResponse, } from "../cosmos/base/query/v1beta1/pagination";
import { Config } from "../wormhole/config";
export const protobufPackage = "certusone.wormholechain.wormhole";
const baseQueryGetGuardianSetRequest = { index: 0 };
export const QueryGetGuardianSetRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== 0) {
            writer.uint32(8).uint32(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
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
export const QueryGetGuardianSetResponse = {
    encode(message, writer = Writer.create()) {
        if (message.GuardianSet !== undefined) {
            GuardianSet.encode(message.GuardianSet, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetGuardianSetResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = {
            ...baseQueryGetGuardianSetResponse,
        };
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            message.GuardianSet = GuardianSet.fromJSON(object.GuardianSet);
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
                ? GuardianSet.toJSON(message.GuardianSet)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetGuardianSetResponse,
        };
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            message.GuardianSet = GuardianSet.fromPartial(object.GuardianSet);
        }
        else {
            message.GuardianSet = undefined;
        }
        return message;
    },
};
const baseQueryAllGuardianSetRequest = {};
export const QueryAllGuardianSetRequest = {
    encode(message, writer = Writer.create()) {
        if (message.pagination !== undefined) {
            PageRequest.encode(message.pagination, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllGuardianSetRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = {
            ...baseQueryAllGuardianSetRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromJSON(object.pagination);
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
                ? PageRequest.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllGuardianSetRequest,
        };
        if (object.pagination !== undefined && object.pagination !== null) {
            message.pagination = PageRequest.fromPartial(object.pagination);
        }
        else {
            message.pagination = undefined;
        }
        return message;
    },
};
const baseQueryAllGuardianSetResponse = {};
export const QueryAllGuardianSetResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.GuardianSet) {
            GuardianSet.encode(v, writer.uint32(10).fork()).ldelim();
        }
        if (message.pagination !== undefined) {
            PageResponse.encode(message.pagination, writer.uint32(18).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryAllGuardianSetResponse,
        };
        message.GuardianSet = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = {
            ...baseQueryAllGuardianSetResponse,
        };
        message.GuardianSet = [];
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            for (const e of object.GuardianSet) {
                message.GuardianSet.push(GuardianSet.fromJSON(e));
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
    toJSON(message) {
        const obj = {};
        if (message.GuardianSet) {
            obj.GuardianSet = message.GuardianSet.map((e) => e ? GuardianSet.toJSON(e) : undefined);
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
    fromPartial(object) {
        const message = {
            ...baseQueryAllGuardianSetResponse,
        };
        message.GuardianSet = [];
        if (object.GuardianSet !== undefined && object.GuardianSet !== null) {
            for (const e of object.GuardianSet) {
                message.GuardianSet.push(GuardianSet.fromPartial(e));
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
const baseQueryGetConfigRequest = {};
export const QueryGetConfigRequest = {
    encode(_, writer = Writer.create()) {
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
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
export const QueryGetConfigResponse = {
    encode(message, writer = Writer.create()) {
        if (message.Config !== undefined) {
            Config.encode(message.Config, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = { ...baseQueryGetConfigResponse };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = { ...baseQueryGetConfigResponse };
        if (object.Config !== undefined && object.Config !== null) {
            message.Config = Config.fromJSON(object.Config);
        }
        else {
            message.Config = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.Config !== undefined &&
            (obj.Config = message.Config ? Config.toJSON(message.Config) : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = { ...baseQueryGetConfigResponse };
        if (object.Config !== undefined && object.Config !== null) {
            message.Config = Config.fromPartial(object.Config);
        }
        else {
            message.Config = undefined;
        }
        return message;
    },
};
export class QueryClientImpl {
    constructor(rpc) {
        this.rpc = rpc;
    }
    GuardianSet(request) {
        const data = QueryGetGuardianSetRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianSet", data);
        return promise.then((data) => QueryGetGuardianSetResponse.decode(new Reader(data)));
    }
    GuardianSetAll(request) {
        const data = QueryAllGuardianSetRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "GuardianSetAll", data);
        return promise.then((data) => QueryAllGuardianSetResponse.decode(new Reader(data)));
    }
    Config(request) {
        const data = QueryGetConfigRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.wormhole.Query", "Config", data);
        return promise.then((data) => QueryGetConfigResponse.decode(new Reader(data)));
    }
}
