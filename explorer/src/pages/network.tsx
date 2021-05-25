import React, { useEffect, useState } from 'react';
import { Typography } from 'antd';
const { Title } = Typography;
import { injectIntl, WrappedComponentProps } from 'gatsby-plugin-intl';
import { grpc } from '@improbable-eng/grpc-web';

import { Layout } from '~/components/Layout';
import { SEO } from '~/components/SEO';
import { GuardiansTable } from '~/components/GuardiansTable'

import { Heartbeat } from '~/proto/gossip/v1/gossip'
import { PublicrpcGetRawHeartbeatsDesc, GetRawHeartbeatsRequest } from '~/proto/publicrpc/v1/publicrpc'

const Network = ({ intl }: WrappedComponentProps) => {
  const [heartbeats, setHeartbeats] = useState<{ [nodeName: string]: Heartbeat }>({})

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
      <div style={{ margin: '1em' }}>
        <Title level={1}>{intl.formatMessage({ id: 'network.title' })}</Title>
        {Object.keys(heartbeats).length === 0 ?
          <Title level={2}>
            {intl.formatMessage({ id: 'network.listening' })}
          </Title>
          :
          <Title level={2}>
            {Object.keys(heartbeats).length}&nbsp;
              {intl.formatMessage({ id: 'network.guardiansFound' })}
          </Title>
        }
        <GuardiansTable heartbeats={heartbeats} intl={intl} />
      </div>
    </Layout>
  )
};

export default injectIntl(Network)
