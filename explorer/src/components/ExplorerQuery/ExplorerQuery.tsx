import React, { useContext, useEffect, useState } from 'react';
import { Spin, Typography } from 'antd'
const { Title } = Typography

import { FormattedMessage } from 'gatsby-plugin-intl'
import { arrayify, isHexString, zeroPad, hexlify } from "ethers/lib/utils";
import { Bech32, toHex, fromHex } from "@cosmjs/encoding"
import { ExplorerSummary } from '~/components/ExplorerSummary';
import { titleStyles } from '~/styles';
import { NetworkContext } from '~/components/NetworkSelect';
import { getEmitterAddressSolana } from "@certusone/wormhole-sdk";
import { ChainIDs, chainIDs } from '~/utils/misc/constants';
import { PublicKey } from '@solana/web3.js';

export interface VAA {
    Version: number | string,
    GuardianSetIndex: number,
    Signatures: { Index: number, Signature: string }[],
    Timestamp: string, // "0001-01-01T00:00:00Z",
    Nonce: number,
    Sequence: number,
    ConsistencyLevel: number,
    EmitterChain: number,
    EmitterAddress: string,
    Payload: string // base64 encoded byte array
}
export interface TokenTransferPayload {
    Amount: string
    OriginAddress: string
    OriginChain: string,
    TargetAddress: string,
    TargetChain: string,
}
export interface TransferDetails {
    Amount: string,             // "1530.000000",
    Decimals: string,           // "6",
    NotionalUSDStr: string,     // "1538.495460",
    TokenPriceUSDStr: string,   // "1.005553",
    TransferTimestamp: string,  // "2021-11-21 16:55:15 +0000 UTC",
    OriginSymbol: string,
    OriginName: string,
    OriginTokenAddress: string,
}
export interface BigTableMessage {
    InitiatingTxID?: string
    SignedVAABytes?: string  // base64 encoded byte array
    SignedVAA?: VAA
    QuorumTime?: string  // "2021-08-11 00:16:11.757 +0000 UTC"
    EmitterChain: keyof ChainIDs
    EmitterAddress: string
    Sequence: string
    TokenTransferPayload?: TokenTransferPayload
    TransferDetails?: TransferDetails
}

