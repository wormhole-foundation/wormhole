// Helpers.ts

import algosdk, {
    Account,
    makePaymentTxnWithSuggestedParamsFromObject,
    Transaction,
} from "algosdk";
import {
    getAlgoClient,
    TESTNET_ACCOUNT_ADDRESS,
    TESTNET_ACCOUNT_MN,
} from "./Algorand";

let KMD_TOKEN =
    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const KMD_ADDRESS: string = "http://localhost";
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
        console.log(element);
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
    console.log("myWalletHandle:", myWalletHandle);

    // Get the 3 addresses associated with the Genesis wallet
    const addresses = (await kmd.listKeys(myWalletHandle)).addresses;
    console.log("addresses:", addresses);
    for (let i = 0; i < addresses.length; i++) {
        const element = addresses[i];
        const myExportedKey: Buffer = (
            await kmd.exportKey(myWalletHandle, KMD_WALLET_PASSWORD, element)
        ).private_key;
        console.log("exported key:", element, myExportedKey);
        let mn = algosdk.secretKeyToMnemonic(myExportedKey);
        let ta = algosdk.mnemonicToSecretKey(mn);

        retval.push(ta);
    }
    kmd.releaseWalletHandle(myWalletHandle);
    console.log("length of genesis accounts:", retval.length);
    return retval;
}

