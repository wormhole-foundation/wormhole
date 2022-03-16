import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface ActiveGuardianSetIndex {
    index: number;
}
export declare const ActiveGuardianSetIndex: {
    encode(message: ActiveGuardianSetIndex, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number | undefined): ActiveGuardianSetIndex;
    fromJSON(object: any): ActiveGuardianSetIndex;
    toJSON(message: ActiveGuardianSetIndex): unknown;
    fromPartial(object: DeepPartial<ActiveGuardianSetIndex>): ActiveGuardianSetIndex;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
