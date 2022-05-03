/**
 * Implement Gatsby's Node APIs in this file.
 *
 * See: https://www.gatsbyjs.com/docs/node-apis/
 */

const webpack = require("webpack");

exports.onCreateWebpackConfig = function addPathMapping({
  stage,
  actions,
  getConfig,
}) {
  actions.setWebpackConfig({
    experiments: {
      asyncWebAssembly: true,
    },
    plugins: [
      // Work around for Buffer is undefined:
      // https://github.com/webpack/changelog-v5/issues/10
      new webpack.ProvidePlugin({
        Buffer: ["buffer", "Buffer"],
      }),
    ],
    resolve: {
      fallback: {
        buffer: require.resolve("buffer"),
        fs: false,
        path: false,
        stream: require.resolve("stream-browserify"),
        crypto: require.resolve("crypto-browserify"),
      },
    },
  });
};

exports.createPages = ({ actions }) => {
  const { createRedirect } = actions;
  createRedirect({
    fromPath: "/en/",
    toPath: "/",
    isPermanent: true,
  });
  createRedirect({
    fromPath: "/en/about/",
    toPath: "/buidl/",
    isPermanent: true,
  });
  createRedirect({
    fromPath: "/en/network/",
    toPath: "/network/",
    isPermanent: true,
  });
  createRedirect({
    fromPath: "/en/explorer/",
    toPath: "/explorer/",
    isPermanent: true,
  });
};
