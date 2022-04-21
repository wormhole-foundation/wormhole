//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface EventGuardianSetUpdate {
    oldIndex: number;
    newIndex: number;
}
export interface EventPostedMessage {
    emitter: Uint8Array;
    sequence: number;
    nonce: number;
    time: number;
    payload: Uint8Array;
}
export interface EventGuardianRegistered {
    guardianKey: Uint8Array;
    validatorKey: Uint8Array;
}
export interface EventConsensusSetUpdate {
    oldIndex: number;
    newIndex: number;
}
export declare const EventGuardianSetUpdate: {
    encode(message: EventGuardianSetUpdate, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventGuardianSetUpdate;
    fromJSON(object: any): EventGuardianSetUpdate;
    toJSON(message: EventGuardianSetUpdate): unknown;
    fromPartial(object: DeepPartial<EventGuardianSetUpdate>): EventGuardianSetUpdate;
};
export declare const EventPostedMessage: {
    encode(message: EventPostedMessage, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventPostedMessage;
    fromJSON(object: any): EventPostedMessage;
    toJSON(message: EventPostedMessage): unknown;
    fromPartial(object: DeepPartial<EventPostedMessage>): EventPostedMessage;
};
export declare const EventGuardianRegistered: {
    encode(message: EventGuardianRegistered, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventGuardianRegistered;
    fromJSON(object: any): EventGuardianRegistered;
    toJSON(message: EventGuardianRegistered): unknown;
    fromPartial(object: DeepPartial<EventGuardianRegistered>): EventGuardianRegistered;
};
export declare const EventConsensusSetUpdate: {
    encode(message: EventConsensusSetUpdate, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): EventConsensusSetUpdate;
    fromJSON(object: any): EventConsensusSetUpdate;
    toJSON(message: EventConsensusSetUpdate): unknown;
    fromPartial(object: DeepPartial<EventConsensusSetUpdate>): EventConsensusSetUpdate;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
