import React from 'react';

import { App } from './src/components/App';

// Duplicated in gatsby-ssr.js for server side rendering during the build
export const wrapRootElement = props => <App {...props} />;
