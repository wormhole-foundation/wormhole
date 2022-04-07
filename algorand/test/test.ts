const algosdk = require('algosdk');
const TestLib = require('./testlib.ts')
const testLib = new TestLib.TestLib()
const fs = require('fs');
const path = require('path');

import  {
    getAlgoClient,
    submitVAA, 
    submitVAAHdr, 
    simpleSignVAA, 
    parseVAA, 
    CORE_ID,
    TOKEN_BRIDGE_ID
} from "../../sdk/js/src/token_bridge/Algorand";

import {
    hexStringToUint8Array,
    uint8ArrayToHexString,
} from "../../sdk/js/src/token_bridge/TmplSig";


import {
    getTempAccounts,
} from "../../sdk/js/src/token_bridge/Helpers";


const guardianKeys = [
    "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
]
const guardianPrivKeys = [
    "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"
]


class AlgoTests {
    constructor() {
    }

    async runTests() {
        let seq = Math.floor(new Date().getTime() / 1000.0);

//        let t = testLib.genAssetMeta(guardianPrivKeys, 2, seq, seq, "4523c3F29447d1f32AEa95BEBD00383c4640F1b4", 1, 8, "USDC", "CircleCoin")
//        console.log(t)
//        console.log(parseVAA(hexStringToUint8Array(t)))
//        process.exit(0)

        console.log("test start");
        let client = getAlgoClient();

        let accounts = await getTempAccounts();
        let player = accounts[0]



        console.log("seq = ", seq);

        console.log("XXX upgrading the the guardian set using untrusted account...", seq)
        let upgradeVAA = testLib.genGuardianSetUpgrade(guardianPrivKeys, 0, 1, seq, seq, guardianKeys)
        console.log(upgradeVAA)
        await submitVAA(hexStringToUint8Array(upgradeVAA), client, player, CORE_ID)

        seq = seq + 1

        console.log("XXX upgrading again...", seq)
        upgradeVAA = testLib.genGuardianSetUpgrade(guardianPrivKeys, 1, 2, seq, seq, guardianKeys)
        console.log(upgradeVAA)
        await submitVAA(hexStringToUint8Array(upgradeVAA), client, player, CORE_ID)

        seq = seq + 1

        console.log("XXX registering chain 2", seq)
        let reg = testLib.genRegisterChain(guardianPrivKeys, 2, 1, seq, 2)
        console.log(reg)
        await submitVAA(hexStringToUint8Array(reg), client, player, TOKEN_BRIDGE_ID)

        seq = seq + 1

        console.log("XXX gen asset meta", seq)
        let a = testLib.genAssetMeta(guardianPrivKeys, 2, seq, seq, "4523c3F29447d1f32AEa95BEBD00383c4640F1b4", 1, 8, "USDC", "CircleCoin")
        console.log(a)
        await submitVAA(hexStringToUint8Array(a), client, player, TOKEN_BRIDGE_ID)

        seq = seq + 1

        console.log("test complete");
    }
};

let t = new AlgoTests()
t.runTests()


