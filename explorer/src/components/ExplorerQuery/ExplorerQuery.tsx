import React, { useEffect, useState } from 'react';
import { Spin, Typography } from 'antd'
const { Title } = Typography

import { FormattedMessage } from 'gatsby-plugin-intl'
import { arrayify, isHexString, zeroPad } from "ethers/lib/utils";
import { ExplorerSummary } from '~/components/ExplorerSummary';
import { titleStyles } from '~/styles';


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
export interface BigTableMessage {
    InitiatingTxID: string
    GuardianAddresses: string[],
    SignedVAABytes: string  // base64 encoded byte array
    SignedVAA: VAA
    QuorumTime: string  // "2021-08-11 00:16:11.757 +0000 UTC"
}

interface ExplorerQuery {
    emitterChain: number,
    emitterAddress: string,
    sequence: string
}
const ExplorerQuery = (props: ExplorerQuery) => {
    const [error, setError] = useState<string>();
    const [loading, setLoading] = useState<boolean>(true);
    const [message, setMessage] = useState<BigTableMessage>();
    const [polling, setPolling] = useState(false);
    const [lastFetched, setLastFetched] = useState<number>()
    const [pollInterval, setPollInterval] = useState<NodeJS.Timeout>()

    const fetchMessage = (
        emitterChain: ExplorerQuery["emitterChain"],
        emitterAddress: ExplorerQuery["emitterAddress"],
        sequence: ExplorerQuery["sequence"]) => {
        let paddedAddress: string

        if (emitterChain === 1) {
            // TODO - zero pad Solana address, if needed.
            paddedAddress = emitterAddress
        } else if (emitterChain === 2 || emitterChain === 4) {
            if (isHexString(emitterAddress)) {

                let paddedAddressArray = zeroPad(arrayify(emitterAddress, { hexPad: "left" }), 32);

                // TODO - properly encode the this to a hex string, Buffer is deprecated.
                let maybeString = new Buffer(paddedAddressArray).toString('hex');

                paddedAddress = maybeString
            } else {
                // must already be padded
                paddedAddress = emitterAddress
            }
        } else {
            // TODO - zero pad Terra address, if needed
            paddedAddress = emitterAddress
        }

        const base = process.env.GATSBY_BIGTABLE_URL
        const url = `${base}/?emitterChain=${emitterChain}&emitterAddress=${paddedAddress}&sequence=${sequence}`

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
        fetchMessage(props.emitterChain, props.emitterAddress, props.sequence)
    }

    if (polling && !pollInterval) {
        let interval = setInterval(() => {
            fetchMessage(props.emitterChain, props.emitterAddress, props.sequence)
        }, 3000)
        setPollInterval(interval)
    } else if (!polling && pollInterval) {
        clearInterval(pollInterval)
        setPollInterval(undefined)
    }

    useEffect(() => {
        if (props.emitterChain && props.emitterAddress && props.sequence) {
            setPolling(false)
            setLoading(true)
            setError(undefined)
            setMessage(undefined)
            fetchMessage(props.emitterChain, props.emitterAddress, props.sequence)
        }

    }, [props.emitterChain, props.emitterAddress, props.sequence])

    useEffect(() => {
        return function cleanup() {
            if (pollInterval) {
                clearInterval(pollInterval)
            }
        };
    }, [polling])


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
