import React from 'react'
import { Result } from 'antd'
import { useIntl } from 'gatsby-plugin-intl';
import { PageProps } from 'gatsby';
const isBrowser = typeof window !== "undefined"

export default (props: PageProps) => {
  // TEMP - handle GCP bucket redirect, remove "index.html" from the route.
  // This handles an edge case where the user lands without a trailing slash,
  // and GCP redirects them to + '/index.html', because the route did not match
  // a file in the bucket.
  // Can remove this when we move away from GCP storage bucket hosting.
  if (isBrowser) {
    if (props.location.href.includes('index.html')) {
      const uri = props.location.pathname.replace('index.html', '') + props.location.search
      window.location.replace(uri)
      return null // don't render anything, user will be redirected.
    }
  }
  const intl = useIntl()
  return <Result
    status="404"
    title="404"
    subTitle={intl.formatMessage({ id: '404.message' })}
  />
}
