# wormhole explorer

A web app built with:
- [GatsbyJS](https://www.gatsbyjs.com/)
- [gatsby-plugin-intl](https://www.gatsbyjs.com/plugins/gatsby-plugin-intl/)
- [Typescript](https://www.typescriptlang.org/)
- [Ant Design](https://ant.design/)


## Notable files

- Supported Languages - add/remove supported languages here [./src/utils/i18n/supportedLanguages.js](../src/utils/i18n/supportedLanguages.js)
- Multilangual copy [./src/locales](./src/locales)
- Top level pages & client side routes. Adding a file here creates a @reach-router route [./src/pages](./src/pages)
- SEO config, inherited by all pages [./src/components/SEO/SEO.tsx](./src/components/SEO/SEO.tsx)
- Main layout HOC, contains top-menu nav and footer [./src/components/Layout/DefaultLayout.tsx](./src/components/Layout/DefaultLayout.tsx)
- Gatsby plugins [./gatsby-config.js](./gatsby-config.js)
- Ant Design theme variables, overrides Antd defaults [./src/AntdTheme.js](./src/AntdTheme.js)


## Repo setup

Installing dependencies with npm:

    npm install

Create a `.env` file for your development environment, from the `.env.sample`:

    cp .env.sample .env.development

## Developing

Start the development server with the npm script:

    npm run dev

Then open the web app in your browser at [http://localhost:8000](http://localhost:8000)

## Debugging
### NodeJs debugging with VSCode

You can debug the Gatsby dev server or build process using VSCode's debbuger. Checkout [.vscode/launch.json](./.vscode/launch.json) to see the NodeJS debugging options.

These debugger configs will let you set breakpoints in the Gatsby node programs ([./gatsby-config.js](./gatsby-config.js), [./gatsby-node.js](./gatsby-node.js)) to debug webpack, Gatsby plugins, etc.

### Browser debugging with VSCode

With the [Debugger for Chrome](https://marketplace.visualstudio.com/items?itemName=msjsdiag.debugger-for-chrome) extension installed, you can inspect the web app and set broswer breakpoints from VSCode. With the dev server (`npm run dev`) running, select & run [Debug in Chrome](./.vscode/launch.json#L12) from the debugger pane.

## Storybook component rendering

[Storybook](https://storybook.js.org/) can render components with sytles and locales, for UI component development.

Run Storybook with:

    npm run storybook

See [./src/components/Button/button.stories.tsx](./src/components/Button/button.stories.tsx)

## eslint linting & formatting

Check linting:

    npm run lint

Fix linting errors:

    npm run format

## Ant Design Theming

Ant Design [default less variables](https://github.com/ant-design/ant-design/blob/master/components/style/themes/default.less) can be overridden in [./src/AntdTheme.js](./src/AntdTheme.js), which is used in [./gatsby-config.js#L51](./gatsby-config.js#L51).


## Programmatic Translations

Translations can be made for the supported languages ([./src/utils/i18n/supportedLanguages.js](../src/utils/i18n/supportedLanguages.js)). The English language definition file ([./src/locales/en.json](./src/locales/en.json)) will be read and used as the source, using either DeepL or Google Translate to supply the translations.

### Translating with DeepL

Pass your DeepL Pro api key to the npm script:

    npm run translate:deepl -- your-DeepL-Pro-api-key-here

### Translating with Google Translate

With your Service Account [credentials](https://github.com/leolabs/json-autotranslate#google-translate) saved to a file locally, pass the path to the .json file to the npm script:

    npm run translate:google -- ./your-GCP-service-account.json

### Protobuf generation

You'll need to generate proto files by running:

    npm run generate-protos

### WASM generation

To generate WASM files run:

    npm run generate-wasm
