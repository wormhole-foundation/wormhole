import * as tinysecp from 'tiny-secp256k1';
import { ECPairFactory, ECPairInterface } from 'ecpair';
import { randomBytes } from 'crypto';
import { Slice, Transaction } from '@ton/ton';
import { findTransactionRequired, FlatTransactionComparable } from '@ton/test-utils';

export type KeyPair = {
    privateKey: Buffer;
    keyPair: ECPairInterface;
};

const ECPair = ECPairFactory(tinysecp);

export class Time {
    static oneHourSeconds = 3600;

    static now = (offsetSeconds?: number): number => {
        return Math.floor(Date.now() / 1000) + (offsetSeconds ?? 0);
    };
    static hours = (hours: number): number => {
        return Time.now(Time.oneHourSeconds * hours);
    };
}

export class Random {
    static id = (bits: number): number => {
        return Math.floor(Math.random() * (2 ** bits - 1));
    };
}

export class Crypto {
    static makeRandomKeyPair = () => {
        const privateKey = randomBytes(32);
        const keyPair = ECPair.fromPrivateKey(privateKey);
        return {
            privateKey,
            keyPair,
        };
    };

    static toXOnly = (publicKey: Buffer) => {
        return publicKey.length === 33 ? publicKey.subarray(1) : publicKey;
    };

    static makeRandomKeyPairs = (count: number): KeyPair[] => {
        return Array.from({ length: count }, () => Crypto.makeRandomKeyPair());
    };

    static mapKeyPairsToXOnlyPublicKeys = (keyPairs: KeyPair[]): Buffer[] => {
        return keyPairs.map((keyPair) => Crypto.toXOnly(keyPair.keyPair.publicKey as Buffer));
    };
}

export class Event {
    static mustFindEvent = (transactions: Transaction[], match: FlatTransactionComparable, eventId: number): Slice => {
        const tx = findTransactionRequired(transactions, match);
        const event = tx.outMessages.values().find((msg) => msg.info.type === 'external-out');
        expect(event).toBeDefined();
        const eventBody = event!.body.beginParse();
        expect(eventBody.loadUint(32)).toBe(eventId);
        return eventBody;
    };
}