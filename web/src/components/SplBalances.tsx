import React, {useContext} from "react"
import {BalanceInfo, SolanaTokenContext} from "../providers/SolanaTokenContext";
import {Table} from "antd";
import {CHAIN_ID_SOLANA} from "../utils/bridge";

function SplBalances() {
    let t = useContext(SolanaTokenContext);

    const columns = [
        {
            title: 'Mint',
            dataIndex: 'mint',
            key: 'mint',
        },
        {
            title: 'Account',
            key: 'account',
            render: (n: any, v: BalanceInfo) => v.account.toString()
        },
        {
            title: 'Balance',
            key: 'balance',
            render: (n: any, v: BalanceInfo) => v.balance.div(Math.pow(10, v.decimals)).toString()
        },
        {
            title: 'Wrapped',
            key: 'wrapped',
            render: (n: any, v: BalanceInfo) => {
                return v.assetMeta.chain != CHAIN_ID_SOLANA ? `Wrapped (${v.assetMeta.chain})` : "Native"
            }
        },
    ];

    return (<>
            <h3>SPL Holdings</h3>
            <Table dataSource={t.balances} columns={columns} pagination={false} scroll={{y: 400}}/>
        </>
    )
}

export default SplBalances
