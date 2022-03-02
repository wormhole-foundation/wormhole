import React from 'react'
import { Helmet, HelmetProps } from 'react-helmet'
import { useStaticQuery, graphql } from 'gatsby'

type Meta = ConcatArray<PropertyMetaObj | NameMetaObj>

type PropertyMetaObj = {
    property: string
    content: string
}

type NameMetaObj = {
    name: string
    content: string
}

type SEOProps = {
    description?: string
    meta?: Meta
    title?: string
    pathname?: string
} & HelmetProps

export function SEO({
    children,
    description = '',
    meta = [],
    title,
    pathname,
    ...props
}: SEOProps) {
    const { site } = useStaticQuery<QueryTypes>(SEOStaticQuery)
    const siteUrl = site.siteMetadata.siteUrl
    const defaultTitle = site.siteMetadata?.defaultTitle
    const twitterUsername = `@wormholecrypto`
    const socialImage = "/wormhole.png"
    const socialImageWidth = '800'
    const socialImageHeight = '400'
    const image = `${siteUrl}${socialImage}`

    const metaDescription = description || 'The best of blockchains'
    const canonical = pathname ? `${siteUrl}${pathname}` : null

    // for social sharing we want a little more context than just title,
    // make a string like "Apps | Wormhole"
    const socialTitle = title ? `${title} | ${defaultTitle}` : title

    return (
        <Helmet
            {...props}
            htmlAttributes={{
                lang: 'en',
            }}
            title={title || defaultTitle}
            link={[
                {
                    rel: `apple-touch-icon`,
                    href: `${siteUrl}/apple-touch-icon.png`,
                    sizes: `180x180`,
                },
                {
                    rel: `icon`,
                    href: `${siteUrl}/favicon-32x32.png`,
                    sizes: `32x32`,
                    type: `image/png`,
                },
                {
                    rel: `icon`,
                    href: `${siteUrl}/favicon-16x16.png`,
                    sizes: `16x16`,
                    type: `image/png`,
                },
                {
                    rel: `manifest`,
                    href: `${siteUrl}/site.webmanifest`,
                },
                {
                    rel: `mask-icon`,
                    href: `${siteUrl}/safari-pinned-tab.svg`,
                    color: `#5bbad5`,
                },

                canonical
                    ? {
                        rel: 'canonical',
                        href: canonical,
                    }
                    : {},
            ]}
            meta={[
                {
                    name: `description`,
                    content: metaDescription,
                },
                {
                    name: `msapplication-TileColor`,
                    content: `#603cba"`,
                },
                {
                    name: `theme-color`,
                    content: `#ffffff`,
                },
                // opengraph metadata
                {
                    property: `og:title`,
                    content: socialTitle,
                },
                {
                    property: `og:description`,
                    content: metaDescription,
                },
                {
                    property: `og:site_name`,
                    content: defaultTitle, // "Wormhole" for all pages
                },
                {
                    property: `og:type`,
                    content: `website`,
                },
                canonical
                    ? {
                        property: `og:url`,
                        content: canonical,
                    }
                    : {},
                {
                    property: 'og:image',
                    content: image,
                },
                {
                    property: 'og:image:secure_url',
                    content: image,
                },
                {
                    property: `og:image:type`,
                    content: `image/png`,
                },
                {
                    property: `og:image:width`,
                    content: socialImageWidth,
                },
                {
                    property: `og:image:height`,
                    content: socialImageHeight,
                },
                {
                    property: `og:image:alt`,
                    content: `Wormhole logo`,
                },
                // twitter metadata
                {
                    name: `twitter:title`,
                    content: socialTitle,
                },
                {
                    name: `twitter:description`,
                    content: metaDescription,
                },
                {
                    name: `twitter:image`,
                    content: image,
                },
                {
                    name: `twitter:image:alt`,
                    content: `Wormhole logo`,
                },
                {
                    name: 'twitter:card',
                    content: 'summary_large_image',
                },
                {
                    name: `twitter:site`,
                    content: twitterUsername,
                },
                {
                    name: `twitter:creator`,
                    content: twitterUsername,
                },
            ]
                // metadata from props
                .concat(meta)}
        >
            
            {children}
        </Helmet>
    )
}

type QueryTypes = {
    site: {
        siteMetadata: {
            siteUrl: string
            defaultTitle: string
        }
    }
}

const SEOStaticQuery = graphql`
  query SEO {
    site {
      siteMetadata {
        siteUrl
        defaultTitle: title
      }
    }
  }
`
