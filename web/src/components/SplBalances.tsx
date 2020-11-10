import React, {useContext} from "react"
import {BalanceInfo, SolanaTokenContext} from "../providers/SolanaTokenContext";
import {Table} from "antd";
import {CHAIN_ID_SOLANA} from "../utils/bridge";
import {BigNumber} from "ethers/utils";
import {PublicKey} from "@solana/web3.js";
import {deriveERC20Address} from "../utils/helpers";


function SplBalances() {
    let t = useContext(SolanaTokenContext);

    const columns = [
        {
            title: 'Account',
            key: 'account',
            render: (n: any, v: BalanceInfo) => v.account.toString()
        },
        {
            title: 'Mint',
            dataIndex: 'mint',
            key: 'mint',
        },
        {
            title: 'Balance',
            key: 'balance',
            render: (n: any, v: BalanceInfo) => v.balance.div(new BigNumber(10).pow(v.decimals)).toString()
        },
        {
            title: 'Wrapped',
            key: 'wrapped',
            render: (n: any, v: BalanceInfo) => {
                return v.assetMeta.chain != CHAIN_ID_SOLANA ? `Wrapped (${v.assetMeta.chain == 2 ? "ETH" : "SOL"} - 0x${v.assetMeta.address.slice(12).toString("hex")})` : `Native (0x${deriveERC20Address(new PublicKey(v.mint))})`
            }
        },
    ];

    return (<>
            <h3>SPL Holdings on Solana</h3>
            <Table dataSource={t.balances} columns={columns} pagination={false} scroll={{y: 400}}/>
        </>
    )
}

export default SplBalances
