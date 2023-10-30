var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { PublicKey, } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function getSignatureSetData(connection, signatureSet, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return connection
            .getAccountInfo(new PublicKey(signatureSet), commitment)
            .then((info) => SignatureSetData.deserialize(utils.getAccountData(info)));
    });
}
export class SignatureSetData {
    constructor(signatures, hash, guardianSetIndex) {
        this.signatures = signatures;
        this.hash = hash;
        this.guardianSetIndex = guardianSetIndex;
    }
    static deserialize(data) {
        const numSignatures = data.readUInt32LE(0);
        const signatures = [...data.subarray(4, 4 + numSignatures)].map((x) => x != 0);
        const hashIndex = 4 + numSignatures;
        const hash = data.subarray(hashIndex, hashIndex + 32);
        const guardianSetIndex = data.readUInt32LE(hashIndex + 32);
        return new SignatureSetData(signatures, hash, guardianSetIndex);
    }
}
