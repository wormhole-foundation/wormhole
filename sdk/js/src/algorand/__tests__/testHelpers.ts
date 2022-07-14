// testHelpers.ts

import { describe, expect, it } from "@jest/globals";
import algosdk, {
  Account,
  Algodv2,
  assignGroupID,
  makePaymentTxnWithSuggestedParamsFromObject,
  Transaction,
  waitForConfirmation,
} from "algosdk";
import { getForeignAssetAlgorand } from "../../token_bridge";
import { ChainId } from "../../utils";
import { TransactionSignerPair, _parseVAAAlgorand } from "../Algorand";

const ci = !!process.env.CI;

const ALGO_TOKEN =
  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const ALGOD_ADDRESS: string = ci ? "http://algorand" : "http://localhost";
const ALGOD_PORT: number = 4001;

/**
 *  Creates a new Algodv2 client using local file consts
 * @returns a newly constructed Algodv2 client
 */
export function getAlgoClient(): Algodv2 {
  const algodClient = new Algodv2(ALGO_TOKEN, ALGOD_ADDRESS, ALGOD_PORT);
  return algodClient;
}

let KMD_TOKEN =
  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const KMD_ADDRESS: string = ci ? "http://algorand" : "http://localhost";
const KMD_PORT: number = 4002;
const KMD_WALLET_NAME: string = "unencrypted-default-wallet";
const KMD_WALLET_PASSWORD: string = "";

export function getKmdClient(): algosdk.Kmd {
  const kmdClient: algosdk.Kmd = new algosdk.Kmd(
    KMD_TOKEN,
    KMD_ADDRESS,
    KMD_PORT
  );
  return kmdClient;
}

export async function getGenesisAccounts(): Promise<Account[]> {
  let retval: Account[] = [];
  const kmd: algosdk.Kmd = getKmdClient();

  // Get list of wallets
  const wallets = (await kmd.listWallets()).wallets;
  if (!wallets) {
    console.error("No wallets found!");
    return retval;
  }

  // Walk walles to find correct wallet
  let myWalletId: string = "";
  wallets.forEach((element: any) => {
    if (element.name === KMD_WALLET_NAME) {
      myWalletId = element.id;
    }
  });
  if (myWalletId.length === 0) {
    console.error("invalid wallet ID");
    return retval;
  }

  // Get the wallet handle for Genesis wallet
  const myWalletHandle = (
    await kmd.initWalletHandle(myWalletId, KMD_WALLET_PASSWORD)
  ).wallet_handle_token;

  // Get the 3 addresses associated with the Genesis wallet
  const addresses = (await kmd.listKeys(myWalletHandle)).addresses;
  for (let i = 0; i < addresses.length; i++) {
    const element = addresses[i];
    const myExportedKey: Buffer = (
      await kmd.exportKey(myWalletHandle, KMD_WALLET_PASSWORD, element)
    ).private_key;
    let mn = algosdk.secretKeyToMnemonic(myExportedKey);
    let ta = algosdk.mnemonicToSecretKey(mn);

    retval.push(ta);
  }
  kmd.releaseWalletHandle(myWalletHandle);
  return retval;
}

export async function firstKmdTransaction() {
  try {
    const genAccounts = await getGenesisAccounts();

    // const walletRsp = await myKmdClient.getWallet(myWalletHandle);
  } catch (e) {
    console.error("KMD transaction error:", e);
  }
}

