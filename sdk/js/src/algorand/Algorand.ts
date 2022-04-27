// Algorand.ts

import algosdk, {
  Account,
  Algodv2,
  assignGroupID,
  bigIntToBytes,
  decodeAddress,
  encodeAddress,
  getApplicationAddress,
  LogicSigAccount,
  makeApplicationCallTxnFromObject,
  makeApplicationOptInTxnFromObject,
  makeAssetTransferTxnWithSuggestedParamsFromObject,
  makePaymentTxnWithSuggestedParamsFromObject,
  OnApplicationComplete,
  signLogicSigTransaction,
  Transaction,
  waitForConfirmation,
} from "algosdk";
import { keccak256 } from "ethers/lib/utils";
import { WormholeWrappedInfo } from "../token_bridge";
import { ChainId } from "../utils";
import {
  hexStringToUint8Array,
  PopulateData,
  TmplSig,
  uint8ArrayToHexString,
} from "./TmplSig";

export let ALGO_TOKEN =
  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
export let ALGOD_ADDRESS: string = "http://localhost";
export let ALGOD_PORT: number = 4001;
export let CORE_ID: number = 4;
export let TOKEN_BRIDGE_ID: number = 6;

export const SEED_AMT: number = 1002000;
const ZERO_PAD_BYTES =
  "0000000000000000000000000000000000000000000000000000000000000000";
const MAX_KEYS: number = 15;
const MAX_BYTES_PER_KEY: number = 127;
const BITS_PER_BYTE: number = 8;

const BITS_PER_KEY: number = MAX_BYTES_PER_KEY * BITS_PER_BYTE;
const MAX_BYTES: number = MAX_BYTES_PER_KEY * MAX_KEYS;
const MAX_BITS: number = BITS_PER_BYTE * MAX_BYTES;
// const COST_PER_VERIF: number = 1000;
const MAX_SIGS_PER_TXN: number = 9;
// const INDEXER_TOKEN: string =
//     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
// const INDEXER_ADDRESS: string = "http://localhost";
// const INDEXER_PORT: number = 8980;

const ALGO_VERIFY_HASH =
  "EZATROXX2HISIRZDRGXW4LRQ46Z6IUJYYIHU3PJGP7P5IQDPKVX42N767A";
const ALGO_VERIFY = new Uint8Array([
  6, 32, 4, 1, 0, 32, 20, 38, 1, 0, 49, 32, 50, 3, 18, 68, 49, 1, 35, 18, 68,
  49, 16, 129, 6, 18, 68, 54, 26, 1, 54, 26, 3, 54, 26, 2, 136, 0, 3, 68, 34,
  67, 53, 2, 53, 1, 53, 0, 40, 53, 240, 40, 53, 241, 52, 0, 21, 53, 5, 35, 53,
  3, 35, 53, 4, 52, 3, 52, 5, 12, 65, 0, 68, 52, 1, 52, 0, 52, 3, 129, 65, 8,
  34, 88, 23, 52, 0, 52, 3, 34, 8, 36, 88, 52, 0, 52, 3, 129, 33, 8, 36, 88, 7,
  0, 53, 241, 53, 240, 52, 2, 52, 4, 37, 88, 52, 240, 52, 241, 80, 2, 87, 12,
  20, 18, 68, 52, 3, 129, 66, 8, 53, 3, 52, 4, 37, 8, 53, 4, 66, 255, 180, 34,
  137,
]);

let accountExistsCache = new Set();

export type Signer = {
  addr: string;
  signTxn(txn: Transaction): Promise<Uint8Array>;
};

export const AccountToSigner = (account: Account): Signer => ({
  addr: account.addr,
  signTxn: (txn) => Promise.resolve(txn.signTxn(account.sk)),
});

/**
 * <p> Creates a new Algodv2 client using local file consts</p>
 * @returns a newly constructed Algodv2 client
 */
export function getAlgoClient(): Algodv2 {
  const algodClient = new Algodv2(ALGO_TOKEN, ALGOD_ADDRESS, ALGOD_PORT);
  return algodClient;
}

export type TealCompileRsp = {
  hash: string; // base32 SHA512_256 of program bytes (Address style)
  result: string; // base64 encoded program bytes
};

// Conversion functions

export function textToHexString(name: string): string {
  return Buffer.from(name, "binary").toString("hex");
}

export function textToUint8Array(name: string): Uint8Array {
  return new Uint8Array(Buffer.from(name, "binary"));
}

export function appIdToAppAddr(appId: number): string {
  const appAddr: string = getApplicationAddress(appId);
  const decAppAddr: Uint8Array = decodeAddress(appAddr).publicKey;
  const aa: string = uint8ArrayToHexString(decAppAddr, false);
  return aa;
}

/**
 * <p> Return the balances of all assets for the supplied account</p>
 * @param client An Algodv2 client
 * @param account The account containing assets
 * @returns Map of asset index to qty
 */
export async function getBalances(
  client: Algodv2,
  account: string
): Promise<Map<number, number>> {
  let balances = new Map<number, number>();
  const accountInfo = await client.accountInformation(account).do();
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
  console.log("balances", balances);
  return balances;
}

/**
 * <p>Return the balance of the supplied asset index for the supplied account</p>
 * @param client An Algodv2 client
 * @param account The account to query for the supplied asset index
 * @param key The asset index
 * @returns The quantity of the asset in the supplied account
 */
export async function getBalance(
  client: Algodv2,
  account: string,
  key: number
): Promise<number> {
  let balances = new Map<number, number>();
  const accountInfo = await client.accountInformation(account).do();

  if (key == 0) {
    return accountInfo.amount;
  }

  let ret = 0;
  const assets: Array<any> = accountInfo.assets;
  assets.forEach((asset) => {
    if (key === asset["asset-id"]) {
      ret = asset.amount;
      return;
    }
  });
  return ret;
}

/**
 * <p>Return the message fee for the core bridge</p>
 * @returns The message fee for the core bridge
 */
export async function getMessageFee(client: Algodv2): Promise<number> {
  const applInfo: Record<string, any> = await client
    .getApplicationByID(CORE_ID)
    .do();
  const globalState = applInfo["params"]["global-state"];
  console.log("globalState:", globalState);
  const key: string = Buffer.from("MessageFee", "binary").toString("base64");
  console.log("key", key);
  let ret = -1;
  globalState.forEach((el: any) => {
    if (el["key"] === key) {
      ret = el["value"]["uint"];
      return;
    }
  });
  return ret;
}

/**
 * <p>Parses the logs of a transaction looking for the sequence number</p>
 * @param txn The transaction containing a sequence number
 * @returns The sequence number found in the supplied transaction
 */
export function parseSeqFromTxn(txn: any): bigint {
  const innerTxns = txn.innerTxns[txn.innerTxns - 1];
  const logs = innerTxns["logs"];
  const seqNum = logs[0];
  const bufSN = Buffer.from(seqNum, "base64");
  const sn = bufSN.readBigUInt64BE();
  return sn;
}

