import * as tinysecp from 'tiny-secp256k1';
import { ECPairFactory } from 'ecpair';
import { randomBytes } from 'crypto';

const ECPair = ECPairFactory(tinysecp);

export const makeRandomId = (bits: number): number => {
    return Math.floor(Math.random() * (2 ** bits - 1));
}

export const makeRandomKeyPair = () => {
    const privateKey = randomBytes(32);
    const keyPair = ECPair.fromPrivateKey(privateKey);
    return {
        privateKey,
        keyPair,
    };
}

export const toXOnly = (publicKey: Buffer) => {
    return publicKey.length === 33 ? publicKey.subarray(1) : publicKey;
}