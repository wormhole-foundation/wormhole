import React, { useEffect } from 'react';
import { Typography } from 'antd'
const { Title } = Typography

export type IWASMModule = typeof import("bridge")

function convertbase64ToBinary(base64: string) {
    var raw = window.atob(base64);
    var rawLength = raw.length;
    var array = new Uint8Array(new ArrayBuffer(rawLength));

    for (let i = 0; i < rawLength; i++) {
        array[i] = raw.charCodeAt(i);
    }
    return array;
}

interface WasmProps {
    base64VAA: string
}

const WasmTest = (props: WasmProps) => {


    const loadWasm = async (base64VAA: string) => {
        const vaa = convertbase64ToBinary(base64VAA)
        try {
            /*eslint no-useless-concat: "off"*/
            const wasm = await import('bridge')
            // debugger
            const parsed = wasm.parse_vaa(vaa)
            console.log('parsed vaa: ', parsed)
            // let addr = 'Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o'
            // let res = wasm.state_address(addr)
            // console.log('res', res)
            // alert('it worked.')
            // debugger
        } catch (err) {
            debugger
            console.error(`Unexpected error in loadWasm. [Message: ${err.message}]`)
        }
    }
    useEffect(() => {
        if (props.base64VAA) {
            loadWasm(props.base64VAA)
        }
    }, [props])

    return <Title level={3}>wasm test</Title>
}

export default WasmTest
