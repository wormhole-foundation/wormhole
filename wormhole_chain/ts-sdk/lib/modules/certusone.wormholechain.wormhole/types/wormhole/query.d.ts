import { Reader, Writer } from "protobufjs/minimal";
import { GuardianSet } from "../wormhole/guardian_set";
import { PageRequest, PageResponse } from "../cosmos/base/query/v1beta1/pagination";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";
import { ActiveGuardianSetIndex } from "../wormhole/active_guardian_set_index";
import { GuardianValidator } from "../wormhole/guardian_validator";
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
export interface QueryGetSequenceCounterRequest {
    index: string;
}
export interface QueryGetSequenceCounterResponse {
    sequenceCounter: SequenceCounter | undefined;
}
export interface QueryAllSequenceCounterRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllSequenceCounterResponse {
    sequenceCounter: SequenceCounter[];
    pagination: PageResponse | undefined;
}
export interface QueryGetActiveGuardianSetIndexRequest {
}
export interface QueryGetActiveGuardianSetIndexResponse {
    ActiveGuardianSetIndex: ActiveGuardianSetIndex | undefined;
}
export interface QueryGetGuardianValidatorRequest {
    guardianKey: Uint8Array;
}
export interface QueryGetGuardianValidatorResponse {
    guardianValidator: GuardianValidator | undefined;
}
export interface QueryAllGuardianValidatorRequest {
    pagination: PageRequest | undefined;
}
export interface QueryAllGuardianValidatorResponse {
    guardianValidator: GuardianValidator[];
    pagination: PageResponse | undefined;
}
export declare const QueryGetGuardianSetRequest: {
    encode(message: QueryGetGuardianSetRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetGuardianSetRequest;
    fromJSON(object: any): QueryGetGuardianSetRequest;
    toJSON(message: QueryGetGuardianSetRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetGuardianSetRequest>): QueryGetGuardianSetRequest;
};
export declare const QueryGetGuardianSetResponse: {
    encode(message: QueryGetGuardianSetResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetGuardianSetResponse;
    fromJSON(object: any): QueryGetGuardianSetResponse;
    toJSON(message: QueryGetGuardianSetResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetGuardianSetResponse>): QueryGetGuardianSetResponse;
};
export declare const QueryAllGuardianSetRequest: {
    encode(message: QueryAllGuardianSetRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllGuardianSetRequest;
    fromJSON(object: any): QueryAllGuardianSetRequest;
    toJSON(message: QueryAllGuardianSetRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllGuardianSetRequest>): QueryAllGuardianSetRequest;
};
export declare const QueryAllGuardianSetResponse: {
    encode(message: QueryAllGuardianSetResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllGuardianSetResponse;
    fromJSON(object: any): QueryAllGuardianSetResponse;
    toJSON(message: QueryAllGuardianSetResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllGuardianSetResponse>): QueryAllGuardianSetResponse;
};
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
export declare const QueryGetSequenceCounterRequest: {
    encode(message: QueryGetSequenceCounterRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetSequenceCounterRequest;
    fromJSON(object: any): QueryGetSequenceCounterRequest;
    toJSON(message: QueryGetSequenceCounterRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetSequenceCounterRequest>): QueryGetSequenceCounterRequest;
};
export declare const QueryGetSequenceCounterResponse: {
    encode(message: QueryGetSequenceCounterResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetSequenceCounterResponse;
    fromJSON(object: any): QueryGetSequenceCounterResponse;
    toJSON(message: QueryGetSequenceCounterResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetSequenceCounterResponse>): QueryGetSequenceCounterResponse;
};
export declare const QueryAllSequenceCounterRequest: {
    encode(message: QueryAllSequenceCounterRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllSequenceCounterRequest;
    fromJSON(object: any): QueryAllSequenceCounterRequest;
    toJSON(message: QueryAllSequenceCounterRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllSequenceCounterRequest>): QueryAllSequenceCounterRequest;
};
export declare const QueryAllSequenceCounterResponse: {
    encode(message: QueryAllSequenceCounterResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllSequenceCounterResponse;
    fromJSON(object: any): QueryAllSequenceCounterResponse;
    toJSON(message: QueryAllSequenceCounterResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllSequenceCounterResponse>): QueryAllSequenceCounterResponse;
};
export declare const QueryGetActiveGuardianSetIndexRequest: {
    encode(_: QueryGetActiveGuardianSetIndexRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetActiveGuardianSetIndexRequest;
    fromJSON(_: any): QueryGetActiveGuardianSetIndexRequest;
    toJSON(_: QueryGetActiveGuardianSetIndexRequest): unknown;
    fromPartial(_: DeepPartial<QueryGetActiveGuardianSetIndexRequest>): QueryGetActiveGuardianSetIndexRequest;
};
export declare const QueryGetActiveGuardianSetIndexResponse: {
    encode(message: QueryGetActiveGuardianSetIndexResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetActiveGuardianSetIndexResponse;
    fromJSON(object: any): QueryGetActiveGuardianSetIndexResponse;
    toJSON(message: QueryGetActiveGuardianSetIndexResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetActiveGuardianSetIndexResponse>): QueryGetActiveGuardianSetIndexResponse;
};
export declare const QueryGetGuardianValidatorRequest: {
    encode(message: QueryGetGuardianValidatorRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetGuardianValidatorRequest;
    fromJSON(object: any): QueryGetGuardianValidatorRequest;
    toJSON(message: QueryGetGuardianValidatorRequest): unknown;
    fromPartial(object: DeepPartial<QueryGetGuardianValidatorRequest>): QueryGetGuardianValidatorRequest;
};
export declare const QueryGetGuardianValidatorResponse: {
    encode(message: QueryGetGuardianValidatorResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryGetGuardianValidatorResponse;
    fromJSON(object: any): QueryGetGuardianValidatorResponse;
    toJSON(message: QueryGetGuardianValidatorResponse): unknown;
    fromPartial(object: DeepPartial<QueryGetGuardianValidatorResponse>): QueryGetGuardianValidatorResponse;
};
export declare const QueryAllGuardianValidatorRequest: {
    encode(message: QueryAllGuardianValidatorRequest, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllGuardianValidatorRequest;
    fromJSON(object: any): QueryAllGuardianValidatorRequest;
    toJSON(message: QueryAllGuardianValidatorRequest): unknown;
    fromPartial(object: DeepPartial<QueryAllGuardianValidatorRequest>): QueryAllGuardianValidatorRequest;
};
export declare const QueryAllGuardianValidatorResponse: {
    encode(message: QueryAllGuardianValidatorResponse, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): QueryAllGuardianValidatorResponse;
    fromJSON(object: any): QueryAllGuardianValidatorResponse;
    toJSON(message: QueryAllGuardianValidatorResponse): unknown;
    fromPartial(object: DeepPartial<QueryAllGuardianValidatorResponse>): QueryAllGuardianValidatorResponse;
};
/** Query defines the gRPC querier service. */
export interface Query {
    /** Queries a guardianSet by index. */
    GuardianSet(request: QueryGetGuardianSetRequest): Promise<QueryGetGuardianSetResponse>;
    /** Queries a list of guardianSet items. */
    GuardianSetAll(request: QueryAllGuardianSetRequest): Promise<QueryAllGuardianSetResponse>;
    /** Queries a config by index. */
    Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
    /** Queries a replayProtection by index. */
    ReplayProtection(request: QueryGetReplayProtectionRequest): Promise<QueryGetReplayProtectionResponse>;
    /** Queries a list of replayProtection items. */
    ReplayProtectionAll(request: QueryAllReplayProtectionRequest): Promise<QueryAllReplayProtectionResponse>;
    /** Queries a sequenceCounter by index. */
    SequenceCounter(request: QueryGetSequenceCounterRequest): Promise<QueryGetSequenceCounterResponse>;
    /** Queries a list of sequenceCounter items. */
    SequenceCounterAll(request: QueryAllSequenceCounterRequest): Promise<QueryAllSequenceCounterResponse>;
    /** Queries a ActiveGuardianSetIndex by index. */
    ActiveGuardianSetIndex(request: QueryGetActiveGuardianSetIndexRequest): Promise<QueryGetActiveGuardianSetIndexResponse>;
    /** Queries a GuardianValidator by index. */
    GuardianValidator(request: QueryGetGuardianValidatorRequest): Promise<QueryGetGuardianValidatorResponse>;
    /** Queries a list of GuardianValidator items. */
    GuardianValidatorAll(request: QueryAllGuardianValidatorRequest): Promise<QueryAllGuardianValidatorResponse>;
}
export declare class QueryClientImpl implements Query {
    private readonly rpc;
    constructor(rpc: Rpc);
    GuardianSet(request: QueryGetGuardianSetRequest): Promise<QueryGetGuardianSetResponse>;
    GuardianSetAll(request: QueryAllGuardianSetRequest): Promise<QueryAllGuardianSetResponse>;
    Config(request: QueryGetConfigRequest): Promise<QueryGetConfigResponse>;
    ReplayProtection(request: QueryGetReplayProtectionRequest): Promise<QueryGetReplayProtectionResponse>;
    ReplayProtectionAll(request: QueryAllReplayProtectionRequest): Promise<QueryAllReplayProtectionResponse>;
    SequenceCounter(request: QueryGetSequenceCounterRequest): Promise<QueryGetSequenceCounterResponse>;
    SequenceCounterAll(request: QueryAllSequenceCounterRequest): Promise<QueryAllSequenceCounterResponse>;
    ActiveGuardianSetIndex(request: QueryGetActiveGuardianSetIndexRequest): Promise<QueryGetActiveGuardianSetIndexResponse>;
    GuardianValidator(request: QueryGetGuardianValidatorRequest): Promise<QueryGetGuardianValidatorResponse>;
    GuardianValidatorAll(request: QueryAllGuardianValidatorRequest): Promise<QueryAllGuardianValidatorResponse>;
}
interface Rpc {
    request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
