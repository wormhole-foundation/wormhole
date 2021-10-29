import React from 'react';
import { Button, Spin, Typography } from 'antd'
const { Title } = Typography
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl'
import { BigTableMessage } from '~/components/ExplorerQuery/ExplorerQuery';
import { DecodePayload } from '~/components/Payload'
import ReactTimeAgo from 'react-time-ago'
import { titleStyles } from '~/styles';
import { CloseOutlined, ReloadOutlined } from '@ant-design/icons';
import { Link } from 'gatsby';
import { contractNameFormatter, nativeExplorerContractUri, nativeExplorerTxUri } from '../ExplorerStats/utils';
import { OutboundLink } from 'gatsby-plugin-google-gtag';
import { chainIDs } from '~/utils/misc/constants';
import { hexToNativeString } from '@certusone/wormhole-sdk';

interface SummaryProps {
    emitterChain?: number,
    emitterAddress?: string,
    sequence?: string
    txId?: string
    message: BigTableMessage
    polling?: boolean
    lastFetched?: number
    refetch: () => void
}

const Summary = (props: SummaryProps) => {

    const intl = useIntl()
    const { SignedVAA, ...message } = props.message

    const { EmitterChain, EmitterAddress, InitiatingTxID } = message
    // get chainId from chain name
    let chainId = chainIDs[EmitterChain]

    let transactionId: string | undefined
    if (InitiatingTxID) {
        if (chainId === chainIDs["ethereum"] || chainId === chainIDs["bsc"] || chainId === chainIDs["polygon"]) {
            transactionId = InitiatingTxID
        } else {
            if (chainId === chainIDs["solana"]) {
                const txId = InitiatingTxID.slice(2) // remove the leading "0x"
                transactionId = hexToNativeString(txId, chainId)
            } else if (chainId === chainIDs["terra"]) {
                transactionId = InitiatingTxID.slice(2) // remove the leading "0x"
            }
        }
    }

    return (
        <>
            <div style={{ display: 'flex', justifyContent: 'space-between', gap: 8, alignItems: 'baseline' }}>
                <Title level={2} style={titleStyles}><FormattedMessage id="explorer.messageSummary" /></Title>
                {props.polling ? (
                    <>
                        <div style={{ flexGrow: 1 }}></div>
                        <Spin />
                        <Title level={2} style={titleStyles}><FormattedMessage id="explorer.listening" /></Title>
                    </>
                ) : (
                    <div>
                        <Button onClick={props.refetch} icon={<ReloadOutlined />} size="large" shape="round" >refresh</Button>
                        <Link to={`/${intl.locale}/explorer`} style={{ marginLeft: 8 }}>
                            <Button icon={<CloseOutlined />} size='large' shape="round">clear</Button>
                        </Link>
                    </div>
                )}
            </div>
            <div className="styled-scrollbar">
                <pre
                    style={{ fontSize: 14, marginBottom: 20 }}
                >{JSON.stringify(message, undefined, 2)}</pre>
            </div>
            <DecodePayload
                base64VAA={props.message.SignedVAABytes}
                emitterChainName={props.message.EmitterChain}
                emitterAddress={props.message.EmitterAddress}
                showPayload={true}
            />
            <div className="styled-scrollbar">
                <Title level={3} style={titleStyles}>Signed VAA</Title>
                <pre
                    style={{ fontSize: 12, marginBottom: 20 }}
                >{JSON.stringify(SignedVAA, undefined, 2)}</pre>
            </div>
            <div style={{ display: 'flex', justifyContent: "flex-end" }}>
                {props.lastFetched ? (
                    <span>
                        <FormattedMessage id="explorer.lastUpdated" />:&nbsp;
                        <ReactTimeAgo date={new Date(props.lastFetched)} locale={intl.locale} timeStyle="round" />
                    </span>

                ) : null}
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', }}>
                {EmitterChain && EmitterAddress && nativeExplorerContractUri(chainId, EmitterAddress) ?
                    <OutboundLink
                        href={nativeExplorerContractUri(chainId, EmitterAddress)}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ fontSize: 16, marginBottom: '6px 0' }}
                    >
                        {'View "'}{contractNameFormatter(EmitterAddress, chainId)}{'" contract on native explorer'}
                    </OutboundLink> : <div />}
                {transactionId && EmitterChain ?
                    <OutboundLink
                        href={nativeExplorerTxUri(chainId, transactionId)}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ fontSize: 16, margin: '6px 0' }}
                    >
                        {'View transaction "'}{transactionId}{'" on native explorer'}
                    </OutboundLink> : <div />}
            </div>
        </>
    )
}

export default Summary
