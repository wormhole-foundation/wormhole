import "dotenv/config";
import * as os from "os"
import { SigningCosmWasmClient, toBinary } from "@cosmjs/cosmwasm-stargate";
import { GasPrice } from "@cosmjs/stargate"
import { fromBase64 } from "cosmwasm";
import { Secp256k1HdWallet } from "@cosmjs/amino";

import { zeroPad } from "ethers/lib/utils.js";
import { keccak256 } from "@cosmjs/crypto"

import * as elliptic from "elliptic"
import { concatArrays, encodeUint8 } from "./utils";

import * as devnetConsts from "./devnet-consts.json"


function signBinary(key: elliptic.ec.KeyPair, binary: string): Uint8Array {
    // base64 string to Uint8Array,
    // so we have bytes to work with for signing, though not sure 100% that's correct.
    const bytes = fromBase64(binary);

    // create the "digest" for signing.
    // The contract will calculate the digest of the "data",
    // then use that with the signature to ec recover the publickey that signed.
    const digest = keccak256(keccak256(bytes));

    // sign the digest
    const signature = key.sign(digest, { canonical: true });

    // create 65 byte signature (64 + 1)
    const signedParts = [
        zeroPad(signature.r.toBuffer(), 32),
        zeroPad(signature.s.toBuffer(), 32),
        encodeUint8(signature.recoveryParam || 0),
    ];

    // combine parts to be Uint8Array with length 65
    const signed = concatArrays(signedParts);

    return signed
}


async function main() {

    /* Set up cosmos client & wallet */

    const WORMCHAIN_ID = 3104

    let host = devnetConsts.chains[3104].tendermintUrlLocal
    if (os.hostname().includes("wormchain-deploy")) {
        // running in tilt devnet
        host = devnetConsts.chains[3104].tendermintUrlTilt
    }
    const denom = devnetConsts.chains[WORMCHAIN_ID].addresses.native.denom
    const mnemonic = devnetConsts.chains[WORMCHAIN_ID].accounts.wormchainNodeOfGuardian0.mnemonic
    const addressPrefix = "wormhole"
    const signerPk = devnetConsts.devnetGuardians[0].private
    const accountingAddress = devnetConsts.chains[WORMCHAIN_ID].contracts.accountingNativeAddress

    const w = await Secp256k1HdWallet.fromMnemonic(mnemonic, { prefix: addressPrefix })

    const gas = GasPrice.fromString(`0${denom}`)
    let cwc = await SigningCosmWasmClient.connectWithSigner(host, w, { prefix: addressPrefix, gasPrice: gas })

    // there is no danger here, just several Cosmos chains in devnet, so check for config issues
    let id = await cwc.getChainId()
    if (id !== "wormchain") {
        throw new Error(`Wormchain CosmWasmClient connection produced an unexpected chainID: ${id}`)
    }

    const signers = await w.getAccounts()
    const signer = signers[0].address
    console.log("wormchain wallet pubkey: ", signer)

    const nativeBalance = await cwc.getBalance(signer, denom)
    console.log("nativeBalance ", nativeBalance.amount)

    const utestBalance = await cwc.getBalance(signer, "utest")
    console.log("utest balance ", utestBalance.amount)


    // create key for guardian0
    const ec = new elliptic.ec("secp256k1");
    // create key from the devnet guardian0's private key
    const key = ec.keyFromPrivate(Buffer.from(signerPk, "hex"));


    // Test empty observation

    // object to json string, then to base64 (serde binary)
    const arrayBinaryString = toBinary([]);

    // combine parts to be Uint8Array with length 65
    const signedEmptyArray = signBinary(key, arrayBinaryString)

    const observeEmptyArray = {
        submit_observations: {
            observations: arrayBinaryString,
            guardian_set_index: 0,
            signature: {
                index: 0,
                signature: Array.from(signedEmptyArray),
            },
        },
    };

    let emptyArrayObsRes = await cwc.execute(signer, accountingAddress, observeEmptyArray, "auto");
    console.log(`emptyArrayObsRes.transactionHash: ${emptyArrayObsRes.transactionHash}`);


    // Test (fake) observation
    const emitter_address = "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16"
    const observations = [
        {
            emitter_chain: 2,
            emitter_address: emitter_address,
            sequence: 2,
            nonce: 1,
            consistency_level: 0,
            timestamp: 1,
            payload:
                Buffer.from("030000000000000000000000000000000000000000000000000000000005f5e1000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c2000000000000000000000000000000000000000000000000000000000000f4240", "hex").toString("base64"),

            tx_hash:
                Buffer.from("9fc68fb0ee735d45c9074a20adef1747b0593803f33b9f3f2252c8e2df567f41", "hex").toString("base64")
        },
    ];

    // object to json string, then to base64 (serde binary)
    const observationsBinaryString = toBinary(observations);

    const signed = signBinary(key, observationsBinaryString)

    const executeMsg = {
        submit_observations: {
            observations: observationsBinaryString,
            guardian_set_index: 0,
            signature: {
                index: 0,
                signature: Array.from(signed),
            },
        },
    };
    console.log(executeMsg);

    let inst = await cwc.execute(
        signer,
        accountingAddress,
        executeMsg,
        "auto"
    );
    let txHash = inst.transactionHash;
    console.log(`executed submit_observation! txHash: ${txHash}`);



    console.log("done, exiting success.")
}

try {
    main()
} catch (e) {
    console.error(e)
    throw e
}
