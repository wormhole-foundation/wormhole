import React, { useEffect, useState } from 'react';
import { Link, PageProps } from "gatsby"
import { Typography, Grid, Button, } from 'antd';
const { Title } = Typography;
const { useBreakpoint } = Grid

import { FormattedMessage, useIntl } from 'gatsby-plugin-intl';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';

import { ExplorerStats } from '~/components/ExplorerStats'
import { contractNameFormatter } from '~/components/ExplorerStats/utils';
import { titleStyles } from '~/styles';
import { WithNetwork, NetworkSelect } from '~/components/NetworkSelect'
import { ExplorerSearchForm, ExplorerTxForm } from '~/components/App/ExplorerSearch';
import { ChainID } from '~/utils/misc/constants';
import { OutboundLink } from 'gatsby-plugin-google-gtag';
import { nativeExplorerContractUri } from '~/components/ExplorerStats/utils';
import { CloseOutlined } from '@ant-design/icons';


// form props
interface ExplorerQueryValues {
    emitterChain: number,
    emitterAddress: string,
    sequence: string
    txId: string
}

interface ExplorerProps extends PageProps { }
const Explorer: React.FC<ExplorerProps> = ({ location, navigate }) => {
    const intl = useIntl()
    const screens = useBreakpoint()
    const [emitterChain, setEmitterChain] = useState<ExplorerQueryValues["emitterChain"]>()
    const [emitterAddress, setEmitterAddress] = useState<ExplorerQueryValues["emitterAddress"]>()
    const [sequence, setSequence] = useState<ExplorerQueryValues["sequence"]>()
    const [txId, setTxId] = useState<ExplorerQueryValues["txId"]>()
    const [showQueryForm, setShowQueryForm] = useState<boolean>(false)
    const [doneReadingQueryParams, setDoneReadingQueryParams] = useState<boolean>(false)

    useEffect(() => {
        if (location.search) {
            // take searchparams from the URL and set the values in the form
            const searchParams = new URLSearchParams(location.search);

            const chain = searchParams.get('emitterChain')
            const address = searchParams.get('emitterAddress')
            const seq = searchParams.get('sequence')
            const tx = searchParams.get('txId')


            // if the search params are different form values, update state
            if (Number(chain) !== emitterChain) {
                setEmitterChain(Number(chain) || undefined)
            }
            if (address !== emitterAddress) {
                setEmitterAddress(address || undefined)
            }
            if (seq !== sequence) {
                setSequence(seq || undefined)
            }
            if (tx !== txId) {
                setTxId(tx || undefined)
            }
            if (!tx && (chain && address && seq)) {
                setShowQueryForm(true)
            }
        } else {
            // clear state
            setEmitterChain(undefined)
            setEmitterAddress(undefined)
            setSequence(undefined)
            setTxId(undefined)
            setShowQueryForm(false)
        }
        // be explicit about when it is ok to render
        setDoneReadingQueryParams(true)
    }, [location.search])

    return (
        <Layout>
            <SEO
                title={intl.formatMessage({ id: 'explorer.title' })}
                description={intl.formatMessage({ id: 'explorer.description' })}
            />
            <div
                className="center-content"
                style={{ paddingTop: screens.md === false ? 24 : 100 }}
            >
                <div
                    className="wider-responsive-padding max-content-width"
                    style={{ width: '100%' }}
                >
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 40 }}>
                        <Title level={1} style={titleStyles}>{intl.formatMessage({ id: 'explorer.title' })}</Title>
                        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', flexDirection: 'column', marginRight: !screens.md ? 0 : 80 }}>
                            <div><FormattedMessage id="networks.network" /></div>
                            <NetworkSelect />
                        </div>
                    </div>
                    <div style={{ width: "100%", display: 'flex', justifyContent: 'flex-start', alignItems: 'center', flexDirection: 'column', marginBottom: 40 }}>
                        <div style={{ width: '100%', maxWidth: 960, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
                            <Title level={3} style={titleStyles}>{intl.formatMessage({ id: 'explorer.lookupPrompt' })}</Title>
                            <div style={{ marginRight: !screens.md ? 0 : 80 }}>
                                {showQueryForm && <a onClick={() => setShowQueryForm(false)}><FormattedMessage id="explorer.queryByTxId" /></a>}
                                {!showQueryForm && <a onClick={() => setShowQueryForm(true)}><FormattedMessage id="explorer.queryByMessageId" /></a>}
                            </div>
                        </div>
                        <div style={{ width: '100%', maxWidth: 900 }}>
                            {showQueryForm ? (
                                <ExplorerSearchForm location={location} navigate={navigate} />
                            ) : (
                                <ExplorerTxForm location={location} navigate={navigate} />
                            )}
                        </div>

                    </div>
                    {!(emitterChain && emitterAddress && sequence) && !txId ? (
                        <>
                            <div
                                style={{
                                    width: '100%',
                                    display: 'flex',
                                    justifyContent: 'space-between',
                                    marginBottom: 40
                                }}
                            >
                                {emitterAddress && emitterChain ? (
                                    // show heading with the context of the address
                                    <Title level={3} style={{ ...titleStyles }}>
                                        Recent messages from {ChainID[emitterChain]}&nbsp;
                                        {nativeExplorerContractUri(emitterChain, emitterAddress) ?
                                            <OutboundLink
                                                href={nativeExplorerContractUri(emitterChain, emitterAddress)}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                            >
                                                {contractNameFormatter(emitterAddress, emitterChain)}
                                            </OutboundLink> : contractNameFormatter(emitterAddress, emitterChain)}
                                        :
                                    </Title>

                                ) : emitterChain ? (
                                    // show heading with the context of the chain
                                    <Title level={3} style={{ ...titleStyles }}>
                                        Recent {ChainID[emitterChain]} activity
                                    </Title>
                                ) : (
                                    // show heading for root view, all chains
                                    <Title level={3} style={{ ...titleStyles }}>
                                        {intl.formatMessage({ id: 'explorer.stats.heading' })}
                                    </Title>

                                )}
                                {emitterAddress || emitterChain ?
                                    <Link to={`/${intl.locale}/explorer`}>
                                        <Button
                                            shape="round"
                                            icon={<CloseOutlined />}
                                            size="large"
                                            style={{ marginRight: !screens.md ? 0 : 40 }}

                                        >clear</Button>
                                    </Link> : null}
                            </div>
                            {doneReadingQueryParams && <ExplorerStats emitterChain={emitterChain} emitterAddress={emitterAddress} />}
                        </>
                    ) : null}
                </div>
            </div>
        </Layout >
    )
};

export default WithNetwork(Explorer)
