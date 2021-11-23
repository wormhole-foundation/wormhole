import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface EventGuardianSetUpdate {
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
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
