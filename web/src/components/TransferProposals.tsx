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
import BN from 'bn.js';
import {PublicKey} from "@solana/web3.js";
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);

interface LockupWithStatus extends Lockup {
    status: LockupStatus,
}

enum LockupStatus {
    AWAITING_VAA,
    UNCLAIMED_VAA,
    COMPLETED
}

function TransferProposals() {
    let s = useContext(SlotContext);
    let t = useContext(SolanaTokenContext);
    let tokens = useContext(SolanaTokenContext);
    let b = useContext(BridgeContext);

    let [lockups, setLockups] = useState<LockupWithStatus[]>([])

    useEffect(() => {
        if (s % 10 !== 0) return;

        let updateLockups = async () => {
            let lockups: LockupWithStatus[] = [];
            for (let account of tokens.balances) {
                let accLockups = await b.fetchTransferProposals(account.account)
                lockups.push(...accLockups.map(v => {
                    return {
                        status: LockupStatus.AWAITING_VAA,
                        ...v
                    }
                }))
            }

            let wormhole = WormholeFactory.connect(BRIDGE_ADDRESS, provider);
            for (let lockup of lockups) {
                if (lockup.vaaTime === undefined || lockup.vaaTime === 0) continue;

                let signingData = lockup.vaa.slice(lockup.vaa[5] * 66 + 6)
                for (let i = signingData.length; i > 0; i--) {
                    if (signingData[i] == 0xff) {
                        signingData = signingData.slice(0, i)
                        break
                    }
                }
                let hash = keccak256(signingData)
                let submissionStatus = await wormhole.consumedVAAs(hash);

                lockup.status = submissionStatus ? LockupStatus.COMPLETED : LockupStatus.UNCLAIMED_VAA;
            }

            setLockups(lockups);
        }
        updateLockups()
    }, [s])

    let statusToPrompt = (v: LockupStatus) => {
        switch (v) {
            case LockupStatus.AWAITING_VAA:
                return ("Awaiting VAA");
            case LockupStatus.UNCLAIMED_VAA:
                return ("Submit to chain");
            case LockupStatus.COMPLETED:
                return ("Completed");
        }
    }

    const columns = [
        {
            title: 'SourceAccount',
            key: 'source',
            render: (n: any, v: LockupWithStatus) => "SOL: " + v.sourceAddress.toString()
        },
        {
            title: 'TargetAccount',
            key: 'target',
            render: (n: any, v: LockupWithStatus) => {
                switch (v.toChain) {
                    case 1:
                        return "SOL: " + new PublicKey(v.targetAddress).toString()
                    case 2:
                        return "ETH: 0x" + new Buffer(v.targetAddress.slice(12)).toString("hex")
                }
            }
        },
        {
            title: 'Asset',
            key: 'assetAddress',
            render: (n: any, v: LockupWithStatus) => {
                switch (v.assetChain) {
                    case 1:
                        return "SOL: " + new PublicKey(v.assetAddress).toString()
                    case 2:
                        return "ETH: 0x" + new Buffer(v.assetAddress.slice(12)).toString("hex")
                }
            }
        },
        {
            title: 'Amount',
            key: 'amount',
            render: (n: any, v: LockupWithStatus) => v.amount.div(new BN(10).pow(new BN(v.assetDecimals))).toString()
        },
        {
            title: 'Status',
            key: 'status',
            render: (n: any, v: LockupWithStatus) => {
                return (<>{statusToPrompt(v.status)}</>)
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
