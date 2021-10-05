import React, { useContext, useEffect, useState } from 'react';
import { Typography, Grid } from 'antd';
const { Title, Paragraph } = Typography;
const { useBreakpoint } = Grid
import { FormattedMessage, injectIntl, WrappedComponentProps } from 'gatsby-plugin-intl';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';
import { GuardiansTable } from '~/components/GuardiansTable'
import { WithNetwork, NetworkSelect, NetworkContext } from '~/components/NetworkSelect'

import { Heartbeat } from '~/proto/gossip/v1/gossip'
import { GrpcWebImpl, PublicRPCServiceClientImpl } from '~/proto/publicrpc/v1/publicrpc'

const networks = { "devnet": {}, "testnet": {}, "mainnet": {} }

const Network = ({ intl }: WrappedComponentProps) => {
  const [heartbeats, setHeartbeats] = useState<{ [networkName: string]: { [nodeName: string]: Heartbeat } }>(networks)
  const screens = useBreakpoint()
  const [pollInterval, setPollInterval] = useState<NodeJS.Timeout>()
  const { activeNetwork } = useContext(NetworkContext)

  const addHeartbeat = (networkName: string, hbObj: Heartbeat) => {
    hbObj.networks.sort((a, b) => a.id - b.id)
    const { nodeName } = hbObj
    heartbeats[networkName][nodeName] = hbObj
    setHeartbeats({ ...heartbeats })
  }

  useEffect(() => {
    if (pollInterval) {
      // stop polling
      clearInterval(pollInterval)
      setHeartbeats({ ...heartbeats, [activeNetwork.name]: {} })
    }
    const rpc = new GrpcWebImpl(String(activeNetwork.endpoints.guardianRpcBase), {});
    const publicRpc = new PublicRPCServiceClientImpl(rpc)

    const interval = setInterval(() => {
      publicRpc.GetLastHeartbeats({}).then(res => {
        res.entries.map(entry => entry.rawHeartbeat ? addHeartbeat(activeNetwork.name, entry.rawHeartbeat) : null)
      }, err => console.error('GetLastHearbeats err: ', err))
    }, 3000)
    setPollInterval(interval)

    return function cleanup() {
      clearInterval(interval)
    }
  }, [activeNetwork.endpoints.guardianRpcBase])

  return (
    <Layout>
      <SEO
        title={intl.formatMessage({ id: 'network.title' })}
        description={intl.formatMessage({ id: 'network.description' })}
      />
      <div
        className="center-content"
        style={{ paddingTop: screens.md === false ? 24 : 100 }}
      >
        <div
          className="wider-responsive-padding max-content-width"
          style={{ width: '100%' }}
        >
          <div style={{ padding: screens.md === false ? '100px 0 0 16px' : '' }} >
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 40 }}>
              <Title level={1} style={{ fontWeight: 'normal' }}>{intl.formatMessage({ id: 'network.title' })}</Title>
              <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', flexDirection: 'column', marginRight: !screens.md ? 0 : 80 }}>
                <div><FormattedMessage id="networks.network" /></div>
                <NetworkSelect />
              </div>
            </div>
            <Paragraph style={{ fontSize: 24, fontWeight: 400, lineHeight: '36px' }} type="secondary">
              {Object.keys(heartbeats[activeNetwork.name]).length === 0 ? (
                intl.formatMessage({ id: 'network.listening' })
              ) :
                <>
                  {Object.keys(heartbeats[activeNetwork.name]).length}&nbsp;
                  {intl.formatMessage({ id: 'network.guardiansFound' })}
                </>}
            </Paragraph>
          </div>
          <GuardiansTable heartbeats={heartbeats[activeNetwork.name]} intl={intl} />
        </div>
      </div>
    </Layout>
  )
};

export default WithNetwork(injectIntl(Network))
