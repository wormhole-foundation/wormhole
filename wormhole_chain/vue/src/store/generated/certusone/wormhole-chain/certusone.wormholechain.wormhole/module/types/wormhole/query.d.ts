import { Reader, Writer } from "protobufjs/minimal";
import { GuardianSet } from "../wormhole/guardian_set";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { Config } from "../wormhole/config";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface QueryGetGuardianSetRequest {
    index: number;
}
export interface QueryGetGuardianSetResponse {
    GuardianSet: GuardianSet | undefined;
}
export interface QueryAllGuardianSetRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllGuardianSetResponse {
    GuardianSet: GuardianSet[];
    pagination: PageResponse | undefined;
}
export interface QueryGetConfigRequest {
}
export interface QueryGetConfigResponse {
    Config: Config | undefined;
}
export declare const QueryGetGuardianSetRequest: {
    encode(message: QueryGetGuardianSetRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetGuardianSetRequest;
    fromJSON(object: any): QueryGetGuardianSetRequest;
    toJSON(message: QueryGetGuardianSetRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetGuardianSetRequest>): QueryGetGuardianSetRequest;
};
export declare const QueryGetGuardianSetResponse: {
    encode(message: QueryGetGuardianSetResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetGuardianSetResponse;
    fromJSON(object: any): QueryGetGuardianSetResponse;
    toJSON(message: QueryGetGuardianSetResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetGuardianSetResponse>): QueryGetGuardianSetResponse;
};
export declare const QueryAllGuardianSetRequest: {
    encode(message: QueryAllGuardianSetRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllGuardianSetRequest;
    fromJSON(object: any): QueryAllGuardianSetRequest;
    toJSON(message: QueryAllGuardianSetRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllGuardianSetRequest>): QueryAllGuardianSetRequest;
};
export declare const QueryAllGuardianSetResponse: {
    encode(message: QueryAllGuardianSetResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryAllGuardianSetResponse;
    fromJSON(object: any): QueryAllGuardianSetResponse;
    toJSON(message: QueryAllGuardianSetResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllGuardianSetResponse>): QueryAllGuardianSetResponse;
};
export declare const QueryGetConfigRequest: {
    encode(_: QueryGetConfigRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetConfigRequest;
    fromJSON(_: any): QueryGetConfigRequest;
    toJSON(_: QueryGetConfigRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetConfigRequest>): QueryGetConfigRequest;
};
export declare const QueryGetConfigResponse: {
    encode(message: QueryGetConfigResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): QueryGetConfigResponse;
    fromJSON(object: any): QueryGetConfigResponse;
    toJSON(message: QueryGetConfigResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetConfigResponse>): QueryGetConfigResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Queries a guardianSet by index. */
    GuardianSet(request: QueryGetGuardianSetRequest): Promise<QueryGetGuardianSetResponse>;
    /** Queries a list of guardianSet items. */
    GuardianSetAll(request: QueryAllGuardianSetRequest): Promise<QueryAllGuardianSetResponse>;
    /** Queries a config by index. */
    Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    constructor(rpc: Rpc);
    GuardianSet(request: QueryGetGuardianSetRequest): Promise<QueryGetGuardianSetResponse>;
    GuardianSetAll(request: QueryAllGuardianSetRequest): Promise<QueryAllGuardianSetResponse>;
    Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
