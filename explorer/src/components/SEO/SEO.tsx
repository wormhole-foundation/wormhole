import React from 'react';
import Helmet, { HelmetProps } from 'react-helmet';
import { useIntl } from 'gatsby-plugin-intl';
import { Location } from '@reach/router';

type Props = {
  /** Description text for the description meta tags */
  description?: string;
} & HelmetProps

/**
 * An SEO component that handles all element in the head that can accept
 */
const SEO: React.FC<Props> = ({ children, description = '', title}) => {
  const metaDescription = description;
  const intl = useIntl()
  return (
    <Location>
      {({ location }) => (
        <Helmet
          htmlAttributes={{
            lang: intl.locale,
          }}
          title={title}
        >
          <meta name="description" content={metaDescription} />

          {/* OG tags */}
          <meta
            property="og:url"
            content={process.env.GATSBY_SITE_URL + location.pathname}
          />
          <meta property="og:type" content="website" />
          <meta property="og:title" content={title} />
          <meta property="og:description" content={metaDescription} />
          <meta property="og:locale" content={intl.locale} />
          {/* "summary" is the type of twitter card. Hardcoded string is OK here. */}
          <meta property="twitter.card" content="summary" />
          <meta property="twitter.creator" content="@CertusOne" />
          <meta property="twitter.title" content={title} />
          <meta property="twitter.description" content={metaDescription} />

          {/* PWA tags */}
          <meta name="theme-color" content="#546e7a"/>

          {children}
        </Helmet>
      )}
    </Location>
  );
};

export default SEO
