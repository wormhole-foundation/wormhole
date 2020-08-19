import React, {useContext, useEffect, useState} from 'react';
import ClientContext from "../providers/ClientContext";
import * as solanaWeb3 from '@solana/web3.js';
import {Account, Connection, PublicKey, Transaction} from '@solana/web3.js';
import {Button, Col, Form, Input, InputNumber, message, Row, Select, Space} from "antd";
import {ethers} from "ethers";
import {Erc20Factory} from "../contracts/Erc20Factory";
import {Arrayish, BigNumber, BigNumberish} from "ethers/utils";
import {WormholeFactory} from "../contracts/WormholeFactory";
import {WrappedAssetFactory} from "../contracts/WrappedAssetFactory";
import {BRIDGE_ADDRESS} from "../config";
import {SlotContext} from "../providers/SlotContext";
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {BridgeContext} from "../providers/BridgeContext";
import SplBalances from "../components/SplBalances";
import {AssetMeta, SolanaBridge} from "../utils/bridge";
import KeyContext from "../providers/KeyContext";


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
        let res = await wh.lockAssets(asset, amount, recipient, target_chain)
        message.loading({content: "Waiting for transaction to be mined...", key: "eth_tx", duration: 1000})
        await res.wait(1);
        message.success({content: "Transfer on ETH succeeded!", key: "eth_tx"})
    } catch (e) {
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
        let mintAccount = await c.getAccountInfo(mint);
        if (!mintAccount) {
            let ix = await b.createWrappedAssetInstruction(key.publicKey, meta);
            tx.add(ix)
        }

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

function Transfer() {
    let c = useContext<solanaWeb3.Connection>(ClientContext);
    let tokenAccounts = useContext(SolanaTokenContext);
    let slot = useContext(SlotContext);
    let bridge = useContext(BridgeContext);
    let k = useContext(KeyContext);

    let [coinInfo, setCoinInfo] = useState({
        balance: new BigNumber(0),
        decimals: 0,
        allowance: new BigNumber(0),
        isWrapped: false,
        chainID: 0,
        assetAddress: new Buffer("")
    });
    let [amount, setAmount] = useState(0);
    let [address, setAddress] = useState("");
    let [addressValid, setAddressValid] = useState(false)
    let [solanaAccount, setSolanaAccount] = useState(false)
    let [wrappedMint, setWrappedMint] = useState("")

    useEffect(() => {
        fetchBalance(address)
    }, [address])

    useEffect(() => {
        if (!addressValid) {
            setWrappedMint("")
            setSolanaAccount(false)
            return
        }

        let getWrappedInfo = async () => {
            let wrappedMint = await bridge.getWrappedAssetMint({
                chain: coinInfo.chainID,
                address: coinInfo.assetAddress
            });
            setWrappedMint(wrappedMint.toString())

            for (let account of tokenAccounts.balances) {
                if (account.mint == wrappedMint.toString()) {
                    setSolanaAccount(true)
                    return;
                }
            }
            setSolanaAccount(false)
        }
        getWrappedInfo();
    }, [address, addressValid, tokenAccounts, bridge])

    async function fetchBalance(token: string) {
        try {
            let e = WrappedAssetFactory.connect(token, provider);
            let addr = await signer.getAddress();
            let balance = await e.balanceOf(addr);
            let decimals = await e.decimals();
            let allowance = await e.allowance(addr, BRIDGE_ADDRESS);

            let info = {
                balance: balance.div(new BigNumber(10).pow(decimals)),
                allowance: allowance.div(new BigNumber(10).pow(decimals)),
                decimals: decimals,
                isWrapped: false,
                chainID: 2,
                assetAddress: new Buffer(token.slice(2), "hex")
            }

            let b = WormholeFactory.connect(BRIDGE_ADDRESS, provider);

            let isWrapped = await b.isWrappedAsset(token)
            if (isWrapped) {
                info.chainID = await e.assetChain()
                info.assetAddress = new Buffer((await e.assetAddress()).slice(2), "hex")
                info.isWrapped = true
            }
            setCoinInfo(info)
            setAddressValid(true)
        } catch (e) {
            setAddressValid(false)
        }
    }

    return (
        <>
            <p>Slot: {slot}</p>
            <Row>
                <Col span={12}>
                    <Form onFinish={(values) => {
                        let recipient = new solanaWeb3.PublicKey(values["recipient"]).toBuffer()
                        let transferAmount = new BigNumber(values["amount"]).mul(new BigNumber(10).pow(coinInfo.decimals));
                        if (coinInfo.allowance.toNumber() >= amount || coinInfo.isWrapped) {
                            lockAssets(values["address"], transferAmount, recipient, values["target_chain"])
                        } else {
                            approveAssets(values["address"], transferAmount)
                        }
                    }} style={{width: "100%"}}>
                        <Form.Item name="address" validateStatus={addressValid ? "success" : "error"}>
                            <Input addonAfter={`Balance: ${coinInfo.balance}`} name="address"
                                   placeholder={"ERC20 address"}
                                   onBlur={(v) => {
                                       setAddress(v.target.value)
                                   }}/>
                        </Form.Item>
                        <Form.Item name="amount" rules={[{
                            required: true, validator: (rule, value, callback) => {
                                let big = new BigNumber(value);
                                callback(big.lte(coinInfo.balance) ? undefined : "Amount exceeds balance")
                            }
                        }]}>
                            <InputNumber name={"amount"} placeholder={"Amount"} type={"number"} onChange={value => {
                                // @ts-ignore
                                setAmount(value || 0)
                            }}/>
                        </Form.Item>
                        <Form.Item name="target_chain"
                                   rules={[{required: true, message: "Please choose a target chain"}]}>
                            <Select placeholder="Target Chain">
                                <Select.Option value={1}>
                                    Solana
                                </Select.Option>
                            </Select>
                        </Form.Item>
                        <Form.Item name="recipient" validateStatus={solanaAccount ? "success" : "error"}>
                            <Input name="recipient" placeholder={"Address of the recipient"}/>
                        </Form.Item>
                        <Form.Item>
                            <Button type="primary" htmlType="submit">
                                {coinInfo.allowance.toNumber() >= amount || coinInfo.isWrapped ? "Transfer" : "Approve"}
                            </Button>
                        </Form.Item>
                    </Form>
                </Col>
            </Row>
            <Col>
                <Space><Button disabled={!addressValid} onClick={() => {
                    createWrapped(c, bridge, k, {
                        chain: coinInfo.chainID,
                        address: coinInfo.assetAddress
                    }, new PublicKey(wrappedMint))
                }}>Create Wrapped account</Button>
                    <p>Mint: {wrappedMint}</p></Space>
                <SplBalances/>
            </Col>
        </>
    );
}

export default Transfer;
