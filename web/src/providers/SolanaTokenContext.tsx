import React, {createContext, FunctionComponent, useContext, useEffect, useState} from "react"
import ClientContext from "../providers/ClientContext";
import KeyContext from "../providers/KeyContext";
import {AccountInfo, ParsedAccountData, PublicKey, RpcResponseAndContext} from "@solana/web3.js";
import {BigNumber} from "ethers/utils";
import {SlotContext} from "./SlotContext";
import {TOKEN_PROGRAM} from "../config";
import {BridgeContext} from "./BridgeContext";
import {message} from "antd";
import {AssetMeta} from "../utils/bridge";

export interface BalanceInfo {
    mint: string,
    account: PublicKey,
    balance: BigNumber,
    decimals: number,
    assetMeta: AssetMeta
}

export interface TokenInfo {
    balances: Array<BalanceInfo>
    loading: boolean
}

export const SolanaTokenContext = createContext<TokenInfo>({
    balances: [],
    loading: false
})

export const SolanaTokenProvider: FunctionComponent = ({children}) => {
    let k = useContext(KeyContext)
    let c = useContext(ClientContext);
    let b = useContext(BridgeContext);
    let slot = useContext(SlotContext);

    let [loading, setLoading] = useState(true)
    let [balances, setBalances] = useState<Array<BalanceInfo>>([]);

    useEffect(() => {
            // @ts-ignore
            setLoading(true)
            let getAccounts = async () => {
                try {
                    let res: RpcResponseAndContext<Array<{ pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }>> = await c.getParsedTokenAccountsByOwner(k.publicKey, {programId: new PublicKey(TOKEN_PROGRAM)}, "single")
                    let meta: AssetMeta[] = [];
                    for (let acc of res.value) {
                        let am = await b?.fetchAssetMeta(new PublicKey(acc.account.data.parsed.info.mint))
                        if (!am) {
                            throw new Error("could not derive asset meta")
                        }
                        meta.push(am)
                    }
                    let balances: Array<BalanceInfo> = await res.value.map((v, i) => {
                        return {
                            mint: v.account.data.parsed.info.mint,
                            account: v.pubkey,
                            balance: new BigNumber(v.account.data.parsed.info.tokenAmount.amount),
                            decimals: v.account.data.parsed.info.tokenAmount.decimals,
                            assetMeta: meta[i],
                        }
                    })
                    setBalances(balances)
                    setLoading(false)
                } catch (e) {
                    setLoading(false)
                    message.error("Failed to load token accounts")
                }
            }
            getAccounts();
        },
        [slot]
    )

    return (
        <SolanaTokenContext.Provider value={{balances, loading}}>
            {children}
        </SolanaTokenContext.Provider>
    )
}