interface ExplorerQuery {
    emitterChain?: number,
    emitterAddress?: string,
    sequence?: string,
    txId?: string,
}
const ExplorerQuery = (props: ExplorerQuery) => {
    const { activeNetwork } = useContext(NetworkContext)
    const [error, setError] = useState<string>();
    const [loading, setLoading] = useState<boolean>(true);
    const [message, setMessage] = useState<BigTableMessage>();
    const [polling, setPolling] = useState(false);
    const [lastFetched, setLastFetched] = useState<number>()
    const [pollInterval, setPollInterval] = useState<NodeJS.Timeout>()

    const fetchMessage = async (
        emitterChain: ExplorerQuery["emitterChain"],
        emitterAddress: ExplorerQuery["emitterAddress"],
        sequence: ExplorerQuery["sequence"],
        txId: ExplorerQuery["txId"]) => {
        let paddedAddress: string = ""
        let paddedSequence: string

        let base = `${activeNetwork.endpoints.bigtableFunctionsBase}`
        let url = ""

        if (emitterChain && emitterAddress && sequence) {
            if (emitterChain === chainIDs["solana"]) {
                if (emitterAddress.length < 64) {
                    try {
                        paddedAddress = await getEmitterAddressSolana(emitterAddress)
                    } catch (_) {
                        // do nothing
                    }
                } else {
                    paddedAddress = emitterAddress
                }
            } else if (emitterChain === chainIDs["ethereum"] || emitterChain === chainIDs["bsc"] || emitterChain === chainIDs["polygon"]) {
                if (isHexString(emitterAddress)) {

                    let paddedAddressArray = zeroPad(arrayify(emitterAddress, { hexPad: "left" }), 32);

                    let maybeString = Buffer.from(paddedAddressArray).toString('hex');

                    paddedAddress = maybeString
                } else {
                    // must already be padded
                    paddedAddress = emitterAddress
                }
            } else if (emitterChain === chainIDs["terra"]) {
                if (emitterAddress.startsWith('terra')) {
                    try {
                        paddedAddress = toHex(zeroPad(Bech32.decode(emitterAddress).data, 32))
                    } catch (_) {
                        // do nothing
                    }
                } else {
                    paddedAddress = emitterAddress
                }
            } else {
                paddedAddress = emitterAddress
            }

            if (sequence.length <= 15) {
                paddedSequence = sequence.padStart(16, "0")
            } else {
                paddedSequence = sequence
            }
            url = `${base}/readrow?emitterChain=${emitterChain}&emitterAddress=${paddedAddress}&sequence=${paddedSequence}`
        } else if (txId) {
            let transformedTxId = txId
            if (isHexString(txId)) {
                // valid hexString, no transformation needed.
            } else {
                try {
                    let pubKey = new PublicKey(txId).toBytes()
                    let solHex = hexlify(pubKey)
                    transformedTxId = solHex
                } catch (_) {
                    // not solana, try terra
                    try {
                        let arr = fromHex(txId)
                        let terraHex = hexlify(arr)
                        transformedTxId = terraHex
                    } catch (_) {
                        // do nothing
                    }
                }
            }
            url = `${base}/transaction?id=${transformedTxId}`
        }

        fetch(url)
            .then<BigTableMessage>(res => {
                if (res.ok) return res.json()
                if (res.status === 404) {
                    // show a specific message to the user if the query returned 404.
                    throw 'explorer.notFound'
                }
                // if res is not ok, and not 404, throw an error with specific message,
                // rather than letting the json decoding throw.
                throw 'explorer.failedFetching'
            })
            .then(result => {

                setMessage(result)
                setLoading(false)
                setLastFetched(Date.now())

                // turn polling on/off
                if (!result.QuorumTime && !polling) {
                    setPolling(true)
                } else if (result.QuorumTime && polling) {
                    setPolling(false)
                }
            }, error => {
                // Note: it's important to handle errors here
                // instead of a catch() block so that we don't swallow
                // exceptions from actual bugs in components.
                setError(error)
                setLoading(false)
                setLastFetched(Date.now())
                if (polling) {
                    setPolling(false)
                }
            })
    }

    const refreshCallback = () => {
        fetchMessage(props.emitterChain, props.emitterAddress, props.sequence, props.txId)
    }

    if (polling && !pollInterval) {
        let interval = setInterval(() => {
            fetchMessage(props.emitterChain, props.emitterAddress, props.sequence, props.txId)
        }, 3000)
        setPollInterval(interval)
    } else if (!polling && pollInterval) {
        clearInterval(pollInterval)
        setPollInterval(undefined)
    }

    useEffect(() => {
        setPolling(false)
        setError(undefined)
        setMessage(undefined)
        setLastFetched(undefined)
        if ((props.emitterChain && props.emitterAddress && props.sequence) || props.txId) {
            setLoading(true)
            fetchMessage(props.emitterChain, props.emitterAddress, props.sequence, props.txId)
        }

    }, [props.emitterChain, props.emitterAddress, props.sequence, props.txId, activeNetwork.endpoints.bigtableFunctionsBase])

    useEffect(() => {
        return function cleanup() {
            if (pollInterval) {
                clearInterval(pollInterval)
            }
        };
    }, [polling, activeNetwork.endpoints.bigtableFunctionsBase])


    return (
        <>
            {loading ? <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                <Spin />
            </div> :
                error ? <Title level={2} style={titleStyles}><FormattedMessage id={error} /></Title> :
                    message ? (
                        <ExplorerSummary
                            {...props}
                            message={message}
                            polling={polling}
                            lastFetched={lastFetched}
                            refetch={refreshCallback}
                        />
                    ) : null
            }
        </>
    )
}

export default ExplorerQuery
