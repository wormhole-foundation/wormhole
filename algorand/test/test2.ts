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
    getIsTransferCompletedAlgorand,
    parseVAA, 
    CORE_ID,
    TOKEN_BRIDGE_ID
} from "@certusone/wormhole-sdk/lib/cjs/algorand/Algorand";

import {
    hexStringToUint8Array,
    uint8ArrayToHexString,
} from "@certusone/wormhole-sdk/lib/cjs/algorand/TmplSig";


import {
    getTempAccounts,
} from "@certusone/wormhole-sdk/lib/cjs/algorand/Helpers";


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

//        let t = "01000000000100bc942f5b6da266078844b26cb01bb541e0b5963da5bae9aadfe717ed5376efa711224796fc9e893dbf6f19ef6472a62f9af9241ece016e42da8a076bbf1ffe3c006250770b625077090001000000000000000000000000000000000000000000000000000000000000000400000000625077092000000000000000000000000000000000000000000000000000000000436f72650200000000000101beFA429d57cD18b7F8A4d91A29AB4AF05d0FBe"
//       console.log(t)
//       console.log(parseVAA(hexStringToUint8Array(t)))
//        process.exit(0)

        console.log("test start");
        let client = getAlgoClient();

        let accounts = await getTempAccounts();
        let player = accounts[0]

        let t = testLib.genAssetMeta(guardianPrivKeys, 0, seq, seq, "4523c3F29447d1f32AEa95BEBD00383c4640F1b4", 1, 8, "USDC", "CircleCoin")
        console.log(t)
        console.log(parseVAA(hexStringToUint8Array(t)))

        await submitVAA(hexStringToUint8Array(t), client, player, TOKEN_BRIDGE_ID)

        process.exit(0)

//        vaaLogs.append(["createWrappedOnAlgorand", attestVAA.hex()])
//        self.submitVAA(attestVAA, client, player, self.tokenid)


        t = testLib.genTransfer(guardianPrivKeys, 1, 1, 1, 1, "4523c3F29447d1f32AEa95BEBD00383c4640F1b4", 2, uint8ArrayToHexString(algosdk.decodeAddress(player.addr).publicKey, false), 8, 0)
        console.log(t)
        console.log(parseVAA(hexStringToUint8Array(t)))

        process.exit(0)

        console.log("seq = ", seq);

        console.log("XXX upgrading the the guardian set using untrusted account...", seq)
        let upgradeVAA = testLib.genGuardianSetUpgrade(guardianPrivKeys, 0, 1, seq, seq, guardianKeys)
        console.log(upgradeVAA)
        console.log(parseVAA(hexStringToUint8Array(upgradeVAA))) 

        let vaa = hexStringToUint8Array(upgradeVAA);
        
        if (await getIsTransferCompletedAlgorand(client, vaa, CORE_ID, player) != false) {
            console.log("assert failed 1");
            process.exit(-1);
        }

        await submitVAA(vaa, client, player, CORE_ID)

        if (await getIsTransferCompletedAlgorand(client, vaa, CORE_ID, player) != true) {
            console.log("assert failed 2");
            process.exit(-1);
        }

        process.exit(0)

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
        let a = testLib.genAssetMeta(guardianPrivKeys, 2, seq, seq, "4523c3F29447d1f32AEa95BEBD00383c4640F1b4", 2, 8, "USDC", "CircleCoin")
        console.log(a)
        await submitVAA(hexStringToUint8Array(a), client, player, TOKEN_BRIDGE_ID)

        seq = seq + 1

        console.log("XXX Transfer the asset ")
        let transferVAA = testLib.genTransfer(guardianPrivKeys, 2, 1, seq, 1, "4523c3F29447d1f32AEa95BEBD00383c4640F1b4", 2, uint8ArrayToHexString(algosdk.decodeAddress(player.addr).publicKey, false), 8, 0)
        await submitVAA(hexStringToUint8Array(transferVAA), client, player, TOKEN_BRIDGE_ID)

        seq = seq + 1

        console.log("test complete");
    }
};

let t = new AlgoTests()
t.runTests()


