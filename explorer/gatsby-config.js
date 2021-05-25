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
];

// Bundle analyzer, dev only
if (process.env.ENABLE_BUNDLE_ANALYZER === '1') {
  plugins.push('gatsby-plugin-webpack-bundle-analyser-v2');
}

export { plugins };
