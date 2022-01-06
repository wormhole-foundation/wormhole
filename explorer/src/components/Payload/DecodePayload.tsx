
import { BigNumber } from "ethers";
import React, { useEffect, useState } from "react";
import { chainEnums, ChainIDs, chainIDs, METADATA_REPLACE } from "~/utils/misc/constants";


import { Statistic, Typography } from 'antd'
import { FormattedMessage } from "gatsby-plugin-intl";
import { titleStyles } from "~/styles";
import { TransferDetails } from "../ExplorerQuery/ExplorerQuery";

const { Title } = Typography

const validChains = Object.values(chainIDs)

// these types match/load the descriptions in src/locales
type TokenTransfer = "tokenTransfer"
type NFTTransfer = "nftTransfer"
type AssetMeta = "assetMeta"
type Governance = "governance"
type Pyth = "pyth"
type UnknownMessage = "unknownMessage"

type PayloadType = TokenTransfer | NFTTransfer | AssetMeta | Governance | Pyth | UnknownMessage

// the payloads this component can decode
const knownPayloads = ["assetMeta", "tokenTransfer", "nftTransfer"]


interface TokenTransferPayload {
    payloadId: number
    amount: string
    originAddress: string
    originChain: number
    targetAddress: string
    targetChain: number
}
interface NFTTransferPayload {
    payloadId: number
    name: string // "Not a PUNK🎸"
    originAddress: string // "0101010101010101010101010101010101010101010101010101010101010101"
    originChain: number // 1
    symbol: string //  "PUNK🎸"
    targetAddress: string // "00000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c1"
    targetChain: number // 2
    tokenId: BigNumber | string // BigNumber { _hex: '0x9c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7', _isBigNumber: true }
    uri: string // "https://wrappedpunks.com:3000/api/punks/metadata/39"
}
interface AssetMetaPayload {
    payloadId: number
    tokenAddress: string
    tokenChain: number
    decimals: number
    symbol: string
    name: string
}
// TODO - figure out how to decode these and what they contain
interface GovernancePayload { }
interface PythPayload { }
interface UnknownPayload { }

type VAAPayload =
    | TokenTransferPayload
    | NFTTransferPayload
    | AssetMetaPayload
    | GovernancePayload
    | PythPayload
    | UnknownPayload

type TokenTransferBundle = { type: TokenTransfer, payload: TokenTransferPayload }
type NFTTransferBundle = { type: NFTTransfer, payload: NFTTransferPayload }
type AssetMetaBundle = { type: AssetMeta, payload: AssetMetaPayload }
type UnknownMessageBundle = { type: UnknownMessage, payload: UnknownPayload }
type PayloadBundle = TokenTransferBundle | NFTTransferBundle | AssetMetaBundle | UnknownMessageBundle


const parseTokenPayload = (arr: Buffer): TokenTransferPayload => ({
    payloadId: arr.readUInt8(0),
    amount: BigNumber.from(arr.slice(1, 1 + 32)).toBigInt().toString(),
    originAddress: arr.slice(33, 33 + 32).toString("hex"),
    originChain: arr.readUInt16BE(65),
    targetAddress: arr.slice(67, 67 + 32).toString("hex"),
    targetChain: arr.readUInt16BE(99),
});
const parseNFTPayload = (arr: Buffer): NFTTransferPayload => {
    const payloadId = arr.readUInt8(0)
    const originAddress = arr.slice(1, 1 + 32).toString("hex");
    const originChain = arr.readUInt16BE(33)
    const symbol = Buffer.from(arr.slice(35, 35 + 32))
        .toString("utf8")
        .replace(METADATA_REPLACE, "");
    const name = Buffer.from(arr.slice(67, 67 + 32))
        .toString("utf8")
        .replace(METADATA_REPLACE, "");
    const tokenId = BigNumber.from(arr.slice(99, 99 + 32)).toString()
    const uri_len = arr.readUInt8(131);
    const uri = Buffer.from(arr.slice(132, 132 + uri_len))
        .toString("utf8")
        .replace(METADATA_REPLACE, "");
    const target_offset = 132 + uri_len;
    const targetAddress = arr
        .slice(target_offset, target_offset + 32)
        .toString("hex");
    const targetChain = arr.readUInt16BE(target_offset + 32);
    return {
        payloadId,
        originAddress,
        originChain,
        symbol,
        name,
        tokenId,
        uri,
        targetAddress,
        targetChain,
    };
};
const parseAssetMetaPayload = (arr: Buffer): AssetMetaPayload => {
    let index = 0
    const payloadId = arr.readUInt8(0)
    index += 1

    const tokenAddress = arr.slice(index, index + 32).toString("hex");
    index += 32

    const tokenChain = arr.readUInt16BE(index)
    index += 1

    const decimals = arr.readUInt8(index)
    index += 1

    const symbol = Buffer.from(arr.slice(index, index + 32))
        .toString("utf8")
        .replace(METADATA_REPLACE, "")
        .replace(new RegExp("\u0012", "g"), "")
        .replace(new RegExp("\u0002", "g"), "")
    index += 32

    const name = Buffer.from(arr.slice(index, index + 32))
        .toString("utf8")
        .replace(METADATA_REPLACE, "");
    index += 32

    return {
        payloadId,
        tokenAddress,
        tokenChain,
        decimals,
        symbol,
        name
    }
}

