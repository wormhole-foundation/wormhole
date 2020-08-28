import React, {useContext, useEffect, useState} from "react"
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {Table} from "antd";
import {Lockup} from "../utils/bridge";
import {BridgeContext} from "../providers/BridgeContext";
import {SlotContext} from "../providers/SlotContext";
import {ethers} from "ethers";
import {WormholeFactory} from "../contracts/WormholeFactory";
import {BRIDGE_ADDRESS} from "../config";
import {keccak256} from "ethers/utils";
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);

function TransferProposals() {
    let s = useContext(SlotContext);
    let t = useContext(SolanaTokenContext);
    let tokens = useContext(SolanaTokenContext);
    let b = useContext(BridgeContext);

    let [lockups, setLockups] = useState<Lockup[]>([])

    useEffect(() => {
        if (s % 10 !== 0) return;

        let updateLockups = async () => {
            let lockups = [];
            for (let account of tokens.balances) {
                let accLockups = await b.fetchTransferProposals(account.account)
                lockups.push(...accLockups)
            }

            let wormhole = WormholeFactory.connect(BRIDGE_ADDRESS, provider);
            for (let lockup of lockups) {
                console.log(lockup)

                if (lockup.vaaTime === undefined || lockup.vaaTime === 0) continue;

                let signingData = lockup.vaa.slice(lockup.vaa[5] * 66 + 6)
                let hash = keccak256(signingData)

                let status = await wormhole.consumedVAAs(hash)
                lockup.initialized = status;
            }

            setLockups(lockups);
        }
        updateLockups()
    }, [s])

    const columns = [
        {
            title: 'SourceAccount',
            key: 'source',
            render: (n: any, v: Lockup) => v.sourceAddress.toString()
        },
        {
            title: 'Mint',
            key: 'assetAddress',
            render: (n: any, v: Lockup) => v.assetAddress.toString()
        },
        {
            title: 'Amount',
            key: 'amount',
            render: (n: any, v: Lockup) => v.amount.toString()
        },
        {
            title: 'Status',
            key: 'status',
            render: (n: any, v: Lockup) => {
                return (<>Pending {v.initialized}</>)
            }
        },
    ];

    return (<>
            <h3>Pending transfers</h3>
            <Table dataSource={lockups} columns={columns} pagination={false} scroll={{y: 400}}/>
        </>
    )
}

export default TransferProposals
