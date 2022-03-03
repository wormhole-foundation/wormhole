import { Secp256k1HdWallet, SigningCosmosClient, Msg, coins, LcdClient,
    setupAuthExtension,
    setupBankExtension,
    setupDistributionExtension,
    setupGovExtension,
    setupMintExtension,
    setupSlashingExtension,
    setupStakingExtension,
    setupSupplyExtension} from "@cosmjs/launchpad";
import { ADDRESS_PREFIX, FAUCET_URL, HOLE_DENOM, NODE_URL } from "../consts";
import axios from 'axios';
import { DeclarationName } from "typescript";

//https://www.npmjs.com/package/@cosmjs/launchpad

export function getClient(){ 
    return LcdClient.withExtensions(
    { apiUrl : NODE_URL },
    setupAuthExtension,
    setupBankExtension,
    setupDistributionExtension,
    setupGovExtension,
    setupMintExtension,
    setupSlashingExtension,
    setupStakingExtension,
    setupSupplyExtension,
  );
}

export async function getWallet(mnemonic : string) : Promise<Secp256k1HdWallet> {
    return await Secp256k1HdWallet.fromMnemonic(mnemonic, {prefix:ADDRESS_PREFIX});
} 

export async function getAddress(wallet : Secp256k1HdWallet) : Promise<string> {
    //There are actually up to 5 accounts in a cosmos wallet. I believe this returns the first wallet.
    const [{ address }] = await wallet.getAccounts();

    return address
}

export async function faucet(denom : string, amount : string, address : string){
    await axios.post(FAUCET_URL, {
        "address": address,
        "coins": [
          amount + denom
        ]
      })
      return;
}

export async function signSendAndConfirm(wallet : Secp256k1HdWallet, msgs : Msg[]) {
    const address = await getAddress(wallet)
    const client = new SigningCosmosClient(NODE_URL, address, wallet);

    //TODO figure out fees
    const fee = {
        amount: coins(0, HOLE_DENOM),
        gas: "0", 
    };
    const result = await client.signAndBroadcast(msgs, fee);

    return result
}

export async function sendTokens(wallet :  Secp256k1HdWallet, denom : string, amount: BigInt, recipient: string) {
    const address = await getAddress(wallet);
    const client = new SigningCosmosClient(NODE_URL, address, wallet);

    const result = await client.sendTokens(recipient, coins(amount.toString(), denom));
    return result
}

export async function getBalance(denom : string, address : string) {
    const client = getClient()
    const balances = await client.bank.balances(address);

    const balance = balances.result.find(x => x.denom === denom);

    return balance ? parseInt(balance.amount) : 0;
}

