import React, { useEffect, useState } from 'react';
import { Typography, Grid } from 'antd';
const { Title, Paragraph } = Typography;
const { useBreakpoint } = Grid
import { injectIntl, WrappedComponentProps } from 'gatsby-plugin-intl';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';
import { GuardiansTable } from '~/components/GuardiansTable'

import { Heartbeat } from '~/proto/gossip/v1/gossip'
import { GrpcWebImpl, PublicRPCServiceClientImpl } from '~/proto/publicrpc/v1/publicrpc'

const rpc = new GrpcWebImpl(String(process.env.GATSBY_APP_RPC_URL), {});
const publicRpc = new PublicRPCServiceClientImpl(rpc)

const Network = ({ intl }: WrappedComponentProps) => {
  const [heartbeats, setHeartbeats] = useState<{ [nodeName: string]: Heartbeat }>({})
  const screens = useBreakpoint()

  const addHeartbeat = (hbObj: Heartbeat) => {
    hbObj.networks.sort((a, b) => b.id - a.id)
    const { nodeName } = hbObj
    heartbeats[nodeName] = hbObj
    setHeartbeats({ ...heartbeats })
  }

  useEffect(() => {

    const interval = setInterval(() => {
      publicRpc.GetLastHeartbeats({}).then(res => {
        res.entries.map(entry => entry.rawHeartbeat ? addHeartbeat(entry.rawHeartbeat) : null)
      }, err => console.error('GetLastHearbeats err: ', err))
    }, 2000)

    return function cleanup() {
      clearInterval(interval)
    }
  }, [])

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
          className="responsive-padding max-content-width"
          style={{ width: '100%' }}
        >
          <div style={{ padding: screens.md === false ? '100px 0 0 16px' : '' }} >
            <Title level={1} style={{ fontWeight: 'normal' }}>{intl.formatMessage({ id: 'network.title' })}</Title>
            <Paragraph style={{ fontSize: 24, fontWeight: 400, lineHeight: '36px' }} type="secondary">
              {Object.keys(heartbeats).length === 0 ? (
                intl.formatMessage({ id: 'network.listening' })
              ) :
                <>
                  {Object.keys(heartbeats).length}&nbsp;
                  {intl.formatMessage({ id: 'network.guardiansFound' })}
                </>}
            </Paragraph>
          </div>
          <GuardiansTable heartbeats={heartbeats} intl={intl} />
        </div>
      </div>
    </Layout>
  )
};

export default injectIntl(Network)
