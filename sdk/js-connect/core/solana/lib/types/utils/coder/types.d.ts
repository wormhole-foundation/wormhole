/// <reference types="node" />
import { Idl, TypesCoder } from '@project-serum/anchor';
export declare class WormholeTypesCoder implements TypesCoder {
    constructor(_idl: Idl);
    encode<T = any>(_name: string, _type: T): Buffer;
    decode<T = any>(_name: string, _typeData: Buffer): T;
}
