import React, { useEffect } from 'react';
import { Button, Spin, Typography } from 'antd'
const { Title } = Typography
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl'
import { BigTableMessage } from '~/components/ExplorerQuery/ExplorerQuery';
import { DecodePayload } from '~/components/Payload'
import ReactTimeAgo from 'react-time-ago'
import { titleStyles } from '~/styles';
import { CloseOutlined, ReloadOutlined } from '@ant-design/icons';
import { Link } from 'gatsby';
import { contractNameFormatter, nativeExplorerUri } from '../ExplorerStats/utils';
import { OutboundLink } from 'gatsby-plugin-google-gtag';

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
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>

                {props.emitterChain && props.emitterAddress && nativeExplorerUri(props.emitterChain, props.emitterAddress) ?
                    <OutboundLink
                        href={nativeExplorerUri(props.emitterChain, props.emitterAddress)}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ fontSize: 16 }}
                    >
                        {'View "'}{contractNameFormatter(props.emitterAddress, props.emitterChain)}{'" emitter contract on native explorer'}
                    </OutboundLink> : <div />}

                {props.lastFetched ? (
                    <span>
                        <FormattedMessage id="explorer.lastUpdated" />:&nbsp;
                        <ReactTimeAgo date={new Date(props.lastFetched)} locale={intl.locale} timeStyle="round" />
                    </span>

                ) : null}
            </div>
        </>
    )
}

export default Summary
