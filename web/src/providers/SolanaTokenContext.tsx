import React, {createContext, FunctionComponent, useContext, useEffect, useState} from "react"
import ClientContext from "../providers/ClientContext";
import KeyContext from "../providers/KeyContext";
import {AccountInfo, ParsedAccountData, PublicKey, RpcResponseAndContext} from "@solana/web3.js";
import {message} from "antd";
import {BigNumber} from "ethers/utils";
import {SlotContext} from "./SlotContext";
import {TOKEN_PROGRAM} from "../config";

export interface BalanceInfo {
    mint: string,
    account: PublicKey,
    balance: BigNumber,
    decimals: number
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
    let slot = useContext(SlotContext);

    let [loading, setLoading] = useState(true)
    let [accounts, setAccounts] = useState<Array<{ pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }>>([]);

    useEffect(() => {
        // @ts-ignore
        setLoading(true)
        c.getParsedTokenAccountsByOwner(k.publicKey, {programId: new PublicKey(TOKEN_PROGRAM)},"single").then((res: RpcResponseAndContext<Array<{ pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }>>) => {
            setAccounts(res.value)
            setLoading(false)
        }).catch(() => {
            setLoading(false)
            message.error("Failed to load token accounts")
        })
    }, [slot])

    let balances: Array<BalanceInfo> = accounts.map((v) => {
        return {
            mint: v.account.data.parsed.info.mint,
            account: v.pubkey,
            balance: new BigNumber(v.account.data.parsed.info.tokenAmount.amount),
            decimals: v.account.data.parsed.info.tokenAmount.decimals
        }
    })
    return (
        <SolanaTokenContext.Provider value={{balances, loading}}>
            {children}
        </SolanaTokenContext.Provider>
    )
}
