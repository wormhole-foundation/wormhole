module.exports = {
  env: {
    browser: true,
    commonjs: true,
    es2021: true,
    mocha: true
  },
  extends: [
    'standard'
  ],
  parser: '@typescript-eslint/parser',
  parserOptions: {
    ecmaVersion: 12
  },
  plugins: [
    '@typescript-eslint'
  ],
  rules: {
  }
}
