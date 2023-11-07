import type { JestConfigWithTsJest } from "ts-jest";

const jestConfig: JestConfigWithTsJest = {
    verbose: true,
    extensionsToTreatAsEsm: ['.ts'],
    testPathIgnorePatterns: [],
    transformIgnorePatterns: [
        "node_modules/*"
    ],
    transform: {
        "^.+\\.ts?$": [
            "ts-jest",
            {
                useESM: true,
            },
        ],
    },
    moduleNameMapper: {
        "^(\\.{1,2}/.*)\\.js$": "$1",
    },
};

export default jestConfig;