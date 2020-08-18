import React, {useContext, useEffect, useState} from 'react';
import ClientContext from "../providers/ClientContext";
import * as solanaWeb3 from '@solana/web3.js';
import {PublicKey, Transaction} from '@solana/web3.js';
import * as spl from '@solana/spl-token';
import {Button, Col, Form, Input, InputNumber, message, Row, Select, Space} from "antd";
import {BigNumber} from "ethers/utils";
import SplBalances from "../components/SplBalances";
import {SlotContext} from "../providers/SlotContext";
import {SolanaTokenContext} from "../providers/SolanaTokenContext";
import {CHAIN_ID_SOLANA} from "../utils/bridge";
import {BridgeContext} from "../providers/BridgeContext";
import KeyContext from "../providers/KeyContext";
import BN from 'bn.js';
import {TOKEN_PROGRAM} from "../config";

function TransferSolana() {
    let c = useContext<solanaWeb3.Connection>(ClientContext);
    let slot = useContext(SlotContext);
    let b = useContext(SolanaTokenContext);
    let bridge = useContext(BridgeContext);
    let k = useContext(KeyContext);

    let [coinInfo, setCoinInfo] = useState({
        balance: new BigNumber(0),
        decimals: 0,
        isWrapped: false,
        chainID: 0,
        wrappedAddress: new Buffer([]),
        mint: ""
    });
    let [amount, setAmount] = useState(new BigNumber(0));
    let [address, setAddress] = useState("");
    let [addressValid, setAddressValid] = useState(false)

    useEffect(() => {
        async function getCoinInfo() {
            let acc = b.balances.find(value => value.account.toString() == address)
            if (!acc) {
                setAmount(new BigNumber(0));
                setAddressValid(false)
                return
            }

            setCoinInfo({
                balance: acc.balance,
                decimals: acc.decimals,
                isWrapped: acc.assetMeta.chain != CHAIN_ID_SOLANA,
                chainID: acc.assetMeta.chain,
                wrappedAddress: acc.assetMeta.address,
                mint: acc.mint
            })
            setAddressValid(true)
        }

        getCoinInfo()
    }, [address])

    return (
        <>
            <p>Slot: {slot}</p>
            <Row>
                <Col>
                    <Space>
                        <Form onFinish={(values) => {
                            let recipient = new Buffer(values["recipient"].slice(2), "hex");

                            let transferAmount = new BN(values["amount"]).mul(new BN(10).pow(new BN(coinInfo.decimals)));
                            let fromAccount = new PublicKey(values["address"])

                            let send = async () => {
                                message.loading({content: "Transferring tokens...", key: "transfer"}, 1000)

                                let lock_ix = await bridge.createLockAssetInstruction(k.publicKey, fromAccount, new PublicKey(coinInfo.mint), transferAmount, values["target_chain"], recipient,
                                    {
                                        chain: coinInfo.chainID,
                                        address: coinInfo.wrappedAddress
                                    }, 2);
                                let ix = spl.Token.createApproveInstruction(new PublicKey(TOKEN_PROGRAM), fromAccount, await bridge.getConfigKey(), k.publicKey, [], transferAmount.toNumber())

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
                            }
                            send()
                        }}>
                            <Form.Item name="address" validateStatus={addressValid ? "success" : "error"}>
                                <Input
                                    addonAfter={`Balance: ${coinInfo.balance.div(new BigNumber(Math.pow(10, coinInfo.decimals)))}`}
                                    name="address"
                                    placeholder={"Token account Pubkey"}
                                    onBlur={(v) => {
                                        setAddress(v.target.value)
                                    }}/>
                            </Form.Item>
                            <Form.Item name="amount" rules={[{
                                required: true, validator: (rule, value, callback) => {
                                    let big = new BigNumber(value).mul(new BigNumber(10).pow(coinInfo.decimals));
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
                                    <Select.Option value={2}>
                                        Ethereum
                                    </Select.Option>
                                </Select>
                            </Form.Item>
                            <Form.Item name="recipient" rules={[{
                                required: true,
                                validator: (rule, value, callback) => {
                                    if (value.length !== 42 || value.indexOf("0x") != 0) {
                                        callback("Invalid address")
                                    } else {
                                        callback()
                                    }
                                }
                            },]}>
                                <Input name="recipient" placeholder={"Address of the recipient"}/>
                            </Form.Item>
                            <Form.Item>
                                <Button type="primary" htmlType="submit">
                                    Transfer
                                </Button>
                            </Form.Item>
                        </Form>
                    </Space>
                </Col>
                <Col>
                    <SplBalances/>
                </Col>
            </Row>
        </>
    );
}

export default TransferSolana;
