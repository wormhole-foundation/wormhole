import { Idl, Instruction, InstructionCoder } from "@project-serum/anchor";
import bs58 from "bs58";
import { Layout } from "buffer-layout";
import { camelCase, upperFirst } from "lodash";
import { IdlField, IdlStateMethod } from "../../anchor";
import * as borsh from "@coral-xyz/borsh";
import { IdlCoder } from "./idl";

// Inspired by  coral-xyz/anchor
//
// https://github.com/coral-xyz/anchor/blob/master/ts/packages/anchor/src/coder/borsh/instruction.ts
export class WormholeInstructionCoder implements InstructionCoder {
  private ixLayout: Map<string, Layout>;

  constructor(idl: Idl) {
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

    return encodeWormholeInstructionData(
      (WormholeInstruction as any)[upperFirst(methodName)],
      data
    );
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

    let name = camelCase(WormholeInstruction[discriminator]);
    let layout = this.ixLayout.get(name);

    if (!layout) {
      return null;
    }
    return { data: this.ixLayout.get(name)?.decode(data), name };
  }
}

/** Solitaire enum of existing the Core Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/bridge/program/src/lib.rs#L92
 */
export enum WormholeInstruction {
  Initialize,
  PostMessage,
  PostVaa,
  SetFees,
  TransferFees,
  UpgradeContract,
  UpgradeGuardianSet,
  VerifySignatures,
  PostMessageUnreliable, // sounds useful
}

function encodeWormholeInstructionData(
  discriminator: number,
  data?: Buffer
): Buffer {
  const instructionData = Buffer.alloc(
    1 + (data === undefined ? 0 : data.length)
  );
  instructionData.writeUInt8(discriminator, 0);
  if (data !== undefined) {
    instructionData.write(data.toString("hex"), 1, "hex");
  }
  return instructionData;
}
