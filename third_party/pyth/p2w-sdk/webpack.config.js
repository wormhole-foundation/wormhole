const path = require('path');

module.exports = {
  entry: './src/index.ts',
  experiments: {
    asyncWebAssembly: true,
  },
  mode: 'development',
  target: 'node',
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: 'ts-loader',
        exclude: /node_modules/,
      },
    ],
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.js'],
  },
  output: {
    filename: 'test.js',
    path: path.resolve(__dirname, 'lib'),
  },
};
