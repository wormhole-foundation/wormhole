import React, {useContext, useEffect, useState} from "react";
import {Button, Form, Input, message, Modal, Select} from "antd";
import solanaWeb3, {Account, Connection, PublicKey, Transaction} from "@solana/web3.js";
import ClientContext from "../providers/ClientContext";
import {SlotContext} from "../providers/SlotContext";
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {BridgeContext} from "../providers/BridgeContext";
import {WrappedAssetFactory} from "../contracts/WrappedAssetFactory";
import {WalletOutlined} from '@ant-design/icons';
import {BRIDGE_ADDRESS} from "../config";
import {WormholeFactory} from "../contracts/WormholeFactory";
import {ethers} from "ethers";
import debounce from "lodash.debounce"
import BN from "bignumber.js";
import {BigNumber} from "ethers/utils";
import {AssetMeta, SolanaBridge} from "../utils/bridge";
import KeyContext from "../providers/KeyContext";

const {confirm} = Modal;

const {Option} = Select;

interface TransferInitiatorParams {
    onFromNetworkChanged?: (v: string) => void
}

// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

const defaultCoinInfo = {
    name: "",
    balance: new BigNumber(0),
    decimals: 0,
    allowance: new BigNumber(0),
    isWrapped: false,
    chainID: 0,
    assetAddress: new Buffer(0),
}

let debounceUpdater = debounce((e) => e(), 500)

async function createWrapped(c: Connection, b: SolanaBridge, key: Account, meta: AssetMeta, mint: PublicKey) {
    try {
        let tx = new Transaction();

        // @ts-ignore
        let [ix_account, newSigner] = await b.createWrappedAssetAndAccountInstructions(key.publicKey, mint, meta);
        let recentHash = await c.getRecentBlockhash();
        tx.recentBlockhash = recentHash.blockhash
        tx.add(...ix_account)
        tx.sign(key, newSigner)
        message.loading({content: "Waiting for transaction to be confirmed...", key: "tx", duration: 1000})
        await c.sendTransaction(tx, [key, newSigner])
        message.success({content: "Creation succeeded!", key: "tx"})
    } catch (e) {
        console.log(e)
        message.error({content: "Creation failed", key: "tx"})
    }
}

