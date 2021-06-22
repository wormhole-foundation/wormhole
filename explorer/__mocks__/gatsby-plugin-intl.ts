import React from 'react';

const gatsbyPluginIntl = jest.requireActual('gatsby-plugin-intl');

module.exports = {
  ...gatsbyPluginIntl,
  Link: jest
    .fn()
    .mockImplementation(
      ({
        activeClassName,
        activeStyle,
        getProps,
        innerRef,
        partiallyActive,
        ref,
        replace,
        to,
        language,
        ...rest
      }) =>
        React.createElement('a', {
          ...rest,
          href: to,
        })
    ),
};
