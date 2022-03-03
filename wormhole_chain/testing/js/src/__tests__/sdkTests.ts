
import { describe, jest, test, expect } from "@jest/globals";
import { TEST_WALLET_MNEMONIC_1 } from "../consts";
import { hexToNativeAddress, nativeToHexAddress } from "../core/utils";
import { getAddress, getWallet } from "../core/walletHelpers";

jest.setTimeout(60000);

describe("SDK tests", () => {
    test("Address manipulation", (done) => {
        const nativeTestAddress = "wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq";

        console.log("nativeTestAddress", nativeTestAddress)
        const hexFormat : string = nativeToHexAddress(nativeTestAddress);
        console.log("Hex", hexFormat)
        const nativeFormat2 = hexToNativeAddress(hexFormat);
        console.log("nativeFormat2", nativeFormat2)

        console.log("hex format length", hexFormat.length)

        expect(
            hexFormat.length === 64
            ).toBe(true);
        expect(nativeFormat2 === nativeTestAddress).toBe(true);

        done();
    });
    test("Wallet instantiation", (done) => {
        (async () => {
            const wallet = await getWallet(TEST_WALLET_MNEMONIC_1);
            const address = await getAddress(wallet);
            console.log("wallet address", address);

            expect(address === "wormhole1cyyzpxplxdzkeea7kwsydadg87357qna3zg3tq").toBe(true);

            done();
        })();
    });
});