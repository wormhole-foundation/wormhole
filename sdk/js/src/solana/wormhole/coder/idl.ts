// Borrowed from coral-xyz/anchor
//
// https://github.com/coral-xyz/anchor/blob/master/ts/packages/anchor/src/coder/borsh/idl.ts

import * as borsh from "@coral-xyz/borsh";
import { Layout } from "buffer-layout";
import { IdlField, IdlTypeDef } from "../../anchor";
import { camelCase } from "lodash";

export class IdlCoder {
  public static fieldLayout(
    field: { name?: string } & Pick<IdlField, "type">,
    types?: IdlTypeDef[]
  ): Layout {
    const fieldName =
      field.name !== undefined ? camelCase(field.name) : undefined;
    switch (field.type) {
      case "bool": {
        return borsh.bool(fieldName);
      }
      case "u8": {
        return borsh.u8(fieldName);
      }
      case "i8": {
        return borsh.i8(fieldName);
      }
      case "u16": {
        return borsh.u16(fieldName);
      }
      case "i16": {
        return borsh.i16(fieldName);
      }
      case "u32": {
        return borsh.u32(fieldName);
      }
      case "i32": {
        return borsh.i32(fieldName);
      }
      case "f32": {
        return borsh.f32(fieldName);
      }
      case "u64": {
        return borsh.u64(fieldName);
      }
      case "i64": {
        return borsh.i64(fieldName);
      }
      case "f64": {
        return borsh.f64(fieldName);
      }
      case "u128": {
        return borsh.u128(fieldName);
      }
      case "i128": {
        return borsh.i128(fieldName);
      }
      case "u256": {
        return borsh.u256(fieldName);
      }
      case "i256": {
        return borsh.i256(fieldName);
      }
      case "bytes": {
        return borsh.vecU8(fieldName);
      }
      case "string": {
        return borsh.str(fieldName);
      }
      case "publicKey": {
        return borsh.publicKey(fieldName);
      }
      default: {
        if ("vec" in field.type) {
          return borsh.vec(
            IdlCoder.fieldLayout(
              {
                name: undefined,
                type: field.type.vec,
              },
              types
            ),
            fieldName
          );
        } else if ("option" in field.type) {
          return borsh.option(
            IdlCoder.fieldLayout(
              {
                name: undefined,
                type: field.type.option,
              },
              types
            ),
            fieldName
          );
        } else if ("array" in field.type) {
          let arrayTy = field.type.array[0];
          let arrayLen = field.type.array[1];
          let innerLayout = IdlCoder.fieldLayout(
            {
              name: undefined,
              type: arrayTy,
            },
            types
          );
          return borsh.array(innerLayout, arrayLen, fieldName);
        } else {
          throw new Error(`Not yet implemented: ${field}`);
        }
      }
    }
  }
}
