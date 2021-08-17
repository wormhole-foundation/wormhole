import path from 'path';

import dotenv from 'dotenv';

dotenv.config({
  path: `.env.${process.env.NODE_ENV}`,
});

export const onCreateWebpackConfig = function addPathMapping({
  stage,
  actions,
  getConfig,
}) {
  actions.setWebpackConfig({
    resolve: {
      alias: {
        '~': path.resolve(__dirname, 'src'),
      },
    },
  });

  // TODO: make sure this only runs in dev
  actions.setWebpackConfig({
    devtool: 'eval-source-map',
  });

  actions.setWebpackConfig({
    module: {
      rules: [
        {
          test: /\.wasm$/,
          use: [
            'wasm-loader'
          ],
          type: "javascript/auto"
        }
      ]
    }
  });
  const config = getConfig();
  config.resolve.extensions.push(".wasm");
  actions.replaceWebpackConfig(config);

  // Attempt to improve webpack vender code splitting
  if (stage === 'build-javascript') {
    const config = getConfig();

    config.optimization.splitChunks.cacheGroups = {
      ...config.optimization.splitChunks.cacheGroups,
      vendors: {
        test: /[\\/]node_modules[\\/]/,
        enforce: true,
        chunks: 'all',
        priority: 1,
      },
    };

    // Ensure Gatsby does not do any css code splitting
    config.optimization.splitChunks.cacheGroups.styles.priority = 10;

    actions.replaceWebpackConfig(config);
  }
};
