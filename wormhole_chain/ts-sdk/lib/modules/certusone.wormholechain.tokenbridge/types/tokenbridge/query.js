"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.QueryClientImpl = exports.QueryAllCoinMetaRollbackProtectionResponse = exports.QueryAllCoinMetaRollbackProtectionRequest = exports.QueryGetCoinMetaRollbackProtectionResponse = exports.QueryGetCoinMetaRollbackProtectionRequest = exports.QueryAllChainRegistrationResponse = exports.QueryAllChainRegistrationRequest = exports.QueryGetChainRegistrationResponse = exports.QueryGetChainRegistrationRequest = exports.QueryAllReplayProtectionResponse = exports.QueryAllReplayProtectionRequest = exports.QueryGetReplayProtectionResponse = exports.QueryGetReplayProtectionRequest = exports.QueryGetConfigResponse = exports.QueryGetConfigRequest = exports.protobufPackage = void 0;
//@ts-nocheck
/* eslint-disable */
const minimal_1 = require("protobufjs/minimal");
const config_1 = require("../tokenbridge/config");
const replay_protection_1 = require("../tokenbridge/replay_protection");
const pagination_1 = require("../cosmos/base/query/v1beta1/pagination");
const chain_registration_1 = require("../tokenbridge/chain_registration");
const coin_meta_rollback_protection_1 = require("../tokenbridge/coin_meta_rollback_protection");
exports.protobufPackage = "certusone.wormholechain.tokenbridge";
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
const baseQueryGetChainRegistrationRequest = { chainID: 0 };
exports.QueryGetChainRegistrationRequest = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.chainID !== 0) {
            writer.uint32(8).uint32(message.chainID);
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
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
exports.QueryGetChainRegistrationResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.chainRegistration !== undefined) {
            chain_registration_1.ChainRegistration.encode(message.chainRegistration, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetChainRegistrationResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.chainRegistration = chain_registration_1.ChainRegistration.decode(reader, reader.uint32());
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
            message.chainRegistration = chain_registration_1.ChainRegistration.fromJSON(object.chainRegistration);
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
                ? chain_registration_1.ChainRegistration.toJSON(message.chainRegistration)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetChainRegistrationResponse,
        };
        if (object.chainRegistration !== undefined &&
            object.chainRegistration !== null) {
            message.chainRegistration = chain_registration_1.ChainRegistration.fromPartial(object.chainRegistration);
        }
        else {
            message.chainRegistration = undefined;
        }
        return message;
    },
};
const baseQueryAllChainRegistrationRequest = {};
exports.QueryAllChainRegistrationRequest = {
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
            ...baseQueryAllChainRegistrationRequest,
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
            ...baseQueryAllChainRegistrationRequest,
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
            ...baseQueryAllChainRegistrationRequest,
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
const baseQueryAllChainRegistrationResponse = {};
exports.QueryAllChainRegistrationResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.chainRegistration) {
            chain_registration_1.ChainRegistration.encode(v, writer.uint32(10).fork()).ldelim();
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
            ...baseQueryAllChainRegistrationResponse,
        };
        message.chainRegistration = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.chainRegistration.push(chain_registration_1.ChainRegistration.decode(reader, reader.uint32()));
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
            ...baseQueryAllChainRegistrationResponse,
        };
        message.chainRegistration = [];
        if (object.chainRegistration !== undefined &&
            object.chainRegistration !== null) {
            for (const e of object.chainRegistration) {
                message.chainRegistration.push(chain_registration_1.ChainRegistration.fromJSON(e));
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
        if (message.chainRegistration) {
            obj.chainRegistration = message.chainRegistration.map((e) => e ? chain_registration_1.ChainRegistration.toJSON(e) : undefined);
        }
        else {
            obj.chainRegistration = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageResponse.toJSON(message.pagination)
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
                message.chainRegistration.push(chain_registration_1.ChainRegistration.fromPartial(e));
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
const baseQueryGetCoinMetaRollbackProtectionRequest = { index: "" };
exports.QueryGetCoinMetaRollbackProtectionRequest = {
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
exports.QueryGetCoinMetaRollbackProtectionResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        if (message.coinMetaRollbackProtection !== undefined) {
            coin_meta_rollback_protection_1.CoinMetaRollbackProtection.encode(message.coinMetaRollbackProtection, writer.uint32(10).fork()).ldelim();
        }
        return writer;
    },
    decode(input, length) {
        const reader = input instanceof Uint8Array ? new minimal_1.Reader(input) : input;
        let end = length === undefined ? reader.len : reader.pos + length;
        const message = {
            ...baseQueryGetCoinMetaRollbackProtectionResponse,
        };
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.coinMetaRollbackProtection = coin_meta_rollback_protection_1.CoinMetaRollbackProtection.decode(reader, reader.uint32());
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
            message.coinMetaRollbackProtection = coin_meta_rollback_protection_1.CoinMetaRollbackProtection.fromJSON(object.coinMetaRollbackProtection);
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
                ? coin_meta_rollback_protection_1.CoinMetaRollbackProtection.toJSON(message.coinMetaRollbackProtection)
                : undefined);
        return obj;
    },
    fromPartial(object) {
        const message = {
            ...baseQueryGetCoinMetaRollbackProtectionResponse,
        };
        if (object.coinMetaRollbackProtection !== undefined &&
            object.coinMetaRollbackProtection !== null) {
            message.coinMetaRollbackProtection = coin_meta_rollback_protection_1.CoinMetaRollbackProtection.fromPartial(object.coinMetaRollbackProtection);
        }
        else {
            message.coinMetaRollbackProtection = undefined;
        }
        return message;
    },
};
const baseQueryAllCoinMetaRollbackProtectionRequest = {};
exports.QueryAllCoinMetaRollbackProtectionRequest = {
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
            ...baseQueryAllCoinMetaRollbackProtectionRequest,
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
            ...baseQueryAllCoinMetaRollbackProtectionRequest,
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
            ...baseQueryAllCoinMetaRollbackProtectionRequest,
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
const baseQueryAllCoinMetaRollbackProtectionResponse = {};
exports.QueryAllCoinMetaRollbackProtectionResponse = {
    encode(message, writer = minimal_1.Writer.create()) {
        for (const v of message.coinMetaRollbackProtection) {
            coin_meta_rollback_protection_1.CoinMetaRollbackProtection.encode(v, writer.uint32(10).fork()).ldelim();
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
            ...baseQueryAllCoinMetaRollbackProtectionResponse,
        };
        message.coinMetaRollbackProtection = [];
        while (reader.pos < end) {
            const tag = reader.uint32();
            switch (tag >>> 3) {
                case 1:
                    message.coinMetaRollbackProtection.push(coin_meta_rollback_protection_1.CoinMetaRollbackProtection.decode(reader, reader.uint32()));
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
            ...baseQueryAllCoinMetaRollbackProtectionResponse,
        };
        message.coinMetaRollbackProtection = [];
        if (object.coinMetaRollbackProtection !== undefined &&
            object.coinMetaRollbackProtection !== null) {
            for (const e of object.coinMetaRollbackProtection) {
                message.coinMetaRollbackProtection.push(coin_meta_rollback_protection_1.CoinMetaRollbackProtection.fromJSON(e));
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
        if (message.coinMetaRollbackProtection) {
            obj.coinMetaRollbackProtection = message.coinMetaRollbackProtection.map((e) => (e ? coin_meta_rollback_protection_1.CoinMetaRollbackProtection.toJSON(e) : undefined));
        }
        else {
            obj.coinMetaRollbackProtection = [];
        }
        message.pagination !== undefined &&
            (obj.pagination = message.pagination
                ? pagination_1.PageResponse.toJSON(message.pagination)
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
                message.coinMetaRollbackProtection.push(coin_meta_rollback_protection_1.CoinMetaRollbackProtection.fromPartial(e));
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
    Config(request) {
        const data = exports.QueryGetConfigRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "Config", data);
        return promise.then((data) => exports.QueryGetConfigResponse.decode(new minimal_1.Reader(data)));
    }
    ReplayProtection(request) {
        const data = exports.QueryGetReplayProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ReplayProtection", data);
        return promise.then((data) => exports.QueryGetReplayProtectionResponse.decode(new minimal_1.Reader(data)));
    }
    ReplayProtectionAll(request) {
        const data = exports.QueryAllReplayProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ReplayProtectionAll", data);
        return promise.then((data) => exports.QueryAllReplayProtectionResponse.decode(new minimal_1.Reader(data)));
    }
    ChainRegistration(request) {
        const data = exports.QueryGetChainRegistrationRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ChainRegistration", data);
        return promise.then((data) => exports.QueryGetChainRegistrationResponse.decode(new minimal_1.Reader(data)));
    }
    ChainRegistrationAll(request) {
        const data = exports.QueryAllChainRegistrationRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "ChainRegistrationAll", data);
        return promise.then((data) => exports.QueryAllChainRegistrationResponse.decode(new minimal_1.Reader(data)));
    }
    CoinMetaRollbackProtection(request) {
        const data = exports.QueryGetCoinMetaRollbackProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "CoinMetaRollbackProtection", data);
        return promise.then((data) => exports.QueryGetCoinMetaRollbackProtectionResponse.decode(new minimal_1.Reader(data)));
    }
    CoinMetaRollbackProtectionAll(request) {
        const data = exports.QueryAllCoinMetaRollbackProtectionRequest.encode(request).finish();
        const promise = this.rpc.request("certusone.wormholechain.tokenbridge.Query", "CoinMetaRollbackProtectionAll", data);
        return promise.then((data) => exports.QueryAllCoinMetaRollbackProtectionResponse.decode(new minimal_1.Reader(data)));
    }
}
exports.QueryClientImpl = QueryClientImpl;
