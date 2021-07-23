import React, { useEffect, useState } from 'react';
import { Typography, Grid } from 'antd';
const { Title, Paragraph } = Typography;
const { useBreakpoint } = Grid
import { injectIntl, WrappedComponentProps } from 'gatsby-plugin-intl';
import { grpc } from '@improbable-eng/grpc-web';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';
import { GuardiansTable } from '~/components/GuardiansTable'

import { Heartbeat } from '~/proto/gossip/v1/gossip'
import { PublicrpcGetRawHeartbeatsDesc, GetRawHeartbeatsRequest } from '~/proto/publicrpc/v1/publicrpc'

const Network = ({ intl }: WrappedComponentProps) => {
  const [heartbeats, setHeartbeats] = useState<{ [nodeName: string]: Heartbeat }>({})
  const screens = useBreakpoint()

  const addHeartbeat = (hb: grpc.ProtobufMessage) => {
    const hbObj = hb.toObject() as Heartbeat
    hbObj.networks.sort((a, b) => b.id - a.id)
    const { nodeName } = hbObj
    heartbeats[nodeName] = hbObj
    setHeartbeats({ ...heartbeats })
  }

  useEffect(() => {
    const client = grpc.client(PublicrpcGetRawHeartbeatsDesc, {
      host: String(process.env.GATSBY_APP_RPC_URL)
    })
    client.onMessage(addHeartbeat)
    client.start()
    client.send({ serializeBinary: () => GetRawHeartbeatsRequest.encode({}).finish(), toObject: () => { return {} } })

    return function cleanup() {
      client.close()
    }
  }, [])

  return (
    <Layout>
      <SEO
        title={intl.formatMessage({ id: 'network.title' })}
        description={intl.formatMessage({ id: 'network.description' })}
      />
      <div
        style={{
          padding: screens.md === false ? 'inherit' : '48px 0 0 100px'
        }} >
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
    </Layout>
  )
};

export default injectIntl(Network)
