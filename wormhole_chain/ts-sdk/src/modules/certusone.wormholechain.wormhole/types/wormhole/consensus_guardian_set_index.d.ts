//@ts-nocheck
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
export interface ConsensusGuardianSetIndex {
    index: number;
}
export declare const ConsensusGuardianSetIndex: {
    encode(message: ConsensusGuardianSetIndex, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): ConsensusGuardianSetIndex;
    fromJSON(object: any): ConsensusGuardianSetIndex;
    toJSON(message: ConsensusGuardianSetIndex): unknown;
    fromPartial(object: DeepPartial<ConsensusGuardianSetIndex>): ConsensusGuardianSetIndex;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