export function parseSeqFromLog(log: Buffer[]): bigint {
  console.log("parseSeqFromLog input:", log);
  console.log("parseSeqFromLog input:", log[0]);
  // const bufSN = Buffer.from(log[-1], "base64");
  const sn = log[0].readBigUInt64BE();
  return sn;
}

/**
 * <p>Attest an already created asset</p>
 * If you create a new asset on algorand and want to transfer it elsewhere,
 * you create an attestation for it on algorand... pass that vaa to the target chain..
 * submit it.. then you can transfer from algorand to that target chain
 * @param client An Algodv2 client
 * @param senderAcct The account paying fees
 * @param assetId The asset index
 * @returns Transaction ID
 */
export async function attestFromAlgorand(
  client: Algodv2,
  senderAcct: Signer,
  assetId: number
): Promise<BigInt> {
  console.log("senderAcct:", senderAcct, "assetId:", assetId);
  const tbAddr: string = getApplicationAddress(TOKEN_BRIDGE_ID);
  const decTbAddr: Uint8Array = decodeAddress(tbAddr).publicKey;
  const aa: string = uint8ArrayToHexString(decTbAddr, false);
  console.log("Getting emitter address...");
  const emitterAddr: string = await optin(
    client,
    senderAcct,
    CORE_ID,
    0,
    aa,
    "attestFromAlgorand::emitterAddr"
  );
  console.log("Got emitter address...", emitterAddr);
  let wormhole: boolean = false;
  let creatorAcctInfo: any = false;
  let creatorAddr = "";
  const bPgmName: Uint8Array = textToUint8Array("attestToken");

  if (assetId != 0) {
    const acctInfo = await client.accountInformation(senderAcct.addr).do();
    console.log("Got sender account info...", acctInfo);
    const assetKey: string = "index: " + assetId.toString();
    // console.log(
    //     assetKey,
    //     "assetKey value:",
    //     acctInfo["created-assets"][assetKey]
    // );
    const createdAssets = acctInfo["created-assets"];
    console.log("createdAssets:", createdAssets);
    class ca {
      index: number = 0;
      params: any;
    }
    // let creatorAddr = acctInfo["created-assets"]["creator"];
    createdAssets.forEach((a: ca) => {
      if (a.index === assetId) {
        // console.log("found asset index", a.index);
        // console.log(a);
        creatorAddr = a.params.creator;
        return;
      }
    });
    console.log("creatorAddr:", creatorAddr);
    creatorAcctInfo = await client.accountInformation(creatorAddr).do();
    // console.log("Got creator account info...");
    wormhole = creatorAcctInfo["auth-addr"] === tbAddr;
  } else {
    wormhole = false;
  }

  if (!wormhole) {
    console.log("Not wormhole.  Need to optin...");
    creatorAddr = await optin(
      client,
      senderAcct,
      TOKEN_BRIDGE_ID,
      assetId,
      textToHexString("native"),
      "notWormhole"
    );
  }
  const suggParams: algosdk.SuggestedParams = await client
    .getTransactionParams()
    .do();

  let txns: algosdk.Transaction[] = [];

  const firstTxn = makeApplicationCallTxnFromObject({
    from: senderAcct.addr,
    appIndex: TOKEN_BRIDGE_ID,
    onComplete: OnApplicationComplete.NoOpOC,
    appArgs: [textToUint8Array("nop")],
    suggestedParams: suggParams,
  });
  txns.push(firstTxn);

  const mfee = await getMessageFee(client);
  if (mfee > 0) {
    console.log("Need to add payment txn...");
    const feeTxn = makePaymentTxnWithSuggestedParamsFromObject({
      from: senderAcct.addr,
      suggestedParams: suggParams,
      to: getApplicationAddress(TOKEN_BRIDGE_ID),
      amount: mfee,
    });
    txns.push(feeTxn);
  }

  console.log("make app call txn...");

  let accts: string[] = [
    emitterAddr,
    creatorAddr,
    getApplicationAddress(CORE_ID),
  ];

  if (creatorAcctInfo) {
    accts.push(creatorAcctInfo["address"]);
  }

  let appTxn = makeApplicationCallTxnFromObject({
    appArgs: [bPgmName, bigIntToBytes(assetId, 8)],
    accounts: accts,
    appIndex: TOKEN_BRIDGE_ID,
    foreignApps: [CORE_ID],
    foreignAssets: [assetId],
    from: senderAcct.addr,
    onComplete: OnApplicationComplete.NoOpOC,
    suggestedParams: suggParams,
  });
  if (mfee > 0) {
    appTxn.fee *= 3;
  } else {
    appTxn.fee *= 2;
  }
  txns.push(appTxn);

  const resp: Buffer[] = await simpleSignVAA(client, senderAcct, txns);
  console.log("resp:", resp);
  let seq: bigint = BigInt(0);
  try {
    seq = parseSeqFromLog(resp);
  } catch (pErr) {
    console.error("parseSeqFromTxn Failed:", pErr);
  }
  console.log("attestFromAlgorand seq:", seq);

  return seq;
  // return txns[txns.length - 1].txID();
}

/**
 * <p>Checks to see it the account exists for the application</p>
 * @param client An Algodv2 client
 * @param appId Application ID
 * @param acctAddr Account address to check
 * @returns true, if account exists for application.  Otherwise, returns false
 */
export async function accountExists(
  client: Algodv2,
  appId: number,
  acctAddr: string
): Promise<boolean> {
  if (accountExistsCache.has([appId, acctAddr])) return true;

  let ret = false;
  try {
    const acctInfo = await client.accountInformation(acctAddr).do();
    console.log("appId", appId, "acctAddr", acctAddr, "acctInfo:", acctInfo);
    const als: Record<string, any>[] = acctInfo["apps-local-state"];
    if (!als) {
      return ret;
    }
    als.forEach((app) => {
      console.log("Inside for loop");

      if (app["id"] === appId) {
        accountExistsCache.add([appId, acctAddr]);
        ret = true;
        return;
      }
    });
  } catch (e) {
    console.error("Failed to check for account existence:", e);
  }

  console.log("returning", ret);
  return ret;
}

export type LogicSigAccountInfo = {
  lsa: LogicSigAccount;
  doesExist: boolean;
};

export async function calcLogicSigAccount(
  client: algosdk.Algodv2,
  appId: number,
  appIndex: number,
  emitterId: string
): Promise<LogicSigAccountInfo> {
  let data: PopulateData = {
    addrIdx: appIndex,
    appAddress: appIdToAppAddr(appId),
    appId: appId,
    emitterId: emitterId,
    seedAmt: SEED_AMT,
  };

  const ts: TmplSig = new TmplSig(client);
  const lsa: LogicSigAccount = await ts.populate(data);
  const sigAddr: string = lsa.address();

  const doesExist: boolean = await accountExists(client, appId, sigAddr);
  return {
    lsa,
    doesExist,
  };
}

