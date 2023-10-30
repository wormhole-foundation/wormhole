/// <reference types="node" />
import { Connection, Commitment, PublicKeyInitData } from '@solana/web3.js';
export declare function getSignatureSetData(connection: Connection, signatureSet: PublicKeyInitData, commitment?: Commitment): Promise<SignatureSetData>;
export declare class SignatureSetData {
    signatures: boolean[];
    hash: Buffer;
    guardianSetIndex: number;
    constructor(signatures: boolean[], hash: Buffer, guardianSetIndex: number);
    static deserialize(data: Buffer): SignatureSetData;
}