// This function creates temporary accounts and funds them with the
// Genesis accounts.
export async function getTempAccounts(): Promise<Account[]> {
  let retval: Account[] = [];
  const algodClient = getAlgoClient();
  const genesisAccounts: Account[] = await getGenesisAccounts();
  const numAccts = genesisAccounts.length;
  if (numAccts === 0) {
    console.error("Failed to get genesisAccounts");
    return retval;
  }
  const params = await algodClient.getTransactionParams().do();
  let transactions: Transaction[] = [];
  for (let i = 0; i < numAccts; i++) {
    let newAcct = createAccount();
    if (!newAcct) {
      throw new Error("failed to create a temp account");
    }
    let fundingAcct = genesisAccounts[i];
    // Create a payment transaction
    const payTxn = makePaymentTxnWithSuggestedParamsFromObject({
      from: fundingAcct.addr,
      to: newAcct.addr,
      amount: 15000000,
      suggestedParams: params,
    });
    // Sign the transaction
    const signedTxn = payTxn.signTxn(fundingAcct.sk);
    const signedTxnId = payTxn.txID().toString();
    // Submit the transaction
    const txId = await algodClient.sendRawTransaction(signedTxn).do();
    // Wait for response
    const confirmedTxn = await algosdk.waitForConfirmation(
      algodClient,
      signedTxnId,
      4
    );
    //Get the completed Transaction
    let accountInfo = await algodClient.accountInformation(newAcct.addr).do();
    retval.push(newAcct);
  }
  return retval;
}

export function createAccount(): Account | undefined {
  try {
    const retval = algosdk.generateAccount();
    return retval;
  } catch (err) {
    console.error("err", err);
  }
}

/**
 *  Return the balances of all assets for the supplied account
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

  // Put the algo balance in key 0
  balances.set(0, accountInfo.amount);

  const assets: Array<any> = accountInfo.assets;
  assets.forEach(function (asset) {
    const assetId = asset["asset-id"];
    const amount = asset.amount;
    balances.set(assetId, amount);
  });
  return balances;
}

/**
 * Return the balance of the supplied asset index for the supplied account
 * @param client An Algodv2 client
 * @param account The account to query for the supplied asset index
 * @param assetId The asset index
 * @returns The quantity of the asset in the supplied account
 */
export async function getBalance(
  client: Algodv2,
  account: string,
  assetId: bigint
): Promise<bigint> {
  const accountInfo = await client.accountInformation(account).do();

  if (assetId === BigInt(0)) {
    return accountInfo.amount;
  }

  let ret = BigInt(0);
  const assets: Array<any> = accountInfo.assets;
  assets.forEach((asset) => {
    if (Number(assetId) === asset["asset-id"]) {
      ret = asset.amount;
      return;
    }
  });
  return ret;
}

export async function createAsset(account: Account): Promise<any> {
  const aClient = getAlgoClient();
  const params = await aClient.getTransactionParams().do();
  const note = undefined; // arbitrary data to be stored in the transaction; here, none is stored
  // Asset creation specific parameters
  const addr = account.addr;
  // Whether user accounts will need to be unfrozen before transacting
  const defaultFrozen = false;
  // integer number of decimals for asset unit calculation
  const decimals = 10;
  // total number of this asset available for circulation
  const totalIssuance = 1000000;
  // Used to display asset units to user
  const unitName = "NORIUM";
  // Friendly name of the asset
  const assetName = "ChuckNorium";
  // Optional string pointing to a URL relating to the asset
  // const assetURL = "http://www.chucknorris.com";
  const assetURL = "";
  // Optional hash commitment of some sort relating to the asset. 32 character length.
  // const assetMetadataHash = "16efaa3924a6fd9d3a4824799a4ac65d";
  const assetMetadataHash = "";
  // The following parameters are the only ones
  // that can be changed, and they have to be changed
  // by the current manager
  // Specified address can change reserve, freeze, clawback, and manager
  const manager = account.addr;
  // Specified address is considered the asset reserve
  // (it has no special privileges, this is only informational)
  const reserve = account.addr;
  // Specified address can freeze or unfreeze user asset holdings
  const freeze = account.addr;
  // Specified address can revoke user asset holdings and send
  // them to other addresses
  const clawback = account.addr;

  // signing and sending "txn" allows "addr" to create an asset
  const txn = algosdk.makeAssetCreateTxnWithSuggestedParams(
    addr,
    note,
    totalIssuance,
    decimals,
    defaultFrozen,
    manager,
    reserve,
    freeze,
    clawback,
    unitName,
    assetName,
    assetURL,
    assetMetadataHash,
    params
  );

  const rawSignedTxn = txn.signTxn(account.sk);
  const tx = await aClient.sendRawTransaction(rawSignedTxn).do();

  // wait for transaction to be confirmed
  const ptx = await algosdk.waitForConfirmation(aClient, tx.txId, 4);
  // Get the new asset's information from the creator account
  const assetID: number = ptx["asset-index"];
  //Get the completed Transaction
  return assetID;
}

