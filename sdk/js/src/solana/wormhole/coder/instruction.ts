import { Idl, Instruction, InstructionCoder } from "@project-serum/anchor";
import { Layout } from "buffer-layout";
import { camelCase } from "lodash";
import { IdlField, IdlStateMethod, IdlTypeDef } from "../../anchor";
import * as borsh from "@coral-xyz/borsh"
import { bs58 } from "@project-serum/anchor/dist/cjs/utils/bytes";
import { toPascalCase } from "@injectivelabs/sdk-ts";

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
  
export class WormholeInstructionCoder implements InstructionCoder {

  private ixLayout : Map<string, Layout>;

  constructor(idl : Idl) {
    this.ixLayout = WormholeInstructionCoder.parseIxLayout(idl);
  }

  private static parseIxLayout(idl: Idl): Map<string, Layout> {
    const stateMethods = idl.state ? idl.state.methods : [];

    const ixLayouts = stateMethods
      .map((m: IdlStateMethod): [string, Layout<unknown>] => {
        let fieldLayouts = m.args.map((arg: IdlField) => {
          return IdlCoder.fieldLayout(
            arg,
            Array.from([...(idl.accounts ?? []), ...(idl.types ?? [])])
          );
        });
        const name = camelCase(m.name);
        return [name, borsh.struct(fieldLayouts, name)];
      })
      .concat(
        idl.instructions.map((ix) => {
          let fieldLayouts = ix.args.map((arg: IdlField) =>
            IdlCoder.fieldLayout(
              arg,
              Array.from([...(idl.accounts ?? []), ...(idl.types ?? [])])
            )
          );
          const name = camelCase(ix.name);
          return [name, borsh.struct(fieldLayouts, name)];
        })
      );
    return new Map(ixLayouts);
  }

  encode(ixName: string, ix: any): Buffer {
    const buffer = Buffer.alloc(1000); // TODO: use a tighter buffer.
    const methodName = camelCase(ixName);
    const layout = this.ixLayout.get(methodName);
    if (!layout) {
      throw new Error(`Unknown method: ${methodName}`);
    }
    const len = layout.encode(ix, buffer);
    const data = buffer.slice(0, len);

    let discriminator = toPascalCase(ixName);
    
        switch (ixName) {
      case "initialize": {
        return encodeWormholeInstructionData(WormholeInstruction.Initialize, data);
      }
      case "postMessage": {
        return encodeWormholeInstructionData(WormholeInstruction.PostMessage, data);
      }
      case "postVaa": {
        return encodeWormholeInstructionData(WormholeInstruction.PostVAA, data);
      }
      case "setFees": {
        return encodeWormholeInstructionData(WormholeInstruction.SetFees, data);
      }
      case "transferFees": {
        return encodeWormholeInstructionData(WormholeInstruction.TransferFees, data);
      }
      case "upgradeContract": {
        return encodeWormholeInstructionData(WormholeInstruction.UpgradeContract, data);
      }
      case "upgradeGuardianSet": {
        return encodeWormholeInstructionData(WormholeInstruction.UpgradeGuardianSet, data);
      }
      case "verifySignatures": {
        return encodeWormholeInstructionData(WormholeInstruction.VerifySignatures, data);
      }
      default: {
        throw new Error(`Invalid instruction: ${ixName}`);
      }
    }

  }


  encodeState(_ixName: string, _ix: any): Buffer {
    throw new Error("Wormhole program does not have state");
  }

  public decode(
    ix: Buffer | string,
    encoding: "hex" | "base58" = "hex"
  ): Instruction | null {
    if (typeof ix === "string") {
      ix = encoding === "hex" ? Buffer.from(ix, "hex") : bs58.decode(ix);
    }
    let discriminator = ix.slice(0, 1).readInt8();
    let data = ix.slice(1);

    let name = camelCase(WormholeInstruction[discriminator])
    let layout = this.ixLayout.get(name)
    
    if (!layout) {
      return null
    }
    return { data: 
        this.ixLayout.get(name)?.decode(data),
      name }
    }
}

/** Solitaire enum of existing the Core Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/dev.v2/solana/bridge/program/src/lib.rs#L92
 */
export enum WormholeInstruction {
  Initialize,
  PostMessage,
  PostVAA,
  SetFees,
  TransferFees,
  UpgradeContract,
  UpgradeGuardianSet,
  VerifySignatures,
  PostMessageUnreliable, // sounds useful
}

function encodeWormholeInstructionData(
  instructionType: WormholeInstruction,
  data?: Buffer
): Buffer {
  const instructionData = Buffer.alloc(
    1 + (data === undefined ? 0 : data.length)
  );
  instructionData.writeUInt8(instructionType, 0);
  if (data !== undefined) {
    instructionData.write(data.toString("hex"), 1, "hex");
  }
  return instructionData;
}
