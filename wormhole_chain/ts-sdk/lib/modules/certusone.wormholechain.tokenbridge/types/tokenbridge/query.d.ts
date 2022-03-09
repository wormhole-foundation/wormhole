import { Reader, Writer } from "protobufjs/minimal";
import { Config } from "../tokenbridge/config";
import { ReplayProtection } from "../tokenbridge/replay_protection";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { ChainRegistration } from "../tokenbridge/chain_registration";
import { CoinMetaRollbackProtection } from "../tokenbridge/coin_meta_rollback_protection";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
export interface QueryGetConfigRequest {
}
export interface QueryGetConfigResponse {
    Config: Config | undefined;
}
export interface QueryGetReplayProtectionRequest {
    index: string;
}
export interface QueryGetReplayProtectionResponse {
    replayProtection: ReplayProtection | undefined;
}
export interface QueryAllReplayProtectionRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllReplayProtectionResponse {
    replayProtection: ReplayProtection[];
    pagination: PageResponse | undefined;
}
export interface QueryGetChainRegistrationRequest {
    chainID: number;
}
export interface QueryGetChainRegistrationResponse {
    chainRegistration: ChainRegistration | undefined;
}
export interface QueryAllChainRegistrationRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllChainRegistrationResponse {
    chainRegistration: ChainRegistration[];
    pagination: PageResponse | undefined;
}
export interface QueryGetCoinMetaRollbackProtectionRequest {
    index: string;
}
export interface QueryGetCoinMetaRollbackProtectionResponse {
    coinMetaRollbackProtection: CoinMetaRollbackProtection | undefined;
}
export interface QueryAllCoinMetaRollbackProtectionRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllCoinMetaRollbackProtectionResponse {
    coinMetaRollbackProtection: CoinMetaRollbackProtection[];
    pagination: PageResponse | undefined;
}
export declare const QueryGetConfigRequest: {
    encode(_: QueryGetConfigRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetConfigRequest;
    fromJSON(_: any): QueryGetConfigRequest;
    toJSON(_: QueryGetConfigRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetConfigRequest>): QueryGetConfigRequest;
};
export declare const QueryGetConfigResponse: {
    encode(message: QueryGetConfigResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetConfigResponse;
    fromJSON(object: any): QueryGetConfigResponse;
    toJSON(message: QueryGetConfigResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetConfigResponse>): QueryGetConfigResponse;
};
export declare const QueryGetReplayProtectionRequest: {
    encode(message: QueryGetReplayProtectionRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetReplayProtectionRequest;
    fromJSON(object: any): QueryGetReplayProtectionRequest;
    toJSON(message: QueryGetReplayProtectionRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetReplayProtectionRequest>): QueryGetReplayProtectionRequest;
};
export declare const QueryGetReplayProtectionResponse: {
    encode(message: QueryGetReplayProtectionResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetReplayProtectionResponse;
    fromJSON(object: any): QueryGetReplayProtectionResponse;
    toJSON(message: QueryGetReplayProtectionResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetReplayProtectionResponse>): QueryGetReplayProtectionResponse;
};
export declare const QueryAllReplayProtectionRequest: {
    encode(message: QueryAllReplayProtectionRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllReplayProtectionRequest;
    fromJSON(object: any): QueryAllReplayProtectionRequest;
    toJSON(message: QueryAllReplayProtectionRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllReplayProtectionRequest>): QueryAllReplayProtectionRequest;
};
export declare const QueryAllReplayProtectionResponse: {
    encode(message: QueryAllReplayProtectionResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllReplayProtectionResponse;
    fromJSON(object: any): QueryAllReplayProtectionResponse;
    toJSON(message: QueryAllReplayProtectionResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllReplayProtectionResponse>): QueryAllReplayProtectionResponse;
};
export declare const QueryGetChainRegistrationRequest: {
    encode(message: QueryGetChainRegistrationRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetChainRegistrationRequest;
    fromJSON(object: any): QueryGetChainRegistrationRequest;
    toJSON(message: QueryGetChainRegistrationRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetChainRegistrationRequest>): QueryGetChainRegistrationRequest;
};
export declare const QueryGetChainRegistrationResponse: {
    encode(message: QueryGetChainRegistrationResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetChainRegistrationResponse;
    fromJSON(object: any): QueryGetChainRegistrationResponse;
    toJSON(message: QueryGetChainRegistrationResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetChainRegistrationResponse>): QueryGetChainRegistrationResponse;
};
export declare const QueryAllChainRegistrationRequest: {
    encode(message: QueryAllChainRegistrationRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllChainRegistrationRequest;
    fromJSON(object: any): QueryAllChainRegistrationRequest;
    toJSON(message: QueryAllChainRegistrationRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllChainRegistrationRequest>): QueryAllChainRegistrationRequest;
};
export declare const QueryAllChainRegistrationResponse: {
    encode(message: QueryAllChainRegistrationResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllChainRegistrationResponse;
    fromJSON(object: any): QueryAllChainRegistrationResponse;
    toJSON(message: QueryAllChainRegistrationResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllChainRegistrationResponse>): QueryAllChainRegistrationResponse;
};
export declare const QueryGetCoinMetaRollbackProtectionRequest: {
    encode(message: QueryGetCoinMetaRollbackProtectionRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetCoinMetaRollbackProtectionRequest;
    fromJSON(object: any): QueryGetCoinMetaRollbackProtectionRequest;
    toJSON(message: QueryGetCoinMetaRollbackProtectionRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetCoinMetaRollbackProtectionRequest>): QueryGetCoinMetaRollbackProtectionRequest;
};
export declare const QueryGetCoinMetaRollbackProtectionResponse: {
    encode(message: QueryGetCoinMetaRollbackProtectionResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetCoinMetaRollbackProtectionResponse;
    fromJSON(object: any): QueryGetCoinMetaRollbackProtectionResponse;
    toJSON(message: QueryGetCoinMetaRollbackProtectionResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetCoinMetaRollbackProtectionResponse>): QueryGetCoinMetaRollbackProtectionResponse;
};
export declare const QueryAllCoinMetaRollbackProtectionRequest: {
    encode(message: QueryAllCoinMetaRollbackProtectionRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllCoinMetaRollbackProtectionRequest;
    fromJSON(object: any): QueryAllCoinMetaRollbackProtectionRequest;
    toJSON(message: QueryAllCoinMetaRollbackProtectionRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllCoinMetaRollbackProtectionRequest>): QueryAllCoinMetaRollbackProtectionRequest;
};
export declare const QueryAllCoinMetaRollbackProtectionResponse: {
    encode(message: QueryAllCoinMetaRollbackProtectionResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllCoinMetaRollbackProtectionResponse;
    fromJSON(object: any): QueryAllCoinMetaRollbackProtectionResponse;
    toJSON(message: QueryAllCoinMetaRollbackProtectionResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllCoinMetaRollbackProtectionResponse>): QueryAllCoinMetaRollbackProtectionResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Queries a config by index. */
    Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
    /** Queries a replayProtection by index. */
    ReplayProtection(request: QueryGetReplayProtectionRequest): Promise<QueryGetReplayProtectionResponse>;
    /** Queries a list of replayProtection items. */
    ReplayProtectionAll(request: QueryAllReplayProtectionRequest): Promise<QueryAllReplayProtectionResponse>;
    /** Queries a chainRegistration by index. */
    ChainRegistration(request: QueryGetChainRegistrationRequest): Promise<QueryGetChainRegistrationResponse>;
    /** Queries a list of chainRegistration items. */
    ChainRegistrationAll(request: QueryAllChainRegistrationRequest): Promise<QueryAllChainRegistrationResponse>;
    /** Queries a coinMetaRollbackProtection by index. */
    CoinMetaRollbackProtection(request: QueryGetCoinMetaRollbackProtectionRequest): Promise<QueryGetCoinMetaRollbackProtectionResponse>;
    /** Queries a list of coinMetaRollbackProtection items. */
    CoinMetaRollbackProtectionAll(request: QueryAllCoinMetaRollbackProtectionRequest): Promise<QueryAllCoinMetaRollbackProtectionResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    constructor(rpc: Rpc);
    Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
    ReplayProtection(request: QueryGetReplayProtectionRequest): Promise<QueryGetReplayProtectionResponse>;
    ReplayProtectionAll(request: QueryAllReplayProtectionRequest): Promise<QueryAllReplayProtectionResponse>;
    ChainRegistration(request: QueryGetChainRegistrationRequest): Promise<QueryGetChainRegistrationResponse>;
    ChainRegistrationAll(request: QueryAllChainRegistrationRequest): Promise<QueryAllChainRegistrationResponse>;
    CoinMetaRollbackProtection(request: QueryGetCoinMetaRollbackProtectionRequest): Promise<QueryGetCoinMetaRollbackProtectionResponse>;
    CoinMetaRollbackProtectionAll(request: QueryAllCoinMetaRollbackProtectionRequest): Promise<QueryAllCoinMetaRollbackProtectionResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
