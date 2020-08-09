import React, {useContext, useEffect, useState} from 'react';
import ClientContext from "../providers/ClientContext";
import * as solanaWeb3 from '@solana/web3.js';
import {Button, Input, InputNumber, Space} from "antd";
import {ethers} from "ethers";
import {Erc20Factory} from "../contracts/Erc20Factory";
import {BigNumber} from "ethers/utils";


// @ts-ignore
window.ethereum.enable();
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

function Transfer() {
    let c = useContext<solanaWeb3.Connection>(ClientContext);

    let [token, setToken] = useState("");
    let [balance, setBalance] = useState("0");
    let [slot, setSlot] = useState(0);
    useEffect(() => {
        c.onSlotChange(value => {
            setSlot(value.slot);
        });
    })
    useEffect(() => {
        async function fetchBalance(){
            let e = Erc20Factory.connect(token, provider);
            try {
                let addr = await signer.getAddress();
                let balance = await e.balanceOf(addr);
                let decimals = await e.decimals();
                setBalance(balance.div(new BigNumber(10).pow(decimals)).toString());
            }catch (e) {

            }
        }
        fetchBalance();
    }, [token])
    return (
        <>
            <p>Slot: {slot}</p>
            <Space>
                <Input.Group>
                    <Input addonAfter={`Balance: ${balance}`} name={"abc"} placeholder={"ERC20 address"}
                           onChange={(e) => {
                                setToken(e.target.value);
                           }}/>
                    <InputNumber name={"amount"} placeholder={"Amount"} type={"number"}/>
                </Input.Group>
                <Button type="primary">Transfer</Button>
            </Space>
        </>
    );
}

export default Transfer;
