import React from 'react';
import { IntlContextProvider } from 'gatsby-plugin-intl/intl-context';

import { locales, messages } from '../preview';

const intlConfig = {
  language: 'en',
  languages: locales,
  messages: messages,
  originalPath: '/',
  redirect: true,
  routed: true,
};

const GatsbyIntlProvider = storyFn => (
  <IntlContextProvider value={intlConfig}>{storyFn()}</IntlContextProvider>
);

export default GatsbyIntlProvider;
