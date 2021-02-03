import React, {useContext, useEffect, useState} from "react";
import {Button, Empty, Form, Input, message, Modal, Select} from "antd";
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
import {ChainID} from "../pages/Assistant";
import WalletContext from "../providers/WalletContext";
import Wallet from "@project-serum/sol-wallet-adapter";

const {confirm} = Modal;

const {Option} = Select;

interface TransferInitiatorParams {
    onFromNetworkChanged?: (v: ChainID) => void
    dataChanged?: (d: TransferInitiatorData) => void
}

export interface CoinInfo {
    address: string,
    name: string,
    balance: BigNumber,
    decimals: number,
    allowance: BigNumber,
    isWrapped: boolean,
    chainID: number,
    assetAddress: Buffer,
    mint: string,
}

export interface TransferInitiatorData {
    fromNetwork: ChainID,
    fromCoinInfo: CoinInfo
    toNetwork: ChainID,
    toAddress: Buffer,
    amount: BigNumber,
}

// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

export const defaultCoinInfo = {
    address: "",
    name: "",
    balance: new BigNumber(0),
    decimals: 0,
    allowance: new BigNumber(0),
    isWrapped: false,
    chainID: 0,
    assetAddress: new Buffer(0),
    mint: ""
}

let debounceUpdater = debounce((e) => e(), 500)

async function createWrapped(c: Connection, b: SolanaBridge, wallet: Wallet, meta: AssetMeta, mint: PublicKey) {
    try {
        let tx = new Transaction();

        // @ts-ignore
        let [ix_account, newSigner] = await b.createWrappedAssetAndAccountInstructions(wallet.publicKey, mint, meta);
        let recentHash = await c.getRecentBlockhash();
        tx.recentBlockhash = recentHash.blockhash
        tx.add(...ix_account)
        tx.feePayer = wallet.publicKey;
        tx.sign(newSigner);
        let signed = await wallet.signTransaction(tx);
        message.loading({content: "Waiting for transaction to be confirmed...", key: "tx", duration: 1000})
        await c.sendRawTransaction(signed.serialize(), {preflightCommitment: "single"})
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
    let wallet = useContext(WalletContext);

    let [fromNetwork, setFromNetwork] = useState(ChainID.ETH);
    let [toNetwork, setToNetwork] = useState(ChainID.SOLANA);
    let [fromAddress, setFromAddress] = useState("");
    let [fromAddressValid, setFromAddressValid] = useState(false)
    let [coinInfo, setCoinInfo] = useState<CoinInfo>(defaultCoinInfo);
    let [toAddress, setToAddress] = useState("");
    let [toAddressValid, setToAddressValid] = useState(false)
    let [amount, setAmount] = useState(new BigNumber(0));
    let [amountValid, setAmountValid] = useState(true);

    let [wrappedMint, setWrappedMint] = useState("")

    const updateBalance = async () => {
        if (fromNetwork == ChainID.SOLANA) {
            let acc = b.balances.find(value => value.account.toString() == fromAddress)
            if (!acc) {
                setFromAddressValid(false);
                setCoinInfo(defaultCoinInfo);
                return
            }

            setCoinInfo({
                address: fromAddress,
                name: "",
                balance: acc.balance,
                allowance: new BigNumber(0),
                decimals: acc.assetMeta.decimals,
                isWrapped: acc.assetMeta.chain != ChainID.SOLANA,
                chainID: acc.assetMeta.chain,
                assetAddress: acc.assetMeta.address,

                // Solana specific
                mint: acc.mint,
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
                    address: fromAddress,
                    name: symbol,
                    balance: balance,
                    allowance: allowance,
                    decimals: decimals,
                    isWrapped: false,
                    chainID: 2,
                    assetAddress: new Buffer(fromAddress.slice(2), "hex"),
                    mint: "",
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
                    decimals: Math.min(decimals, 9),
                });
                console.log(decimals)

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
        if (toNetwork == ChainID.ETH) {
            setToAddressValid(toAddress.length == 42 && toAddress.match(/0[xX][0-9a-fA-F]+/) != null)
        } else {
            setToAddressValid(toAddress != "")
        }
    }, [toNetwork, toAddress])

    useEffect(() => {
        setAmountValid(amount.lte(coinInfo.balance) && amount.gt(0))
    }, [amount])

    useEffect(() => {
        if (params.dataChanged) {
            params.dataChanged({
                fromCoinInfo: coinInfo,
                fromNetwork,
                toNetwork,
                toAddress: toAddressValid ? (toNetwork == ChainID.ETH ? new Buffer(toAddress.slice(2), "hex") : new PublicKey(toAddress).toBuffer()) : new Buffer(0),
                amount: amount,
            });
        }
    }, [fromNetwork, fromAddressValid, coinInfo, toNetwork, toAddress, toAddressValid, amount])

    return (
        <>
            <Form layout={"vertical"}>
                <Form.Item label="From" name="layout" validateStatus={fromAddressValid ? "success" : "error"}>
                    <Input.Group compact={true}>
                        <Select style={{width: '30%'}} defaultValue={ChainID.ETH} className="select-before"
                                value={fromNetwork}
                                onChange={(v: ChainID) => {
                                    setFromNetwork(v);
                                    setFromAddress("");
                                    if (v === toNetwork) {
                                        setToNetwork(v == ChainID.ETH ? ChainID.SOLANA : ChainID.ETH);
                                    }
                                    if (params.onFromNetworkChanged) params.onFromNetworkChanged(v);
                                }}>
                            <Option value={ChainID.ETH}>Ethereum</Option>
                            <Option value={ChainID.SOLANA}>Solana</Option>
                        </Select>
                        {fromNetwork == ChainID.ETH &&

                        <Input style={{width: '70%'}} placeholder="ERC20 address"
                               onChange={(e) => setFromAddress(e.target.value)}
                               suffix={coinInfo.name}/>}
                        {fromNetwork == ChainID.SOLANA &&
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
                           addonAfter={`Balance: ${new BN(coinInfo.balance.toString()).div(new BN(10).pow(coinInfo.decimals))}`}
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
                        <Select style={{width: '30%'}} defaultValue={ChainID.SOLANA} className="select-before"
                                value={toNetwork}
                                onChange={(v: ChainID) => {
                                    setToNetwork(v)
                                    if (v === fromNetwork) {
                                        setFromNetwork(v == ChainID.ETH ? ChainID.SOLANA : ChainID.ETH);
                                    }
                                    setToAddress("");
                                }}>
                            <Option value={ChainID.ETH}>Ethereum</Option>
                            <Option value={ChainID.SOLANA}>Solana</Option>
                        </Select>
                        {toNetwork == ChainID.ETH &&

                        <Input style={{width: '70%'}} placeholder="Account address"
                               onChange={(e) => setToAddress(e.target.value)}/>}
                        {toNetwork == ChainID.SOLANA &&
                        <>
                            <Select style={{width: '60%'}} onChange={(e) => setToAddress(e.toString())}
                                    placeholder="Pick a token account or create a new one"
                                    notFoundContent={<Empty description="No accounts. Create a new one."/>}>
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
                                        createWrapped(c, bridge, wallet, {
                                            chain: coinInfo.chainID,
                                            address: coinInfo.assetAddress,
                                            decimals: Math.min(coinInfo.decimals, 9)
                                        }, new PublicKey(wrappedMint))
                                    },
                                    onCancel() {
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
