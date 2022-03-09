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
    payload: Uint8Array;
}
export declare const EventGuardianSetUpdate: {
    encode(message: EventGuardianSetUpdate, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): EventGuardianSetUpdate;
    fromJSON(object: any): EventGuardianSetUpdate;
    toJSON(message: EventGuardianSetUpdate): unknown;
    fromPartial(object: DeepPartial<EventGuardianSetUpdate>): EventGuardianSetUpdate;
};
export declare const EventPostedMessage: {
    encode(message: EventPostedMessage, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): EventPostedMessage;
    fromJSON(object: any): EventPostedMessage;
    toJSON(message: EventPostedMessage): unknown;
    fromPartial(object: DeepPartial<EventPostedMessage>): EventPostedMessage;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
