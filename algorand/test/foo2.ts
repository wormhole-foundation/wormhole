const algosdk = require('algosdk');
const TestLib = require('./testlib.ts')
const testLib = new TestLib.TestLib()
const fs = require('fs');
const path = require('path');

import {
       submitVAA, 
       submitVAAHdr, 
       simpleSignVAA, 
       parseVAA, 
//       Account,
} from "../../sdk/js/src/token_bridge/Algorand";

//const AlgorandLib = require('../../sdk/js/src/token_bridge/Algorand.ts')
//const algorandLib = new AlgorandLib.AlgorandLib()

const guardianKeys = [
    "beFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe"
]
const guardianPrivKeys = [
    "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0"
]

const PYTH_EMITTER = '0x3afda841c1f43dd7d546c8a581ba1f92a139f4133f9f6ab095558f6a359df5d4'
const PYTH_PAYLOAD = '0x50325748000101230abfe0ec3b460bd55fc4fb36356716329915145497202b8eb8bf1af6a0a3b9fe650f0367d4a7ef9815a593ea15d36593f0643aaaf0149bb04be67ab851decd010000002f17254388fffffff70000002eed73d9000000000070d3b43f0000000037faa03d000000000e9e555100000000894af11c0000000037faa03d000000000dda6eb801000000000061a5ff9a'

async function firstTransaction() {
    try {
        // This is a funded account... 
        let myAccount = algosdk.mnemonicToSecretKey("intact frozen tooth wealth syrup elevator list book property census imitate attend draft silly fortune afford injury poem section wait main bench feel absent giraffe")

        console.log(myAccount)

        // Connect your client
        const algodToken = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
        const algodServer = 'http://localhost';
        const algodPort = 4001;
        let algodClient = new algosdk.Algodv2(algodToken, algodServer, algodPort);

        //Check your balance
        let accountInfo = await algodClient.accountInformation(myAccount.addr).do();
        console.log("Account balance: %d microAlgos", accountInfo.amount);

//        let vaa = testLib.genGuardianSetUpgrade(guardianPrivKeys, 0, 1, 1, 1, guardianKeys)
//        console.log(vaa)
//        console.log(parseVAA(new Uint8Array(Buffer.from(vaa, "hex"))))
//        process.exit(0)

        let  vaa = testLib.createSignedVAA(0, guardianPrivKeys, 1, 1, 1, PYTH_EMITTER, 0, 0, PYTH_PAYLOAD)
        console.log(vaa)
        let evaa = new Uint8Array(Buffer.from(vaa, "hex"))

        let sstate = await submitVAAHdr(evaa, algodClient, myAccount, 4);
        console.log(await simpleSignVAA(algodClient, myAccount, sstate.txns));
    }
    catch (err) {
        console.log("err", err);
    }
    process.exit();
};

firstTransaction();