export async function optin(
  client: algosdk.Algodv2,
  sender: Signer,
  appId: number,
  appIndex: number,
  emitterId: string,
  why: string
): Promise<string> {
  console.log("optin called with ", appIndex, emitterId, why);
  const appAddr: string = getApplicationAddress(appId);

  // Check to see if we need to create this
  const { doesExist, lsa } = await calcLogicSigAccount(
    client,
    appId,
    appIndex,
    emitterId
  );
  const sigAddr: string = lsa.address();
  if (!doesExist) {
    // console.log("Account does not exist.");
    // These are the suggested params from the system
    // console.log("Getting parms...");
    const params = await client.getTransactionParams().do();
    // console.log("Creating payment txn...");
    const seedTxn = makePaymentTxnWithSuggestedParamsFromObject({
      from: sender.addr,
      to: sigAddr,
      amount: SEED_AMT,
      suggestedParams: params,
    });
    // console.log("Creating optin txn...");
    const optinTxn = makeApplicationOptInTxnFromObject({
      from: sigAddr,
      suggestedParams: params,
      appIndex: appId,
    });
    // console.log("Creating rekey txn...");
    const rekeyTxn = makePaymentTxnWithSuggestedParamsFromObject({
      from: sigAddr,
      to: sigAddr,
      amount: 0,
      suggestedParams: params,
      rekeyTo: appAddr,
    });

    // console.log("Assigning group ID...");
    let txns = [seedTxn, optinTxn, rekeyTxn];
    assignGroupID(txns);

    // console.log("Signing seed for optin...");
    const signedSeedTxn = await sender.signTxn(seedTxn);
    // console.log("Signing optin for optin...");
    const signedOptinTxn = signLogicSigTransaction(optinTxn, lsa);
    // console.log("Signing rekey for optin...");
    const signedRekeyTxn = signLogicSigTransaction(rekeyTxn, lsa);

    // console.log(
    //     "Sending txns for optin...",
    //     appIndex,
    //     "-",
    //     emitterId,
    //     "-",
    //     sigAddr
    // );
    const txnId = await client
      .sendRawTransaction([
        signedSeedTxn,
        signedOptinTxn.blob,
        signedRekeyTxn.blob,
      ])
      .do();

    // console.log(
    //     "Awaiting confirmation for optin...",
    //     appIndex,
    //     "-",
    //     emitterId
    // );
    const confirmedTxns = await algosdk.waitForConfirmation(
      client,
      txns[txns.length - 1].txID(),
      1
    );
    console.log("optin confirmation", confirmedTxns);

    accountExistsCache.add([appId, lsa.address()]);
  }
  return sigAddr;
}

export function getLogicSigAccount(program: Uint8Array): LogicSigAccount {
  const lsa = new LogicSigAccount(program);
  return lsa;
}

function extract3(buffer: any, start: number, size: number) {
  return buffer.slice(start, start + size);
}

/**
 * <p>Parses the VAA into a Map</p>
 * @param vaa The VAA to be parsed
 * @returns The Map<string, any> containing the parsed elements of the VAA
 */
