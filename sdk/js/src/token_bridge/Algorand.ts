// Algorand.ts

import algosdk, {
    assignGroupID,
    computeGroupID,
    decodeAddress,
    getApplicationAddress,
    LogicSigAccount,
    makeApplicationCallTxnFromObject,
    makeApplicationOptInTxnFromObject,
    makeAssetCreateTxnWithSuggestedParamsFromObject,
    makePaymentTxnWithSuggestedParams,
    makePaymentTxnWithSuggestedParamsFromObject,
    OnApplicationComplete,
    signLogicSigTransaction,
    Transaction,
} from "algosdk";
import account from "algosdk/dist/types/src/account";
import AlgodClient from "algosdk/dist/types/src/client/v2/algod/algod";
import SuggestedParamsRequest from "algosdk/dist/types/src/client/v2/algod/suggestedParams";
import { sensitiveHeaders } from "http2";
import internal from "stream";

export const ALGO_TOKEN =
    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
export const KMD_ADDRESS: string = "http://localhost";
export const KMD_PORT: number = 4002;
export const KMD_WALLET_NAME: string = "unencrypted-default-wallet";
export const KMD_WALLET_PASSWORD: string = "";
export const ALGOD_ADDRESS: string = "http://localhost";
export const ALGOD_PORT: number = 4001;
export const CORE_ID: number = 4;
export const TOKEN_BRIDGE_ID: number = 6;
export const SEED_AMT: number = 1002000;

// Generated Testnet wallet
export const TESTNET_ACCOUNT_ADDRESS =
    "RWVYXYLSV32QIHFUMBEBW4BQZR7FDVJGKTVZIVYECMQWU7CZUAK5Q4WMP4";
export const TESTNET_ACCOUNT_MN =
    "enforce sail meat library retreat rain praise run floor drastic flat end true olympic boy dune dust regular feed allow top universe borrow able ginger";

export function getKmdClient(): algosdk.Kmd {
    const kmdClient: algosdk.Kmd = new algosdk.Kmd(
        ALGO_TOKEN,
        KMD_ADDRESS,
        KMD_PORT
    );
    return kmdClient;
}

export function getAlgoClient(): algosdk.Algodv2 {
    const algodClient = new algosdk.Algodv2(
        ALGO_TOKEN,
        ALGOD_ADDRESS,
        ALGOD_PORT
    );
    return algodClient;
}

export class Account {
    pk: Buffer;
    addr: string;
    mn: string;

    constructor(address: string, privateKey: Buffer) {
        this.pk = privateKey;
        this.addr = address;
        this.mn = algosdk.secretKeyToMnemonic(privateKey);
    }

    getPrivateKey(): Buffer {
        return this.pk;
    }
    getAddress(): string {
        return this.addr;
    }
    getMnemonic(): string {
        return this.mn;
    }
}

export async function getBalances(
    account: string
): Promise<Map<number, number>> {
    let balances = new Map<number, number>();
    const aClient = getAlgoClient();
    const accountInfo = await aClient.accountInformation(account).do();
    console.log("Account Info:", accountInfo);
    console.log("Account Info|created-assets:", accountInfo["created-assets"]);

    // Put the algo balance in key 0
    balances.set(0, accountInfo.amount);

    const assets: Array<any> = accountInfo.assets;
    console.log("assets", assets);
    assets.forEach(function (asset) {
        console.log("inside foreach", asset);
        const assetId = asset["asset-id"];
        const amount = asset.amount;
        balances.set(assetId, amount);
    });
    return balances;
}


