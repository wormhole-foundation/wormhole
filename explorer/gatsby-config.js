import dotenv from 'dotenv';

import { getThemeVariables } from 'antd/dist/theme';
import supportedLanguages from './src/utils/i18n/supportedLanguages';
import antdThemeOverrides from './src/AntdTheme';

dotenv.config({
  path: `.env.${process.env.NODE_ENV}`,
});

const languages = supportedLanguages.map(language => language.languageTag);
const plugins = [
  'gatsby-plugin-react-helmet',
  'gatsby-plugin-typescript',
  'gatsby-plugin-remove-serviceworker',
  'gatsby-plugin-svgr',
  {
    resolve: 'gatsby-plugin-intl',
    options: {
      path: `${__dirname}/src/locales`,
      languages,
      defaultLanguage: 'en',
      redirect: true,
    },
  },
  {
    resolve: 'gatsby-plugin-antd',
    options: {
      style: true,
    },
  },
  {
    resolve: `gatsby-plugin-less`,
    options: {
      lessOptions: {
        javascriptEnabled: true,
        modifyVars: {
          ...getThemeVariables({
            dark: true, // Enable dark mode
            compact: true, // Enable compact mode,
          }),
          ...antdThemeOverrides,
        },
      },
    },
  },
  {
    resolve: 'gatsby-plugin-robots-txt',
    options: {
      host: process.env.GATSBY_SITE_URL,
      sitemap: `${process.env.GATSBY_SITE_URL}/sitemap.xml`,
      env: {
        development: {
          policy: [{ userAgent: '*', disallow: ['/'] }]
        },
        production: {
          policy: [{ userAgent: '*', allow: '/' }]
        }
      }
    }
  },
  {
    resolve: "gatsby-plugin-sitemap",
    options: {
      serialize: ({ site, allSitePage }) => {
        // filter out pages that do not include a locale, along with locale specific 404 pages.
        const edges = allSitePage.edges.filter(page => languages.some(lang => page.node.path.includes(lang)) && !page.node.path.includes('404'))
        // return sitemap entries
        return edges.map(page => {
          return {
            url: `${site.siteMetadata.siteUrl}${page.node.path}`,
            // changefreq: `daily`,
            // priority: 0.7,
            // lastmod: modifiedGmt,
          }
        })
      },
      exclude: [
        process.env.ENABLE_NETWORK_PAGE !== 'true' ? '/*/network/' : '/',
        process.env.ENABLE_EXPLORER_PAGE !== 'true' ? '/*/explorer/' : '/',
      ]
    },
  },
  {
    resolve: `gatsby-plugin-google-gtag`,
    options: {
      trackingIds: [String(process.env.GATSBY_GA_TAG)],
    },
  },
];

// Bundle analyzer, dev only
if (process.env.ENABLE_BUNDLE_ANALYZER === '1') {
  plugins.push('gatsby-plugin-webpack-bundle-analyser-v2');
}

const siteMetadata = {
  siteUrl: process.env.GATSBY_SITE_URL,
}

export { plugins, siteMetadata };
