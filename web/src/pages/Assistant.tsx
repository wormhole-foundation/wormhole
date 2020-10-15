import React, {useContext, useState} from 'react';
import ClientContext from "../providers/ClientContext";
import * as solanaWeb3 from '@solana/web3.js';
import {PublicKey, Transaction} from '@solana/web3.js';
import {Button, message, Progress, Space, Spin, Steps} from "antd";
import {ethers} from "ethers";
import {Erc20Factory} from "../contracts/Erc20Factory";
import {Arrayish, BigNumber, BigNumberish} from "ethers/utils";
import {WormholeFactory} from "../contracts/WormholeFactory";
import {BRIDGE_ADDRESS, TOKEN_PROGRAM} from "../config";
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {BridgeContext} from "../providers/BridgeContext";
import KeyContext from "../providers/KeyContext";
import TransferInitiator, {defaultCoinInfo, TransferInitiatorData} from "../components/TransferInitiator";
import * as spl from "@solana/spl-token";
import BN from "bn.js"
import {SlotContext} from "../providers/SlotContext";


const {Step} = Steps;

// @ts-ignore
window.ethereum.enable();
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

interface LoadingInfo {
    loading: boolean,
    message: string,
    progress?: ProgressInfo,
}

interface ProgressInfo {
    completion: number,
    content: string
}

export enum ChainID {
    SOLANA = 1,
    ETH
}

async function approveAssets(asset: string,
                             amount: BigNumberish) {
    let e = Erc20Factory.connect(asset, signer);
    try {
        message.loading({content: "Signing transaction...", key: "eth_tx", duration: 1000})
        let res = await e.approve(BRIDGE_ADDRESS, amount)
        message.loading({content: "Waiting for transaction to be mined...", key: "eth_tx", duration: 1000})
        await res.wait(1);
        message.success({content: "Approval on ETH succeeded!", key: "eth_tx"})
    } catch (e) {
        message.error({content: "Approval failed", key: "eth_tx"})
    }
}

