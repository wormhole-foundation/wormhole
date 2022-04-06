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
    CORE_ID
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
        let client = getAlgoClient();

        let accounts = await getTempAccounts();
        let player = accounts[0]

        let seq = 2

        console.log("upgrading the the guardian set using untrusted account...")
        let upgradeVAA = testLib.genGuardianSetUpgrade(guardianPrivKeys, 0, 1, seq, seq, guardianKeys)
        console.log(upgradeVAA)
        submitVAA(hexStringToUint8Array(upgradeVAA), client, player, CORE_ID)
    }
};

let t = new AlgoTests()
t.runTests()


