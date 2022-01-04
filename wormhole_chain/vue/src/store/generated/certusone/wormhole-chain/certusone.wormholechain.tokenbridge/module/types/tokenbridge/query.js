/* eslint-disable */
import { Reader, Writer } from "protobufjs/minimal";
import { Config } from "../tokenbridge/config";
import { ReplayProtection } from "../tokenbridge/replay_protection";
import { PageRequest, PageResponse, } from "../cosmos/base/query/v1beta1/pagination";
import { ChainRegistration } from "../tokenbridge/chain_registration";
import { CoinMetaRollbackProtection } from "../tokenbridge/coin_meta_rollback_protection";
export const protobufPackage = "certusone.wormholechain.tokenbridge";
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
const baseQueryGetReplayProtectionRequest = { index: "" };
export const QueryGetReplayProtectionRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
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
export const QueryGetReplayProtectionResponse = {
    encode(message, writer = Writer.create()) {
        if (message.replayProtection !== undefined) {
            ReplayProtection.encode(message.replayProtection, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetReplayProtectionResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = {
            ...baseQueryGetReplayProtectionResponse,
        };
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            message.replayProtection = ReplayProtection.fromJSON(object.replayProtection);
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
                ? ReplayProtection.toJSON(message.replayProtection)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetReplayProtectionResponse,
        };
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
const baseQueryAllReplayProtectionRequest = {};
export const QueryAllReplayProtectionRequest = {
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
            ...baseQueryAllReplayProtectionRequest,
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
            ...baseQueryAllReplayProtectionRequest,
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
            ...baseQueryAllReplayProtectionRequest,
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
const baseQueryAllReplayProtectionResponse = {};
export const QueryAllReplayProtectionResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.replayProtection) {
            ReplayProtection.encode(v, writer.uint32(10).fork()).ldelim();
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
            ...baseQueryAllReplayProtectionResponse,
        };
        message.replayProtection = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
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
    fromJSON(object) {
        const message = {
            ...baseQueryAllReplayProtectionResponse,
        };
        message.replayProtection = [];
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            for (const e of object.replayProtection) {
                message.replayProtection.push(ReplayProtection.fromJSON(e));
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
        if (message.replayProtection) {
            obj.replayProtection = message.replayProtection.map((e) => e ? ReplayProtection.toJSON(e) : undefined);
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
    fromPartial(object) {
        const message = {
            ...baseQueryAllReplayProtectionResponse,
        };
        message.replayProtection = [];
        if (object.replayProtection !== undefined &&
            object.replayProtection !== null) {
            for (const e of object.replayProtection) {
                message.replayProtection.push(ReplayProtection.fromPartial(e));
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
const baseQueryGetChainRegistrationRequest = { chainID: 0 };
export const QueryGetChainRegistrationRequest = {
    encode(message, writer = Writer.create()) {
        if (message.chainID !== 0) {
            writer.uint32(8).uint32(message.chainID);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetChainRegistrationRequest,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.chainID = reader.uint32();
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
            ...baseQueryGetChainRegistrationRequest,
        };
        if (object.chainID !== undefined && object.chainID !== null) {
            message.chainID = Number(object.chainID);
        }
        else {
            message.chainID = 0;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.chainID !== undefined && (obj.chainID = message.chainID);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetChainRegistrationRequest,
        };
        if (object.chainID !== undefined && object.chainID !== null) {
            message.chainID = object.chainID;
        }
        else {
            message.chainID = 0;
        }
        return message;
    },
};
const baseQueryGetChainRegistrationResponse = {};
export const QueryGetChainRegistrationResponse = {
    encode(message, writer = Writer.create()) {
        if (message.chainRegistration !== undefined) {
            ChainRegistration.encode(message.chainRegistration, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetChainRegistrationResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.chainRegistration = ChainRegistration.decode(reader, reader.uint32());
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
            ...baseQueryGetChainRegistrationResponse,
        };
        if (object.chainRegistration !== undefined &&
            object.chainRegistration !== null) {
            message.chainRegistration = ChainRegistration.fromJSON(object.chainRegistration);
        }
        else {
            message.chainRegistration = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.chainRegistration !== undefined &&
            (obj.chainRegistration = message.chainRegistration
                ? ChainRegistration.toJSON(message.chainRegistration)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetChainRegistrationResponse,
        };
        if (object.chainRegistration !== undefined &&
            object.chainRegistration !== null) {
            message.chainRegistration = ChainRegistration.fromPartial(object.chainRegistration);
        }
        else {
            message.chainRegistration = undefined;
        }
        return message;
    },
};
const baseQueryAllChainRegistrationRequest = {};
export const QueryAllChainRegistrationRequest = {
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
            ...baseQueryAllChainRegistrationRequest,
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
            ...baseQueryAllChainRegistrationRequest,
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
            ...baseQueryAllChainRegistrationRequest,
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
const baseQueryAllChainRegistrationResponse = {};
export const QueryAllChainRegistrationResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.chainRegistration) {
            ChainRegistration.encode(v, writer.uint32(10).fork()).ldelim();
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
            ...baseQueryAllChainRegistrationResponse,
        };
        message.chainRegistration = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.chainRegistration.push(ChainRegistration.decode(reader, reader.uint32()));
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
            ...baseQueryAllChainRegistrationResponse,
        };
        message.chainRegistration = [];
        if (object.chainRegistration !== undefined &&
            object.chainRegistration !== null) {
            for (const e of object.chainRegistration) {
                message.chainRegistration.push(ChainRegistration.fromJSON(e));
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
        if (message.chainRegistration) {
            obj.chainRegistration = message.chainRegistration.map((e) => e ? ChainRegistration.toJSON(e) : undefined);
        }
        else {
            obj.chainRegistration = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllChainRegistrationResponse,
        };
        message.chainRegistration = [];
        if (object.chainRegistration !== undefined &&
            object.chainRegistration !== null) {
            for (const e of object.chainRegistration) {
                message.chainRegistration.push(ChainRegistration.fromPartial(e));
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
const baseQueryGetCoinMetaRollbackProtectionRequest = { index: "" };
export const QueryGetCoinMetaRollbackProtectionRequest = {
    encode(message, writer = Writer.create()) {
        if (message.index !== "") {
            writer.uint32(10).string(message.index);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetCoinMetaRollbackProtectionRequest,
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
            ...baseQueryGetCoinMetaRollbackProtectionRequest,
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
            ...baseQueryGetCoinMetaRollbackProtectionRequest,
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
const baseQueryGetCoinMetaRollbackProtectionResponse = {};
export const QueryGetCoinMetaRollbackProtectionResponse = {
    encode(message, writer = Writer.create()) {
        if (message.coinMetaRollbackProtection !== undefined) {
            CoinMetaRollbackProtection.encode(message.coinMetaRollbackProtection, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetCoinMetaRollbackProtectionResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.coinMetaRollbackProtection = CoinMetaRollbackProtection.decode(reader, reader.uint32());
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
            ...baseQueryGetCoinMetaRollbackProtectionResponse,
        };
        if (object.coinMetaRollbackProtection !== undefined &&
            object.coinMetaRollbackProtection !== null) {
            message.coinMetaRollbackProtection = CoinMetaRollbackProtection.fromJSON(object.coinMetaRollbackProtection);
        }
        else {
            message.coinMetaRollbackProtection = undefined;
        }
        return message;
    },
    toJSON(message) {
        const obj = {};
        message.coinMetaRollbackProtection !== undefined &&
            (obj.coinMetaRollbackProtection = message.coinMetaRollbackProtection
                ? CoinMetaRollbackProtection.toJSON(message.coinMetaRollbackProtection)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetCoinMetaRollbackProtectionResponse,
        };
        if (object.coinMetaRollbackProtection !== undefined &&
            object.coinMetaRollbackProtection !== null) {
            message.coinMetaRollbackProtection = CoinMetaRollbackProtection.fromPartial(object.coinMetaRollbackProtection);
        }
        else {
            message.coinMetaRollbackProtection = undefined;
        }
        return message;
    },
};
const baseQueryAllCoinMetaRollbackProtectionRequest = {};
export const QueryAllCoinMetaRollbackProtectionRequest = {
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
            ...baseQueryAllCoinMetaRollbackProtectionRequest,
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
            ...baseQueryAllCoinMetaRollbackProtectionRequest,
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
            ...baseQueryAllCoinMetaRollbackProtectionRequest,
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
const baseQueryAllCoinMetaRollbackProtectionResponse = {};
export const QueryAllCoinMetaRollbackProtectionResponse = {
    encode(message, writer = Writer.create()) {
        for (const v of message.coinMetaRollbackProtection) {
            CoinMetaRollbackProtection.encode(v, writer.uint32(10).fork()).ldelim();
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
            ...baseQueryAllCoinMetaRollbackProtectionResponse,
        };
        message.coinMetaRollbackProtection = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.coinMetaRollbackProtection.push(CoinMetaRollbackProtection.decode(reader, reader.uint32()));
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
            ...baseQueryAllCoinMetaRollbackProtectionResponse,
        };
        message.coinMetaRollbackProtection = [];
        if (object.coinMetaRollbackProtection !== undefined &&
            object.coinMetaRollbackProtection !== null) {
            for (const e of object.coinMetaRollbackProtection) {
                message.coinMetaRollbackProtection.push(CoinMetaRollbackProtection.fromJSON(e));
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
        if (message.coinMetaRollbackProtection) {
            obj.coinMetaRollbackProtection = message.coinMetaRollbackProtection.map((e) => (e ? CoinMetaRollbackProtection.toJSON(e) : undefined));
        }
        else {
            obj.coinMetaRollbackProtection = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? PageResponse.toJSON(message.pagination)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryAllCoinMetaRollbackProtectionResponse,
        };
        message.coinMetaRollbackProtection = [];
        if (object.coinMetaRollbackProtection !== undefined &&
            object.coinMetaRollbackProtection !== null) {
            for (const e of object.coinMetaRollbackProtection) {
                message.coinMetaRollbackProtection.push(CoinMetaRollbackProtection.fromPartial(e));
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
export class QueryClientImpl {
    constructor(rpc) {
        this.rpc = rpc;
    }
    Config(request) {
        const data = QueryGetConfigRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "Config", data);
        return promise.then((data) => QueryGetConfigResponse.decode(new Reader(data)));
    }
    ReplayProtection(request) {
        const data = QueryGetReplayProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ReplayProtection", data);
        return promise.then((data) => QueryGetReplayProtectionResponse.decode(new Reader(data)));
    }
    ReplayProtectionAll(request) {
        const data = QueryAllReplayProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ReplayProtectionAll", data);
        return promise.then((data) => QueryAllReplayProtectionResponse.decode(new Reader(data)));
    }
    ChainRegistration(request) {
        const data = QueryGetChainRegistrationRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ChainRegistration", data);
        return promise.then((data) => QueryGetChainRegistrationResponse.decode(new Reader(data)));
    }
    ChainRegistrationAll(request) {
        const data = QueryAllChainRegistrationRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ChainRegistrationAll", data);
        return promise.then((data) => QueryAllChainRegistrationResponse.decode(new Reader(data)));
    }
    CoinMetaRollbackProtection(request) {
        const data = QueryGetCoinMetaRollbackProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "CoinMetaRollbackProtection", data);
        return promise.then((data) => QueryGetCoinMetaRollbackProtectionResponse.decode(new Reader(data)));
    }
    CoinMetaRollbackProtectionAll(request) {
        const data = QueryAllCoinMetaRollbackProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "CoinMetaRollbackProtectionAll", data);
        return promise.then((data) => QueryAllCoinMetaRollbackProtectionResponse.decode(new Reader(data)));
    }
}
