import React from 'react';
import dotenv from 'dotenv';
import { App } from './src/components/App';

import supportedLanguages from './src/utils/i18n/supportedLanguages';

dotenv.config({
  path: `.env.${process.env.NODE_ENV}`,
});

// Duplicated in gatsby-browser.js for client side rendering
export const wrapRootElement = props => <App {...props} />;

export const onRenderBody = ({ pathname, setHeadComponents }) => {
  // Create a string to allow a regex replacement for SEO hreflang links: https://support.google.com/webmasters/answer/189077?hl=en
  const supportedLocaleRegexGroups = supportedLanguages
    .map(language => language.languageTag)
    .join('|');

  const hrefLangLinks = [
    ...supportedLanguages.map(language => {
      // Must be a fully qualified site URL
      const href = `${process.env.GATSBY_SITE_URL}/${language.languageTag +
        pathname.replace(new RegExp(`^/(${supportedLocaleRegexGroups})`), '')}`;

      return (
        <link
          hrefLang={language.languageTag}
          href={href}
          key={`href-lang-${language.languageTag}`}
        />
      );
    }),
  ];

  setHeadComponents(hrefLangLinks);
};