export async function createNFT(account: Account): Promise<number> {
  const aClient = getAlgoClient();
  const params = await aClient.getTransactionParams().do();
  // Asset creation specific parameters
  const addr = account.addr;
  // Whether user accounts will need to be unfrozen before transacting
  const defaultFrozen = false;
  // integer number of decimals for asset unit calculation
  const decimals = 0;
  // total number of this asset available for circulation
  const total = 1;
  // Used to display asset units to user
  const unitName = "CNART";
  // Friendly name of the asset
  const assetName = "ChuckNoriumArtwork@arc3";
  // Optional string pointing to a URL relating to the asset
  const assetURL = "http://www.chucknorris.com";
  // Optional hash commitment of some sort relating to the asset. 32 character length.
  const assetMetadataHash = "16efaa3924a6fd9d3a4824799a4ac65d";
  // The following parameters are the only ones
  // that can be changed, and they have to be changed
  // by the current manager
  // Specified address can change reserve, freeze, clawback, and manager
  const manager = account.addr;
  // Specified address is considered the asset reserve
  // (it has no special privileges, this is only informational)
  const reserve = account.addr;
  // Specified address can freeze or unfreeze user asset holdings
  const freeze = account.addr;
  // Specified address can revoke user asset holdings and send
  // them to other addresses
  const clawback = account.addr;

  // signing and sending "txn" allows "addr" to create an asset
  const txn = algosdk.makeAssetCreateTxnWithSuggestedParamsFromObject({
    from: addr,
    total,
    decimals,
    assetName,
    unitName,
    assetURL,
    assetMetadataHash,
    defaultFrozen,
    freeze,
    manager,
    clawback,
    reserve,
    suggestedParams: params,
  });

  const rawSignedTxn = txn.signTxn(account.sk);
  const tx = await aClient.sendRawTransaction(rawSignedTxn).do();

  // wait for transaction to be confirmed
  const ptx = await algosdk.waitForConfirmation(aClient, tx.txId, 4);
  // Get the new asset's information from the creator account
  const assetID: number = ptx["asset-index"];
  return assetID;
}

export async function getForeignAssetFromVaaAlgorand(
  client: Algodv2,
  tokenBridgeId: bigint,
  vaa: Uint8Array
): Promise<bigint | null> {
  const parsedVAA = _parseVAAAlgorand(vaa);
  if (parsedVAA.Contract === undefined) {
    throw "parsedVAA.Contract is undefined";
  }
  return await getForeignAssetAlgorand(
    client,
    tokenBridgeId,
    parsedVAA.FromChain as ChainId,
    parsedVAA.Contract
  );
}

export async function signSendAndConfirmAlgorand(
  algodClient: Algodv2,
  txs: TransactionSignerPair[],
  wallet: Account
) {
  assignGroupID(txs.map((tx) => tx.tx));
  const signedTxns: Uint8Array[] = [];
  for (const tx of txs) {
    if (tx.signer) {
      signedTxns.push(await tx.signer.signTxn(tx.tx));
    } else {
      signedTxns.push(tx.tx.signTxn(wallet.sk));
    }
  }
  await algodClient.sendRawTransaction(signedTxns).do();
  const result = await waitForConfirmation(
    algodClient,
    txs[txs.length - 1].tx.txID(),
    1
  );
  return result;
}

describe("testHelpers exists in test folder", () => {
  it("pass test", () => {
    expect(true).toBe(true);
  });
});
