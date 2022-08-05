import { Idl, InstructionCoder } from "@project-serum/anchor";

export class TokenBridgeInstructionCoder implements InstructionCoder {
  constructor(_: Idl) {}

  encode(ixName: string, ix: any): Buffer {
    switch (ixName) {
      default: {
        throw new Error(`Invalid instruction: ${ixName}`);
      }
    }
  }

  encodeState(_ixName: string, _ix: any): Buffer {
    throw new Error("Token Bridge program does not have state");
  }
}

/** Solitaire enum of existing the Token Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/dev.v2/solana/modules/token_bridge/program/src/lib.rs#L100
 */
export enum TokenBridgeInstruction {
  AttestToken,
  CompleteNative,
  CompleteWrapped,
  TransferWrapped,
  TransferNative,
  RegisterChain,
  CreateWrapped,
  UpgradeContract,
  CompleteNativeWithPayload,
  CompleteWrappedWithPayload,
  TransferWrappedWithPayload,
  TransferNativeWithPayload,
}

function encodeTokenBridgeInstructionData(
  instructionType: TokenBridgeInstruction,
  data?: Buffer
): Buffer {
  const instructionData = Buffer.alloc(
    1 + (data == undefined ? 0 : data.length)
  );
  instructionData.writeUInt8(instructionType, 0);
  if (data != undefined) {
    instructionData.write(data.toString("hex"), 1, "hex");
  }
  return instructionData;
}
