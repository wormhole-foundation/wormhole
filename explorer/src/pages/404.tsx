import React from 'react'
import { Result } from 'antd'
import { useIntl } from 'gatsby-plugin-intl';

export default () => {
  const intl = useIntl()
  return <Result
    status="404"
    title="404"
    subTitle={intl.formatMessage({id: '404.message'})}
  />
}
