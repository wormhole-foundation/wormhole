import { test, describe, expect } from "@jest/globals";
import { toUint8Array, hex, b64, b58 } from "../src/utils/encoding";

// A table of base64 encoded strings and their decoded string equivalents
const base64Table = [
    ["", ""],
    ["f", "Zg=="],
    ["fo", "Zm8="],
    ["foo", "Zm9v"],
    ["foob", "Zm9vYg=="],
    ["fooba", "Zm9vYmE="],
    ["foobar", "Zm9vYmFy"]
];

// A table of hex encoded strings and their decoded string equivalents
const hexTable = [
    ["", ""],
    ["f", "66"],
    ["fo", "666f"],
    ["foo", "666f6f"],
    ["foob", "666f6f62"],
    ["fooba", "666f6f6261"],
    ["foobar", "666f6f626172"]
];

// A table of base58 encoded strings and their decoded string equivalents
const base58Table = [
    ["", ""],
    ["f", "2m"],
    ["fo", "8o8"],
    ["foo", "bQbp"],
    ["foob", "3csAg9"],
    ["fooba", "CZJRhmz"],
    ["foobar", "t1Zv2yaZ"]
];


describe("codec Tests", function () {

    describe("base64", function () {
        test.each(base64Table)("encodes properly", function (plain, expected) {
            const actual = b64.encode(plain)
            expect(actual).toEqual(expected)
        });

        test.each(base64Table)("decodes properly", function (expected, encoded) {
            const actual = b64.decode(encoded)
            expect(actual).toEqual(toUint8Array(expected))
        });
    })

    describe("hex", function () {
        test.each(hexTable)("encodes properly", function (plain, expected) {
            const actual = hex.encode(plain)
            expect(actual).toEqual(expected)
        });

        test.each(hexTable)("decodes properly", function (expected, encoded) {
            const actual = hex.decode(encoded)
            expect(actual).toEqual(toUint8Array(expected))
        });
    })


    describe("base58", function () {
        test.each(base58Table)("encodes properly", function (plain, expected) {
            const actual = b58.encode(plain)
            expect(actual).toEqual(expected)
        });

        test.each(base58Table)("decodes properly", function (expected, encoded) {
            const actual = b58.decode(encoded)
            expect(actual).toEqual(toUint8Array(expected))
        });
    })
});