export default function TransferInitiator(params: TransferInitiatorParams) {
    let c = useContext<solanaWeb3.Connection>(ClientContext);
    let slot = useContext(SlotContext);
    let b = useContext(SolanaTokenContext);
    let bridge = useContext(BridgeContext);
    let k = useContext(KeyContext);

    let [fromNetwork, setFromNetwork] = useState("eth");
    let [toNetwork, setToNetwork] = useState("solana");
    let [fromAddress, setFromAddress] = useState("");
    let [fromAddressValid, setFromAddressValid] = useState(true)
    let [coinInfo, setCoinInfo] = useState(defaultCoinInfo);
    let [toAddress, setToAddress] = useState("");
    let [toAddressValid, setToAddressValid] = useState(true)
    let [amount, setAmount] = useState(new BigNumber(0));
    let [amountValid, setAmountValid] = useState(true);

    let [wrappedMint, setWrappedMint] = useState("")

    const updateBalance = async () => {
        if (fromNetwork == "solana") {
            let acc = b.balances.find(value => value.account.toString() == fromAddress)
            if (!acc) {
                setFromAddressValid(false);
                setCoinInfo(defaultCoinInfo);
                return
            }

            setCoinInfo({
                name: "",
                balance: acc.balance,
                allowance: new BigNumber(0),
                decimals: acc.decimals,
                isWrapped: false,
                chainID: 1,
                assetAddress: new PublicKey(fromAddress).toBuffer(),
            })
            setFromAddressValid(true);
        } else {
            try {
                let e = WrappedAssetFactory.connect(fromAddress, provider);
                let addr = await signer.getAddress();
                let balance = await e.balanceOf(addr);
                let decimals = await e.decimals();
                let symbol = await e.symbol();
                let allowance = await e.allowance(addr, BRIDGE_ADDRESS);

                let info = {
                    name: symbol,
                    balance: balance,
                    allowance: allowance,
                    decimals: decimals,
                    isWrapped: false,
                    chainID: 2,
                    assetAddress: new Buffer(fromAddress.slice(2), "hex")
                }

                let b = WormholeFactory.connect(BRIDGE_ADDRESS, provider);

                let isWrapped = await b.isWrappedAsset(fromAddress)
                if (isWrapped) {
                    info.chainID = await e.assetChain()
                    info.assetAddress = new Buffer((await e.assetAddress()).slice(2), "hex")
                    info.isWrapped = true
                }

                let wrappedMint = await bridge.getWrappedAssetMint({
                    chain: info.chainID,
                    address: info.assetAddress,
                    decimals: Math.min(decimals, 8),
                });

                setWrappedMint(wrappedMint.toString())
                setCoinInfo(info)
                setFromAddressValid(true)
            } catch (e) {
                setCoinInfo(defaultCoinInfo);
                setFromAddressValid(false)
            }
        }
    }
    useEffect(() => {
        debounceUpdater(updateBalance)
    }, [fromNetwork, fromAddress])

    useEffect(() => {
        if (toNetwork == "eth") {
            setToAddressValid(toAddress.length == 42 && toAddress.match(/0[xX][0-9a-fA-F]+/) != null)
        } else {

        }
    }, [toNetwork, toAddress])

    useEffect(() => {
        setAmountValid(amount.lte(coinInfo.balance) && amount.gt(0))
    }, [amount])

    return (
        <>
            <Form layout={"vertical"}>
                <Form.Item label="From" name="layout" validateStatus={fromAddressValid ? "success" : "error"}>
                    <Input.Group compact={true}>
                        <Select style={{width: '30%'}} defaultValue="eth" className="select-before"
                                value={fromNetwork}
                                onChange={(v) => {
                                    setFromNetwork(v);
                                    if (v === toNetwork) {
                                        setToNetwork(v == "eth" ? "solana" : "eth");
                                    }
                                    if (params.onFromNetworkChanged) params.onFromNetworkChanged(v);
                                }}>
                            <Option value="eth">Ethereum</Option>
                            <Option value="solana">Solana</Option>
                        </Select>
                        {fromNetwork == "eth" &&

                        <Input style={{width: '70%'}} placeholder="ERC20 address"
                               onChange={(e) => setFromAddress(e.target.value)}
                               suffix={coinInfo.name}/>}
                        {fromNetwork == "solana" &&
                        <>
                            <Select style={{width: '70%'}} placeholder="Pick a token account"
                                    onChange={(e) => {
                                        setFromAddress(e.toString())
                                    }}>
                                {b.balances.map((v) => <Option
                                    value={v.account.toString()}>{v.account.toString()}</Option>)}
                            </Select>
                        </>
                        }
                    </Input.Group>
                </Form.Item>
                <Form.Item label="Amount" name="layout"
                           validateStatus={amountValid ? "success" : "error"}>
                    <Input type={"number"} placeholder={"Amount"}
                           addonAfter={`Balance: ${coinInfo.balance.div(Math.pow(10, coinInfo.decimals))}`}
                           onChange={(v) => {
                               if (v.target.value === "") {
                                   setAmount(new BigNumber(0));
                                   return
                               }
                               setAmount(new BigNumber(new BN(v.target.value).multipliedBy(new BN(Math.pow(10, coinInfo.decimals))).toFixed(0)))
                           }}/>
                </Form.Item>
                <Form.Item label="Recipient" name="layout" validateStatus={toAddressValid ? "success" : "error"}>
                    <Input.Group compact={true}>
                        <Select style={{width: '30%'}} defaultValue="solana" className="select-before" value={toNetwork}
                                onChange={(v) => {
                                    setToNetwork(v)
                                    if (v === fromNetwork) {
                                        setFromNetwork(v == "eth" ? "solana" : "eth");
                                    }
                                }}>
                            <Option value="eth">Ethereum</Option>
                            <Option value="solana">Solana</Option>
                        </Select>
                        {toNetwork == "eth" &&

                        <Input style={{width: '70%'}} placeholder="Account address"
                               onChange={(e) => setToAddress(e.target.value)}/>}
                        {toNetwork == "solana" &&
                        <>
                            <Select style={{width: '60%'}} placeholder="Pick a token account or create a new one">
                                {b.balances.filter((v) => v.mint == wrappedMint).map((v) =>
                                    <Option
                                        value={v.account.toString()}>{v.account.toString()}</Option>)}
                            </Select>
                            <Button style={{width: '10%'}} disabled={!fromAddressValid} onClick={() => {
                                confirm({
                                    title: 'Do you want to create a new token account?',
                                    icon: <WalletOutlined/>,
                                    content: (<>This will create a new token account for the
                                        token: <code>{wrappedMint}</code></>),
                                    onOk() {
                                        createWrapped(c, bridge, k, {
                                            chain: coinInfo.chainID,
                                            address: coinInfo.assetAddress,
                                            decimals: Math.min(coinInfo.decimals, 8)
                                        }, new PublicKey(wrappedMint))
                                    },
                                    onCancel() {
                                        console.log('Cancel');
                                    },
                                })
                            }}>+</Button>
                        </>
                        }
                    </Input.Group>
                </Form.Item>
            </Form>
        </>
    );
}
