import React from 'react';
import { Recent } from './ExplorerStats';
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl'
import { Grid, Table, Typography } from 'antd'
const { useBreakpoint } = Grid
const { Title } = Typography
import ReactTimeAgo from 'react-time-ago'
import { BigTableMessage } from '../ExplorerQuery/ExplorerQuery';
import { chainIDs, ChainID } from '~/utils/misc/constants';
import { Link } from 'gatsby';
import { ColumnsType } from 'antd/es/table';
import { titleStyles } from '~/styles';
import { DecodePayload } from '../Payload';
import { contractNameFormatter } from './utils';
import './RecentMessages.less'

import { ReactComponent as BinanceChainIcon } from '~/icons/binancechain.svg';
import { ReactComponent as EthereumIcon } from '~/icons/ethereum.svg';
import { ReactComponent as SolanaIcon } from '~/icons/solana.svg';
import { ReactComponent as TerraIcon } from '~/icons/terra.svg';
import { ReactComponent as PolygonIcon } from '~/icons/polygon.svg'
import { formatQuorumDate } from '~/utils/misc/utils';

interface RecentMessagesProps {
    recent: Recent
    lastFetched?: number
    title: string
    hideTableTitles?: boolean
}

const networkIcons = [
    <></>,
    <SolanaIcon key="1" style={{ height: 18, maxWidth: 18, margin: '0 4px' }} />,
    <EthereumIcon key="2" style={{ height: 24, margin: '0 4px' }} />,
    <TerraIcon key="3" style={{ height: 18, margin: '0 4px' }} />,
    <BinanceChainIcon key="4" style={{ height: 18, margin: '0 4px' }} />,
    <PolygonIcon key="5" style={{ height: 18, margin: '0 4px' }} />,
]


const RecentMessages = (props: RecentMessagesProps) => {
    const intl = useIntl()
    const screens = useBreakpoint()
    const columns: ColumnsType<BigTableMessage> = [
        { title: '', key: 'icon', render: (item: BigTableMessage) => networkIcons[chainIDs[item.EmitterChain]], responsive: ['sm'] },
        {
            title: "contract",
            key: "contract",
            render: (item: BigTableMessage) => {
                const name = contractNameFormatter(item.EmitterAddress, chainIDs[item.EmitterChain])
                return <div>{name}</div>
            },
            responsive: ['sm']
        },
        {
            title: "message",
            key: "payload",
            render: (item: BigTableMessage) => item.SignedVAABytes ? <DecodePayload
                base64VAA={item.SignedVAABytes}
                emitterChainName={item.EmitterChain}
                emitterAddress={item.EmitterAddress}
                showType={true}
                showSummary={true}
                transferDetails={item.TransferDetails}
            /> : null
        },
        {
            title: "sequence",
            key: "sequence",
            render: (item: BigTableMessage) => {
                let sequence = item.Sequence.replace(/^0+/, "")
                if (!sequence) sequence = "0"

                return sequence
            },
            responsive: ['md']
        },
        {
            title: "attested",
            dataIndex: "QuorumTime",
            key: "time",
            render: QuorumTime => <ReactTimeAgo date={QuorumTime ? Date.parse(formatQuorumDate(QuorumTime)) : new Date()} locale={intl.locale} timeStyle={!screens.md ? "twitter" : "round"} />
        },
        {
            title: "",
            key: "view",
            render: (item: BigTableMessage) => <Link to={`/${intl.locale}/explorer/?emitterChain=${chainIDs[item.EmitterChain]}&emitterAddress=${item.EmitterAddress}&sequence=${item.Sequence}`}>View</Link>
        },
    ]


    const formatKey = (key: string) => {
        if (props.hideTableTitles) {
            return null
        }
        if (key.includes(":")) {
            const parts = key.split(":")
            const link = `/${intl.locale}/explorer/?emitterChain=${parts[0]}&emitterAddress=${parts[1]}`
            return <Title level={4} style={titleStyles}>From {ChainID[Number(parts[0])]} contract: <Link to={link}>{contractNameFormatter(parts[1], Number(parts[0]))}</Link></Title>
        } else if (key === "*") {
            return <Title level={4} style={titleStyles}>From all chains and addresses</Title>
        } else {
            return <Title level={4} style={titleStyles}>From {ChainID[Number(key)]}</Title>
        }
    }

    return (
        <>
            <Title level={3} style={titleStyles} >{props.title}</Title>
            {Object.keys(props.recent).map(key => (
                <Table<BigTableMessage>
                    key={key}
                    rowKey={(item) => item.EmitterAddress + item.Sequence}
                    style={{ marginBottom: 40 }}
                    size={screens.lg ? "large" : "small"}
                    columns={columns}
                    dataSource={props.recent[key]}
                    title={() => formatKey(key)}
                    pagination={false}
                    rowClassName="highlight-new-row"
                    footer={() => {
                        return props.lastFetched ? (
                            <span>
                                <FormattedMessage id="explorer.lastUpdated" />:&nbsp;
                                <ReactTimeAgo date={new Date(props.lastFetched)} locale={intl.locale} timeStyle="twitter" />
                            </span>

                        ) : null
                    }}
                />
            ))}
        </>
    )
}

export default RecentMessages
