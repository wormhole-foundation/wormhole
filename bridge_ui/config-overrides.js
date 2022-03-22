const { ProvidePlugin } = require("webpack");

module.exports = function override(config, env) {
  return {
    ...config,
    module: {
      ...config.module,
      rules: [
        ...config.module.rules,
        {
          test: /\.js$/,
          enforce: "pre",
          use: ["source-map-loader"],
        },
        {
          test: /\.wasm$/,
          type: "webassembly/async",
        },
      ],
    },
    plugins: [
      ...config.plugins,
      new ProvidePlugin({
        Buffer: ["buffer", "Buffer"],
        process: "process/browser",
      }),
    ],
    resolve: {
      ...config.resolve,
      fallback: {
        assert: "assert",
        buffer: "buffer",
        console: "console-browserify",
        constants: "constants-browserify",
        crypto: "crypto-browserify",
        domain: "domain-browser",
        events: "events",
        fs: false,
        http: "stream-http",
        https: "https-browserify",
        os: "os-browserify/browser",
        path: "path-browserify",
        punycode: "punycode",
        process: "process/browser",
        querystring: "querystring-es3",
        stream: "stream-browserify",
        _stream_duplex: "readable-stream/duplex",
        _stream_passthrough: "readable-stream/passthrough",
        _stream_readable: "readable-stream/readable",
        _stream_transform: "readable-stream/transform",
        _stream_writable: "readable-stream/writable",
        string_decoder: "string_decoder",
        sys: "util",
        timers: "timers-browserify",
        tty: "tty-browserify",
        url: "url",
        util: "util",
        vm: "vm-browserify",
        zlib: "browserify-zlib",
      },
    },
    experiments: {
      asyncWebAssembly: true,
    },
    ignoreWarnings: [/Failed to parse source map/],
  };
};
