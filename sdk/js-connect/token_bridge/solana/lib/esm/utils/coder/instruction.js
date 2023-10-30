import { PublicKey } from '@solana/web3.js';
export class TokenBridgeInstructionCoder {
    constructor(_) { }
    encode(ixName, ix) {
        switch (ixName) {
            case 'initialize': {
                return encodeInitialize(ix);
            }
            case 'attestToken': {
                return encodeAttestToken(ix);
            }
            case 'completeNative': {
                return encodeCompleteNative(ix);
            }
            case 'completeWrapped': {
                return encodeCompleteWrapped(ix);
            }
            case 'transferWrapped': {
                return encodeTransferWrapped(ix);
            }
            case 'transferNative': {
                return encodeTransferNative(ix);
            }
            case 'registerChain': {
                return encodeRegisterChain(ix);
            }
            case 'createWrapped': {
                return encodeCreateWrapped(ix);
            }
            case 'upgradeContract': {
                return encodeUpgradeContract(ix);
            }
            case 'transferWrappedWithPayload': {
                return encodeTransferWrappedWithPayload(ix);
            }
            case 'transferNativeWithPayload': {
                return encodeTransferNativeWithPayload(ix);
            }
            default: {
                throw new Error(`Invalid instruction: ${ixName}`);
            }
        }
    }
    encodeState(_ixName, _ix) {
        throw new Error('Token Bridge program does not have state');
    }
}
/** Solitaire enum of existing the Token Bridge's instructions.
 *
 * https://github.com/certusone/wormhole/blob/main/solana/modules/token_bridge/program/src/lib.rs#L100
 */
export var TokenBridgeInstruction;
(function (TokenBridgeInstruction) {
    TokenBridgeInstruction[TokenBridgeInstruction["Initialize"] = 0] = "Initialize";
    TokenBridgeInstruction[TokenBridgeInstruction["AttestToken"] = 1] = "AttestToken";
    TokenBridgeInstruction[TokenBridgeInstruction["CompleteNative"] = 2] = "CompleteNative";
    TokenBridgeInstruction[TokenBridgeInstruction["CompleteWrapped"] = 3] = "CompleteWrapped";
    TokenBridgeInstruction[TokenBridgeInstruction["TransferWrapped"] = 4] = "TransferWrapped";
    TokenBridgeInstruction[TokenBridgeInstruction["TransferNative"] = 5] = "TransferNative";
    TokenBridgeInstruction[TokenBridgeInstruction["RegisterChain"] = 6] = "RegisterChain";
    TokenBridgeInstruction[TokenBridgeInstruction["CreateWrapped"] = 7] = "CreateWrapped";
    TokenBridgeInstruction[TokenBridgeInstruction["UpgradeContract"] = 8] = "UpgradeContract";
    TokenBridgeInstruction[TokenBridgeInstruction["CompleteNativeWithPayload"] = 9] = "CompleteNativeWithPayload";
    TokenBridgeInstruction[TokenBridgeInstruction["CompleteWrappedWithPayload"] = 10] = "CompleteWrappedWithPayload";
    TokenBridgeInstruction[TokenBridgeInstruction["TransferWrappedWithPayload"] = 11] = "TransferWrappedWithPayload";
    TokenBridgeInstruction[TokenBridgeInstruction["TransferNativeWithPayload"] = 12] = "TransferNativeWithPayload";
})(TokenBridgeInstruction || (TokenBridgeInstruction = {}));
function encodeTokenBridgeInstructionData(instructionType, data) {
    const dataLen = data === undefined ? 0 : data.length;
    const instructionData = Buffer.alloc(1 + dataLen);
    instructionData.writeUInt8(instructionType, 0);
    if (dataLen > 0) {
        instructionData.write(data.toString('hex'), 1, 'hex');
    }
    return instructionData;
}
function encodeInitialize({ wormhole }) {
    const serialized = Buffer.alloc(32);
    serialized.write(new PublicKey(wormhole).toBuffer().toString('hex'), 0, 'hex');
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.Initialize, serialized);
}
function encodeAttestToken({ nonce }) {
    const serialized = Buffer.alloc(4);
    serialized.writeUInt32LE(nonce, 0);
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.AttestToken, serialized);
}
function encodeCompleteNative({}) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.CompleteNative);
}
function encodeCompleteWrapped({}) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.CompleteWrapped);
}
function encodeTransferData({ nonce, amount, fee, targetAddress, targetChain, }) {
    if (typeof amount != 'bigint') {
        amount = BigInt(amount);
    }
    if (typeof fee != 'bigint') {
        fee = BigInt(fee);
    }
    if (!Buffer.isBuffer(targetAddress)) {
        throw new Error('targetAddress must be Buffer');
    }
    const serialized = Buffer.alloc(54);
    serialized.writeUInt32LE(nonce, 0);
    serialized.writeBigUInt64LE(amount, 4);
    serialized.writeBigUInt64LE(fee, 12);
    serialized.write(targetAddress.toString('hex'), 20, 'hex');
    serialized.writeUInt16LE(targetChain, 52);
    return serialized;
}
function encodeTransferWrapped({ nonce, amount, fee, targetAddress, targetChain, }) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.TransferWrapped, encodeTransferData({ nonce, amount, fee, targetAddress, targetChain }));
}
function encodeTransferNative({ nonce, amount, fee, targetAddress, targetChain, }) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.TransferNative, encodeTransferData({ nonce, amount, fee, targetAddress, targetChain }));
}
function encodeRegisterChain({}) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.RegisterChain);
}
function encodeCreateWrapped({}) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.CreateWrapped);
}
function encodeUpgradeContract({}) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.UpgradeContract);
}
function encodeTransferWithPayloadData({ nonce, amount, targetAddress, targetChain, payload, }) {
    if (typeof amount != 'bigint') {
        amount = BigInt(amount);
    }
    if (!Buffer.isBuffer(targetAddress)) {
        throw new Error('targetAddress must be Buffer');
    }
    if (!Buffer.isBuffer(payload)) {
        throw new Error('payload must be Buffer');
    }
    const serializedWithPayloadLen = Buffer.alloc(50);
    serializedWithPayloadLen.writeUInt32LE(nonce, 0);
    serializedWithPayloadLen.writeBigUInt64LE(amount, 4);
    serializedWithPayloadLen.write(targetAddress.toString('hex'), 12, 'hex');
    serializedWithPayloadLen.writeUInt16LE(targetChain, 44);
    serializedWithPayloadLen.writeUInt32LE(payload.length, 46);
    return Buffer.concat([
        serializedWithPayloadLen,
        payload,
        Buffer.alloc(1), // option == None
    ]);
}
function encodeTransferWrappedWithPayload({ nonce, amount, fee, targetAddress, targetChain, payload, }) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.TransferWrappedWithPayload, encodeTransferWithPayloadData({
        nonce,
        amount,
        fee,
        targetAddress,
        targetChain,
        payload,
    }));
}
function encodeTransferNativeWithPayload({ nonce, amount, fee, targetAddress, targetChain, payload, }) {
    return encodeTokenBridgeInstructionData(TokenBridgeInstruction.TransferNativeWithPayload, encodeTransferWithPayloadData({
        nonce,
        amount,
        fee,
        targetAddress,
        targetChain,
        payload,
    }));
}