export function parseVAA(vaa: Uint8Array): Map<string, any> {
  let ret = new Map<string, any>();
  let buf = Buffer.from(vaa);
  ret.set("version", buf.readIntBE(0, 1));
  ret.set("index", buf.readIntBE(1, 4));
  ret.set("siglen", buf.readIntBE(5, 1));
  const siglen = ret.get("siglen");
  if (siglen) {
    ret.set("signatures", extract3(vaa, 6, siglen * 66));
  }
  const sigs = [];
  for (let i = 0; i < siglen; i++) {
    const start = 6 + i * 66;
    const len = 66;
    const sigBuf = extract3(vaa, start, len);
    sigs.push(sigBuf);
  }
  ret.set("sigs", sigs);
  let off = siglen * 66 + 6;
  ret.set("digest", vaa.slice(off)); // This is what is actually signed...
  ret.set("timestamp", buf.readIntBE(off, 4));
  off += 4;
  ret.set("nonce", buf.readIntBE(off, 4));
  off += 4;
  ret.set("chainRaw", Buffer.from(extract3(vaa, off, 2)).toString("hex"));
  ret.set("chain", buf.readIntBE(off, 2));
  off += 2;
  ret.set("emitter", Buffer.from(extract3(vaa, off, 32)).toString("hex"));
  off += 32;
  ret.set("sequence", buf.readBigUInt64BE(off));
  off += 8;
  ret.set("consistency", buf.readIntBE(off, 1));
  off += 1;

  ret.set("Meta", "Unknown");

  if (
    !Buffer.compare(
      extract3(buf, off, 32),
      Buffer.from(
        "000000000000000000000000000000000000000000546f6b656e427269646765",
        "hex"
      )
    )
  ) {
    ret.set("Meta", "TokenBridge");
    ret.set("module", extract3(vaa, off, 32));
    off += 32;
    ret.set("action", buf.readIntBE(off, 1));
    off += 1;
    if (ret.get("action") === 1) {
      ret.set("Meta", "TokenBridge RegisterChain");
      ret.set("targetChain", buf.readIntBE(off, 2));
      off += 2;
      ret.set("EmitterChainID", buf.readIntBE(off, 2));
      off += 2;
      ret.set("targetEmitter", extract3(vaa, off, 32));
      off += 32;
    } else if (ret.get("action") === 2) {
      ret.set("Meta", "TokenBridge UpgradeContract");
      ret.set("targetChain", buf.readIntBE(off, 2));
      off += 2;
      ret.set("newContract", extract3(vaa, off, 32));
      off += 32;
    }
  } else if (
    !Buffer.compare(
      extract3(buf, off, 32),
      Buffer.from(
        "00000000000000000000000000000000000000000000000000000000436f7265",
        "hex"
      )
    )
  ) {
    ret.set("Meta", "CoreGovernance");
    ret.set("module", extract3(vaa, off, 32));
    off += 32;
    ret.set("action", buf.readIntBE(off, 1));
    off += 1;
    ret.set("targetChain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("NewGuardianSetIndex", buf.readIntBE(off, 4));
  }

  //    ret.set("len", vaa.slice(off).length)
  //    ret.set("act", buf.readIntBE(off, 1))

  if (vaa.slice(off).length === 100 && buf.readIntBE(off, 1) === 2) {
    ret.set("Meta", "TokenBridge Attest");
    ret.set("Type", buf.readIntBE(off, 1));
    off += 1;
    ret.set("Contract", uint8ArrayToHexString(extract3(vaa, off, 32), false));
    off += 32;
    ret.set("FromChain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("Decimals", buf.readIntBE(off, 1));
    off += 1;
    ret.set("Symbol", extract3(vaa, off, 32));
    off += 32;
    ret.set("Name", extract3(vaa, off, 32));
  }

  if (vaa.slice(off).length === 133 && buf.readIntBE(off, 1) === 1) {
    ret.set("Meta", "TokenBridge Transfer");
    ret.set("Type", buf.readIntBE(off, 1));
    off += 1;
    ret.set("Amount", extract3(vaa, off, 32));
    off += 32;
    ret.set("Contract", uint8ArrayToHexString(extract3(vaa, off, 32), false));
    off += 32;
    ret.set("FromChain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("ToAddress", extract3(vaa, off, 32));
    off += 32;
    ret.set("ToChain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("Fee", extract3(vaa, off, 32));
  }

  if (off >= buf.length) {
    console.log("No payload.");
    return ret;
  }
  if (buf.readIntBE(off, 1) === 3) {
    ret.set("Meta", "TokenBridge Transfer With Payload");
    ret.set("Type", buf.readIntBE(off, 1));
    off += 1;
    ret.set("Amount", extract3(vaa, off, 32));
    off += 32;
    ret.set("Contract", uint8ArrayToHexString(extract3(vaa, off, 32), false));
    off += 32;
    ret.set("FromChain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("ToAddress", extract3(vaa, off, 32));
    off += 32;
    ret.set("ToChain", buf.readIntBE(off, 2));
    off += 2;
    ret.set("Fee", extract3(vaa, off, 32));
    off += 32;
    ret.set("Payload", vaa.slice(off));
    ret.set("appid", buf.readIntBE(off, 8));
    ret.set("body", uint8ArrayToHexString(vaa.slice(off + 8), false));
  }

  return ret;
}

export async function decodeLocalState(
  client: Algodv2,
  appId: number,
  address: string
): Promise<Uint8Array> {
  let app_state = null;
  const ai = await client.accountInformation(address).do();
  for (const app of ai["apps-local-state"]) {
    if (app["id"] === appId) {
      app_state = app["key-value"];
      break;
    }
  }

  // console.log("decodeLocalState", JSON.stringify(app_state));

  let ret = Buffer.alloc(0);
  let empty = Buffer.alloc(0);
  if (app_state) {
    const e = Buffer.alloc(127);
    const m = Buffer.from("meta");

    let sk: string[] = [];
    let vals: Map<string, Buffer> = new Map<string, Buffer>();
    for (const kv of app_state) {
      const k = Buffer.from(kv["key"], "base64");
      const key: number = k.readInt8();
      if (!Buffer.compare(k, m)) {
        continue;
      }
      const v: Buffer = Buffer.from(kv["value"]["bytes"], "base64");
      if (Buffer.compare(v, e)) {
        vals.set(key.toString(), v);
        sk.push(key.toString());
      }
    }

    sk.sort((a, b) => a.localeCompare(b, "en", { numeric: true }));

    sk.forEach((v) => {
      ret = Buffer.concat([ret, vals.get(v) || empty]);
    });
  }
  return new Uint8Array(ret);
}

export async function assetOptinCheck(
  client: Algodv2,
  asset: number,
  receiver: string
): Promise<boolean> {
  const acctInfo = await client.accountInformation(receiver).do();
  const assets: Array<any> = acctInfo.assets;
  let ret = false;
  assets.forEach((a) => {
    const assetId = a["asset-id"];
    if (assetId === asset) {
      ret = true;
      return;
    }
  });
  return ret;
}

export async function assetOptin(
  client: Algodv2,
  sender: Signer,
  asset: number,
  receiver: string
) {
  const params: algosdk.SuggestedParams = await client
    .getTransactionParams()
    .do();

  // Create transaction
  const optinTxn: Transaction =
    makeAssetTransferTxnWithSuggestedParamsFromObject({
      amount: 0,
      assetIndex: asset,
      from: sender.addr,
      suggestedParams: params,
      to: receiver,
    });

  // Sign transaction
  const signedOptinTxn: Uint8Array = await sender.signTxn(optinTxn);

  // Send transaction
  const txId: string = await client.sendRawTransaction(signedOptinTxn).do();

  // Wait for response
  const confirmedTxn = await waitForConfirmation(client, optinTxn.txID(), 1);

  console.log("assetOptin confirmation:", asset, receiver, confirmedTxn);

  // Double check the result
  if (!(await assetOptinCheck(client, asset, receiver))) {
    throw new Error("assetOptin() failed ");
  }
}

class SubmitVAAState {
  vaaMap: Map<string, any>;
  accounts: string[];
  txns: Array<algosdk.Transaction>;
  guardianAddr: string;

  constructor(
    vaaMap: Map<string, any>,
    accounts: string[],
    txns: Array<algosdk.Transaction>,
    guardianAddr: string
  ) {
    this.vaaMap = vaaMap;
    this.accounts = accounts;
    this.txns = txns;
    this.guardianAddr = guardianAddr;
  }
}

/**
 * <p>Submits just the header of the VAA</p>
 * @param vaa The VAA (Just the header is used)
 * @param client AlgodV2 client
 * @param sender Sending account
 * @param appid Application ID
 * @returns Current VAA state
 */
export async function submitVAAHdr(
  vaa: Uint8Array,
  client: Algodv2,
  sender: Signer,
  appid: number
): Promise<SubmitVAAState> {
  // A lot of our logic here depends on parseVAA and knowing what the payload is..
  const parsedVAA: Map<string, any> = parseVAA(vaa);
  console.log(Number(parsedVAA.get("sequence")));
  const seq: number = Math.floor(Number(parsedVAA.get("sequence")) / MAX_BITS);
  const chainRaw: string = parsedVAA.get("chainRaw"); // TODO: this needs to be a hex string
  const em: string = parsedVAA.get("emitter"); // TODO: this needs to be a hex string
  const index: number = parsedVAA.get("index");
  console.log(parsedVAA);

  const seqAddr: string = await optin(
    client,
    sender,
    appid,
    seq,
    chainRaw + em,
    "seqAddr"
  );
  const guardianPgmName = textToHexString("guardian");
  // And then the signatures to help us verify the vaa_s
  const guardianAddr: string = await optin(
    client,
    sender,
    CORE_ID,
    index,
    guardianPgmName,
    "guardianAddr"
  );

  let accts: string[] = [seqAddr, guardianAddr];

  // When we attest for a new token, we need some place to store the info... later we will need to
  // mirror the other way as well
  const keys: Uint8Array = await decodeLocalState(
    client,
    CORE_ID,
    guardianAddr
  );

  const params: algosdk.SuggestedParams = await client
    .getTransactionParams()
    .do();
  let txns: Array<algosdk.Transaction> = [];

  // We don't pass the entire payload in but instead just pass it pre digested.  This gets around size
  // limitations with lsigs AND reduces the cost of the entire operation on a conjested network by reducing the
  // bytes passed into the transaction
  // This is a 2 pass digest
  const digest = keccak256(keccak256(parsedVAA.get("digest"))).slice(2);

  //    const data = parsedVAA.get("digest")
  //    process.stdout.write("srcDigest" + `${inspect(data, { maxArrayLength: 1000 })}\n`)
  //    console.log("digest", digest);
  //    console.log("adigest", hexStringToUint8Array(digest));

  // How many signatures can we process in a single txn... we can do 9!
  // There are likely upwards of 19 signatures.  So, we ned to split things up
  const numSigs: number = parsedVAA.get("siglen");
  let numTxns: number = Math.floor(numSigs / MAX_SIGS_PER_TXN) + 1;

  const SIG_LEN: number = 66;
  const BSIZE: number = SIG_LEN * MAX_SIGS_PER_TXN;
  const signatures: Uint8Array = parsedVAA.get("signatures");
  const verifySigArg: Uint8Array = textToUint8Array("verifySigs");
  for (let nt = 0; nt < numTxns; nt++) {
    let sigs: Uint8Array = signatures.slice(nt * BSIZE);
    if (sigs.length > BSIZE) {
      sigs = sigs.slice(0, BSIZE);
    }

    // The keyset is the set of guardians that correspond
    // to the current set of signatures in this loop.
    // Each signature in 20 bytes and comes from decodeLocalState()
    const GuardianKeyLen: number = 20;
    const numSigsThisTxn = sigs.length / SIG_LEN;
    let arraySize: number = numSigsThisTxn * GuardianKeyLen;
    let keySet: Uint8Array = new Uint8Array(arraySize);
    for (let i = 0; i < numSigsThisTxn; i++) {
      // The first byte of the sig is the relative index of that signature in the signatures array
      // Use that index to get the appropriate guardian key
      const idx = sigs[i * SIG_LEN];
      const key = keys.slice(
        idx * GuardianKeyLen + 1,
        (idx + 1) * GuardianKeyLen + 1
      );
      keySet.set(key, i * 20);
    }

    console.log("keySet:", keySet);

    const appTxn = makeApplicationCallTxnFromObject({
      appArgs: [verifySigArg, sigs, keySet, hexStringToUint8Array(digest)],
      accounts: accts,
      appIndex: CORE_ID,
      from: ALGO_VERIFY_HASH,
      onComplete: OnApplicationComplete.NoOpOC,
      suggestedParams: params,
    });
    appTxn.fee = 0;
    txns.push(appTxn);
  }
  const appTxn = makeApplicationCallTxnFromObject({
    appArgs: [textToUint8Array("verifyVAA"), vaa],
    accounts: accts,
    appIndex: CORE_ID,
    from: sender.addr,
    onComplete: OnApplicationComplete.NoOpOC,
    suggestedParams: params,
  });
  appTxn.fee = appTxn.fee * (1 + numTxns);
  txns.push(appTxn);

  return new SubmitVAAState(parsedVAA, accts, txns, guardianAddr);
}

/**
 * <p>Groups the transactions together, signs them, sends them, waits for a confirmation</p>
 * @param client AlgodV2 client
 * @param sender Sending account
 * @param txns One or more transactions to send
 * @returns Confirmation log(s)
 */
export async function simpleSignVAA(
  client: Algodv2,
  sender: Signer,
  txns: Array<algosdk.Transaction>
): Promise<Buffer[]> {
  console.log("simpleSignVAA");
  //    console.log(txns)
  assignGroupID(txns);
  const signedTxnsPromises: Promise<Uint8Array>[] = [];
  txns.forEach((txn) => {
    // console.log(txn);
    if (
      txn.appArgs &&
      txn.appArgs?.length > 0 &&
      JSON.stringify(txn.appArgs[0]) ===
        JSON.stringify(textToUint8Array("verifySigs"))
    ) {
      console.log("Signing logic sig...");
      const lsa = new LogicSigAccount(ALGO_VERIFY);
      const stxn = signLogicSigTransaction(txn, lsa);
      signedTxnsPromises.push(Promise.resolve(stxn.blob));
    } else {
      console.log("Signing normal txn...");
      signedTxnsPromises.push(sender.signTxn(txn));
    }
  });
  const signedTxns = await Promise.all(signedTxnsPromises);

  console.log("sendRawTransaction", signedTxns);
  const resp = await client.sendRawTransaction(signedTxns).do();

  // console.log("waiting for confirmation", txns[txns.length - 1].txID());
  // const response = await waitForConfirmation(
  //     client,
  //     txns[txns.length - 1].txID(),
  //     1
  // );
  let ret: Buffer[] = [];
  console.log("waitForConfirmation on", txns.length, "transactions...");
  let response: Record<string, any>;
  for (let i: number = 0; i < txns.length; i++) {
    response = await waitForConfirmation(client, txns[i].txID(), 1);
    console.log("txn loop resp", response);
    if (response["inner-txns"]) {
      console.log("Got inner txns.  Looking for logs...");
      const innerTxns: [] = response["inner-txns"];
      console.log("innerTxns:", innerTxns);
      class iTxn {
        "local-state-delta": [[Object]];
        logs: Buffer[] | undefined;
        "pool-eror": string;
        txn: { txn: [Object] } | undefined;
      }
      innerTxns.forEach((txn: iTxn) => {
        console.log("txn:", txn.logs);
        if (txn.logs) {
          txn.logs.forEach((l) => {
            ret.push(l);
          });
          const seq = parseSeqFromLog(txn.logs);
          console.log("sequence:", seq);
        }
      });
    }
  }
  console.log(".. done");

  // console.log("submitVAA confirmation", response);
  // const innerTxns: [] = response["inner-txns"];
  // console.log("innerTxns:", innerTxns);
  // class iTxn {
  //     "local-state-delta": [[Object]];
  //     logs: Buffer[] | undefined;
  //     "pool-eror": string;
  //     txn: { txn: [Object] } | undefined;
  // }
  // innerTxns.forEach((txn: iTxn) => {
  //     console.log("txn:", txn.logs);
  //     if (txn.logs) {
  //         const seq = parseSeqFromLog(txn.logs);
  //         console.log("sequence:", seq);
  //     }
  // });
  // if (response["inner-txns"]["logs"]) {
  //     ret = response["inner-txns"]["logs"];
  //     console.log("submitVAA logs:", ret);
  // }
  ret.forEach((log) => {
    console.log("logs:", new TextDecoder().decode(Buffer.from(log)));
  });

  return ret;
}

export async function getForeignAssetAlgo(
  client: Algodv2,
  chain: number,
  contract: string
): Promise<number | null> {
  if (chain == 8) {
    return parseInt(contract, 16);
  } else {
    let { lsa, doesExist } = await calcLogicSigAccount(
      client,
      TOKEN_BRIDGE_ID,
      chain,
      contract
    );
    if (!doesExist) {
      return null;
    }
    let asset: Uint8Array = await decodeLocalState(
      client,
      TOKEN_BRIDGE_ID,
      lsa.address()
    );
    if (asset.length > 8) {
      const tmp = Buffer.from(asset.slice(0, 8));
      return Number(tmp.readBigUInt64BE(0));
    } else return null;
  }
}

export async function getForeignAssetFromVaaAlgo(
  client: Algodv2,
  vaa: Uint8Array
): Promise<number | null> {
  const parsedVAA: Map<string, any> = parseVAA(vaa);
  return await getForeignAssetAlgo(
    client,
    parsedVAA.get("FromChain"),
    parsedVAA.get("Contract")
  );
}

/**
 * </p>Submits the VAA to the application
 * @param vaa The VAA to be submitted
 * @param client AlgodV2 client
 * @param sender Sending account
 * @param appid Application ID
 * @returns Confirmation log
 */
export async function submitVAA(
  vaa: Uint8Array,
  client: Algodv2,
  sender: Signer,
  appid: number
): Promise<Buffer[]> {
  let sstate = await submitVAAHdr(vaa, client, sender, appid);

  let parsedVAA = sstate.vaaMap;
  console.log("submitVAA:", parsedVAA);
  let accts = sstate.accounts;
  let txns = sstate.txns;

  // If this happens to be setting up a new guardian set, we probably need it as well...
  if (
    parsedVAA.get("Meta") === "CoreGovernance" &&
    parsedVAA.get("action") === 2
  ) {
    const ngsi = parsedVAA.get("NewGuardianSetIndex");
    const guardianPgmName = textToHexString("guardian");
    const newGuardianAddr = await optin(
      client,
      sender,
      CORE_ID,
      ngsi,
      guardianPgmName,
      "newGuardianAddr"
    );
    accts.push(newGuardianAddr);
  }

  // When we attest for a new token, we need some place to store the info... later we will need to
  // mirror the other way as well
  const meta = parsedVAA.get("Meta");
  let chainAddr: string = "";
  if (
    meta === "TokenBridge Attest" ||
    meta === "TokenBridge Transfer" ||
    meta === "TokenBridge Transfer With Payload"
  ) {
    if (parsedVAA.get("FromChain") != 8) {
      chainAddr = await optin(
        client,
        sender,
        TOKEN_BRIDGE_ID,
        parsedVAA.get("FromChain"),
        parsedVAA.get("Contract"),
        "TokenBridge chainAddr"
      );
    } else {
      console.log("nativeAssetOptin::parseVAA", parsedVAA);
      console.log(parsedVAA.get("Contract"));
      const contract: string = parsedVAA.get("Contract");
      const assetId = parseInt(contract, 16);
      console.log("nativeAssetOptin::assetId", assetId);
      chainAddr = await optin(
        client,
        sender,
        TOKEN_BRIDGE_ID,
        assetId,
        textToHexString("native"),
        "TokenBridge native chainAddr"
      );
    }
    accts.push(chainAddr);
  }

  const params: algosdk.SuggestedParams = await client
    .getTransactionParams()
    .do();

  if (meta === "CoreGovernance") {
    txns.push(
      makeApplicationCallTxnFromObject({
        appArgs: [textToUint8Array("governance"), vaa],
        accounts: accts,
        appIndex: CORE_ID,
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );
    txns.push(
      makeApplicationCallTxnFromObject({
        appArgs: [textToUint8Array("nop"), bigIntToBytes(5, 8)],
        appIndex: CORE_ID,
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );
  }
  if (
    meta === "TokenBridge RegisterChain" ||
    meta === "TokenBridge UpgradeContract"
  ) {
    txns.push(
      makeApplicationCallTxnFromObject({
        appArgs: [textToUint8Array("governance"), vaa],
        accounts: accts,
        appIndex: TOKEN_BRIDGE_ID,
        foreignApps: [CORE_ID],
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );
  }

  if (meta === "TokenBridge Attest") {
    let asset: Uint8Array = await decodeLocalState(
      client,
      TOKEN_BRIDGE_ID,
      chainAddr
    );
    let foreignAssets: number[] = [];
    if (asset.length > 8) {
      const tmp = Buffer.from(asset.slice(0, 8));
      foreignAssets.push(Number(tmp.readBigUInt64BE(0)));
    }
    txns.push(
      makePaymentTxnWithSuggestedParamsFromObject({
        from: sender.addr,
        to: chainAddr,
        amount: 100000,
        suggestedParams: params,
      })
    );
    let buf: Uint8Array = new Uint8Array(1);
    buf[0] = 0x01;
    txns.push(
      makeApplicationCallTxnFromObject({
        appArgs: [textToUint8Array("nop"), buf],
        appIndex: TOKEN_BRIDGE_ID,
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );

    buf = new Uint8Array(1);
    buf[0] = 0x02;
    txns.push(
      makeApplicationCallTxnFromObject({
        appArgs: [textToUint8Array("nop"), buf],
        appIndex: TOKEN_BRIDGE_ID,
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );

    txns.push(
      makeApplicationCallTxnFromObject({
        accounts: accts,
        appArgs: [textToUint8Array("receiveAttest"), vaa],
        appIndex: TOKEN_BRIDGE_ID,
        foreignAssets: foreignAssets,
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );
    txns[txns.length - 1].fee = txns[txns.length - 1].fee * 2;
  }

  if (
    meta === "TokenBridge Transfer" ||
    meta === "TokenBridge Transfer With Payload"
  ) {
    let foreignAssets = [];
    let a: number = 0;
    if (parsedVAA.get("FromChain") != 8) {
      let asset = await decodeLocalState(client, TOKEN_BRIDGE_ID, chainAddr);

      console.log("bilbo", chainAddr, asset);

      if (asset.length > 8) {
        const tmp = Buffer.from(asset.slice(0, 8));
        a = Number(tmp.readBigUInt64BE(0));
        console.log("a", a);
      }
    } else {
      a = parseInt(parsedVAA.get("Contract"), 16);
    }

    // The receiver needs to be optin in to receive the coins... Yeah, the relayer pays for this

    const addr = encodeAddress(
      hexStringToUint8Array(parsedVAA.get("ToAddress"))
    );

    console.log("before assetOptin", a, addr);

    if (a != 0) {
      foreignAssets.push(a);
      console.log("inside assetOptin", a, addr);
      await assetOptin(client, sender, foreignAssets[0], addr);
      // And this is how the relayer gets paid...
      if (parsedVAA.get("Fee") != ZERO_PAD_BYTES) {
        await assetOptin(client, sender, foreignAssets[0], sender.addr);
      }
      console.log("done assetOptin", a, addr);
    }
    console.log("after assetOptin", a, addr);
    accts.push(addr);
    txns.push(
      makeApplicationCallTxnFromObject({
        accounts: accts,
        appArgs: [textToUint8Array("completeTransfer"), vaa],
        appIndex: TOKEN_BRIDGE_ID,
        foreignAssets: foreignAssets,
        from: sender.addr,
        onComplete: OnApplicationComplete.NoOpOC,
        suggestedParams: params,
      })
    );

    // We need to cover the inner transactions
    if (parsedVAA.get("Fee") != ZERO_PAD_BYTES) {
      txns[txns.length - 1].fee = txns[txns.length - 1].fee * 3;
    } else {
      txns[txns.length - 1].fee = txns[txns.length - 1].fee * 2;
    }

    if (meta === "TokenBridge Transfer With Payload") {
      txns[txns.length - 1].appForeignApps = [parsedVAA.get("appid")];

      txns.push(
        makeApplicationCallTxnFromObject({
          appArgs: [textToUint8Array("completeTransfer"), vaa],
          appIndex: parsedVAA.get("appid"),
          foreignAssets: foreignAssets,
          from: sender.addr,
          onComplete: OnApplicationComplete.NoOpOC,
          suggestedParams: params,
        })
      );
    }
  }

  console.log("simpleSignVAA start");
  let ret = await simpleSignVAA(client, sender, txns);
  console.log("simpleSignVAA done", ret);
  return ret;
}

/**
 * <p>Transfers an asset from Algorand to a receiver on another chain</p>
 * @param client AlgodV2 client
 * @param sender Sending account
 * @param assetId Asset index
 * @param qty Quantity to transfer
 * @param receiver Receiving account
 * @param chain Reeiving chain
 * @param fee Transfer fee
 * @returns Sequence number of confirmation
 */
export async function transferFromAlgorand(
  client: Algodv2,
  sender: Signer,
  assetId: number,
  qty: number,
  receiver: string,
  chain: number,
  fee: number
) {
  const tokenAddr: string = getApplicationAddress(TOKEN_BRIDGE_ID);
  const applAddr: string = appIdToAppAddr(TOKEN_BRIDGE_ID);
  console.log("Getting emitter addr for core...");
  const emitterAddr: string = await optin(
    client,
    sender,
    CORE_ID,
    0,
    applAddr,
    "transferAsset"
  );
  console.log("Getting assetInfo...");
  let creator;
  let creatorAcctInfo: any;
  let wormhole: boolean = false;
  if (assetId != 0) {
    const assetInfo: Record<string, any> = await client
      .getAssetByID(assetId)
      .do();
    creator = assetInfo["params"]["creator"];
    creatorAcctInfo = await client.accountInformation(creator).do();
    const authAddr: string = creatorAcctInfo["auth-addr"];
    if (authAddr === tokenAddr) {
      wormhole = true;
    }
  }

  const params: algosdk.SuggestedParams = await client
    .getTransactionParams()
    .do();
  let txns: algosdk.Transaction[] = [];
  const msgFee: number = await getMessageFee(client);
  if (msgFee > 0) {
    const payTxn: algosdk.Transaction =
      makePaymentTxnWithSuggestedParamsFromObject({
        from: sender.addr,
        suggestedParams: params,
        to: getApplicationAddress(TOKEN_BRIDGE_ID),
        amount: msgFee,
      });
    txns.push(payTxn);
  }
  if (!wormhole) {
    console.log("Not wormhole...");
    const bNat = Buffer.from("native", "binary").toString("hex");
    creator = await optin(
      client,
      sender,
      TOKEN_BRIDGE_ID,
      assetId,
      bNat,
      "creator"
    );
  }
  console.log("AssetOptinCheck...");
  if (assetId != 0 && !(await assetOptinCheck(client, assetId, creator))) {
    // Looks like we need to optin
    console.log("Creating payTxn...");
    const payTxn: algosdk.Transaction =
      makePaymentTxnWithSuggestedParamsFromObject({
        from: sender.addr,
        to: creator,
        amount: 100000,
        suggestedParams: params,
      });
    txns.push(payTxn);
    // The tokenid app needs to do the optin since it has signature authority
    console.log("Creating call Txn...");
    const bOptin: Uint8Array = textToUint8Array("optin");
    let txn = makeApplicationCallTxnFromObject({
      from: sender.addr,
      appIndex: TOKEN_BRIDGE_ID,
      onComplete: OnApplicationComplete.NoOpOC,
      appArgs: [bOptin, bigIntToBytes(assetId, 8)],
      foreignAssets: [assetId],
      accounts: [creator],
      suggestedParams: params,
    });
    txn.fee *= 2;
    txns.push(txn);
    console.log("Calling simpleSignVAA...");
    await simpleSignVAA(client, sender, txns);
    txns = [];
  }
  console.log("Creating NOP...");
  const t = makeApplicationCallTxnFromObject({
    from: sender.addr,
    appIndex: TOKEN_BRIDGE_ID,
    onComplete: OnApplicationComplete.NoOpOC,
    appArgs: [textToUint8Array("nop")],
    suggestedParams: params,
  });
  txns.push(t);

  let accounts: string[] = [];
  if (assetId === 0) {
    console.log("Algo asset");
    const t = makePaymentTxnWithSuggestedParamsFromObject({
      from: sender.addr,
      to: creator,
      amount: qty,
      suggestedParams: params,
    });
    txns.push(t);
    accounts = [emitterAddr, creator, creator];
  } else {
    console.log("non Algo asset. sender:", sender.addr, "creator:", creator);
    const t = makeAssetTransferTxnWithSuggestedParamsFromObject({
      from: sender.addr,
      to: creator,
      suggestedParams: params,
      amount: qty,
      assetIndex: assetId,
    });
    txns.push(t);
    accounts = [emitterAddr, creator, creatorAcctInfo["address"]];
  }
  console.log("receiver", receiver);
  let args = [
    textToUint8Array("sendTransfer"),
    bigIntToBytes(assetId, 8),
    bigIntToBytes(qty, 8),
    // decodeAddress(receiver.addr).publicKey,
    hexStringToUint8Array(receiver),
    bigIntToBytes(chain, 8),
    bigIntToBytes(fee, 8),
  ];
  console.log("creating transfer txn...");
  let acTxn = makeApplicationCallTxnFromObject({
    from: sender.addr,
    appIndex: TOKEN_BRIDGE_ID,
    onComplete: OnApplicationComplete.NoOpOC,
    appArgs: args,
    foreignApps: [CORE_ID],
    foreignAssets: [assetId],
    accounts: accounts,
    suggestedParams: params,
  });
  acTxn.fee *= 2;
  txns.push(acTxn);
  console.log("calling simpleSignVAA...");
  const resp: Buffer[] = await simpleSignVAA(client, sender, txns);
  console.log("about to parse seq from txn...");
  let seq: bigint = BigInt(0);
  try {
    seq = parseSeqFromLog(resp);
  } catch (pErr) {
    console.error("parseSeqFromLog Failed:", pErr);
  }
  console.log("transferAsset seq:", seq);
  return seq;
}

export async function createWrappedOnAlgorand(
  client: algosdk.Algodv2,
  sender: Signer,
  attestVAA: Uint8Array
): Promise<Buffer[]> {
  return await submitVAA(attestVAA, client, sender, TOKEN_BRIDGE_ID);
}

export async function updateWrappedOnAlgorand(
  client: algosdk.Algodv2,
  sender: Signer,
  vaa: Uint8Array
) {
  const parsedVAA: Map<string, any> = parseVAA(vaa);
  console.log("updateWrappedOnAlgorand::parseVAA", parsedVAA);

  await submitVAA(vaa, client, sender, TOKEN_BRIDGE_ID);
}

/**
 * <p>This basically just submits the VAA to Algorand</p>
 * @param vaa The VAA to be redeemed
 * @param client AlgodV2 client
 * @param acct Sending account
 * @param tokenId Token bridge ID
 * @returns Transaction ID(s)
 */
export async function redeemOnAlgorand(
  vaa: Uint8Array,
  client: Algodv2,
  acct: Signer,
  tokenBridgeId: number
): Promise<Buffer[]> {
  const parsedVAA: Map<string, any> = parseVAA(vaa);
  console.log("redeemOnAlgorand::parseVAA", parsedVAA);

  return await submitVAA(vaa, client, acct, tokenBridgeId);
}

/**
 * <p>This function is used to check if a VAA has been redeemed by looking at a specific bit.<p>
 * @param client AlgodV2 client
 * @param appId Application Id
 * @param addr Wallet address. Someone has to pay for this.
 * @param seq The sequence number of the redemption
 * @returns true, if the bit was set and VAA was redeemed, false otherwise.
 */
async function checkBitsSet(
  client: Algodv2,
  appId: number,
  addr: string,
  seq: bigint
): Promise<boolean> {
  let retval: boolean = false;
  let appState: any[] = [];
  const acctInfo = await client.accountInformation(addr).do();
  const als = acctInfo["apps-local-state"];
  als.forEach((app: any) => {
    if (app["id"] === appId) {
      appState = app["key-value"];
    }
  });
  if (appState.length === 0) {
    console.log("appState is empty");
    return retval;
  }

  const BIG_MAX_BITS: bigint = BigInt(MAX_BITS);
  const BIG_EIGHT: bigint = BigInt(8);
  // Start on a MAX_BITS boundary
  const start: bigint = (seq / BIG_MAX_BITS) * BIG_MAX_BITS;
  // beg should be in the range [0..MAX_BITS]
  const beg: number = Number(seq - start);
  // s should be in the range [0..15]
  const s: number = Math.floor(beg / BITS_PER_KEY);
  const b: number = Math.floor((beg - s * BITS_PER_KEY) / 8);

  const key = Buffer.from(bigIntToBytes(s, 1)).toString("base64");
  console.log("key:", key);
  appState.forEach((kv) => {
    if (kv["key"] === key) {
      const v = Buffer.from(kv["value"]["bytes"], "base64");
      const bt = 1 << Number(seq % BIG_EIGHT);
      retval = (v[b] & bt) != 0;
      return;
    }
  });
  return retval;
}

/**
 *
 * @param client AlgodV2 client
 * @param signedVAA VAA to check
 * @param appId Most likely the Token bridge ID
 * @param wallet The account paying the bill for this (it isn't free)
 * @returns true if VAA has been redeemed, false otherwise
 */
export async function getIsTransferCompletedAlgorand(
  client: Algodv2,
  signedVAA: Uint8Array,
  appId: number,
  wallet: Signer
): Promise<boolean> {
  const parsedVAA: Map<string, any> = parseVAA(signedVAA);
  const seq: number = Number(parsedVAA.get("sequence"));
  console.log("seq:", seq);
  const chainRaw: string = parsedVAA.get("chainRaw"); // this needs to be a hex string
  const em: string = parsedVAA.get("emitter"); // this needs to be a hex string
  console.log("chainRaw:", chainRaw, "em:", em);
  const index: number = parsedVAA.get("index");
  console.log(parsedVAA);

  const seqAddr: string = await optin(
    client,
    wallet,
    appId,
    Math.floor(Number(parsedVAA.get("sequence")) / MAX_BITS),
    chainRaw + em,
    "Getting seqAddr from getIsTransferCompletedAlgorand"
  );
  console.log("seqAddr:", seqAddr);
  const retVal: boolean = await checkBitsSet(
    client,
    appId,
    seqAddr,
    BigInt(seq)
  );
  return retVal;
}

export async function getOriginalAssetAlgorand(
  client: Algodv2,
  assetId: number
): Promise<WormholeWrappedInfo> {
  let retVal: WormholeWrappedInfo = {
    isWrapped: false,
    chainId: 8,
    assetAddress: new Uint8Array(),
  };
  retVal.isWrapped = await getIsWrappedAssetAlgorand(client, assetId);
  const assetInfo = await client.getAssetByID(assetId).do();
  console.log("assetInfo", assetInfo);
  const assetName = assetInfo.params.name;
  console.log("assetName", assetName);
  const lsa = assetInfo.params.creator;
  const aInfo = await client.accountInformation(lsa).do();
  console.log("aInfo", aInfo);
  console.log(aInfo["apps-local-state"]);
  const dls = await decodeLocalState(client, TOKEN_BRIDGE_ID, lsa);
  console.log(dls);
  const cid2 = Buffer.from(dls).readInt16BE(92);
  console.log("chainId", cid2);
  retVal.chainId = cid2 as ChainId;
  retVal.assetAddress = new Uint8Array(Buffer.from(dls).slice(60, 60 + 32));

  console.log(retVal);
  return retVal;
}

export async function getIsWrappedAssetAlgorand(
  client: Algodv2,
  assetId: number
): Promise<boolean> {
  if (assetId === 0) {
    return false;
  }
  const tbAddr: string = getApplicationAddress(TOKEN_BRIDGE_ID);
  const assetInfo = await client.getAssetByID(assetId).do();
  const creatorAddr = assetInfo.params.creator;
  console.log("creatorAddr:", creatorAddr);
  const creatorAcctInfo = await client.accountInformation(creatorAddr).do();
  const wormhole: boolean = creatorAcctInfo["auth-addr"] === tbAddr;
  return wormhole;
}

export function hexToNativeStringAlgo(s: string): string {
  return encodeAddress(hexStringToUint8Array(s));
}

export function nativeStringToHexAlgo(s: string): string {
  return uint8ArrayToHexString(decodeAddress(s).publicKey, false);
}
