import React, { useEffect } from 'react';
import { Button, Spin, Typography } from 'antd'
const { Title } = Typography
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl'
import { BigTableMessage } from '~/components/ExplorerQuery/ExplorerQuery';
// import { WasmTest } from '~/components/wasm'
import ReactTimeAgo from 'react-time-ago'
import { buttonStylesLg, titleStyles } from '~/styles';

interface SummaryProps {
    emitterChain: number,
    emitterAddress: string,
    sequence: string
    message: BigTableMessage
    polling?: boolean
    lastFetched?: number
    refetch: () => void
}

const Summary = (props: SummaryProps) => {

    const intl = useIntl()

    useEffect(() => {
        // TODO: decode the payload. if applicable lookup other relevant messages.
    }, [props])

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
                    <Button style={buttonStylesLg} onClick={props.refetch} size="large"><FormattedMessage id="explorer.refresh" /></Button>
                )}
            </div>
            <pre>{JSON.stringify(props.message, undefined, 2)}</pre>
            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>

                {props.lastFetched ? (
                    <span>
                        <FormattedMessage id="explorer.lastUpdated" />:&nbsp;
                        <ReactTimeAgo date={new Date(props.lastFetched)} locale={intl.locale} timeStyle="round" />
                    </span>

                ) : null}
            </div>
            {/* <WasmTest base64VAA={props.message.SignedVAA} /> */}
        </>
    )
}

export default Summary
