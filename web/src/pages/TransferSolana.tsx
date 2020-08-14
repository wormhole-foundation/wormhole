import React, {useContext, useEffect, useState} from 'react';
import ClientContext from "../providers/ClientContext";
import * as solanaWeb3 from '@solana/web3.js';
import {Button, Col, Form, Input, InputNumber, Row, Select, Space} from "antd";
import {BigNumber} from "ethers/utils";
import SplBalances from "../components/SplBalances";
import {SlotContext} from "../providers/SlotContext";
import {SolanaTokenContext} from "../providers/SolanaTokenContext";

function TransferSolana() {
    let c = useContext<solanaWeb3.Connection>(ClientContext);
    let slot = useContext(SlotContext);
    let b = useContext(SolanaTokenContext);

    let [coinInfo, setCoinInfo] = useState({
        balance: new BigNumber(0),
        decimals: 0,
        allowance: new BigNumber(0),
        isWrapped: false,
        chainID: 0,
        wrappedAddress: ""
    });
    let [amount, setAmount] = useState(0);
    let [address, setAddress] = useState("");
    let [addressValid, setAddressValid] = useState(false)

    useEffect(() => {
        async function getCoinInfo(): Promise<BigNumber> {
            let acc = b.balances.find(value => value.account.toString() == address)
            if (!acc) {
                return new BigNumber(0)
            }

            return acc.balance
        }
    }, [address])

    return (
        <>
            <p>Slot: {slot}</p>
            <Row>
                <Col>
                    <Space>
                        <Form onFinish={(values) => {
                            let recipient = new solanaWeb3.PublicKey(values["recipient"]).toBuffer()
                            let transferAmount = new BigNumber(values["amount"]).mul(new BigNumber(10).pow(coinInfo.decimals));
                            if (coinInfo.allowance.toNumber() >= amount || coinInfo.isWrapped) {
                                //lockAssets(values["address"], transferAmount, recipient, values["target_chain"])
                            } else {
                                //approveAssets(values["address"], transferAmount)
                            }
                        }}>
                            <Form.Item name="address" validateStatus={addressValid ? "success" : "error"}>
                                <Input addonAfter={`Balance: ${coinInfo.balance}`} name="address"
                                       placeholder={"Token account Pubkey"}
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
                                    {coinInfo.allowance.toNumber() >= amount || coinInfo.isWrapped ? "Transfer" : "Approve"}
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