function Assistant() {
    let c = useContext<solanaWeb3.Connection>(ClientContext);
    let tokenAccounts = useContext(SolanaTokenContext);
    let bridge = useContext(BridgeContext);
    let k = useContext(KeyContext);
    let slot = useContext(SlotContext);

    let [fromNetwork, setFromNetwork] = useState(ChainID.ETH)
    let [transferData, setTransferData] = useState<TransferInitiatorData>({
        fromCoinInfo: defaultCoinInfo,
        fromNetwork: 0,
        toNetwork: 0,
        toAddress: new Buffer(0),
        amount: new BigNumber(0),
    });
    let [loading, setLoading] = useState<LoadingInfo>({
        loading: false,
        message: "",
        progress: undefined
    })
    let [current, setCurrent] = useState(0);

    let nextStep = (from: string) => {
        setLoading({
            ...loading,
            loading: false
        })
        if (from == "approve") {
            lockAssets(transferData.fromCoinInfo?.address, transferData.amount, transferData.toAddress, transferData.toNetwork)
        } else if (from == "lock") {
            // Await approvals or allow to submit guardian shit
            if (fromNetwork == ChainID.ETH && transferData.toNetwork == ChainID.SOLANA) {
                awaitCompletionEth()
            } else if (fromNetwork == ChainID.SOLANA && transferData.toNetwork == ChainID.ETH) {
                awaitCompletionSolana()
            }
        } else if (from == "vaa") {
            postVAAOnEth()
        }

        setCurrent((v) => v + 1)
    }

    const lockAssets = async function (asset: string,
                                       amount: BigNumberish,
                                       recipient: Arrayish,
                                       target_chain: BigNumberish) {
        let wh = WormholeFactory.connect(BRIDGE_ADDRESS, signer);
        try {
            setLoading({
                ...loading,
                loading: true,
                message: "Allow transfer in Metamask...",
            })
            let res = await wh.lockAssets(asset, amount, recipient, target_chain, 10, false)
            setLoading({
                ...loading,
                loading: true,
                message: "Waiting for transaction to be mined...",
            })
            await res.wait(1);
            message.success({content: "Transfer on ETH succeeded!", key: "eth_tx"})
            nextStep("lock");
        } catch (e) {
            message.error({content: "Transfer failed", key: "eth_tx"})
            setCurrent(0);
            setLoading({
                ...loading,
                loading: false,
            })
        }
    }

    const approveAssets = async function (asset: string,
                                          amount: BigNumberish) {
        let e = Erc20Factory.connect(asset, signer);
        try {
            setLoading({
                ...loading,
                loading: true,
                message: "Allow approval in Metamask...",
            })
            let res = await e.approve(BRIDGE_ADDRESS, amount)
            setLoading({
                ...loading,
                loading: true,
                message: "Waiting for transaction to be mined...",
            })
            await res.wait(1);
            message.success({content: "Approval on ETH succeeded!", key: "eth_tx"})
            nextStep("approve")
        } catch (e) {
            message.error({content: "Approval failed", key: "eth_tx"})
            setCurrent(0);
            setLoading({
                loading: false,
                ...loading
            })
        }
    }

    const initiateTransfer = () => {
        if (fromNetwork == ChainID.ETH && transferData.fromCoinInfo) {
            nextStep("init")
            if (transferData.fromCoinInfo?.allowance.lt(transferData.amount)) {
                approveAssets(transferData.fromCoinInfo?.address, transferData.amount)
            } else {
                lockAssets(transferData.fromCoinInfo?.address, transferData.amount, transferData.toAddress, transferData.toNetwork)
            }
        } else if (fromNetwork == ChainID.SOLANA && transferData.fromCoinInfo) {
            nextStep("init")
            solanaTransfer();
        }
    }

    let transferProposal: PublicKey;
    let transferVAA = new Uint8Array(0);
    const solanaTransfer = async () => {
        setLoading({
            ...loading,
            loading: true,
            message: "Locking tokens on Solana...",
        })

        let {ix: lock_ix, transferKey} = await bridge.createLockAssetInstruction(k.publicKey, new PublicKey(transferData.fromCoinInfo.address),
            new PublicKey(transferData.fromCoinInfo.mint), new BN(transferData.amount.toString()),
            transferData.toNetwork, transferData.toAddress,
            {
                chain: transferData.fromCoinInfo.chainID,
                address: transferData.fromCoinInfo.assetAddress,
                decimals: transferData.fromCoinInfo.decimals,
            }, Math.random() * 100000);
        let ix = spl.Token.createApproveInstruction(TOKEN_PROGRAM, new PublicKey(transferData.fromCoinInfo.address), await bridge.getConfigKey(), k.publicKey, [], transferData.amount.toNumber())

        let recentHash = await c.getRecentBlockhash();
        let tx = new Transaction();
        tx.recentBlockhash = recentHash.blockhash
        tx.add(ix)
        tx.add(lock_ix)
        tx.sign(k)
        try {
            await c.sendTransaction(tx, [k])
            message.success({content: "Transfer succeeded", key: "transfer"})
        } catch (e) {
            message.error({content: "Transfer failed", key: "transfer"})
        }
        transferProposal = transferKey
        nextStep("lock")
    }

    const executeVAAOnETH = async (vaa: Uint8Array) => {
        let wh = WormholeFactory.connect(BRIDGE_ADDRESS, signer)

        setLoading({
            ...loading,
            loading: true,
            message: "Sign the claim...",
        })
        let tx = await wh.submitVAA(vaa)
        setLoading({
            ...loading,
            loading: true,
            message: "Waiting for tokens unlock to be mined...",
        })
        await tx.wait(1)
        message.success({content: "Execution of VAA succeeded", key: "eth_tx"})
        nextStep("submit")
    }

    const awaitCompletionEth = () => {
        let startBlock = provider.blockNumber;
        let completed = false;
        let blockHandler = (blockNumber: number) => {
            if (blockNumber - startBlock < 5) {
                setLoading({
                    loading: true,
                    message: "Awaiting ETH confirmations",
                    progress: {
                        completion: (blockNumber - startBlock) / 5 * 100,
                        content: `${blockNumber - startBlock}/${5}`
                    }
                })
            } else if (!completed) {
                provider.removeListener("block", blockHandler)
                setLoading({loading: true, message: "Awaiting completion on Solana"})
            }
        }
        provider.on("block", blockHandler)

        let accountChangeListener = c.onAccountChange(new PublicKey(transferData.toAddress), () => {
            if (completed) return;

            completed = true;
            provider.removeListener("block", blockHandler)
            c.removeAccountChangeListener(accountChangeListener);
            nextStep("await")
        }, "single")
    }

    const awaitCompletionSolana = () => {
        let completed = false;
        let startSlot = slot;

        let slotUpdateListener = c.onSlotChange((slot) => {
            if (completed) return;
            if (slot.slot - startSlot < 32) {
                setLoading({
                    loading: true,
                    message: "Awaiting confirmations",
                    progress: {
                        completion: (slot.slot - startSlot) / 32 * 100,
                        content: `${slot.slot - startSlot}/${32}`
                    }
                })
            } else {
                setLoading({loading: true, message: "Awaiting guardians (TODO ping)"})
            }
        })

        let accountChangeListener = c.onAccountChange(transferProposal, async (a) => {
            if (completed) return;

            let lockup = bridge.parseLockup(transferProposal, a.data);
            let vaa = lockup.vaa;

            console.log(lockup)

            for (let i = vaa.length; i > 0; i--) {
                if (vaa[i] == 0xff) {
                    vaa = vaa.slice(0, i)
                    break
                }
            }

            // Probably a poke
            if (vaa.filter(v => v != 0).length == 0) {
                return
            }

            completed = true;
            c.removeAccountChangeListener(accountChangeListener);
            c.removeSlotChangeListener(slotUpdateListener);

            let signatures = await bridge.fetchSignatureStatus(lockup.signatureAccount);
            let sigData = Buffer.of(...signatures.reduce((previousValue, currentValue) => {
                previousValue.push(currentValue.index)
                previousValue.push(...currentValue.signature)

                return previousValue
            }, new Array<number>()))

            vaa = Buffer.concat([vaa.slice(0, 5), Buffer.of(signatures.length), sigData, vaa.slice(6)])
            transferVAA = vaa

            nextStep("vaa")
        }, "single")
    }

    const postVAAOnEth = () => {
        executeVAAOnETH(transferVAA);
    }

    const steps = [
        {
            title: 'Initiate Transfer',
            content: (
                <>
                    <TransferInitiator onFromNetworkChanged={setFromNetwork} dataChanged={(d) => {
                        setTransferData(d);
                    }}/>
                    <Button onClick={initiateTransfer}>Transfer</Button>
                </>),
        },
    ];
    if (fromNetwork == ChainID.ETH) {
        if (transferData.fromCoinInfo && transferData.fromCoinInfo.allowance.lt(transferData.amount)) {
            steps.push({
                title: 'Approval',
                content: (<></>),
            })
        }
        steps.push(...[
            {
                title: 'Transfer',
                content: (<></>),
            },
            {
                title: 'Wait for confirmations',
                content: (<></>),
            },
            {
                title: 'Done',
                content: (<><Space align="center" style={{width: "100%", paddingTop: "128px", paddingBottom: "128px"}}
                                   direction="vertical">
                    <Progress type="circle" percent={100} format={() => 'Done'}/>
                    <b>Your transfer has been completed</b>
                </Space></>),
            },
        ])
    } else {
        steps.push(...[
            {
                title: 'Transfer',
                content: (<></>),
            },
            {
                title: 'Wait for approval',
                content: (<></>),
            },
            {
                title: 'Claim tokens on ETH',
                content: (<></>),
            },
            {
                title: 'Done',
                content: (<><Space align="center" style={{width: "100%", paddingTop: "128px", paddingBottom: "128px"}}
                                   direction="vertical">
                    <Progress type="circle" percent={100} format={() => 'Done'}/>
                    <b>Your transfer has been completed</b>
                </Space></>),
            },
        ])
    }

    return (
        <>
            <Steps current={current}>
                {steps.map(item => (
                    <Step key={item.title} title={item.title}/>
                ))}
            </Steps>
            <div className="steps-content"
                 style={{marginTop: "24px", marginBottom: "24px"}}>
                {loading.loading ? loading.progress ? (
                        <Space align="center" style={{width: "100%", paddingTop: "128px", paddingBottom: "128px"}}
                               direction="vertical">
                            <ProgressIndicator {...loading.progress}/>
                            <b>{loading.message}</b>
                        </Space>) :
                    <Space align="center" style={{width: "100%", paddingTop: "128px", paddingBottom: "128px"}}
                           direction="vertical">
                        <Spin size={"large"}/>
                        <b>{loading.message}</b>
                    </Space> : steps[current].content}
            </div>

        </>
    );
}

let ProgressIndicator = (params: { completion: number, content: string }) => {
    return (<Progress type="circle" percent={params.completion} format={() => params.content}/>)
}

export default Assistant;
