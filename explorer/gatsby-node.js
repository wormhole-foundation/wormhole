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

  const wasmExtensionRegExp = /\.wasm$/;

  actions.setWebpackConfig({
    module: {
      rules: [
        {
          test: wasmExtensionRegExp,
          include: /node_modules\/(bridge|token-bridge|nft)/,
          use: ['wasm-loader'],
          type: "javascript/auto"
        }
      ]
    }
  });

  if (stage === 'build-html') {
    // exclude wasm from SSR
    actions.setWebpackConfig({
      externals: getConfig().externals.concat(function (context, request, callback) {
        const regex = wasmExtensionRegExp;
        // exclude wasm from being bundled in SSR html, it will be loaded async at runtime.
        if (regex.test(request)) {
          return callback(null, 'commonjs ' + request); // use commonjs for wasm modules
        }
        const bridge = new RegExp('/wormhole-sdk/')
        if (bridge.test(request)) {
          return callback(null, 'commonjs ' + request);
        }
        callback();
      })
    });
  }

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
