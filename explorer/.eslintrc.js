const {
  rules: baseImportsRules,
} = require('eslint-config-airbnb-base/rules/imports');

module.exports = {
  globals: {
    // Gatsby Config
    __PATH_PREFIX__: true,
  },
  env: {
    // Allow `window` global
    browser: true,
  },
  // Global ESLint Settings
  // =================================
  settings: {
    'import/resolver': {
      node: {
        paths: ['./', 'src'],
        extensions: ['.js', '.jsx', '.ts', '.tsx', 'json'],
      },
      // Resolve Aliases
      // =================================
      alias: {
        map: [
          ['~', './src'],
          ['@theme/styled', './src/styled'],
        ],
        extensions: ['.js', '.jsx', '.ts', '.tsx', 'json', '.d.ts'],
      },
    },
  },

  // ===========================================
  // Set up ESLint for .js / .jsx files
  // ===========================================
  // .js / .jsx uses babel-eslint
  parser: 'babel-eslint',

  // Plugins
  // =================================
  plugins: ['no-only-tests'],

  // Extend Other Configs
  // =================================
  extends: [
    'eslint:recommended',
    'airbnb',
    // Disable rules that conflict with Prettier
    // !!! Prettier must be last to override other configs
    'prettier/react',
    'plugin:prettier/recommended',
  ],
  rules: {
    // This project uses TS. Disable prop-types check
    'react/prop-types': 0,
    // Allow snake_case due to inconsistent APIs
    camelcase: 0,
    // Prevents exclusion of tests from passing lint check
    'no-only-tests/no-only-tests': 'error',
    // dont enforce semicolon usage either way
    semi: 0
  },

  // https://eslint.org/docs/user-guide/configuring#report-unused-eslint-disable-comments
  reportUnusedDisableDirectives: true,

  // =================================
  // Overrides for Specific Files
  // =================================
  overrides: [
    // =================================
    // TypeScript Files
    // =================================
    {
      files: ['**/*.{ts,tsx}'],
      // allow ESLint to understand TypeScript syntax
      // https://github.com/iamturns/eslint-config-airbnb-typescript/blob/master/lib/shared.js#L10
      parserOptions: {
        // Lint with Type Information
        // https://github.com/typescript-eslint/typescript-eslint/blob/master/docs/getting-started/linting/TYPED_LINTING.md
        tsconfigRootDir: __dirname,
        project: './tsconfig.json',
      },

      extends: [
        // ESLint's inbuilt 'recommended' config
        'eslint:recommended',
        // Disables rules from the 'eslint:recommended' that are already covered by TypeScript's typechecker
        'plugin:@typescript-eslint/eslint-recommended',
        // Turns on rules from @typescript-eslint/eslint-plugin
        'plugin:@typescript-eslint/recommended',
        // Lint with Type Information
        // https://github.com/typescript-eslint/typescript-eslint/blob/master/docs/getting-started/linting/TYPED_LINTING.md
        'plugin:@typescript-eslint/recommended-requiring-type-checking',
        'airbnb-typescript',
        // Disable rules that conflict with Prettier
        // !!! Prettier must be last to override other configs
        'prettier/react',
        'prettier/@typescript-eslint',
        'plugin:prettier/recommended',
      ],
      rules: {
        // This project uses TS. Disable prop-types check
        'react/prop-types': 'off',
        // Allow snake_case due to inconsistent APIs
        '@typescript-eslint/camelcase': 0,
        // Makes no sense to allow type inferrence for expression parameters, but require typing the response
        '@typescript-eslint/explicit-function-return-type': 0,
        // Reduce props spreading rule to a warning, not an error
        'react/jsx-props-no-spreading': 1,
        'no-restricted-imports': [
          'warn',
          {
            paths: [
            ],
          },
        ],
      },
    },
    // =================================
    // index.ts Files (Re-exporting a directory's files)
    // =================================
    {
      files: ['**/index.{js,ts,tsx}'],
      rules: {
        // Allow named exports in a directory's index files
        'import/prefer-default-export': 0,
      },
    },
    // =================================
    // Gatsby Files
    // =================================
    {
      files: ['**/**/gatsby-*.js'],
      rules: {
        'no-console': 0,
        // Allow import devDependencies in Gatsby files.
        'import/no-extraneous-dependencies': [
          2,
          {
            devDependencies: true,
            // Tells ESLint where the path to the folder containing package.json is for nested files like /plugin/**/gatsby-*.js
            packageDir: './',
          },
        ],
        'react/no-danger': 0,
        'react/jsx-props-no-spreading': 0,
        // Allow 'jsx' in .js files
        'react/jsx-filename-extension': [1, { extensions: ['.js', '.jsx'] }],
        'import/prefer-default-export': 0,
        // Append 'ts' and 'tsx' when importing files from a folder/index.ts
        'import/extensions': [
          baseImportsRules['import/extensions'][0],
          baseImportsRules['import/extensions'][1],
          {
            ...baseImportsRules['import/extensions'][2],
            ts: 'never',
            tsx: 'never',
          },
        ],
      },
    },
    // =================================
    // Test Files
    // =================================
    {
      files: ['**/test-utils/*.{js,ts,tsx}', '**/**/*.test.{js,ts,tsx}'],
      // Allow `jest` global
      extends: ['plugin:jest/recommended'],
      rules: {
        // Allow import devDependencies in tests
        'import/no-extraneous-dependencies': 0,
        'react/jsx-props-no-spreading': 0,
        'jsx-a11y/alt-text': 0,
      },
    },
    // =================================
    // Storybook Files
    // =================================
    {
      files: ['**/*.stories.{js,ts,tsx}'],
      rules: {
        // Allow import devDependencies in stories
        'import/no-extraneous-dependencies': 0,
        'react/jsx-props-no-spreading': 0,
        'jsx-a11y/alt-text': 0,
      },
    },
  ],
};
