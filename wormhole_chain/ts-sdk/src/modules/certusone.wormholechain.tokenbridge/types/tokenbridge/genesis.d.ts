//@ts-nocheck
import { Config } from "../tokenbridge/config";
import { ReplayProtection } from "../tokenbridge/replay_protection";
import { ChainRegistration } from "../tokenbridge/chain_registration";
import { CoinMetaRollbackProtection } from "../tokenbridge/coin_meta_rollback_protection";
import { Writer, Reader } from "protobufjs/minimal";
export declare const protobufPackage = "certusone.wormholechain.tokenbridge";
/** GenesisState defines the tokenbridge module's genesis state. */
export interface GenesisState {
    config: Config | undefined;
    replayProtectionList: ReplayProtection[];
    chainRegistrationList: ChainRegistration[];
    /** this line is used by starport scaffolding # genesis/proto/state */
    coinMetaRollbackProtectionList: CoinMetaRollbackProtection[];
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
