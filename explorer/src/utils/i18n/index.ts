type Language = {
    label: string
    languageTag: string
}

const supportedLanguages = require('./supportedLanguages') as Language[]

export { supportedLanguages };
