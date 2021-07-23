import React from 'react';
import Helmet, { HelmetProps } from 'react-helmet';
import { useIntl } from 'gatsby-plugin-intl';
import { Location } from '@reach/router';

type Props = {
  /** Description text for the description meta tags */
  description?: string;
} & HelmetProps


const socialAvatar = '/images/WH_Avatar_DarkBackground.webp'
const socialAvatarSrc = process.env.GATSBY_SITE_URL + socialAvatar
const socialAvatarHeight = '441'
const socialAvatarWidth = '375'

const socialLogo = '/images/WH_Logo_DarkBackground.webp'
const socialLogoSrc = process.env.GATSBY_SITE_URL + socialLogo
const socialLogoHeight = '543'
const socialLogoWidth = '2193'

/**
 * An SEO component that handles all element in the head that can accept
 */
const SEO: React.FC<Props> = ({ children, description = '', title }) => {
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
          <meta property="og:image" content={socialAvatarSrc} />
          <meta property="og:image:type" content="image/webp" />
          <meta property="og:image:height" content={socialAvatarHeight} />
          <meta property="og:image:width" content={socialAvatarWidth} />
          <meta property="og:image:alt" content={metaDescription} />
          <meta property="og:locale" content={intl.locale} />
          {/* "summary" is the type of twitter card. Hardcoded string is OK here. */}
          <meta name="twitter:card" content="summary" />
          <meta name="twitter:site" content="@wormholecrypto" />
          <meta name="twitter:creator" content="@CertusOne" />
          <meta name="twitter:title" content={title} />
          <meta name="twitter:description" content={metaDescription} />
          <meta name="twitter:image" content={socialAvatarSrc} />
          <meta name="twitter:image:alt" content={metaDescription} />

          <script type="application/ld+json">
            {`
              {
                "@context": "https://schema.org",
                "@type": "Organization",
                "@id": "wormhole-org",
                "url": "https://wormholenetwork.com",
                "name": "Wormhole Network",
                "sameAs": [
                    "https://github.com/certusone/wormhole",
                    "https://t.me/wormholecrypto",
                    "https://twitter.com/wormholecrypto",
                    "https://www.wormholebridge.com"
                ],
                "alternateName": [
                  "wormhole",
                  "wormhole protocol",
                  "wormhole bridge",
                  "wormhole crypto",
                  "solana wormhole",
                  "SOL wormhole",
                  "terra wormhole",
                  "LUNA wormhole",
                  "ethereum wormhole",
                  "ETH wormhole",
                  "binance wormhole",
                  "BSC wormhole",
                  "certus one wormhole"
                ],
                "description": "A cross-chain messaging protocol.",
                "image": {
                    "@type": "ImageObject",
                    "height": "${socialAvatarHeight}",
                    "url": "${socialAvatarSrc}",
                    "width": "${socialAvatarWidth}"
                },
                "logo": {
                    "@type": "ImageObject",
                    "height": "${socialLogoHeight}",
                    "url": "${socialLogoSrc}",
                    "width": "${socialLogoWidth}"
                }
              }
            `}
          </script>

          {/* PWA tags */}
          <link rel="apple-touch-icon" sizes="180x180" href="/apple-touch-icon.png" />
          <link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png" />
          <link rel="icon" type="image/png" sizes="16x16" href="/favicon-16x16.png" />
          <link rel="manifest" href="/site.webmanifest" />
          <link rel="mask-icon" href="/safari-pinned-tab.svg" color="#000000" />
          <meta name="msapplication-TileColor" content="#00aba9" />
          <meta name="theme-color" content="#141449" />

          {children}
        </Helmet>
      )}
    </Location>
  );
};

export default SEO
