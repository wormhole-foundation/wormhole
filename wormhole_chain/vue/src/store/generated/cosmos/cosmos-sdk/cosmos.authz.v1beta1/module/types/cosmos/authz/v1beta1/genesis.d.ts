import { Any } from '../../../google/protobuf/any';
import { Writer, Reader } from 'protobufjs/minimal';
export declare const protobufPackage = "cosmos.authz.v1beta1";
/** GenesisState defines the authz module's genesis state. */
export interface GenesisState {
    authorization: GrantAuthorization[];
}
/** GrantAuthorization defines the GenesisState/GrantAuthorization type. */
export interface GrantAuthorization {
    granter: string;
    grantee: string;
    authorization: Any | undefined;
    expiration: Date | undefined;
}
export declare const GenesisState: {
    encode(message: GenesisState, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GenesisState;
    fromJSON(object: any): GenesisState;
    toJSON(message: GenesisState): unknown;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
export declare const GrantAuthorization: {
    encode(message: GrantAuthorization, writer?: Writer): Writer;
    decode(input: Reader | Uint8Array, length?: number): GrantAuthorization;
    fromJSON(object: any): GrantAuthorization;
    toJSON(message: GrantAuthorization): unknown;
    fromPartial(object: DeepPartial<GrantAuthorization>): GrantAuthorization;
};
declare type Builtin = Date | Function | Uint8Array | string | number | undefined;
export declare type DeepPartial<T> = T extends Builtin ? T : T extends Array<infer U> ? Array<DeepPartial<U>> : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>> : T extends {} ? {
    [K in keyof T]?: DeepPartial<T[K]>;
} : Partial<T>;
export {};
