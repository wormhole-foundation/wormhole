import React, { useState, useEffect } from 'react';
import { Form, Input, Button, List } from 'antd';
import { ethers } from "ethers";
import { BRIDGE_ADDRESS } from "../config";
import { ImplementationFactory } from '../contracts/ImplementationFactory';

// @ts-ignore
if (window.ethereum === undefined) {
    alert("Please install the MetaMask extension before using this experimental demo web UI");
}

// @ts-ignore
window.ethereum.enable();
// @ts-ignore
const provider = new ethers.providers.Web3Provider(window.ethereum);
const signer = provider.getSigner();

function Message() {
    const [form] = Form.useForm();
    const [, forceUpdate] = useState({});

    // map: { txHash: payloadString }
    const [txHashToPayload, setTxHashToPayload] = useState<{ [txHash: string]: string }>({})

    // To disable submit button at the beginning.
    useEffect(() => {
        forceUpdate({});
    }, []);

    const sendMessage = async ({ payload }: { payload: string }) => {

        let nonceConst = Math.random() * 100000
        let nonceBuffer = Buffer.alloc(4);
        nonceBuffer.writeUInt32LE(nonceConst, 0)

        let i = ImplementationFactory.connect(BRIDGE_ADDRESS, signer)

        let res = await i.publishMessage(nonceBuffer, Buffer.from(payload, 'utf16le'), true)

        await res.wait(1)

        if (res.hash) {
            setTxHashToPayload({ ...txHashToPayload, [res.hash]: payload })
        }

        form.resetFields(['payload'])
    }
    const rmTxHash = (txHash: string) => {
        const { [txHash]: rm, ...others } = txHashToPayload
        setTxHashToPayload(others)
        return undefined  // for typescript
    }

    return (
        <>
            <Form form={form} name="publish_message" layout="inline" onFinish={sendMessage}>
                <Form.Item>
                    <h1><code>publishMessage</code></h1>
                </Form.Item>
                <Form.Item
                    name="payload"
                    rules={[{ required: true, message: 'Please enter a payload for the message.' }]}
                >
                    <Input.TextArea placeholder="Payload to write to ETH" />
                </Form.Item>
                <Form.Item shouldUpdate>
                    {() => (
                        <Button
                            type="primary"
                            htmlType="submit"
                            disabled={
                                !form.isFieldsTouched(true) ||
                                !!form.getFieldsError().filter(({ errors }) => errors.length).length
                            }
                        >
                            Send to MetaMask
                        </Button>
                    )}
                </Form.Item>
            </Form>
            {Object.keys(txHashToPayload).length >= 1 ? (
                <List
                    dataSource={Object.keys(txHashToPayload)}
                    renderItem={item => (
                        <List.Item
                            actions={[<a onClick={() => rmTxHash(item)} >X</a>]}
                        >
                            <div style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
                                <h4><code>{item}</code></h4>
                                <h4><code><pre>{txHashToPayload[item]}</pre></code></h4>
                            </div>
                        </List.Item>
                    )}
                />

            ) : null}
        </>
    );
}

export default Message;