export async function attestFromAlgorand(
    senderAcct: Account,
    assetId: number,
    appId: number
) {
    const algodClient = getAlgoClient();
    const appAddr: string = getApplicationAddress(appId);
    const acctInfo = await algodClient
        .accountInformation(senderAcct.getAddress())
        .do();
    const creatorAddr = acctInfo["created-assets"]["creator"];
    const creatorAcctInfo = await algodClient
        .accountInformation(creatorAddr)
        .do();
    const wormhole: boolean = creatorAcctInfo["auth-addr"] === appAddr;
    if (!wormhole) {
        // TODO:   creator = self.optin(client, sender, self.tokenid, asset_id, b"native".hex())
    }
    const params: algosdk.SuggestedParams = await algodClient
        .getTransactionParams()
        .do();
    const PgmName: string = "attestToken";
    const encoder: TextEncoder = new TextEncoder();
    const bPgmName: Uint8Array = encoder.encode(PgmName);
    const emitterAddr = await optin(senderAcct, bPgmName);
    const appTxn = makeApplicationCallTxnFromObject({
        appArgs: [bPgmName, numberToUint8Array(assetId)],
        accounts: [creatorAddr, creatorAcctInfo["address"]], // TODO: add emitterAddr
        appIndex: TOKEN_BRIDGE_ID,
        foreignApps: [CORE_ID],
        foreignAssets: [assetId],
        from: senderAcct.getAddress(),
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
    });
    const rawSignedTxn = appTxn.signTxn(senderAcct.getPrivateKey());
    console.log("rawSignedTxn:", rawSignedTxn);
    const tx = await algodClient.sendRawTransaction(rawSignedTxn).do();
    // wait for transaction to be confirmed
    const ptx = await algosdk.waitForConfirmation(algodClient, tx.txId, 4);

    // TODO: return something?
}

export async function optin(sender: Account, pgmName: Uint8Array) {
    const algodClient = getAlgoClient();
    const appAddr: string = getApplicationAddress(CORE_ID);
    const decodedAddr = decodeAddress(appAddr);
    const params = await algodClient.getTransactionParams().do();
    const seedTxn = makePaymentTxnWithSuggestedParamsFromObject({
        from: sender.getAddress(),
        to: appAddr,
        amount: SEED_AMT,
        suggestedParams: params,
    });
    const optinTxn = makeApplicationOptInTxnFromObject({
        from: sender.getAddress(),
        suggestedParams: params,
        appIndex: CORE_ID,
    });
    const rekeyTxn = makePaymentTxnWithSuggestedParamsFromObject({
        from: sender.getAddress(),
        to: sender.getAddress(),
        amount: 0,
        suggestedParams: params,
        rekeyTo: appAddr,
    });

    assignGroupID([seedTxn, optinTxn, rekeyTxn]);

    const logicSigAcct: LogicSigAccount = getLogicSigAccount(pgmName);
    const signedSeedTxn = seedTxn.signTxn(sender.getPrivateKey());
    const signedOptinTxn = signLogicSigTransaction(optinTxn, logicSigAcct);
    const signedRekeyTxn = signLogicSigTransaction(rekeyTxn, logicSigAcct);

    const txnId = await algodClient
        .sendRawTransaction([
            signedSeedTxn,
            signedOptinTxn.blob,
            signedRekeyTxn.blob,
        ])
        .do();
    const confirmedTxns = await algosdk.waitForConfirmation(
        algodClient,
        txnId,
        4
    );
}

export function getLogicSigAccount(program: Uint8Array): LogicSigAccount {
    const lsa = new LogicSigAccount(program);
    return lsa;
}

export function numberToUint8Array(n: number) {
    if (!n) return new Uint8Array(0);
    const a = [];
    a.unshift(n & 255);
    while (n >= 256) {
        n = n >>> 8;
        a.unshift(n & 255);
    }
    return new Uint8Array(a);
}

