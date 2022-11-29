import { Idl, InstructionCoder } from "@project-serum/anchor";
import { PublicKey } from "@solana/web3.js";

export class TokenBridgeInstructionCoder implements InstructionCoder {
  constructor(_: Idl) {}

  encode(ixName: string, ix: any): Buffer {
    switch (ixName) {
      case "initialize": {
        return encodeInitialize(ix);
      }
      case "attestToken": {
        return encodeAttestToken(ix);
      }
      case "completeNative": {
        return encodeCompleteNative(ix);
      }
      case "completeWrapped": {
        return encodeCompleteWrapped(ix);
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
      case "createWrapped": {
        return encodeCreateWrapped(ix);
      }
      case "upgradeContract": {
        return encodeUpgradeContract(ix);
      }
      case "transferWrappedWithPayload": {
        return encodeTransferWrappedWithPayload(ix);
      }
      case "transferNativeWithPayload": {
        return encodeTransferNativeWithPayload(ix);
      }
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
 * https://github.com/certusone/wormhole/blob/main/solana/modules/token_bridge/program/src/lib.rs#L100
 */
export enum TokenBridgeInstruction {
  Initialize,
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
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.Initialize,
    serialized
  );
}

function encodeAttestToken({ nonce }: any) {
  const serialized = Buffer.alloc(4);
  serialized.writeUInt32LE(nonce, 0);
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.AttestToken,
    serialized
  );
}

function encodeCompleteNative({}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.CompleteNative
  );
}

function encodeCompleteWrapped({}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.CompleteWrapped
  );
}

function encodeTransferData({
  nonce,
  amount,
  fee,
  targetAddress,
  targetChain,
}: any) {
  if (typeof amount != "bigint") {
    amount = BigInt(amount);
  }
  if (typeof fee != "bigint") {
    fee = BigInt(fee);
  }
  if (!Buffer.isBuffer(targetAddress)) {
    throw new Error("targetAddress must be Buffer");
  }
  const serialized = Buffer.alloc(54);
  serialized.writeUInt32LE(nonce, 0);
  serialized.writeBigUInt64LE(amount, 4);
  serialized.writeBigUInt64LE(fee, 12);
  serialized.write(targetAddress.toString("hex"), 20, "hex");
  serialized.writeUInt16LE(targetChain, 52);
  return serialized;
}

function encodeTransferWrapped({
  nonce,
  amount,
  fee,
  targetAddress,
  targetChain,
}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.TransferWrapped,
    encodeTransferData({ nonce, amount, fee, targetAddress, targetChain })
  );
}

function encodeTransferNative({
  nonce,
  amount,
  fee,
  targetAddress,
  targetChain,
}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.TransferNative,
    encodeTransferData({ nonce, amount, fee, targetAddress, targetChain })
  );
}

function encodeRegisterChain({}: any) {
  return encodeTokenBridgeInstructionData(TokenBridgeInstruction.RegisterChain);
}

function encodeCreateWrapped({}: any) {
  return encodeTokenBridgeInstructionData(TokenBridgeInstruction.CreateWrapped);
}

function encodeUpgradeContract({}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.UpgradeContract
  );
}

function encodeTransferWithPayloadData({
  nonce,
  amount,
  targetAddress,
  targetChain,
  payload,
}: any) {
  if (typeof amount != "bigint") {
    amount = BigInt(amount);
  }
  if (!Buffer.isBuffer(targetAddress)) {
    throw new Error("targetAddress must be Buffer");
  }
  if (!Buffer.isBuffer(payload)) {
    throw new Error("payload must be Buffer");
  }
  const serializedWithPayloadLen = Buffer.alloc(50);
  serializedWithPayloadLen.writeUInt32LE(nonce, 0);
  serializedWithPayloadLen.writeBigUInt64LE(amount, 4);
  serializedWithPayloadLen.write(targetAddress.toString("hex"), 12, "hex");
  serializedWithPayloadLen.writeUInt16LE(targetChain, 44);
  serializedWithPayloadLen.writeUInt32LE(payload.length, 46);
  return Buffer.concat([
    serializedWithPayloadLen,
    payload,
    Buffer.alloc(1), // option == None
  ]);
}

function encodeTransferWrappedWithPayload({
  nonce,
  amount,
  fee,
  targetAddress,
  targetChain,
  payload,
}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.TransferWrappedWithPayload,
    encodeTransferWithPayloadData({
      nonce,
      amount,
      fee,
      targetAddress,
      targetChain,
      payload,
    })
  );
}

function encodeTransferNativeWithPayload({
  nonce,
  amount,
  fee,
  targetAddress,
  targetChain,
  payload,
}: any) {
  return encodeTokenBridgeInstructionData(
    TokenBridgeInstruction.TransferNativeWithPayload,
    encodeTransferWithPayloadData({
      nonce,
      amount,
      fee,
      targetAddress,
      targetChain,
      payload,
    })
  );
}
