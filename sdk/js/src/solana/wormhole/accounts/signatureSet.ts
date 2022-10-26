import {
  Connection,
  PublicKey,
  Commitment,
  PublicKeyInitData,
} from "@solana/web3.js";
import { getAccountData } from "../../utils";

export async function getSignatureSetData(
  connection: Connection,
  signatureSet: PublicKeyInitData,
  commitment?: Commitment
): Promise<SignatureSetData> {
  return connection
    .getAccountInfo(new PublicKey(signatureSet), commitment)
    .then((info) => SignatureSetData.deserialize(getAccountData(info)));
}

export class SignatureSetData {
  signatures: boolean[];
  hash: Buffer;
  guardianSetIndex: number;

  constructor(signatures: boolean[], hash: Buffer, guardianSetIndex: number) {
    this.signatures = signatures;
    this.hash = hash;
    this.guardianSetIndex = guardianSetIndex;
  }

  static deserialize(data: Buffer): SignatureSetData {
    const numSignatures = data.readUInt32LE(0);
    const signatures = [...data.subarray(4, 4 + numSignatures)].map(
      (x) => x != 0
    );
    const hashIndex = 4 + numSignatures;
    const hash = data.subarray(hashIndex, hashIndex + 32);
    const guardianSetIndex = data.readUInt32LE(hashIndex + 32);
    return new SignatureSetData(signatures, hash, guardianSetIndex);
  }
}
