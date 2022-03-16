//@ts-nocheck
import { GuardianSet } from "../wormhole/guardian_set";
import { Config } from "../wormhole/config";
import { ReplayProtection } from "../wormhole/replay_protection";
import { SequenceCounter } from "../wormhole/sequence_counter";
import { ActiveGuardianSetIndex } from "../wormhole/active_guardian_set_index";
import { GuardianValidator } from "../wormhole/guardian_validator";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.wormhole";
/** GenesisState defines the wormhole module's genesis state. */
export interface GenesisState {
    guardianSetList: GuardianSet[];
    config: Config | undefined;
    replayProtectionList: ReplayProtection[];
    sequenceCounterList: SequenceCounter[];
    activeGuardianSetIndex: ActiveGuardianSetIndex | undefined;
    /** this line is used by starport scaffolding # genesis/proto/state */
    guardianValidatorList: GuardianValidator[];
}
export declare const GenesisState: {
    encode(message: GenesisState, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GenesisState;
    fromJSON(object: any): GenesisState;
    toJSON(message: GenesisState): unknown;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