function useBase64ToBuffer(base64VAA: string = "") {
    const [buf, setBuf] = useState<Buffer>()

    function convertbase64ToBinary(base64: string) {
        var raw = window.atob(base64);
        var rawLength = raw.length;
        var array = new Uint8Array(new ArrayBuffer(rawLength));

        for (let i = 0; i < rawLength; i++) {
            array[i] = raw.charCodeAt(i);
        }
        return array;
    }

    useEffect(() => {
        async function asyncWork(vaaString: string) {
            const vaa = convertbase64ToBinary(vaaString)
            const bridgeWasm = await import('bridge')

            const parsedVaa = bridgeWasm.parse_vaa(vaa)

            setBuf(Buffer.from(parsedVaa.payload))
        }
        asyncWork(base64VAA)
    }, [base64VAA])
    return buf
}
interface DecodePayloadProps {
    base64VAA?: string
    emitterChainName: keyof ChainIDs
    emitterAddress: string
    showType?: boolean
    showSummary?: boolean
    showPayload?: boolean
    transferDetails?: TransferDetails
}

const DecodePayload = (props: DecodePayloadProps) => {
    const buf = useBase64ToBuffer(props.base64VAA)
    const [payloadBundle, setPayloadBundle] = useState<PayloadBundle>()

    const determineType = (payloadBuffer: Buffer) => {
        let payload: PayloadBundle["payload"] = {}
        let type: PayloadBundle["type"] = "unknownMessage"

        let unknown: UnknownMessageBundle = { type: "unknownMessage", payload: {} }
        let bundle: PayloadBundle = unknown

        // try the types, do some logic on the results
        let parsedTokenPayload: TokenTransferPayload | undefined
        let parsedNftPayload: NFTTransferPayload | undefined
        let parsedAssetMeta: AssetMetaPayload | undefined
        try {
            parsedTokenPayload = parseTokenPayload(payloadBuffer)
            // console.log('parsedTokenPayload: ', parsedTokenPayload)
        } catch (_) {
            // do nothing
        }

        try {
            parsedNftPayload = parseNFTPayload(payloadBuffer)
            // console.log('parsedNftPayload ', parsedNftPayload)
        } catch (_) {
            // do nothing
        }

        try {
            parsedAssetMeta = parseAssetMetaPayload(payloadBuffer)
            // console.log('parsedAssetMeta ', parsedAssetMeta)
        } catch (_) {
            // do nothing
        }

        // determine which type of payload this is by asserting values
        if (parsedNftPayload?.uri) {
            try {
                // test for valid url
                new URL(parsedNftPayload.uri);
                type = "nftTransfer"
                payload = parsedNftPayload
                bundle = { type: "nftTransfer", payload: parsedNftPayload }
            } catch (_) {
                // probably not an NFT transfer, continue
            }
        } else if (parsedTokenPayload && validChains.includes(parsedTokenPayload?.originChain) && validChains.includes(parsedTokenPayload?.targetChain)) {
            type = "tokenTransfer"
            payload = parsedTokenPayload
            bundle = { type: "tokenTransfer", payload: parsedTokenPayload }
        } else if (parsedAssetMeta && chainIDs[props.emitterChainName] === parsedAssetMeta.tokenChain) {
            payload = parsedAssetMeta
            type = "assetMeta"
            bundle = { type: "assetMeta", payload: parsedAssetMeta }
        }

        setPayloadBundle(bundle)
    }

    useEffect(() => {
        if (buf) {
            determineType(buf)
        }
    }, [buf])



    return (
        <>
            {props.showType && payloadBundle ?
                <span>

                    {props.showSummary && payloadBundle ? (
                        payloadBundle.type === "assetMeta" ? (<>
                            {"AssetMeta:"}&nbsp;{chainEnums[payloadBundle.payload.tokenChain]}&nbsp; {payloadBundle.payload.symbol} {payloadBundle.payload.name}
                        </>) :
                            payloadBundle.type === "tokenTransfer" ?
                                props.transferDetails && props.transferDetails.OriginSymbol ? (<>
                                    {Math.round(Number(props.transferDetails.Amount) * 100) / 100}{' '}{props.transferDetails.OriginSymbol}{' -> '}{chainEnums[payloadBundle.payload.targetChain]}
                                </>) : (<>
                                    {"Native "}{chainEnums[payloadBundle.payload.originChain]}{' asset -> '}{chainEnums[payloadBundle.payload.targetChain]}
                                </>) :
                                payloadBundle.type === "nftTransfer" ? (<>
                                    {payloadBundle.payload.symbol || "?"}&nbsp;{"-"}&nbsp;{chainEnums[payloadBundle.payload.originChain]}{' -> '}{chainEnums[payloadBundle.payload.targetChain]}
                                </>) : null
                    ) : null}
                </span> : props.showPayload && payloadBundle ? (
                    <>
                        <div style={{ margin: "20px 0" }} className="styled-scrollbar">
                            <Title level={3} style={titleStyles}><FormattedMessage id={`explorer.payloads.${payloadBundle.type}`} /> payload</Title>
                            <pre style={{ fontSize: 14 }}>{JSON.stringify(payloadBundle.payload, undefined, 2)}</pre>
                        </div>
                        {/* TODO - prettier formatting of payload data. POC below. */}
                        {/* {payloadBundle && payloadBundle.payload && knownPayloads.includes(payloadBundle.type) ? (
                            Object.entries(payloadBundle.payload).map(([key, value]) => {
                                return <Statistic title={key} key={key} value={value} />
                            })
                        ) : <span>Can't decode unknown payloads</span>} */}

                    </>
                ) : null}

        </>
    )



}


export { DecodePayload }
