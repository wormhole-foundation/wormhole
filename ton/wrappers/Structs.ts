import { Address, beginCell, Cell, Dictionary } from '@ton/core';
import { GuardianSet, GuardianSetDictionaryValue, Signature, SignatureDictionaryValue } from './Wormhole';
import { randomBytes } from 'crypto';
import { TON_CHAIN_ID } from './Constants';

export type CommentPayload = {
    chainId: number;
    to: Buffer;
    comment: string;
};

export const createEmptyGuardianSet = (): Dictionary<number, GuardianSet> => {
    return Dictionary.empty(Dictionary.Keys.Uint(8), GuardianSetDictionaryValue);
};

export const createEmptySignatures = (): Dictionary<number, Signature> => {
    return Dictionary.empty(Dictionary.Keys.Uint(8), SignatureDictionaryValue);
};

export const randomSignature = (index: number): Signature => {
    return { signature: randomBytes(65), guardianIndex: index };
};

export const generateVAACell = (signaturesCount: number, payload?: Cell) => {
    // Create a test VM that follows the contract's parsing order
    const signaturesDict = createEmptySignatures();
    for (let i = 0; i < signaturesCount; i++) {
        signaturesDict.set(i, randomSignature(i));
    }
    const vmData = beginCell()
        .storeUint(1, 8) // version
        .storeUint(0, 32) // guardianSetIndex
        .storeUint(signaturesDict.size, 8) // signaturesCount
        .storeDict(signaturesDict)
        .storeUint(Math.floor(Date.now() / 1000), 32) // timestamp
        .storeUint(123, 32) // nonce
        .storeUint(TON_CHAIN_ID, 16) // emitterChainId
        .storeUint(0, 256) // emitterAddress
        .storeUint(1, 64) // sequence
        .storeUint(1, 8) // consistencyLevel
        .storeRef(payload || beginCell().storeStringTail('test payload').endCell()) // payload
        .endCell();
    return vmData;
};

export const decodeCommentPayload = (payload: Cell): CommentPayload => {
    const slice = payload.beginParse();
    return { chainId: slice.loadUint(16), to: slice.loadBuffer(32), comment: slice.loadStringRefTail() };
};