export function parseVAA(vaa: Uint8Array) {
    let ret = new Map<string, any>();
    let buf = Buffer.from(vaa);
    ret.set("version", buf.readIntBE(0, 1));
    ret.set("index", buf.readIntBE(1, 4));
    ret.set("siglen", buf.readIntBE(5, 1));
    const siglen = ret.get("siglen");
    if (siglen) {
        ret.set("signatures", buf.readIntBE(6, siglen * 66));
    }
    let sigs = [];
    for (let i = 0; i < siglen; i++) {
        // TODO:  finish figuring this out.
        const start = 6 + i * 66;
        const len = 66;
        const sigBuf = Buffer.from(vaa, start, len);
        sigs.push(sigBuf);
        // ret["sigs"].append(vaa[(6 + (i * 66)):(6 + (i * 66)) + 66].hex())
    }
    ret.set("sigs", sigs);
    let off = siglen * 66 + 6;
    ret.set("digest", Buffer.from(vaa, off)); // This is what is actually signed...
    ret.set("timestamp", buf.readIntBE(off, 4));
    off += 4;
    ret.set("nonce", buf.readIntBE(off, 4));
    off += 4;
    ret.set("chainRaw", Buffer.from(vaa, off, 2));
    ret.set("chain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("emitter", Buffer.from(vaa, off, 32));
    off += 32;
    ret.set("sequence", buf.readIntBE(off, 8));
    off += 8;
    ret.set("consistency", buf.readIntBE(off, 1));
    off += 1;

    ret.set("Meta", "Unknown");

    if (
        Buffer.from(vaa, off, 32) ===
        Buffer.from(
            "000000000000000000000000000000000000000000546f6b656e427269646765"
        )
    ) {
        ret.set("Meta", "TokenBridge");
        ret.set("module", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("action", buf.readIntBE(off, 1));
        off += 1;
        if (ret.get("action") === 1) {
            ret.set("Meta", "TokenBridge RegisterChain");
            ret.set("targetChain", buf.readIntBE(off, 2));
            off += 2;
            ret.set("EmitterChainID", buf.readIntBE(off, 2));
            off += 2;
            ret.set("targetEmitter", Buffer.from(vaa, off, 32));
            off += 32;
        } else if (ret.get("action") === 2) {
            ret.set("Meta", "TokenBridge UpgradeContract");
            ret.set("targetChain", buf.readIntBE(off, 2));
            off += 2;
            ret.set("newContract", Buffer.from(vaa, off, 32));
            off += 32;
        }
    }

    if (
        Buffer.from(vaa, off, 32) ===
        Buffer.from(
            "00000000000000000000000000000000000000000000000000000000436f7265"
        )
    ) {
        ret.set("Meta", "CoreGovernance");
        ret.set("module", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("action", buf.readIntBE(off, 1));
        off += 1;
        ret.set("targetChain", buf.readIntBE(off, 2));
        off += 2;
        ret.set("NewGuardianSetIndex", buf.readIntBE(off, 4));
    }
    if (Buffer.from(vaa, off).length === 100 && buf.readIntBE(off, 1) === 2) {
        ret.set("Meta", "TokenBridge Attest");
        ret.set("Type", buf.readIntBE(off, 1));
        off += 1;
        ret.set("Contract", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("FromChain", buf.readIntBE(off, 2));
        off += 2;
        ret.set("Decimals", buf.readIntBE(off, 1));
        off += 1;
        ret.set("Symbol", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("Name", Buffer.from(vaa, off, 32));
    }

    if (Buffer.from(vaa, off).length === 133 && buf.readIntBE(off, 1) === 1) {
        ret.set("Meta", "TokenBridge Transfer");
        ret.set("Type", buf.readIntBE(off, 1));
        off += 1;
        ret.set("Amount", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("Contract", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("FromChain", buf.readIntBE(off, 2));
        off += 2;
        ret.set("ToAddress", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("ToChain", buf.readIntBE(off, 2));
        off += 2;
        ret.set("Fee", Buffer.from(vaa, off, 32));
    }

    if (buf.readIntBE(off, 1) === 3) {
        ret.set("Meta", "TokenBridge Transfer With Payload");
        ret.set("Type", buf.readIntBE(off, 1));
        off += 1;
        ret.set("Amount", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("Contract", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("FromChain", buf.readIntBE(off, 2));
        off += 2;
        ret.set("ToAddress", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("ToChain", buf.readIntBE(off, 2));
        off += 2;
        ret.set("Fee", Buffer.from(vaa, off, 32));
        off += 32;
        ret.set("Payload", Buffer.from(vaa, off));
    }

    return ret;
}
