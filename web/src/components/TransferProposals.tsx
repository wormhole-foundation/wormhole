import React, {useContext, useEffect, useState} from "react"
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {Button, message, Table} from "antd";
import {Lockup} from "../utils/bridge";
import {BridgeContext} from "../providers/BridgeContext";
import {SlotContext} from "../providers/SlotContext";
import {ethers} from "ethers";
import {WormholeFactory} from "../contracts/WormholeFactory";
import {BRIDGE_ADDRESS} from "../config";
import {keccak256} from "ethers/utils";
import BN from 'bn.js';
import {PublicKey, Transaction} from "@solana/web3.js";
import KeyContext from "../providers/KeyContext";
import ClientContext from "../providers/ClientContext";

// @ts-ignore
window.ethereum.enable();
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

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
    let k = useContext(KeyContext);
    let c = useContext(ClientContext);

    let [lockups, setLockups] = useState<LockupWithStatus[]>([])

    useEffect(() => {
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

    let executeVAA = async (v: LockupWithStatus) => {
        let wh = WormholeFactory.connect(BRIDGE_ADDRESS, signer)
        let vaa = v.vaa;
        for (let i = vaa.length; i > 0; i--) {
            if (vaa[i] == 0xff) {
                vaa = vaa.slice(0, i)
                break
            }
        }
        message.loading({content: "Signing transaction...", key: "eth_tx", duration: 1000},)
        let tx = await wh.submitVAA(vaa)
        message.loading({content: "Waiting for transaction to be mined...", key: "eth_tx", duration: 1000})
        await tx.wait(1)
        message.success({content: "Execution of VAA succeeded", key: "eth_tx"})
    }

    let pokeProposal = async (proposalAddress: PublicKey) => {
        message.loading({content: "Poking lockup ...", key: "poke"}, 1000)

        let ix = await b.createPokeProposalInstruction(proposalAddress);
        let recentHash = await c.getRecentBlockhash();
        let tx = new Transaction();
        tx.recentBlockhash = recentHash.blockhash
        tx.add(ix)
        tx.sign(k)
        try {
            await c.sendTransaction(tx, [k])
            message.success({content: "Poke succeeded", key: "poke"})
        } catch (e) {
            message.error({content: "Poke failed", key: "poke"})
        }
    }

    let statusToPrompt = (v: LockupWithStatus) => {
        switch (v.status) {
            case LockupStatus.AWAITING_VAA:
                return (<>Awaiting VAA (<a onClick={() => {
                    pokeProposal(v.lockupAddress)
                }}>poke</a>)</>);
            case LockupStatus.UNCLAIMED_VAA:
                return (<Button onClick={() => {
                    executeVAA(v)
                }}>Execute</Button>);
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
                return (<>{statusToPrompt(v)}</>)
            }
        },
    ];

    return (<>
            <h3>Pending transfers</h3>
            <Table dataSource={lockups} columns={columns}/>
        </>
    )
}

export default TransferProposals
