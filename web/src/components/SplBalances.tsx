import React, {useContext} from "react"
import {BalanceInfo, SolanaTokenContext} from "../providers/SolanaTokenContext";
import {Table} from "antd";

function SplBalances() {
    let t = useContext(SolanaTokenContext);

    const dataSource = [
        {
            key: '1',
            name: 'Mike',
            age: 32,
            address: '10 Downing Street',
        },
        {
            key: '2',
            name: 'John',
            age: 42,
            address: '10 Downing Street',
        },
    ];

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
    ];

    return (<>
            <h3>SPL Holdings</h3>
            <Table dataSource={t.balances} columns={columns} pagination={false} scroll={{y: 400}}/>
        </>
    )
}

export default SplBalances
