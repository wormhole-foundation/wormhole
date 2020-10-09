import React, {useContext, useState} from 'react';
import ClientContext from "../providers/ClientContext";
import * as solanaWeb3 from '@solana/web3.js';
import {Account, Connection, PublicKey, Transaction} from '@solana/web3.js';
import {Button, message, Steps} from "antd";
import {ethers} from "ethers";
import {Erc20Factory} from "../contracts/Erc20Factory";
import {Arrayish, BigNumberish} from "ethers/utils";
import {WormholeFactory} from "../contracts/WormholeFactory";
import {BRIDGE_ADDRESS} from "../config";
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {BridgeContext} from "../providers/BridgeContext";
import {AssetMeta, SolanaBridge} from "../utils/bridge";
import KeyContext from "../providers/KeyContext";
import TransferInitiator from "../components/TransferInitiator";

const {Step} = Steps;

// @ts-ignore
window.ethereum.enable();
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

async function lockAssets(asset: string,
                          amount: BigNumberish,
                          recipient: Arrayish,
                          target_chain: BigNumberish) {
    let wh = WormholeFactory.connect(BRIDGE_ADDRESS, signer);
    try {
        message.loading({content: "Signing transaction...", key: "eth_tx", duration: 1000},)
        let res = await wh.lockAssets(asset, amount, recipient, target_chain, 10, false)
        message.loading({content: "Waiting for transaction to be mined...", key: "eth_tx", duration: 1000})
        await res.wait(1);
        message.success({content: "Transfer on ETH succeeded!", key: "eth_tx"})
    } catch (e) {
        console.log(e)
        message.error({content: "Transfer failed", key: "eth_tx"})
    }
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

async function createWrapped(c: Connection, b: SolanaBridge, key: Account, meta: AssetMeta, mint: PublicKey) {
    try {
        let tx = new Transaction();

        // @ts-ignore
        let [ix_account, newSigner] = await b.createWrappedAssetAndAccountInstructions(key.publicKey, mint);
        let recentHash = await c.getRecentBlockhash();
        tx.recentBlockhash = recentHash.blockhash
        tx.add(...ix_account)
        tx.sign(key, newSigner)
        message.loading({content: "Waiting for transaction to be confirmed...", key: "tx", duration: 1000})
        await c.sendTransaction(tx, [key, newSigner])
        message.success({content: "Creation succeeded!", key: "tx"})
    } catch (e) {
        message.error({content: "Creation failed", key: "tx"})
    }
}

function Assistant() {
    let c = useContext<solanaWeb3.Connection>(ClientContext);
    let tokenAccounts = useContext(SolanaTokenContext);
    let bridge = useContext(BridgeContext);
    let k = useContext(KeyContext);

    let [fromNetwork, setFromNetwork] = useState("eth")

    const steps = fromNetwork == "eth" ? [
        {
            title: 'Initiate Transfer',
            content: (
                <>
                    <TransferInitiator onFromNetworkChanged={setFromNetwork}/>
                </>),
        },
        {
            title: 'Wait for guardian approval',
            content: 'Second-content',
        },
        {
            title: 'Done',
            content: 'Last-content',
        },
    ] : [
        {
            title: 'Initiate Transfer',
            content: (
                <>
                    <TransferInitiator onFromNetworkChanged={setFromNetwork}/>
                </>),
        },
        {
            title: 'Wait for guardian approval',
            content: 'Second-content',
        },
        {
            title: 'Unlock tokens on Ethereum',
            content: 'Second-content',
        },
        {
            title: 'Done',
            content: 'Last-content',
        },
    ];
    let [current, setCurrent] = useState(0);

    let prevStep = () => {
        setCurrent(current - 1)
    }

    let nextStep = () => {
        setCurrent(current + 1)
    }

    return (
        <>
            <Steps current={current}>
                {steps.map(item => (
                    <Step key={item.title} title={item.title}/>
                ))}
            </Steps>
            <div className="steps-content"
                 style={{marginTop: "24px", marginBottom: "24px"}}>{steps[current].content}</div>
            <div className="steps-action">
                {current < steps.length - 1 && (
                    <Button type="primary" onClick={() => nextStep()}>
                        Next
                    </Button>
                )}
                {current === steps.length - 1 && (
                    <Button type="primary" onClick={() => message.success('Processing complete!')}>
                        Done
                    </Button>
                )}
                {current > 0 && (
                    <Button style={{margin: '0 8px'}} onClick={() => prevStep()}>
                        Previous
                    </Button>
                )}
            </div>
        </>
    );
}

export default Assistant;
