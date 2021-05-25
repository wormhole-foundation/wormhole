import React from 'react';

import { FormattedMessage } from 'gatsby-plugin-intl';

import { Button } from 'antd';

export default {
  title: 'Button',
};

export const Link = () => (
  <Button type="link">
    <FormattedMessage id="homepage.title" />
  </Button>
);

export const Primary = () => (
  <Button type="primary" >
    <FormattedMessage id="homepage.title" />
  </Button>
);

export const Large = () => (
  <Button type="primary" size="large">
    <FormattedMessage id="homepage.title" />
  </Button>
);

export const Loading = () => (
  <Button type="primary" loading>
    <FormattedMessage id="homepage.title" />
  </Button>
);

export const Outline = () => (
  <Button type="ghost">
    <FormattedMessage id="homepage.title" />
  </Button>
);

export const Danger = () => (
  <Button danger>
    <FormattedMessage id="homepage.title" />
  </Button>
);
