import { Idl, TypesCoder } from "@project-serum/anchor";

export class TokenBridgeTypesCoder implements TypesCoder {
  constructor(_idl: Idl) {}

  encode<T = any>(_name: string, _type: T): Buffer {
    throw new Error("Token Bridge program does not have user-defined types");
  }
  decode<T = any>(_name: string, _typeData: Buffer): T {
    throw new Error("Token Bridge program does not have user-defined types");
  }
}
