import {
    MockTokenBridge,
    MockGuardians,
} from "@certusone/wormhole-sdk/lib/cjs/mock";
import { BN, web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import { ethers } from "ethers";
import { GUARDIAN_KEYS } from "./tests/helpers";


describe("Core Bridge: Legacy Verify Signatures and Post VAA", () => {

    const guardians = new MockGuardians(0, GUARDIAN_KEYS);

    const foreignEmitter = new MockTokenBridge(
        "95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491",
        1,
        32,
        0
    );

    it("wtf", () => {
        // TODO
        const published = foreignEmitter.publishAttestMeta("000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", 18, "WETH", "Wrapped ether", 2095245887);
        published.writeBigUInt64BE(BigInt("11833801757748136510"), 42);
        published.writeUInt16BE(2, 51 + 33)
        const signedVaa = guardians.addSignatures(published, [0]);

        console.log("yay", signedVaa.toString("hex"))
    });
});