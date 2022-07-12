import { Idl, InstructionCoder } from "@project-serum/anchor";

export class WormholeInstructionCoder implements InstructionCoder {
  constructor(_: Idl) {}

  encode(ixName: string, ix: any): Buffer {
    switch (ixName) {
      case "postVaa": {
        return encodePostVaa(ix);
      }
      case "setFees": {
        return encodeSetFees(ix);
      }
      case "transferFees": {
        return encodeTransferFees(ix);
      }
      case "upgradeContract": {
        return encodeUpgradeContract(ix);
      }
      case "upgradeGuardianSet": {
        return encodeUpgradeGuardianSet(ix);
      }
      case "verifySignatures": {
        return encodeVerifySignatures(ix);
      }
      default: {
        throw new Error(`Invalid instruction: ${ixName}`);
      }
    }
  }

  encodeState(_ixName: string, _ix: any): Buffer {
    throw new Error("Wormhole program does not have state");
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

function encodeWormholeInstructionData(instructionType: WormholeInstruction, data?: Buffer): Buffer {
  const instructionData = Buffer.alloc(1 + (data == undefined ? 0 : data.length));
  instructionData.writeUInt8(instructionType, 0);
  if (data != undefined) {
    instructionData.write(data.toString("hex"), 1, "hex");
  }
  return instructionData;
}

function encodePostVaa({
  version,
  guardianSetIndex,
  timestamp,
  nonce,
  emitterChain,
  emitterAddress,
  sequence,
  consistencyLevel,
  payload,
}: any) {
  if (!Buffer.isBuffer(emitterAddress)) {
    throw Error("emitterAddress must be Buffer");
  }
  if (!Buffer.isBuffer(payload)) {
    throw Error("payload must be Buffer");
  }
  if (typeof sequence != "number") {
    sequence = BigInt(sequence);
  }
  const serialized = Buffer.alloc(60 + payload.length);
  serialized.writeUInt8(version, 0);
  serialized.writeUInt32LE(guardianSetIndex, 1);
  serialized.writeUInt32LE(timestamp, 5);
  serialized.writeUInt32LE(nonce, 9);
  serialized.writeUInt16LE(emitterChain, 13);
  serialized.write(emitterAddress.toString("hex"), 15, "hex");
  serialized.writeBigInt64LE(sequence, 47);
  serialized.writeUInt8(consistencyLevel, 55);
  serialized.writeUInt32LE(payload.length, 56);
  serialized.write(payload.toString("hex"), 60, "hex");
  return encodeWormholeInstructionData(WormholeInstruction.PostVAA, serialized);
}

function encodeSetFees({}: any) {
  return encodeWormholeInstructionData(WormholeInstruction.SetFees);
}

function encodeTransferFees({}: any) {
  return encodeWormholeInstructionData(WormholeInstruction.TransferFees);
}

function encodeUpgradeContract({}: any) {
  return encodeWormholeInstructionData(WormholeInstruction.UpgradeContract);
}

function encodeUpgradeGuardianSet({}: any) {
  return encodeWormholeInstructionData(WormholeInstruction.UpgradeGuardianSet);
}

function encodeVerifySignatures({ signatureStatus }: any) {
  return encodeWormholeInstructionData(WormholeInstruction.VerifySignatures, Buffer.from(signatureStatus, "hex"));
}
