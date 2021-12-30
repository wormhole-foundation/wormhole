import { Coins, LCDClient, MnemonicKey, Msg, MsgExecuteContract, StdFee, TxInfo, Wallet } from "@terra-money/terra.js";
import axios from "axios";
import {
    ETH_NODE_URL,
    TERRA_CHAIN_ID,
    TERRA_GAS_PRICES_URL,
    TERRA_NODE_URL,
    TERRA_PRIVATE_KEY
} from './consts';
import Web3 from 'web3';

export const lcd = new LCDClient({
    URL: TERRA_NODE_URL,
    chainID: TERRA_CHAIN_ID,
});
export const terraWallet: Wallet = lcd.wallet(new MnemonicKey({
    mnemonic: TERRA_PRIVATE_KEY,
}));

export const web3 = new Web3(ETH_NODE_URL);

export async function getGasPrices() {
    return axios
        .get(TERRA_GAS_PRICES_URL)
        .then((result) => result.data);
}

export async function estimateTerraFee(gasPrices: Coins.Input, msgs: Msg[]): Promise<StdFee> {
    const feeEstimate = await lcd.tx.estimateFee(
        terraWallet.key.accAddress,
        msgs,
        {
            memo: "localhost",
            feeDenoms: ["uluna"],
            gasPrices,
        }
    );
    return feeEstimate;
}


export async function mint_cw721(contract_address: string, token_id: number, token_uri: any): Promise<void> {
    await terraWallet
        .createAndSignTx({
            msgs: [
                new MsgExecuteContract(
                    terraWallet.key.accAddress,
                    contract_address,
                    {
                        mint: {
                            token_id: token_id.toString(),
                            owner: terraWallet.key.accAddress,
                            token_uri: token_uri,
                        },
                    },
                    { uluna: 1000 }
                ),
            ],
            memo: "",
            fee: new StdFee(2000000, {
                uluna: "100000",
            }),
        })
        .then((tx) => lcd.tx.broadcast(tx));
}

export async function waitForTerraExecution(txHash: string): Promise<TxInfo> {
    let info: TxInfo | undefined = undefined;
    while (!info) {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        try {
            info = await lcd.tx.txInfo(txHash);
        } catch (e) {
            console.error(e);
        }
    }
    if (info.code !== undefined) {
        // error code
        throw new Error(
            `Tx ${txHash}: error code ${info.code}: ${info.raw_log}`
        );
    }
    return info;
}
