const config = {
  // all ts or tsx files need to be transformed using jest-preprocess.js
  // Set up Babel config in jest-preprocess.js
  transform: {
    // Allow tests in TypeScript using the .ts or .tsx
    '^.+\\.[jt]sx?$': '<rootDir>/test-utils/jest-preprocess.js',
  },
  testRegex: '(/__tests__/.*(test|spec))\\.([tj]sx?)$',
  moduleDirectories: ['node_modules', __dirname],
  // Works like webpack rules. Tells Jest how to handle imports
  moduleNameMapper: {
    // Mock static file imports and assets which Jest can’t handle
    // stylesheets use the package identity-obj-proxy
    '.+\\.(css|styl|less|sass|scss)$': 'identity-obj-proxy',
    // Manual mock other files using file-mock.js
    '.+\\.(jpg|jpeg|png|gif|eot|otf|webp|ttf|woff|woff2|mp4|webm|wav|mp3|m4a|aac|oga)$':
      '<rootDir>/__mocks__/file-mock.js',
    // Mock SVG
    '\\.svg': '<rootDir>/__mocks__/svgr-mock.js',
    '^~/(.*)$': '<rootDir>/src/$1',
  },
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  testPathIgnorePatterns: ['node_modules', '.cache', 'public'],
  // Gatsby includes un-transpiled ES6 code. Exclude the gatsby module.
  transformIgnorePatterns: ['node_modules/(?!(gatsby)/)'],
  globals: {
    __PATH_PREFIX__: '',
  },
  collectCoverageFrom: [
    'src/**/*.{js,jsx,ts,tsx}',
    '!<rootDir>/src/**/*.stories.{ts,tsx}',
    '!<rootDir>/src/**/__tests__/**/*',
    '!<rootDir>/src/components/**/index.ts',
    '!<rootDir>/node_modules/',
    '!<rootDir>/test-utils/',
  ],
  testURL: 'http://localhost',
  setupFiles: ['<rootDir>/test-utils/loadershim.js', 'jest-localstorage-mock'],
  setupFilesAfterEnv: ['<rootDir>/test-utils/setup-test-env.ts'],
};

module.exports = config;
