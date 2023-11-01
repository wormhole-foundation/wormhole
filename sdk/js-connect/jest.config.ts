import type { JestConfigWithTsJest } from "ts-jest";

const jestConfig: JestConfigWithTsJest = {
    verbose: true,
    extensionsToTreatAsEsm: ['.ts'],
    testPathIgnorePatterns: [
        "utils",
        "eth-integration.ts",
        "solana-integration.ts",
        "algorand-integration.ts",
        "aptos-integration.ts",
        "near-integration.ts",
        "sui-integration.ts",
        "terra-integration.ts",
        "terra2-integration.ts",
    ],
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