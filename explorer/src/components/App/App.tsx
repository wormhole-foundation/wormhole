import { ReactNode } from 'react';

import "@fontsource/sora"

import TimeAgo from 'javascript-time-ago'

import en from 'javascript-time-ago/locale/en'

TimeAgo.addDefaultLocale(en)

import './App.less'

/**
 * This component exists to provide a reusable application wrapper for use in Gatsby API's, testing, etc.
 */
const App = ({ element }: { element: ReactNode }) => {
  return element;
};

export default App;
