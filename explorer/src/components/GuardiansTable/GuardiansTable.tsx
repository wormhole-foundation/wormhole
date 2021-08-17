import React from 'react';
import { Table } from 'antd';
import { ColumnsType } from 'antd/es/table'

import { IntlShape } from 'gatsby-plugin-intl';
import ReactTimeAgo from 'react-time-ago'

import { Heartbeat, Heartbeat_Network } from '~/proto/gossip/v1/gossip'

import { ReactComponent as BinanceChainIcon } from '~/icons/binancechain.svg';
import { ReactComponent as EthereumIcon } from '~/icons/ethereum.svg';
import { ReactComponent as SolanaIcon } from '~/icons/solana.svg';
import { ReactComponent as TerraIcon } from '~/icons/terra.svg';

import './GuardiansTable.less'

const networkEnums = ['', 'Solana', 'Ethereum', 'Terra', 'BSC']
const networkIcons = [
  <></>,
  <SolanaIcon key="1" style={{ height: 18, maxWidth: 18, margin: '0 4px' }} />,
  <EthereumIcon key="2" style={{ height: 24, margin: '0 4px' }} />,
  <TerraIcon key="3" style={{ height: 18, margin: '0 4px' }} />,
  <BinanceChainIcon key="4" style={{ height: 18, margin: '0 4px' }} />,
]

const expandedRowRender = (intl: IntlShape) => (item: Heartbeat) => {
  const columns: ColumnsType<Heartbeat_Network> = [
    { title: '', dataIndex: 'id', key: 'icon', render: (id: number) => networkIcons[id] },
    {
      title: intl.formatMessage({ id: 'network.network' }), dataIndex: 'id', key: 'id', responsive: ['md'],
      render: (id: number) => networkEnums[id]
    },
    { title: intl.formatMessage({ id: 'network.address' }), dataIndex: 'bridgeAddress', key: 'bridgeAddress' },
    { title: intl.formatMessage({ id: 'network.blockHeight' }), dataIndex: 'height', key: 'height', responsive: ['md'], }
  ];

  return (
    <Table<Heartbeat_Network>
      rowKey="id"
      columns={columns}
      dataSource={item.networks}
      pagination={false}
    />
  )
};

const GuardiansTable = ({ heartbeats, intl }: { heartbeats: { [nodeName: string]: Heartbeat }, intl: IntlShape }) => {
  const columns: ColumnsType<Heartbeat> = [
    {
      title: intl.formatMessage({ id: 'network.guardian' }), key: 'guardian',
      render: (item: Heartbeat) => <>{item.nodeName}<br />{item.guardianAddr}</>
    },
    { title: intl.formatMessage({ id: 'network.version' }), dataIndex: 'version', key: 'version', responsive: ['lg'] },
    {
      title: intl.formatMessage({ id: 'network.networks' }), dataIndex: 'networks', key: 'networks', responsive: ['md'],
      render: (networks: Heartbeat_Network[]) => networks.map(network => networkIcons[network.id])
    },
    { title: intl.formatMessage({ id: 'network.heartbeat' }), dataIndex: 'counter', key: 'counter', responsive: ['xl'] },
    {
      title: intl.formatMessage({ id: 'network.lastHeartbeat' }), dataIndex: 'timestamp', key: 'timestamp', responsive: ['sm'],
      render: (timestamp: string) =>
        <ReactTimeAgo date={new Date(Number(timestamp.slice(0, -6)))} locale={intl.locale} timeStyle="round" />
    }
  ];
  return (
    <Table<Heartbeat>
      columns={columns}
      size="small"
      expandable={{
        expandedRowRender: expandedRowRender(intl),
        expandRowByClick: true,
      }}
      dataSource={Object.values(heartbeats)}
      loading={Object.keys(heartbeats).length === 0}
      rowKey="nodeName"
      pagination={false}
    />
  )

}

export default GuardiansTable
