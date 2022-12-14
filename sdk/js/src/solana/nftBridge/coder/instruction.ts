import { Idl, InstructionCoder } from "@project-serum/anchor";
import { PublicKey } from "@solana/web3.js";

export class NftBridgeInstructionCoder implements InstructionCoder {
  constructor(_: Idl) {}

  encode(ixName: string, ix: any): Buffer {
    switch (ixName) {
      case "initialize": {
        return encodeInitialize(ix);
      }
      case "completeNative": {
        return encodeCompleteNative(ix);
      }
      case "completeWrapped": {
        return encodeCompleteWrapped(ix);
      }
      case "completeWrappedMeta": {
        return encodeCompleteWrappedMeta(ix);
      }
      case "transferWrapped": {
        return encodeTransferWrapped(ix);
      }
      case "transferNative": {
        return encodeTransferNative(ix);
      }
      case "registerChain": {
        return encodeRegisterChain(ix);
      }
      case "upgradeContract": {
        return encodeUpgradeContract(ix);
      }
      default: {
        throw new Error(`Invalid instruction: ${ixName}`);
      }
    }
  }

  encodeState(_ixName: string, _ix: any): Buffer {
    throw new Error("NFT Bridge program does not have state");
  }
}

/** Solitaire enum of existing the NFT Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/modules/nft_bridge/program/src/lib.rs#L74
 */
export enum NftBridgeInstruction {
  Initialize,
  CompleteNative,
  CompleteWrapped,
  CompleteWrappedMeta,
  TransferWrapped,
  TransferNative,
  RegisterChain,
  UpgradeContract,
}

function encodeNftBridgeInstructionData(
  instructionType: NftBridgeInstruction,
  data?: Buffer
): Buffer {
  const dataLen = data === undefined ? 0 : data.length;
  const instructionData = Buffer.alloc(1 + dataLen);
  instructionData.writeUInt8(instructionType, 0);
  if (dataLen > 0) {
    instructionData.write(data!.toString("hex"), 1, "hex");
  }
  return instructionData;
}

function encodeInitialize({ wormhole }: any): Buffer {
  const serialized = Buffer.alloc(32);
  serialized.write(
    new PublicKey(wormhole).toBuffer().toString("hex"),
    0,
    "hex"
  );
  return encodeNftBridgeInstructionData(
    NftBridgeInstruction.Initialize,
    serialized
  );
}

function encodeCompleteNative({}: any) {
  return encodeNftBridgeInstructionData(NftBridgeInstruction.CompleteNative);
}

function encodeCompleteWrapped({}: any) {
  return encodeNftBridgeInstructionData(NftBridgeInstruction.CompleteWrapped);
}

function encodeCompleteWrappedMeta({}: any) {
  return encodeNftBridgeInstructionData(
    NftBridgeInstruction.CompleteWrappedMeta
  );
}

function encodeTransferData({ nonce, targetAddress, targetChain }: any) {
  if (!Buffer.isBuffer(targetAddress)) {
    throw new Error("targetAddress must be Buffer");
  }
  const serialized = Buffer.alloc(38);
  serialized.writeUInt32LE(nonce, 0);
  serialized.write(targetAddress.toString("hex"), 4, "hex");
  serialized.writeUInt16LE(targetChain, 36);
  return serialized;
}

function encodeTransferWrapped({ nonce, targetAddress, targetChain }: any) {
  return encodeNftBridgeInstructionData(
    NftBridgeInstruction.TransferWrapped,
    encodeTransferData({ nonce, targetAddress, targetChain })
  );
}

function encodeTransferNative({ nonce, targetAddress, targetChain }: any) {
  return encodeNftBridgeInstructionData(
    NftBridgeInstruction.TransferNative,
    encodeTransferData({ nonce, targetAddress, targetChain })
  );
}

function encodeRegisterChain({}: any) {
  return encodeNftBridgeInstructionData(NftBridgeInstruction.RegisterChain);
}

function encodeUpgradeContract({}: any) {
  return encodeNftBridgeInstructionData(NftBridgeInstruction.UpgradeContract);
}