export async function firstKmdTransaction() {
    try {
        const genAccounts = await getGenesisAccounts();

        // const walletRsp = await myKmdClient.getWallet(myWalletHandle);
        // console.log("walletRsp:", walletRsp);
    } catch (e) {
        console.log("KMD transaction error:", e);
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
    console.log("About to construct txns...");
    const params = await algodClient.getTransactionParams().do();
    let transactions: Transaction[] = [];
    for (let i = 0; i < numAccts; i++) {
        let newAcct = createAccount();
        if (!newAcct) {
            throw new Error("failed to create a temp account");
        }
        let fundingAcct = genesisAccounts[i];
        // Create a payment transaction
        console.log(
            "Creating paytxn with fundAcct",
            fundingAcct,
            "newAcct",
            newAcct
        );
        const payTxn = makePaymentTxnWithSuggestedParamsFromObject({
            from: fundingAcct.addr,
            to: newAcct.addr,
            amount: 15000000,
            suggestedParams: params,
        });
        // Sign the transaction
        console.log("signing paytxn...");
        const signedTxn = payTxn.signTxn(fundingAcct.sk);
        const signedTxnId = payTxn.txID().toString();
        console.log("signedTxnId:", signedTxnId);
        // Submit the transaction
        console.log("submitting transaction...");
        const txId = await algodClient.sendRawTransaction(signedTxn).do();
        console.log("submitted txId:", txId);
        // Wait for response
        const confirmedTxn = await algosdk.waitForConfirmation(
            algodClient,
            signedTxnId,
            4
        );
        //Get the completed Transaction
        console.log(
            "Transaction " +
                txId +
                " confirmed in round " +
                confirmedTxn["confirmed-round"]
        );
        console.log("Confirmation response:", confirmedTxn);
        // let mytxinfo = JSON.stringify(confirmedTxn.txn.txn, undefined, 2);
        // console.log("Transaction information: %o", mytxinfo);
//        let string = new TextDecoder().decode(confirmedTxn.txn.txn.note);
//        console.log("Note field: ", string);
        let accountInfo = await algodClient
            .accountInformation(newAcct.addr)
            .do();
        console.log(
            "Transaction Amount: %d microAlgos",
            confirmedTxn.txn.txn.amt
        );
        console.log("Transaction Fee: %d microAlgos", confirmedTxn.txn.txn.fee);
        console.log("Account balance: %d microAlgos", accountInfo.amount);
        retval.push(newAcct);
    }
    return retval;
}

export function createAccount(): Account | undefined {
    try {
        const retval = algosdk.generateAccount();
        // let retval = new Account(tempAcct.addr, Buffer.from(tempAcct.sk));
        // let account_mnemonic = algosdk.secretKeyToMnemonic(tempAcct.sk);
        // console.log("Account Address = " + retval.addr);
        // console.log("Account Mnemonic = " + account_mnemonic);
        console.log("Class Account:", retval);

        return retval;
    } catch (err) {
        console.log("err", err);
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

export async function firstAlgoTransaction() {
    try {
        // Create an Account
        // const myAccount = createAccount();
        // if (!myAccount) {
        //     throw new Error("Failed to createAccount");
        // }
        // console.log("account:", myAccount);

        // Connect client
        const algodClient = getAlgoClient();
        const mySecretKey = algosdk.mnemonicToSecretKey(TESTNET_ACCOUNT_MN);

        // Check your balance
        const accountInfo = await algodClient
            // .accountInformation(myAccount.addr)
            .accountInformation(TESTNET_ACCOUNT_ADDRESS)
            .do();
        console.log("Account balance: %d microAlgos", accountInfo.amount);
        console.log("accountInfo:", accountInfo);

        // Construct the transaction
        let params = await algodClient.getTransactionParams().do();
        console.log("params:", params);
        // comment out the next two lines to use suggested fee
        // params.fee = algosdk.ALGORAND_MIN_TX_FEE;
        // params.flatFee = true;

        // receiver defined as TestNet faucet address
        // const receiver = "HZ57J3K46JIJXILONBBZOHX6BKPXEM2VVXNRFSUED6DKFD5ZD24PMJ3MVA";
        // const enc = new TextEncoder();
        // const note = enc.encode("Hello World");
        // let amount = 1000000;
        // let sender = TESTNET_ACCOUNT_ADDRESS;
        // let txn = algosdk.makePaymentTxnWithSuggestedParamsFromObject({
        //     from: sender,
        //     to: receiver,
        //     amount: amount,
        //     note: note,
        //     suggestedParams: params
        // });

        // Sign the transaction
        // let signedTxn = txn.signTxn(myAccount.sk);
        // let txId = txn.txID().toString();
        // console.log("Signed transaction with txID: %s", txId);

        // // Submit the transaction
        // await algodClient.sendRawTransaction(signedTxn).do();

        // // Wait for confirmation
        // let confirmedTxn = await waitForConfirmation(algodClient, txId, 4);
        // //Get the completed Transaction
        // console.log("Transaction " + txId + " confirmed in round " + confirmedTxn["confirmed-round"]);
        // var string = new TextDecoder().decode(confirmedTxn.txn.txn.note);
        // console.log("Note field: ", string);
        // accountInfo = await algodClient.accountInformation(myAccount.addr).do();
        // console.log("Transaction Amount: %d microAlgos", confirmedTxn.txn.txn.amt);
        // console.log("Transaction Fee: %d microAlgos", confirmedTxn.txn.txn.fee);

        // console.log("Account balance: %d microAlgos", accountInfo.amount);
    } catch (err) {
        console.log("err", err);
    }
    // process.exit();
}

export async function testFn() {
    const tempAccts = await getTempAccounts();
    const numAccts = tempAccts.length;
    for (let i = 0; i < numAccts; i++) {
        const bal = await getBalances(tempAccts[i].addr);
        console.log("balance:", bal);
    }
    await createAsset(tempAccts[0]);
    const remBal = await getBalances(tempAccts[0].addr);
    console.log("remBal:", remBal);
}

export async function createAsset(account: Account): Promise<any> {
    console.log("Creating asset...");
    const aClient = getAlgoClient();
    const params = await aClient.getTransactionParams().do();
    const note = undefined; // arbitrary data to be stored in the transaction; here, none is stored
    // Asset creation specific parameters
    const addr = account.addr;
    // Whether user accounts will need to be unfrozen before transacting
    const defaultFrozen = false;
    // integer number of decimals for asset unit calculation
    const decimals = 0;
    // total number of this asset available for circulation
    const totalIssuance = 1000;
    // Used to display asset units to user
    const unitName = "NORIUM";
    // Friendly name of the asset
    const assetName = "norium";
    // Optional string pointing to a URL relating to the asset
    const assetURL = "http://someurl";
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
    console.log("rawSignedTxn:", rawSignedTxn);
    const tx = await aClient.sendRawTransaction(rawSignedTxn).do();

    // wait for transaction to be confirmed
    const ptx = await algosdk.waitForConfirmation(aClient, tx.txId, 4);
    console.log("createAsset() - ptx:", ptx);
    // Get the new asset's information from the creator account
    const assetID: number = ptx["asset-index"];
    //Get the completed Transaction
    console.log(
        "createAsset() - Transaction " +
            tx.txId +
            " confirmed in round " +
            ptx["confirmed-round"]
    );
    return assetID;
}
